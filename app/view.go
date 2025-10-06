package app

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ---------- Tabs (single row that always fits) ----------

func truncRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max == 1 {
		return string(r[:1])
	}
	return string(r[:max-1]) + "…"
}

func renderTabTitle(label string, active bool) string {
	if active {
		return activeTab.Copy().Underline(true).Render(label)
	}
	return tabStyle.Render(label)
}

func (m Model) renderTabRow() string {
	w := m.width
	if w < 10 {
		if w < 0 {
			w = 0
		}
		return strings.Repeat(" ", w)
	}
	labels := []string{"Requested Reviews", "My PRs", "Actions"}

	sep := " │ "
	sepW := lipgloss.Width(sep)
	avail := w - sepW*2
	if avail < 6 { // at least 2 cols per tab
		avail = 6
	}
	slot := avail / 3
	if slot < 2 {
		slot = 2
	}

	t0 := truncRunes(labels[0], slot)
	t1 := truncRunes(labels[1], slot)
	t2 := truncRunes(labels[2], slot)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderTabTitle(t0, int(m.activeTab) == 0),
		sep,
		renderTabTitle(t1, int(m.activeTab) == 1),
		sep,
		renderTabTitle(t2, int(m.activeTab) == 2),
	)

	// Hard fit to exactly w columns (no wrap)
	rowW := lipgloss.Width(row)
	switch {
	case rowW < w:
		row += strings.Repeat(" ", w-rowW)
	case rowW > w:
		diff := rowW - w
		r := []rune(row)
		if diff < len(r) {
			row = string(r[:len(r)-diff])
		} else {
			row = strings.Repeat(" ", w)
		}
	}
	return row
}

// ---------- Content clamping (width & height) ----------

// clampContent trims each line to <= width and caps total lines to <= height.
// If trimmed by height, the last visible line becomes "…".
func clampContent(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")

	// width clamp per line
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w > width {
			if width == 1 {
				lines[i] = "…"
			} else {
				lines[i] = truncRunes(l, width-1) + "…"
			}
		}
	}

	// height clamp
	if len(lines) > height {
		if height == 1 {
			return "…"
		}
		cut := lines[:height]
		cut[len(cut)-1] = "…"
		return strings.Join(cut, "\n")
	}
	return strings.Join(lines, "\n")
}

// Should we render the help line? Hide when the terminal is short to protect the tab row.
func showHelp(m Model) bool {
	return m.height >= 12
}

// ---------- View (boxes sized by INNER widths/heights) ----------

func (m Model) View() string {
	// 1) Tab row (1 line), fits exactly in m.width
	tabRow := m.renderTabRow()

	// 2) Box size math
	bodyTotalH := m.bodyHeight() // TOTAL rows for the pane row incl. borders
	innerH := bodyTotalH - 2     // content height inside a bordered box
	if innerH < 1 {
		innerH = 1
	}

	leftOuter := m.leftBoxWidth() // OUTER width of left box (borders+padding included)
	leftInner := leftOuter - 4    // content width (2 borders + 2 padding)
	if leftInner < 1 {
		leftInner = 1
	}

	// 3) Left content (strict width/height clamp)
	leftContent := clampContent(m.lists[m.activeTab].View(), leftInner, innerH)
	leftBox := boxStyle.Copy().
		Width(leftInner). // content width
		Height(innerH).   // content height
		Render(leftContent)

	// 4) Optional right details
	var body string
	if m.ShowDetails {
		rightOuter := m.rightBoxWidth()
		rightInner := rightOuter - 4
		if rightInner < 1 {
			rightInner = 1
		}
		rightContent := clampContent(m.Vp.View(), rightInner, innerH)
		rightBox := boxStyle.Copy().
			Width(rightInner).
			Height(innerH).
			Render(rightContent)

		// Join two inner-sized boxes; their OUTER totals will equal m.width
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	} else {
		body = leftBox
	}

	// 5) Footers (help is adaptive)
	var rows []string
	rows = append(rows, tabRow)
	rows = append(rows, body)
	if showHelp(m) {
		rows = append(rows, subtleStyle.Render("[Tab/Shift+Tab] switch  [j/k/Up/Down] move  [o/Enter] open  [d] details  [r] refresh  [t] tail actions  [q] quit"))
	}
	rows = append(rows, subtleStyle.Render(m.StatusMsg))

	screen := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// 6) FINAL SAFETY: clamp the ENTIRE screen to terminal width & height
	return finalizeScreen(screen, m.width, m.height)
}

// ---------- Final full-screen clamp ----------------------------------

// finalizeScreen ensures the full rendered screen fits exactly within width x height.
// We keep TOP lines (so the tabs never disappear), and trim each line to width.
func finalizeScreen(s string, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	lines := strings.Split(s, "\n")

	// Keep TOP 'height' lines (protect the tab row). If we cut, end with an ellipsis line.
	if len(lines) > height {
		if height == 1 {
			lines = []string{"…"}
		} else {
			lines = append(lines[:height-1], "…")
		}
	}

	// Trim each line to <= width (protect the right border region)
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w > width {
			if width == 1 {
				lines[i] = "…"
			} else {
				lines[i] = truncRunes(l, width-1) + "…"
			}
		} else if w < width {
			lines[i] = l + strings.Repeat(" ", width-w)
		}
	}

	return strings.Join(lines, "\n")
}
