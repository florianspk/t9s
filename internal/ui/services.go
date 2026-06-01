package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleServicesKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "up", "k":
		if app.svcCur > 0 {
			app.svcCur--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.svcCur, len(app.filteredServices()), app.mainHeight()-3)
		}

	case "down", "j":
		if app.svcCur < len(app.filteredServices())-1 {
			app.svcCur++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.svcCur, len(app.filteredServices()), app.mainHeight()-3)
		}

	case "enter", "l":
		svcs := app.filteredServices()
		if len(svcs) == 0 || app.selNode == nil {
			return app, nil
		}
		svc := svcs[app.svcCur]
		app.logService = svc.ID
		app.logLines = nil
		app.logCur = 0
		app.logStreaming = true
		app = app.goTo(StateLogs)
		app.logCh = make(chan string, 500)
		app.logCtx, app.logCancel = context.WithCancel(context.Background())
		client := app.client
		node := app.selNode.IP
		service := app.logService
		logCh := app.logCh
		logCtx := app.logCtx
		go func() {
			defer close(logCh)
			client.StreamLogs(logCtx, node, service, logCh)
		}()
		return app, waitForLine(app.logCh)

	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderServices(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Services on %s\n", titleStyle.Render(node))

	if app.svcLoading && len(app.services) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading services..."))
	}

	svcs := app.filteredServices()
	if len(svcs) == 0 {
		msg := "No services found."
		if app.searchInput.Value() != "" {
			msg = "No match for \"" + app.searchInput.Value() + "\""
		}
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render(msg))
	}

	const colState = 10
	// SERVICE column expands to fill available width; STATE + HEALTH + separators = 26
	colID := app.width - 26
	if colID < 20 {
		colID = 20
	}

	hdr := colHeaderStyle.Render(
		"  " + col("SERVICE", colID) + "  " + col("STATE", colState) + "  HEALTH",
	)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	maxRows := height - 3
	start := clampScrollStart(app.viewScrollStart, app.svcCur, len(svcs), maxRows)

	for i := start; i < len(svcs) && i < start+maxRows; i++ {
		s := svcs[i]
		selected := i == app.svcCur

		id    := col(truncate(s.ID, colID), colID)
		state := col(truncate(s.State, colState), colState)

		cursor := "  "
		if selected {
			cursor = "▶ "
		}

		var row string
		if selected {
			row = cursor + id + "  " + state + "  " + s.Healthy
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			row = "  " + id + "  " + colorState(state) + "  " + colorHealth(s.Healthy)
			sb.WriteString(row)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
