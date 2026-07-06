package ui

import (
	"testing"
	"time"

	"connrs/internal/collector"
)

func testSnapshot(processCount int) collector.Snapshot {
	processes := make([]collector.ProcessSnapshot, 0, processCount)
	for index := range processCount {
		processes = append(processes, collector.ProcessSnapshot{
			Name: "process-" + itoa32(int32(index)),
			PID:  int32(1000 + index),
			Connections: []collector.ConnectionSnapshot{
				{
					Protocol:       "TCP4",
					State:          "ESTABLISHED",
					LocalIP:        "127.0.0.1",
					LocalPort:      uint32(5000 + index),
					RemoteIP:       "1.1.1.1",
					RemotePort:     443,
					BandwidthKnown: true,
				},
			},
		})
	}

	return collector.Snapshot{
		CapturedAt:       time.Now(),
		IsAdmin:          true,
		Processes:        processes,
		TotalConnections: len(processes),
		BandwidthTracked: len(processes),
	}
}

func TestApplySnapshotKeepsSelectionByProcessIdentity(t *testing.T) {
	model := Model{
		selected: 1,
		snapshot: collector.Snapshot{
			CapturedAt: time.Now(),
			Processes: []collector.ProcessSnapshot{
				{Name: "chrome.exe", PID: 100},
				{Name: "discord.exe", PID: 200},
			},
		},
	}

	model.applySnapshot(collector.Snapshot{
		CapturedAt: time.Now(),
		Processes: []collector.ProcessSnapshot{
			{Name: "discord.exe", PID: 200},
			{Name: "steam.exe", PID: 300},
		},
	})

	if got, want := model.selected, 0; got != want {
		t.Fatalf("expected selection to follow discord.exe/200 to index %d, got %d", want, got)
	}
}

func TestToggleExpandedCurrentUsesStableProcessKey(t *testing.T) {
	model := Model{
		snapshot: collector.Snapshot{
			CapturedAt: time.Now(),
			Processes: []collector.ProcessSnapshot{
				{Name: "discord.exe", PID: 200},
			},
		},
		expanded: map[string]bool{},
	}

	model.toggleExpandedCurrent()

	key := processKey(model.snapshot.Processes[0])
	if !model.expanded[key] {
		t.Fatalf("expected process %q to be expanded", key)
	}

	model.toggleExpandedCurrent()

	if model.expanded[key] {
		t.Fatalf("expected process %q to collapse on second toggle", key)
	}
}

func TestRefreshViewportClampsYOffsetWhenContentShrinks(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 20
	model.viewport.YOffset = 99
	model.snapshot = collector.Snapshot{}

	model.refreshViewport(false)

	if got, want := model.viewport.YOffset, 0; got != want {
		t.Fatalf("expected viewport y-offset %d after shrink, got %d", want, got)
	}
	if got := model.viewport.View(); got == "" {
		t.Fatalf("expected viewport to render empty-state content after shrink")
	}
}

func TestRenderContentLineRangesMatchRenderedLines(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 16
	model.snapshot = testSnapshot(8)

	model.refreshViewport(false)

	if got, want := len(model.lines), 8; got != want {
		t.Fatalf("expected %d line ranges, got %d", want, got)
	}
	for index, lr := range model.lines {
		if index == 0 {
			continue
		}
		if got, want := lr.start, model.lines[index-1].end+1; got != want {
			t.Fatalf("expected range %d to start at line %d, got %d", index, want, got)
		}
	}
	if got, want := model.lines[len(model.lines)-1].end, model.viewport.TotalLineCount()-1; got != want {
		t.Fatalf("expected last range to end at content line %d, got %d", want, got)
	}
}

func TestApplySnapshotPrunesExpandedStateForDepartedProcesses(t *testing.T) {
	model := Model{
		expanded: map[string]bool{
			"chrome.exe#100":  true,
			"discord.exe#200": true,
		},
	}

	model.applySnapshot(collector.Snapshot{
		CapturedAt: time.Now(),
		Processes: []collector.ProcessSnapshot{
			{Name: "discord.exe", PID: 200},
		},
	})

	if model.expanded["chrome.exe#100"] {
		t.Fatalf("expected expansion state for departed process to be pruned")
	}
	if !model.expanded["discord.exe#200"] {
		t.Fatalf("expected expansion state for surviving process to be kept")
	}
}

func TestMoveSelectionStopsAtListBounds(t *testing.T) {
	model := Model{
		selected: 1,
		snapshot: collector.Snapshot{
			Processes: []collector.ProcessSnapshot{
				{Name: "chrome.exe", PID: 100},
				{Name: "discord.exe", PID: 200},
			},
		},
	}

	model.moveSelection(1)
	if got, want := model.selected, 1; got != want {
		t.Fatalf("expected selection to stay at bottom index %d, got %d", want, got)
	}

	model.selected = 0
	model.moveSelection(-1)
	if got, want := model.selected, 0; got != want {
		t.Fatalf("expected selection to stay at top index %d, got %d", want, got)
	}
}

func TestRefreshViewportPreservesManualScrollOffset(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 16
	model.snapshot = testSnapshot(24)

	model.refreshViewport(false)
	model.viewport.SetYOffset(8)
	model.refreshViewport(false)

	if got, want := model.viewport.YOffset, 8; got != want {
		t.Fatalf("expected manual viewport offset %d to be preserved on refresh, got %d", want, got)
	}
}

func TestViewDoesNotPanicWhenViewportYOffsetIsInvalid(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 16
	model.snapshot = testSnapshot(24)
	model.refreshViewport(false)
	model.viewport.YOffset = model.viewport.TotalLineCount() + 2

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("expected View not to panic, got %v", recovered)
		}
	}()

	_ = model.View()
}
