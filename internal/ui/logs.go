package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wrap"
)

func waitForLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg(line)
	}
}

func (app App) handleLogsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	n := len(app.logLines)
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app.stopLogs()
		app = app.goBack()
		return app, nil
	case "up", "k":
		if app.logCur > 0 {
			app.logCur--
		}
	case "down", "j":
		if app.logCur < n-1 {
			app.logCur++
		}
	case "pgup":
		app.logCur = max(0, app.logCur-app.mainHeight()/2)
	case "pgdown":
		app.logCur = min(max(0, n-1), app.logCur+app.mainHeight()/2)
	case "g":
		app.logCur = 0
	case "G":
		app.logCur = max(0, n-1)
	}
	return app, nil
}

func (app App) renderLogs(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	streaming := ""
	if app.logStreaming {
		streaming = infoStyle.Render(" [streaming]")
	} else {
		streaming = dimStyle.Render(" [stopped]")
	}
	title := fmt.Sprintf("  Logs: %s on %s%s\n",
		titleStyle.Render(app.logService),
		titleStyle.Render(node),
		streaming,
	)

	if len(app.logLines) == 0 {
		return title + "  " + infoStyle.Render("Waiting for logs…")
	}

	return title + renderLinesCursor(app.logLines, app.logCur, app.width, height-2)
}

// renderLinesCursor renders a scrollable, cursor-highlighted list of lines.
func renderLinesCursor(lines []string, cur, width, maxRows int) string {
	if len(lines) == 0 || maxRows <= 0 {
		return ""
	}
	if cur < 0 {
		cur = 0
	}
	if cur >= len(lines) {
		cur = len(lines) - 1
	}

	w := width
	if w <= 0 {
		w = 80
	}

	// Walk backward from cur, consuming up to maxRows/2 physical lines,
	// so the cursor lands near the middle regardless of line wrapping.
	start := cur
	budget := maxRows / 2
	for start > 0 && budget > 0 {
		prev := start - 1
		colored := colorLogLine(lines[prev])
		n := strings.Count(wrap.String(colored, max(1, w-2)), "\n") + 1
		if n > budget {
			break
		}
		budget -= n
		start = prev
	}

	var sb strings.Builder
	lineCount := 0

	for i := start; i < len(lines) && lineCount < maxRows; i++ {
		colored := colorLogLine(lines[i])
		wrapped := wrap.String(colored, max(1, w-2))
		physLines := strings.Split(wrapped, "\n")

		// Trim to fit remaining budget
		remaining := maxRows - lineCount
		if len(physLines) > remaining {
			physLines = physLines[:remaining]
		}

		for j, pl := range physLines {
			if i == cur && j == 0 {
				sb.WriteString(selectedStyle.Width(w).Render("▶ " + pl))
			} else {
				sb.WriteString("  " + pl)
			}
			sb.WriteByte('\n')
			lineCount++
		}
	}

	return sb.String()
}
