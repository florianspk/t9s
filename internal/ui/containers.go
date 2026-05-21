package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"

	"github.com/florianspk/t9s/internal/talos"
)

func (app App) handleContainersKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		if app.contCur > 0 {
			app.contCur--
		}
	case "down", "j":
		if app.contCur < len(app.containers)-1 {
			app.contCur++
		}
	case "r":
		if app.selNode != nil {
			app.contLoading = true
			return app, app.loadContainers()
		}
	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func colorStatus(s string) string {
	switch s {
	case "Running", "RUNNING", "READY":
		return okStyle.Render(s)
	case "Stopped", "STOPPED", "Exited", "EXITED", "NOT_READY":
		return errStyle.Render(s)
	default:
		return warnStyle.Render(s)
	}
}

func (app App) renderContainers(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Containers on %s\n", titleStyle.Render(node))

	if app.contLoading && len(app.containers) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading containers…"))
	}
	if len(app.containers) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No containers found."))
	}

	const (
		colNS     = 10
		colID     = 14
		colPID    = 7
		colStatus = 10
	)
	const fixedW = 2 + colNS + 2 + colID + 2 + 2 + colPID + 2 + colStatus
	imageW := app.width - fixedW
	if imageW < 15 {
		imageW = 15
	}

	hdr := colHeaderStyle.Render(
		"  " + col("NAMESPACE", colNS) + "  " + col("ID", colID) + "  " +
			col("IMAGE", imageW) + "  " + col("PID", colPID) + "  STATUS",
	)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	maxRows := height - 3
	var start int
	if app.wrapMode {
		start = app.contCur
		budget := maxRows / 3
		for start > 0 && budget > 0 {
			prev := start - 1
			// Estimate physical lines from image length vs column width
			imgLines := max(1, (len(app.containers[prev].Image)+imageW-1)/imageW)
			if imgLines > budget {
				break
			}
			budget -= imgLines
			start = prev
		}
	} else {
		start = computeScrollStart(app.contCur, len(app.containers), maxRows)
	}

	lineCount := 0

	for i := start; i < len(app.containers) && lineCount < maxRows; i++ {
		c := app.containers[i]
		selected := i == app.contCur
		cursor := "  "
		if selected {
			cursor = okStyle.Render("▶ ")
		}

		ns := col(truncate(c.Namespace, colNS), colNS)
		id := col(truncate(c.ID, colID), colID)
		img := c.Image
		if !app.wrapMode {
			img = truncate(img, imageW)
		}
		img = col(img, imageW)
		pid := col(c.PID, colPID)

		var row string
		if selected {
			row = cursor + ns + "  " + id + "  " + img + "  " + pid + "  " + c.Status
		} else {
			row = cursor + ns + "  " + id + "  " + dimStyle.Render(img) + "  " + pid + "  " + colorStatus(c.Status)
		}

		var physLines []string
		if app.wrapMode {
			physLines = strings.Split(wrap.String(row, app.width), "\n")
		} else {
			physLines = []string{row}
		}

		if lineCount+len(physLines) > maxRows {
			physLines = physLines[:maxRows-lineCount]
		}
		if len(physLines) == 0 {
			break
		}

		if selected {
			sb.WriteString(selectedStyle.Width(app.width).Render(physLines[0]))
			sb.WriteByte('\n')
			lineCount++
			for _, pl := range physLines[1:] {
				sb.WriteString(pl)
				sb.WriteByte('\n')
				lineCount++
			}
		} else {
			for _, pl := range physLines {
				sb.WriteString(pl)
				sb.WriteByte('\n')
				lineCount++
			}
		}
	}
	return sb.String()
}

// ensure import used
var _ = talos.ContainerInfo{}
