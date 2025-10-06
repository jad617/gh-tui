package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"gh-tui/app"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
