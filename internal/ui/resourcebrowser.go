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
// row. Columns are read by their offset in the header rather than by field
// index, so an ID containing spaces still parses. ok is false when the listing
// is not in the NODE/NAMESPACE/TYPE/ID shape (yaml mode, `talosctl mounts`, …).
func resourceRowKey(header, row string) (node, id string, ok bool) {
	h := strings.Fields(header)
	if len(h) < 5 || h[0] != "NODE" || h[1] != "NAMESPACE" || h[2] != "TYPE" || h[3] != "ID" {
		return "", "", false
	}
	idStart := strings.Index(header, " ID ")
	verStart := strings.Index(header, h[4])
	if idStart < 0 || verStart <= idStart {
		return "", "", false
	}
	idStart++ // step over the space before "ID"

	f := strings.Fields(row)
	if len(f) < 4 || idStart >= len(row) {
		return "", "", false
	}
	id = strings.TrimSpace(row[idStart:min(verStart, len(row))])
	if id == "" {
		return "", "", false
	}
	return f[0], id, true
}

// detailOpen reports whether the drill-in YAML pane is showing.
func (app App) detailOpen() bool {
	return app.browser.detailID != "" || app.browser.detailLoad
}

func (app App) handleResourceBrowserKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.detailOpen() {
		return app.handleResourceDetailKey(msg)
	}

	n := len(app.browser.lines)
	if app.scrollList(msg.String(), n) {
		return app, nil
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app = app.goBack()
		return app, nil
	case "r":
		app.browser.load = true
		return app, app.loadResourceListing(app.browser.kind, app.browser.yaml)
	case "y":
		// Toggle the whole listing between table and YAML.
		if pseudoKinds[app.browser.kind] != nil {
			return app, nil // not a resource — no YAML form
		}
		app.browser.yaml = !app.browser.yaml
		app.browser.load = true
		app.listScroll, app.viewScrollStart = 0, 0
		return app, app.loadResourceListing(app.browser.kind, app.browser.yaml)
	case "enter":
		if app.browser.yaml || n == 0 {
			return app, nil
		}
		node, id, ok := resourceRowKey(app.browser.lines[0], app.browser.lines[app.listScroll])
		if !ok || id == "ID" {
			app.statusMsg = warnStyle.Render("no resource on this row")
			return app, nil
		}
		app.browser.detailID = id
		app.browser.detailLoad = true
		app.browser.detail = nil
		app.browser.listScroll, app.browser.listStart = app.listScroll, app.viewScrollStart
		app.listScroll, app.viewScrollStart = 0, 0
		return app, app.loadResourceYAML(app.browser.kind, id, node)
	}
	return app, nil
}

// handleResourceDetailKey drives the drill-in YAML pane; Esc returns to the
// listing rather than leaving the view.
func (app App) handleResourceDetailKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.scrollList(msg.String(), len(app.browser.detail)) {
		return app, nil
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app.browser.detailID = ""
		app.browser.detail = nil
		app.browser.detailErr = ""
		app.browser.detailLoad = false
		app.listScroll, app.viewScrollStart = app.browser.listScroll, app.browser.listStart
		return app, nil
	case "r":
		app.browser.detailLoad = true
		return app, app.loadResourceYAML(app.browser.kind, app.browser.detailID, "")
	}
	return app, nil
}

func (app App) renderResourceBrowser(height int) string {
	if app.detailOpen() {
		return app.renderResourceDetail(height)
	}

	kind := app.browser.kind
	count := max(0, len(app.browser.lines)-1)
	if app.browser.yaml {
		kind += " [yaml]"
		count = -1
	}
	title := renderTitleBar(kind, count, 0, nodeName(app.selNode))

	if app.browser.err != "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.browser.err))
	}
	if app.browser.load && len(app.browser.lines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading "+app.browser.kind+"…"))
	}
	if len(app.browser.lines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No rows."))
	}
	return title + renderLinesCursor(app.browser.lines, app.listScroll, app.width, height-2, app.viewScrollStart, "")
}

func (app App) renderResourceDetail(height int) string {
	title := renderTitleBar(app.browser.kind+" / "+app.browser.detailID, -1, 0, nodeName(app.selNode))

	if app.browser.detailErr != "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.browser.detailErr))
	}
	if len(app.browser.detail) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading "+app.browser.detailID+"…"))
	}
	return title + renderLinesCursor(app.browser.detail, app.listScroll, app.width, height-2, app.viewScrollStart, "")
}
