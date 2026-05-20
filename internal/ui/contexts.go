package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleContextsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "up", "k":
		if app.ctxCur > 0 {
			app.ctxCur--
		}

	case "down", "j":
		if app.ctxCur < len(app.filteredContexts())-1 {
			app.ctxCur++
		}

	case "enter":
		ctxs := app.filteredContexts()
		if len(ctxs) == 0 {
			return app, nil
		}
		newCtx := ctxs[app.ctxCur]
		app.talosCtx = newCtx
		app.client.Context = newCtx
		// Reset all state
		app.nodes = nil
		app.selNode = nil
		app.services = nil
		app.extensions = nil
		app.stats = nil
		app.nodeCur = 0
		app.statusMsg = fmt.Sprintf("Switched to context: %s", okStyle.Render(newCtx))
		app.state = StateNodeList
		app.nodeLoading = true
		return app, app.loadNodes()

	case "esc", "q":
		app.state = app.prev
	}
	return app, nil
}

func (app App) renderContextSwitcher(height int) string {
	title := "  Switch Context\n"

	ctxs := app.filteredContexts()
	if len(ctxs) == 0 {
		msg := "No contexts found."
		if app.searchInput.Value() != "" {
			msg = "No match for \"" + app.searchInput.Value() + "\""
		}
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render(msg))
	}

	current := app.talosCtx
	if current == "" {
		current = app.cfg.Context
	}

	header := colHeaderStyle.Render(fmt.Sprintf("  %-30s %s", "CONTEXT", "STATUS"))

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(header)
	sb.WriteByte('\n')

	start := 0
	if app.ctxCur >= height-3 {
		start = app.ctxCur - (height - 4)
	}

	for i := start; i < len(ctxs) && i-start < height-3; i++ {
		c := ctxs[i]
		cursor := "  "
		if i == app.ctxCur {
			cursor = okStyle.Render("▶ ")
		}
		active := ""
		if c == current {
			active = okStyle.Render("● active")
		}
		row := fmt.Sprintf("%s%-30s %s", cursor, truncate(c, 30), active)

		if i == app.ctxCur {
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
