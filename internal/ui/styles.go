package ui

import "github.com/charmbracelet/lipgloss"

var (
	cardBorder = lipgloss.Border{
		Top:          "─",
		Bottom:       "─",
		Left:         "│",
		Right:        "│",
		TopLeft:      "╭",
		TopRight:     "╮",
		BottomLeft:   "╰",
		BottomRight:  "╯",
	}
)

type styles struct {
	app            lipgloss.Style
	title          lipgloss.Style
	livePill       lipgloss.Style
	metaPill       lipgloss.Style
	infoPill       lipgloss.Style
	meta           lipgloss.Style
	muted          lipgloss.Style
	rx             lipgloss.Style
	tx             lipgloss.Style
	row            lipgloss.Style
	selectedRow    lipgloss.Style
	rowIndicator   lipgloss.Style
	rowDivider     lipgloss.Style
	processName    lipgloss.Style
	connectionKey  lipgloss.Style
	connectionMeta lipgloss.Style
	warningBox     lipgloss.Style
	errorBox       lipgloss.Style
	emptyBox       lipgloss.Style
	footer         lipgloss.Style
}

func newStyles() styles {
	return styles{
		app:            lipgloss.NewStyle().Padding(1, 2),
		title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7dd3fc")),
		livePill:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#081018")).Background(lipgloss.Color("#67e8f9")).Padding(0, 1),
		metaPill:       lipgloss.NewStyle().Foreground(lipgloss.Color("#cbd5e1")).Background(lipgloss.Color("#172033")).Padding(0, 1),
		infoPill:       lipgloss.NewStyle().Foreground(lipgloss.Color("#dbeafe")).Background(lipgloss.Color("#111827")).Padding(0, 1),
		meta:           lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")),
		muted:          lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")),
		rx:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#93c5fd")),
		tx:             lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fbbf24")),
		row:            lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#1f2937")).PaddingLeft(1).PaddingRight(1),
		selectedRow:    lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#67e8f9")).Background(lipgloss.Color("#0b1220")).PaddingLeft(1).PaddingRight(1),
		rowIndicator:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#67e8f9")),
		rowDivider:     lipgloss.NewStyle().Foreground(lipgloss.Color("#1e293b")),
		processName:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8fafc")),
		connectionKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("#cbd5e1")),
		connectionMeta: lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")),
		warningBox:     lipgloss.NewStyle().Border(cardBorder).BorderForeground(lipgloss.Color("#f59e0b")).Foreground(lipgloss.Color("#fbbf24")).Padding(0, 1),
		errorBox:       lipgloss.NewStyle().Border(cardBorder).BorderForeground(lipgloss.Color("#ef4444")).Foreground(lipgloss.Color("#fca5a5")).Padding(0, 1),
		emptyBox:       lipgloss.NewStyle().Border(cardBorder).BorderForeground(lipgloss.Color("#334155")).Foreground(lipgloss.Color("#e2e8f0")).Padding(1, 2),
		footer:         lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b")),
	}
}
