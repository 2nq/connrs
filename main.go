package main

import (
	"fmt"
	"os"

	"connrs/internal/collector"
	"connrs/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	poller, err := collector.NewDefault()
	if err != nil {
		fmt.Fprintln(os.Stderr, "connrs:", err)
		os.Exit(1)
	}

	program := tea.NewProgram(
		ui.NewModel(poller),
		tea.WithAltScreen(),
	)

	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "connrs:", err)
		os.Exit(1)
	}
}
