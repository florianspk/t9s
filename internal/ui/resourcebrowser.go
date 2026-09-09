package ui

import (
	"strings"

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

// resourceRowKey extracts the node and resource ID from a `talosctl get` table
// row. ok is false when the listing is not in the NODE/NAMESPACE/TYPE/ID shape
// (yaml mode, `talosctl mounts`, …).
func resourceRowKey(header, row string) (node, id string, ok bool) {
	h := strings.Fields(header)
	if len(h) < 4 || h[0] != "NODE" || h[1] != "NAMESPACE" || h[2] != "TYPE" || h[3] != "ID" {
		return "", "", false
	}
	f := strings.Fields(row)
	if len(f) < 4 {
		return "", "", false
	}
	return f[0], f[3], true
}

// detailOpen reports whether the drill-in YAML pane is showing.
func (app App) detailOpen() bool {
	return app.resBrowserDetailID != "" || app.resBrowserDetailLoad
}

func (app App) handleResourceBrowserKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.detailOpen() {
		return app.handleResourceDetailKey(msg)
	}

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
		return app, app.loadResourceListing(app.resBrowserKind, app.resBrowserYAML)
	case "y":
		// Toggle the whole listing between table and YAML.
		app.resBrowserYAML = !app.resBrowserYAML
		app.resBrowserLoad = true
		app.listScroll, app.viewScrollStart = 0, 0
		return app, app.loadResourceListing(app.resBrowserKind, app.resBrowserYAML)
	case "enter":
		if app.resBrowserYAML || n == 0 {
			return app, nil
		}
		node, id, ok := resourceRowKey(app.resBrowserLines[0], app.resBrowserLines[app.listScroll])
		if !ok || id == "ID" {
			app.statusMsg = warnStyle.Render("no resource on this row")
			return app, nil
		}
		app.resBrowserDetailID = id
		app.resBrowserDetailLoad = true
		app.resBrowserDetail = nil
		app.resBrowserListScroll, app.resBrowserListStart = app.listScroll, app.viewScrollStart
		app.listScroll, app.viewScrollStart = 0, 0
		return app, app.loadResourceYAML(app.resBrowserKind, id, node)
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

// handleResourceDetailKey drives the drill-in YAML pane; Esc returns to the
// listing rather than leaving the view.
func (app App) handleResourceDetailKey(msg tea.KeyMsg) (App, tea.Cmd) {
	n := len(app.resBrowserDetail)
	maxRows := max(1, app.mainHeight()-3)
	scroll := func() {
		app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, n, maxRows)
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app.resBrowserDetailID = ""
		app.resBrowserDetail = nil
		app.resBrowserDetailErr = ""
		app.resBrowserDetailLoad = false
		app.listScroll, app.viewScrollStart = app.resBrowserListScroll, app.resBrowserListStart
		return app, nil
	case "r":
		app.resBrowserDetailLoad = true
		return app, app.loadResourceYAML(app.resBrowserKind, app.resBrowserDetailID, "")
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
	if app.detailOpen() {
		return app.renderResourceDetail(height)
	}

	kind := app.resBrowserKind
	count := max(0, len(app.resBrowserLines)-1)
	if app.resBrowserYAML {
		kind += " [yaml]"
		count = -1
	}
	title := renderTitleBar(kind, count, 0, nodeName(app.selNode))

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

func (app App) renderResourceDetail(height int) string {
	title := renderTitleBar(app.resBrowserKind+" / "+app.resBrowserDetailID, -1, 0, nodeName(app.selNode))

	if app.resBrowserDetailErr != "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.resBrowserDetailErr))
	}
	if len(app.resBrowserDetail) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading "+app.resBrowserDetailID+"…"))
	}
	return title + renderLinesCursor(app.resBrowserDetail, app.listScroll, app.width, height-2, app.viewScrollStart, "")
}
