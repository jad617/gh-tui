package app

// bodyHeight = total rows available for the pane row (including borders).
// We reserve lines for: tabs + help + status. When details are open,
// reserve ONE EXTRA safety row to ensure the tab line never scrolls off.
func (m Model) bodyHeight() int {
	// base reservation: tabs(1) + help(1) + status(1) = 3
	reserve := 3
	if m.ShowDetails {
		// extra cushion for double-box layout on some terminals
		reserve++
	}
	h := m.height - reserve
	if h < 3 {
		h = 3
	}
	return h
}

// leftBoxWidth returns the OUTER width (including 2 borders + 2 padding)
// so the two pane boxes sum to exactly the terminal width.
func (m Model) leftBoxWidth() int {
	total := m.width
	if total < 1 {
		return 1
	}
	if !m.ShowDetails {
		// Single pane spans full width.
		return total
	}
	const min = 24
	left := (total * 2) / 5
	if left < min {
		left = min
	}
	right := total - left
	if right < min {
		right = min
		left = total - right
	}
	// final clamp for odd/even off-by-one
	if left+right > total {
		left = total - right
		if left < min {
			left = min
		}
	}
	return left
}

// rightBoxWidth returns the OUTER width (including borders+padding).
func (m Model) rightBoxWidth() int {
	if !m.ShowDetails {
		return 0
	}
	total := m.width
	if total < 1 {
		return 0
	}
	const min = 24
	left := m.leftBoxWidth()
	right := total - left
	if right < min {
		right = min
		left = total - right
	}
	if left < min {
		left = min
		right = total - left
	}
	if left+right > total {
		right = total - left
	}
	if right < 0 {
		right = 0
	}
	return right
}
