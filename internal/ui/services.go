package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleServicesKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.scrollCursor(msg.String(), &app.svcCur, len(app.filteredServices()), app.mainHeight()-3) {
		return app, nil
	}
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "enter", "l":
		svcs := app.filteredServices()
		if len(svcs) == 0 || app.selNode == nil {
			return app, nil
		}
		return app.startLogStream(svcs[app.svcCur].ID)

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
	title := renderTitleBar("Services", len(app.services), 0, node)

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

		id := col(truncate(s.ID, colID), colID)
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
