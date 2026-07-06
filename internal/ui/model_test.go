package ui

import (
	"testing"
	"time"

	"connrs/internal/collector"

	tea "github.com/charmbracelet/bubbletea"
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

func filterTestSnapshot() collector.Snapshot {
	return collector.Snapshot{
		CapturedAt: time.Now(),
		Processes: []collector.ProcessSnapshot{
			{
				Name: "chrome.exe", PID: 100,
				Connections: []collector.ConnectionSnapshot{
					{Protocol: "TCP4", RemoteIP: "1.1.1.1", RemotePort: 443},
				},
			},
			{
				Name: "discord.exe", PID: 200,
				Connections: []collector.ConnectionSnapshot{
					{Protocol: "TCP4", RemoteIP: "162.159.128.233", RemotePort: 443},
					{Protocol: "UDP4", RemoteIP: "66.22.212.5", RemotePort: 50001},
				},
			},
		},
	}
}

func TestVisibleProcessesFiltersByNameRemoteIPAndPort(t *testing.T) {
	model := Model{snapshot: filterTestSnapshot()}

	cases := []struct {
		filter string
		want   []string
	}{
		{"disc", []string{"discord.exe"}},
		{"162.159", []string{"discord.exe"}},
		{"1.1.1.1", []string{"chrome.exe"}},
		{"443", []string{"chrome.exe", "discord.exe"}},
		{"nomatch", nil},
		{"", []string{"chrome.exe", "discord.exe"}},
	}

	for _, tc := range cases {
		model.filter = tc.filter
		visible := model.visibleProcesses()
		if got, want := len(visible), len(tc.want); got != want {
			t.Fatalf("filter %q: expected %d processes, got %d", tc.filter, want, got)
		}
		for index, name := range tc.want {
			if visible[index].Name != name {
				t.Fatalf("filter %q: expected %q at index %d, got %q", tc.filter, name, index, visible[index].Name)
			}
		}
	}
}

func TestVisibleProcessesSortModes(t *testing.T) {
	model := Model{
		snapshot: collector.Snapshot{
			Processes: []collector.ProcessSnapshot{
				{Name: "zebra.exe", PID: 1, SentBps: 900, Connections: make([]collector.ConnectionSnapshot, 1)},
				{Name: "alpha.exe", PID: 2, SentBps: 100, Connections: make([]collector.ConnectionSnapshot, 3)},
			},
		},
	}

	model.sort = sortByName
	if got := model.visibleProcesses()[0].Name; got != "alpha.exe" {
		t.Fatalf("expected name sort to put alpha.exe first, got %q", got)
	}

	model.sort = sortByConnections
	if got := model.visibleProcesses()[0].Name; got != "alpha.exe" {
		t.Fatalf("expected connection-count sort to put alpha.exe (3 conns) first, got %q", got)
	}

	model.sort = sortByBandwidth
	if got := model.visibleProcesses()[0].Name; got != "zebra.exe" {
		t.Fatalf("expected bandwidth sort to keep collector order (zebra.exe first), got %q", got)
	}
}

func TestPausePreventsScheduledPolls(t *testing.T) {
	model := NewModel(stubPoller{})
	model.paused = true

	model.Update(refreshTickMsg(time.Now()))
	if model.polling {
		t.Fatalf("expected no poll to start while paused")
	}

	model.paused = false
	model.Update(refreshTickMsg(time.Now()))
	if !model.polling {
		t.Fatalf("expected poll to start once unpaused")
	}
}

func TestFilterPromptTypingAndClearing(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 20
	model.snapshot = filterTestSnapshot()

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !model.filtering {
		t.Fatalf("expected / to open the filter prompt")
	}

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("disc")})
	if got, want := model.filter, "disc"; got != want {
		t.Fatalf("expected typed filter %q, got %q", want, got)
	}
	if got := len(model.visibleProcesses()); got != 1 {
		t.Fatalf("expected 1 visible process while typing, got %d", got)
	}

	// q must be typed into the prompt, not quit the program.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if got, want := model.filter, "discq"; got != want {
		t.Fatalf("expected q to append to filter, got %q", got)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.filtering {
		t.Fatalf("expected enter to close the filter prompt")
	}
	if got, want := model.filter, "disc"; got != want {
		t.Fatalf("expected enter to keep the filter, got %q", got)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.filter != "" {
		t.Fatalf("expected esc to clear the applied filter, got %q", model.filter)
	}
}

func TestSortCycleFollowsSelectedProcess(t *testing.T) {
	model := NewModel(nil)
	model.width = 90
	model.height = 20
	model.snapshot = collector.Snapshot{
		Processes: []collector.ProcessSnapshot{
			{Name: "zebra.exe", PID: 1},
			{Name: "alpha.exe", PID: 2},
		},
	}
	model.selected = 0 // zebra.exe under default bandwidth order

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if got, want := model.sort, sortByName; got != want {
		t.Fatalf("expected sort mode to cycle to name, got %v", got)
	}
	process, ok := model.currentProcess()
	if !ok || process.Name != "zebra.exe" {
		t.Fatalf("expected selection to follow zebra.exe across re-sort, got %+v ok=%v", process, ok)
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
