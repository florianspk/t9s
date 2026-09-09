package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/florianspk/t9s/internal/talos"
)

// nodeName returns the hostname of n, or "" when n is nil (cluster-wide view).
func nodeName(n *talos.Node) string {
	if n == nil {
		return ""
	}
	return n.Hostname
}

func (app App) handleResourceBrowserKey(msg tea.KeyMsg) (App, tea.Cmd) {
	n := len(app.resBrowserLines)
	maxRows := max(1, app.mainHeight()-3)

	scroll := func() {
		app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, n, maxRows)
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app = app.goBack()
		return app, nil
	case "r":
		app.resBrowserLoad = true
		return app, app.loadResourceTable(app.resBrowserKind)
	case "up", "k":
		if app.listScroll > 0 {
			app.listScroll--
		}
		scroll()
	case "down", "j":
		if app.listScroll < n-1 {
			app.listScroll++
		}
		scroll()
	case "pgup":
		app.listScroll = max(0, app.listScroll-maxRows/2)
		scroll()
	case "pgdown":
		app.listScroll = min(max(0, n-1), app.listScroll+maxRows/2)
		scroll()
	case "g":
		app.listScroll = 0
		scroll()
	case "G":
		app.listScroll = max(0, n-1)
		scroll()
	}
	return app, nil
}

func (app App) renderResourceBrowser(height int) string {
	title := renderTitleBar(app.resBrowserKind, max(0, len(app.resBrowserLines)-1), 0, nodeName(app.selNode))

	if app.resBrowserErr != "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.resBrowserErr))
	}
	if app.resBrowserLoad && len(app.resBrowserLines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading "+app.resBrowserKind+"…"))
	}
	if len(app.resBrowserLines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No rows."))
	}
	return title + renderLinesCursor(app.resBrowserLines, app.listScroll, app.width, height-2, app.viewScrollStart, "")
}
