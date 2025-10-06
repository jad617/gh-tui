package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type loadedRequestedMsg []list.Item

type loadedMineMsg []list.Item

type loadedRunsMsg []list.Item

type detailsMsg string

type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchRequested(m), fetchMine(m), fetchRuns(m))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeTab = (m.activeTab + 1) % tab(len(m.lists))
			return m, m.loadDetailsForSelection()
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + tab(len(m.lists))) % tab(len(m.lists))
			return m, m.loadDetailsForSelection()
		case "ctrl+r", "r":
			return m, tea.Batch(fetchRequested(m), fetchMine(m), fetchRuns(m), m.loadDetailsForSelection())
		case "o", "enter":
			cur := m.lists[m.activeTab].SelectedItem()
			if cur == nil {
				break
			}
			if op, ok := cur.(openable); ok {
				if u := op.url(); u != "" && isURL(u) {
					_ = openURL(u)
				}
			}
		case "d":
			m.ShowDetails = !m.ShowDetails
			m.resize()
			return m, m.loadDetailsForSelection()
		case "t":
			if m.activeTab == tabActions {
				m.Tailing = !m.Tailing
				if m.Tailing {
					return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
				}
			}
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()

	case loadedRequestedMsg:
		m.lists[0].SetItems([]list.Item(msg))
		m.StatusMsg = fmt.Sprintf("Loaded %d requested reviews", len(msg))
		return m, m.loadDetailsForSelection()

	case loadedMineMsg:
		m.lists[1].SetItems([]list.Item(msg))
		m.StatusMsg = fmt.Sprintf("Loaded %d of your PRs", len(msg))
		return m, m.loadDetailsForSelection()

	case loadedRunsMsg:
		m.lists[2].SetItems([]list.Item(msg))
		m.StatusMsg = fmt.Sprintf("Loaded %d workflow runs for %s/%s", len(msg), m.Owner, m.Repo)
		return m, m.loadDetailsForSelection()

	case detailsMsg:
		m.Vp.SetContent(string(msg))

	case errMsg:
		m.StatusMsg = fmt.Sprintf("Error: %v", msg)

	case tickMsg:
		if m.Tailing && m.activeTab == tabActions {
			return m, tea.Batch(fetchRuns(m), m.loadDetailsForSelection(),
				tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }))
		}
	}

	var cmd tea.Cmd
	m.lists[m.activeTab], cmd = m.lists[m.activeTab].Update(msg)
	return m, tea.Batch(cmd, m.loadDetailsForSelection())
}

// --- FIXED RESIZE ---
func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Available height for the main body area (between tab + help + status)
	bodyH := m.bodyHeight()
	innerH := bodyH - 2 // minus box borders
	if innerH < 1 {
		innerH = 1
	}

	if m.ShowDetails {
		// Two boxes
		leftBoxW := m.leftBoxWidth()
		rightBoxW := m.rightBoxWidth()

		leftInnerW := leftBoxW - 4 // borders + padding
		if leftInnerW < 10 {
			leftInnerW = 10
		}
		rightInnerW := rightBoxW - 4
		if rightInnerW < 10 {
			rightInnerW = 10
		}

		for i := range m.lists {
			m.lists[i].SetSize(leftInnerW, innerH)
		}
		m.Vp.Width = rightInnerW
		m.Vp.Height = innerH

	} else {
		// Single box (list only)
		boxW := m.leftBoxWidth()
		innerW := boxW - 4
		if innerW < 10 {
			innerW = 10
		}
		for i := range m.lists {
			m.lists[i].SetSize(innerW, innerH)
		}
		m.Vp.Width, m.Vp.Height = 0, 0
	}
}

// --- DETAILS LOADER ---
func (m Model) loadDetailsForSelection() tea.Cmd {
	if !m.ShowDetails {
		return nil
	}
	cur := m.lists[m.activeTab].SelectedItem()
	if cur == nil {
		return nil
	}

	switch m.activeTab {
	case tabRequested, tabMine:
		if p, ok := cur.(prItem); ok {
			return fetchPRDetails(m, p)
		}
	case tabActions:
		if r, ok := cur.(runItem); ok {
			return fetchRunDetails(m, r)
		}
	}
	return nil
}
