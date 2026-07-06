package ui

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"connrs/internal/collector"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const refreshInterval = time.Second

type refreshTickMsg time.Time

type snapshotMsg struct {
	snapshot collector.Snapshot
	err      error
}

type lineRange struct {
	start int
	end   int
}

type sortMode int

const (
	sortByBandwidth sortMode = iota
	sortByName
	sortByConnections

	sortModeCount = 3
)

func (s sortMode) label() string {
	switch s {
	case sortByName:
		return "name"
	case sortByConnections:
		return "conns"
	default:
		return "bw"
	}
}

type Model struct {
	poller   collector.Poller
	snapshot collector.Snapshot
	selected int
	expanded map[string]bool

	filter    string
	filtering bool
	sort      sortMode
	paused    bool
	rdns      *rdnsCache

	width    int
	height   int
	loading  bool
	polling  bool
	showHelp bool

	lastErr error
	lines   []lineRange

	viewport viewport.Model
	help     help.Model
	keys     keyMap
	spinner  spinner.Model
	styles   styles
}

func NewModel(poller collector.Poller) *Model {
	spin := spinner.New()
	spin.Spinner = spinner.Spinner{
		Frames: []string{"-", "\\", "|", "/"},
		FPS:    time.Second / 10,
	}

	return &Model{
		poller:   poller,
		expanded: map[string]bool{},
		rdns:     newRDNSCache(),
		help:     help.New(),
		keys:     newKeyMap(),
		spinner:  spin,
		styles:   newStyles(),
		loading:  true,
		viewport: viewport.New(0, 0),
	}
}

func (m *Model) Init() tea.Cmd {
	m.polling = true
	return tea.Batch(
		pollCmd(m.poller),
		tickCmd(),
		m.spinner.Tick,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.refreshViewport(false)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case refreshTickMsg:
		cmds = append(cmds, tickCmd())
		if !m.paused && !m.polling {
			m.polling = true
			cmds = append(cmds, pollCmd(m.poller))
		}
	case snapshotMsg:
		m.polling = false
		m.loading = false
		if msg.err != nil {
			m.lastErr = msg.err
		} else {
			m.lastErr = nil
			m.applySnapshot(msg.snapshot)
		}
		m.refreshViewport(false)
	case tea.KeyMsg:
		if m.filtering {
			if cmd := m.handleFilterKey(msg); cmd != nil {
				return m, cmd
			}
			return m, tea.Batch(cmds...)
		}

		switch {
		case keyMatches(msg, m.keys.Quit):
			return m, tea.Quit
		case keyMatches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			m.refreshViewport(false)
		case keyMatches(msg, m.keys.Up):
			m.moveSelection(-1)
			m.refreshViewport(true)
		case keyMatches(msg, m.keys.Down):
			m.moveSelection(1)
			m.refreshViewport(true)
		case keyMatches(msg, m.keys.Expand):
			m.toggleExpandedCurrent()
			m.refreshViewport(true)
		case keyMatches(msg, m.keys.Filter):
			m.filtering = true
			m.refreshViewport(false)
		case keyMatches(msg, m.keys.ClearFilter):
			if m.filter != "" {
				key := m.currentKey()
				m.filter = ""
				m.restoreSelection(key)
				m.refreshViewport(true)
			}
		case keyMatches(msg, m.keys.Sort):
			key := m.currentKey()
			m.sort = (m.sort + 1) % sortModeCount
			m.restoreSelection(key)
			m.refreshViewport(true)
		case keyMatches(msg, m.keys.Pause):
			m.paused = !m.paused
			m.refreshViewport(false)
		case keyMatches(msg, m.keys.Refresh):
			if !m.polling {
				m.polling = true
				cmds = append(cmds, pollCmd(m.poller))
			}
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleFilterKey consumes all keys while the filter prompt is active, so
// navigation shortcuts like q/j/k can be typed as filter text.
func (m *Model) handleFilterKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		m.filtering = false
		m.filter = ""
		m.restoreSelection("")
	case tea.KeyEnter:
		m.filtering = false
	case tea.KeyBackspace:
		if m.filter != "" {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.selected = 0
		}
	case tea.KeySpace:
		m.filter += " "
		m.selected = 0
	case tea.KeyRunes:
		m.filter += string(msg.Runes)
		m.selected = 0
	}

	m.refreshViewport(true)
	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(at time.Time) tea.Msg {
		return refreshTickMsg(at)
	})
}

