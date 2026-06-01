package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

func (app App) handleProcessesKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		if app.listScroll > 0 {
			app.listScroll--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, len(app.processes), app.mainHeight()-3)
		}
	case "down", "j":
		if app.listScroll < len(app.processes)-1 {
			app.listScroll++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, len(app.processes), app.mainHeight()-3)
		}
	case "r":
		if app.selNode != nil {
			app.procLoading = true
			app.listScroll = 0
			return app, app.loadProcesses()
		}
	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderProcesses(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Processes on %s\n", titleStyle.Render(node))

	if app.procLoading && len(app.processes) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading processes…"))
	}
	if len(app.processes) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No processes found."))
	}

	const (
		colPID   = 7
		colState = 5
		colCPU   = 10
		colMem   = 10
	)
	const fixedW = 2 + colPID + 2 + colState + 2 + colCPU + 2 + colMem + 2
	cmdAvail := app.width - fixedW
	if cmdAvail < 10 {
		cmdAvail = 10
	}

	hdr := colHeaderStyle.Render(
		"  " + col("PID", colPID) + "  " + col("STATE", colState) + "  " +
			col("CPU-TIME", colCPU) + "  " + col("MEM", colMem) + "  COMMAND",
	)

	maxRows := height - 3
	cur := app.listScroll
	var start int
	if app.wrapMode {
		// Walk backward from cur until maxRows/3 physical lines are consumed.
		start = cur
		budget := maxRows / 3
		for start > 0 && budget > 0 {
			prev := start - 1
			n := strings.Count(wrap.String(app.processes[prev].Command, max(1, cmdAvail)), "\n") + 1
			if n > budget {
				break
			}
			budget -= n
			start = prev
		}
	} else {
		start = clampScrollStart(app.viewScrollStart, cur, len(app.processes), maxRows)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	lineCount := 0
	indent := strings.Repeat(" ", fixedW)

	for i := start; i < len(app.processes) && lineCount < maxRows; i++ {
		p := app.processes[i]
		selected := i == cur

		cursor := "  "
		if selected {
			cursor = okStyle.Render("▶ ")
		}

		fixedPart := cursor +
			col(truncate(p.PID, colPID), colPID) + "  " +
			col(truncate(p.State, colState), colState) + "  " +
			col(truncate(p.CPUTime, colCPU), colCPU) + "  " +
			col(truncate(p.ResMem, colMem), colMem) + "  "

		var cmdLines []string
		if app.wrapMode {
			cmdLines = strings.Split(wrap.String(p.Command, max(1, cmdAvail)), "\n")
		} else {
			cmdLines = []string{truncate(p.Command, cmdAvail)}
		}

		// Trim to remaining budget
		if lineCount+len(cmdLines) > maxRows {
			cmdLines = cmdLines[:maxRows-lineCount]
		}
		if len(cmdLines) == 0 {
			break
		}

		firstLine := fixedPart + dimStyle.Render(cmdLines[0])
		if selected {
			sb.WriteString(selectedStyle.Width(app.width).Render(firstLine))
		} else {
			sb.WriteString(firstLine)
		}
		sb.WriteByte('\n')
		lineCount++

		for _, cont := range cmdLines[1:] {
			sb.WriteString(indent + dimStyle.Render(cont))
			sb.WriteByte('\n')
			lineCount++
		}
	}
	return sb.String()
}
