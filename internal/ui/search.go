package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/florianspk/t9s/internal/talos"
)

func (app App) searchQuery() string {
	return strings.ToLower(app.searchInput.Value())
}

func matchSearch(q string, fields ...string) bool {
	if q == "" {
		return true
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func (app App) filteredNodes() []talos.Node {
	q := app.searchQuery()
	if q == "" {
		return app.nodes
	}
	var out []talos.Node
	for _, n := range app.nodes {
		if matchSearch(q, n.Hostname, n.DisplayIP, n.Role, n.Version, n.Status) {
			out = append(out, n)
		}
	}
	return out
}

func (app App) filteredServices() []talos.Service {
	q := app.searchQuery()
	if q == "" {
		return app.services
	}
	var out []talos.Service
	for _, s := range app.services {
		if matchSearch(q, s.ID, s.State, s.Healthy) {
			out = append(out, s)
		}
	}
	return out
}

func (app App) filteredExtensions() []talos.Extension {
	q := app.searchQuery()
	if q == "" {
		return app.extensions
	}
	var out []talos.Extension
	for _, e := range app.extensions {
		if matchSearch(q, e.Name, e.Version, e.Description) {
			out = append(out, e)
		}
	}
	return out
}

func (app App) filteredCatalog() []talos.CatalogExtension {
	q := app.searchQuery()
	if q == "" {
		return app.catalog
	}
	var out []talos.CatalogExtension
	for _, e := range app.catalog {
		if matchSearch(q, e.Name, e.Author, e.Description) {
			out = append(out, e)
		}
	}
	return out
}

func (app App) filteredStats() []talos.StatsResult {
	q := app.searchQuery()
	if q == "" {
		return app.stats
	}
	var out []talos.StatsResult
	for _, s := range app.stats {
		if matchSearch(q, s.ID) {
			out = append(out, s)
		}
	}
	return out
}

func (app App) filteredContexts() []string {
	q := app.searchQuery()
	if q == "" {
		return app.contexts
	}
	var out []string
	for _, c := range app.contexts {
		if matchSearch(q, c) {
			out = append(out, c)
		}
	}
	return out
}

func (app App) handleSearchKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc":
		app.searchActive = false
		app.searchInput.Reset()
		app.searchInput.Blur()
		app.clampCursor()
		return app, nil
	case "enter":
		app.searchActive = false
		app.searchInput.Blur()
		return app, nil
	default:
		var cmd tea.Cmd
		app.searchInput, cmd = app.searchInput.Update(msg)
		app.clampCursor()
		return app, cmd
	}
}

// clampCursor resets the cursor for the current state to stay within filtered bounds.
func (app *App) clampCursor() {
	switch app.state {
	case StateNodeList:
		app.nodeCur = clamp(app.nodeCur, 0, max(0, len(app.filteredNodes())-1))
	case StateServices:
		app.svcCur = clamp(app.svcCur, 0, max(0, len(app.filteredServices())-1))
	case StateExtensions:
		app.extCur = clamp(app.extCur, 0, max(0, len(app.filteredExtensions())-1))
	case StateExtCatalog:
		app.catalogCur = clamp(app.catalogCur, 0, max(0, len(app.filteredCatalog())-1))
	case StateContextSwitcher:
		app.ctxCur = clamp(app.ctxCur, 0, max(0, len(app.filteredContexts())-1))
	}
}
