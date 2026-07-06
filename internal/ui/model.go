package ui

import (
	"context"
	"errors"
	"strconv"
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

type Model struct {
	poller   collector.Poller
	snapshot collector.Snapshot
	selected int
	expanded map[string]bool

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
		if !m.polling {
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
	currentKey := ""
	if len(m.snapshot.Processes) > 0 && m.selected >= 0 && m.selected < len(m.snapshot.Processes) {
		currentKey = processKey(m.snapshot.Processes[m.selected])
	}

	m.snapshot = snapshot
	m.pruneExpanded()
	if len(m.snapshot.Processes) == 0 {
		m.selected = 0
		return
	}

	if currentKey == "" {
		if m.selected >= len(m.snapshot.Processes) {
			m.selected = len(m.snapshot.Processes) - 1
		}
		return
	}

	for index, process := range m.snapshot.Processes {
		if processKey(process) == currentKey {
			m.selected = index
			return
		}
	}

	if m.selected >= len(m.snapshot.Processes) {
		m.selected = len(m.snapshot.Processes) - 1
	}
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
	if m.selected < 0 || m.selected >= len(m.snapshot.Processes) {
		return
	}
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}

	key := processKey(m.snapshot.Processes[m.selected])
	m.expanded[key] = !m.expanded[key]
}

func (m *Model) moveSelection(delta int) {
	count := len(m.snapshot.Processes)
	if count == 0 {
		return
	}

	m.selected = max(0, min(m.selected+delta, count-1))
}

func (m *Model) currentProcess() (collector.ProcessSnapshot, bool) {
	if len(m.snapshot.Processes) == 0 {
		return collector.ProcessSnapshot{}, false
	}
	if m.selected < 0 || m.selected >= len(m.snapshot.Processes) {
		return collector.ProcessSnapshot{}, false
	}
	return m.snapshot.Processes[m.selected], true
}

func itoa32(value int32) string {
	return strconv.FormatInt(int64(value), 10)
}
