package ui

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"connrs/internal/collector"

	tea "github.com/charmbracelet/bubbletea"
)

type stubPoller struct{}

func (stubPoller) Poll(context.Context) (collector.Snapshot, error) {
	return collector.Snapshot{
			CapturedAt: time.Now(),
			IsAdmin:    true,
		},
		nil
}

func TestProgramStartsAndQuits(t *testing.T) {
	program := tea.NewProgram(
		NewModel(stubPoller{}),
		tea.WithInput(strings.NewReader("q")),
		tea.WithOutput(io.Discard),
	)

	if _, err := program.Run(); err != nil {
		t.Fatalf("expected program to exit cleanly, got error: %v", err)
	}
}