func pollCmd(poller collector.Poller) tea.Cmd {
	return func() tea.Msg {
		if poller == nil {
			return snapshotMsg{err: errors.New("collector is unavailable")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		snapshot, err := poller.Poll(ctx)
		return snapshotMsg{
			snapshot: snapshot,
			err:      err,
		}
	}
}

func processKey(process collector.ProcessSnapshot) string {
	return process.Name + "#" + itoa32(process.PID)
}

func (m *Model) applySnapshot(snapshot collector.Snapshot) {
	key := m.currentKey()
	m.snapshot = snapshot
	m.pruneExpanded()
	m.restoreSelection(key)
}

// visibleProcesses returns the processes the UI operates on: the snapshot
// list narrowed by the active filter and ordered by the active sort mode.
// Selection indexes refer to this list, not to the raw snapshot.
func (m *Model) visibleProcesses() []collector.ProcessSnapshot {
	processes := m.snapshot.Processes

	if query := strings.ToLower(strings.TrimSpace(m.filter)); query != "" {
		filtered := make([]collector.ProcessSnapshot, 0, len(processes))
		for _, process := range processes {
			if processMatches(process, query) {
				filtered = append(filtered, process)
			}
		}
		processes = filtered
	}

	switch m.sort {
	case sortByName:
		processes = slices.Clone(processes)
		slices.SortStableFunc(processes, func(left, right collector.ProcessSnapshot) int {
			if c := cmp.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)); c != 0 {
				return c
			}
			return cmp.Compare(left.PID, right.PID)
		})
	case sortByConnections:
		processes = slices.Clone(processes)
		slices.SortStableFunc(processes, func(left, right collector.ProcessSnapshot) int {
			if c := cmp.Compare(len(right.Connections), len(left.Connections)); c != 0 {
				return c
			}
			if c := cmp.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)); c != 0 {
				return c
			}
			return cmp.Compare(left.PID, right.PID)
		})
	}

	return processes
}

func processMatches(process collector.ProcessSnapshot, query string) bool {
	if strings.Contains(strings.ToLower(process.Name), query) {
		return true
	}
	if strings.Contains(itoa32(process.PID), query) {
		return true
	}
	for _, connection := range process.Connections {
		if strings.Contains(strings.ToLower(connection.RemoteIP), query) {
			return true
		}
		if connection.RemotePort != 0 &&
			strings.Contains(strconv.FormatUint(uint64(connection.RemotePort), 10), query) {
			return true
		}
	}
	return false
}

func (m *Model) currentKey() string {
	processes := m.visibleProcesses()
	if m.selected >= 0 && m.selected < len(processes) {
		return processKey(processes[m.selected])
	}
	return ""
}

// restoreSelection re-points the selection at the process identified by key
// after the visible list changed (new snapshot, filter edit, or sort cycle),
// falling back to clamping the current index.
func (m *Model) restoreSelection(key string) {
	processes := m.visibleProcesses()
	if key != "" {
		for index, process := range processes {
			if processKey(process) == key {
				m.selected = index
				return
			}
		}
	}
	m.selected = max(0, min(m.selected, len(processes)-1))
}

// pruneExpanded drops expansion state for processes that left the snapshot,
// so the map does not grow without bound while the monitor runs.
func (m *Model) pruneExpanded() {
	if len(m.expanded) == 0 {
		return
	}

	alive := make(map[string]struct{}, len(m.snapshot.Processes))
	for _, process := range m.snapshot.Processes {
		alive[processKey(process)] = struct{}{}
	}
	for key := range m.expanded {
		if _, ok := alive[key]; !ok {
			delete(m.expanded, key)
		}
	}
}

func (m *Model) toggleExpandedCurrent() {
	processes := m.visibleProcesses()
	if m.selected < 0 || m.selected >= len(processes) {
		return
	}
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}

	key := processKey(processes[m.selected])
	m.expanded[key] = !m.expanded[key]
}

func (m *Model) moveSelection(delta int) {
	count := len(m.visibleProcesses())
	if count == 0 {
		return
	}

	m.selected = max(0, min(m.selected+delta, count-1))
}

func (m *Model) currentProcess() (collector.ProcessSnapshot, bool) {
	processes := m.visibleProcesses()
	if m.selected < 0 || m.selected >= len(processes) {
		return collector.ProcessSnapshot{}, false
	}
	return processes[m.selected], true
}

func itoa32(value int32) string {
	return strconv.FormatInt(int64(value), 10)
}
