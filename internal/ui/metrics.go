package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/florianspk/t9s/internal/talos"
)

func (app App) handleMetricsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app = app.goBack()
	case "r":
		if app.selNode != nil {
			app.statsLoading = true
			return app, app.loadStats()
		}
	}
	return app, nil
}

func (app App) renderMetrics(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Container Metrics: %s\n", titleStyle.Render(node))

	if app.statsLoading && len(app.stats) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading metrics…"))
	}
	if len(app.stats) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No stats available."))
	}

	const (
		colID  = 36
		colCPU = 8
		colMem = 10
	)

	hdr := colHeaderStyle.Render(
		"  " + col("CONTAINER", colID) + "  " + col("CPU%", colCPU) + "  " + "MEMORY",
	)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	// elapsed = interval between the two stat samples (fixed, not render-time).
	elapsed := app.statsAt.Sub(app.prevStatsAt).Nanoseconds()

	prevMap := make(map[string]int64, len(app.prevStats))
	for _, p := range app.prevStats {
		prevMap[p.ID] = p.CPUNanos
	}

	for i, s := range app.stats {
		if i >= height-3 {
			break
		}

		cpuStr := "–"
		if elapsed > 0 {
			if prev, ok := prevMap[s.ID]; ok && s.CPUNanos >= prev {
				pct := float64(s.CPUNanos-prev) / float64(elapsed) * 100.0
				cpuStr = fmt.Sprintf("%.1f%%", pct)
				switch {
				case pct >= 80:
					cpuStr = errStyle.Render(cpuStr)
				case pct >= 50:
					cpuStr = warnStyle.Render(cpuStr)
				default:
					cpuStr = okStyle.Render(cpuStr)
				}
			}
		}

		memStr := formatMem(s.MemoryMB)

		row := "  " +
			col(truncate(s.ID, colID), colID) + "  " +
			col(cpuStr, colCPU) + "  " +
			memStr

		sb.WriteString(row)
		sb.WriteByte('\n')
	}

	if len(app.prevStats) == 0 {
		sb.WriteString("\n  " + dimStyle.Render("CPU% available after first refresh (5s)…"))
	}

	return sb.String()
}

func formatMem(mb float64) string {
	if mb >= 1024 {
		gb := mb / 1024
		s := fmt.Sprintf("%.2f GB", gb)
		if gb >= 4 {
			return warnStyle.Render(s)
		}
		return infoStyle.Render(s)
	}
	s := fmt.Sprintf("%.0f MB", mb)
	return dimStyle.Render(s)
}

// col for colored strings: pad based on visual width, not byte length.
// Used when the string may already contain ANSI codes.
func colStyled(s string, w int) string {
	_ = talos.StatsResult{} // ensure import used
	return padRight(s, w)
}
