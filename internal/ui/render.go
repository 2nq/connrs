package ui

import (
	"fmt"
	"strconv"
	"strings"

	"connrs/internal/collector"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "connrs\n\nwaiting for terminal size..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	// Re-clamp in case YOffset was assigned directly, bypassing SetYOffset;
	// the viewport panics on out-of-range offsets when rendering.
	m.viewport.SetYOffset(m.viewport.YOffset)

	return m.styles.app.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			m.viewport.View(),
			footer,
		),
	)
}

func (m *Model) refreshViewport(focusSelection bool) {
	if m.width == 0 || m.height == 0 {
		return
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	contentWidth := max(40, m.width-4)
	contentHeight := max(6, m.height-lipgloss.Height(header)-lipgloss.Height(footer)-2)

	m.viewport.Width = contentWidth
	m.viewport.Height = contentHeight

	content, lines := m.renderContent(contentWidth)
	m.lines = lines
	m.viewport.SetContent(content)
	m.clampViewportOffset(content)
	if focusSelection {
		m.ensureSelectionVisible()
	}
}

func (m *Model) clampViewportOffset(content string) {
	contentHeight := lipgloss.Height(content)
	if contentHeight <= m.viewport.Height {
		m.viewport.SetYOffset(0)
		return
	}

	maxOffset := contentHeight - m.viewport.Height
	if m.viewport.YOffset > maxOffset {
		m.viewport.SetYOffset(maxOffset)
		return
	}
	m.viewport.SetYOffset(m.viewport.YOffset)
}

func (m *Model) ensureSelectionVisible() {
	if len(m.lines) == 0 || m.selected < 0 || m.selected >= len(m.lines) {
		return
	}

	target := m.lines[m.selected]
	if target.start < m.viewport.YOffset {
		m.viewport.SetYOffset(target.start)
		return
	}

	limit := m.viewport.YOffset + m.viewport.Height - 1
	if target.end > limit {
		m.viewport.SetYOffset(max(0, target.end-m.viewport.Height+1))
	}
}

func (m *Model) renderHeader() string {
	timestamp := "waiting for first sample"
	if !m.snapshot.CapturedAt.IsZero() {
		timestamp = "updated " + m.snapshot.CapturedAt.Local().Format("15:04:05")
	}

	status := m.spinner.View() + " live"
	if !m.polling && !m.loading {
		status = "o live"
	}

	titleLine := alignLine(
		max(48, m.width-4),
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			m.styles.title.Render("connrs"),
			"  ",
			m.styles.meta.Render("windows connection monitor"),
		),
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			m.styles.livePill.Render(strings.ToUpper(status)),
			" ",
			m.styles.metaPill.Render("REFRESH 1S"),
		),
	)

	parts := []string{titleLine, m.renderSummary(timestamp)}

	if !m.loading && !m.snapshot.IsAdmin {
		parts = append(parts, m.styles.warningBox.Render(
			"Administrator privileges not detected. Process visibility and TCP byte counters may be incomplete.",
		))
	}

	if m.lastErr != nil {
		parts = append(parts, m.styles.errorBox.Render("poll error: "+m.lastErr.Error()))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *Model) renderSummary(timestamp string) string {
	pills := []string{
		m.styles.infoPill.Render(strconv.Itoa(len(m.snapshot.Processes)) + " processes"),
		m.styles.infoPill.Render(strconv.Itoa(m.snapshot.TotalConnections) + " connections"),
		m.styles.infoPill.Render(strconv.Itoa(m.snapshot.BandwidthTracked) + " tcp tracked"),
		m.styles.metaPill.Render(strings.ToUpper(timestamp)),
	}

	return joinBlocks(pills, " ")
}

func (m *Model) renderFooter() string {
	m.help.ShowAll = m.showHelp
	helpView := m.styles.footer.Render(m.help.View(m.keys))

	if process, ok := m.currentProcess(); ok {
		selected := m.styles.muted.Render(
			fmt.Sprintf(
				"selected %s (%d)  %d connections",
				process.Name,
				process.PID,
				len(process.Connections),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, selected, helpView)
	}

	return helpView
}

func (m *Model) renderContent(width int) (string, []lineRange) {
	if m.loading && len(m.snapshot.Processes) == 0 {
		content := m.styles.emptyBox.Width(max(28, width-2)).Render("Collecting live connection data...")
		return content, nil
	}

	if len(m.snapshot.Processes) == 0 {
		content := m.styles.emptyBox.Width(max(28, width-2)).Render("No active TCP or UDP connections detected.")
		return content, nil
	}

	rowWidth := max(32, width)
	blocks := make([]string, 0, len(m.snapshot.Processes))
	ranges := make([]lineRange, 0, len(m.snapshot.Processes))

	cursor := 0
	for index, process := range m.snapshot.Processes {
		block := m.renderProcessRow(process, index == m.selected, m.expanded[processKey(process)], rowWidth)
		blocks = append(blocks, block)

		height := lipgloss.Height(block)
		ranges = append(ranges, lineRange{
			start: cursor,
			end:   cursor + height - 1,
		})
		cursor += height
	}

	return strings.Join(blocks, "\n"), ranges
}

func (m *Model) renderProcessRow(
	process collector.ProcessSnapshot,
	selected bool,
	expanded bool,
	width int,
) string {
	name := displayProcessName(process.Name, process.PID)
	indicator := "▸"
	if expanded {
		indicator = "▾"
	}
	if selected {
		indicator = "▶"
	}

	left := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.styles.rowIndicator.Render(indicator),
		" ",
		m.styles.processName.Render(name),
		" ",
		m.styles.meta.Render("pid "+itoa32(process.PID)),
	)
	right := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.styles.rx.Render("rx "+formatRate(process.ReceivedBps)),
		"  ",
		m.styles.tx.Render("tx "+formatRate(process.SentBps)),
	)

	stats := []string{
		m.styles.meta.Render(fmt.Sprintf("%d connections", len(process.Connections))),
	}
	if process.BandwidthTracked < len(process.Connections) {
		stats = append(stats, m.styles.muted.Render(fmt.Sprintf("%d tracked", process.BandwidthTracked)))
	}
	if selected {
		stats = append(stats, m.styles.muted.Render("enter to toggle details"))
	}

	lines := []string{
		alignLine(max(24, width-4), left, right),
		"  " + strings.Join(stats, "  "),
	}

	if expanded {
		lines = append(lines, m.styles.rowDivider.Render(strings.Repeat("─", max(12, width-6))))
		for _, connection := range process.Connections {
			lines = append(lines, m.renderConnection(connection))
		}
	}

	style := m.styles.row
	if selected {
		style = m.styles.selectedRow
	}

	return style.Width(width).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderConnection(connection collector.ConnectionSnapshot) string {
	endpoints := formatEndpoint(connection.LocalIP, connection.LocalPort) +
		" -> " +
		formatEndpoint(connection.RemoteIP, connection.RemotePort)

	header := "  " + m.styles.connectionKey.Render(
		strings.Join([]string{
			connection.Protocol,
			connection.State,
			endpoints,
		}, "  "),
	)

	metrics := []string{}
	if connection.BandwidthKnown {
		metrics = append(metrics,
			"    "+m.styles.rx.Render("rx "+formatRate(connection.ReceivedBps)),
			m.styles.tx.Render("tx "+formatRate(connection.SentBps)),
			m.styles.connectionMeta.Render(
				"totals rx "+formatBytes(connection.Totals.Received)+"  tx "+formatBytes(connection.Totals.Sent),
			),
		)
	} else {
		metrics = append(metrics, "    "+m.styles.muted.Render("bw n/a"))
	}

	return strings.Join([]string{
		header,
		strings.Join(metrics, "  "),
	}, "\n")
}

func formatRate(value uint64) string {
	return formatUnit(value, "/s")
}

func formatBytes(value uint64) string {
	return formatUnit(value, "")
}

func formatUnit(value uint64, suffix string) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(value)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%d %s%s", value, units[unitIndex], suffix)
	}

	return fmt.Sprintf("%.1f %s%s", size, units[unitIndex], suffix)
}

func formatEndpoint(ip string, port uint32) string {
	host := strings.TrimSpace(ip)
	if host == "" {
		host = "*"
	}
	if port == 0 {
		return host + ":*"
	}
	return host + ":" + strconv.FormatUint(uint64(port), 10)
}

func joinBlocks(blocks []string, separator string) string {
	if len(blocks) == 0 {
		return ""
	}

	var builder strings.Builder
	for index, block := range blocks {
		if index > 0 {
			builder.WriteString(separator)
		}
		builder.WriteString(block)
	}
	return builder.String()
}

func alignLine(width int, left string, right string) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 2 {
		return left + "\n" + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func displayProcessName(name string, pid int32) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		return trimmed
	}
	return "PID " + itoa32(pid)
}
