package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleExtensionsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "up", "k":
		if app.extCur > 0 {
			app.extCur--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.extCur, len(app.filteredExtensions()), app.mainHeight()-3)
		}

	case "down", "j":
		if app.extCur < len(app.filteredExtensions())-1 {
			app.extCur++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.extCur, len(app.filteredExtensions()), app.mainHeight()-3)
		}

	case "C":
		if app.selNode != nil {
			app.catalog = nil
			app.catalogCur = 0
			app.catalogLoading = true
			app.catalogVersion = app.selNode.Version
			app.statusMsg = "Loading extension catalog..."
			app = app.goTo(StateExtCatalog)
			return app, app.loadCatalog()
		}

	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderExtensions(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Extensions on %s\n", titleStyle.Render(node))

	if app.extLoading && len(app.extensions) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading extensions..."))
	}

	exts := app.filteredExtensions()
	if len(exts) == 0 {
		msg := "No extensions installed."
		if app.searchInput.Value() != "" {
			msg = "No match for \"" + app.searchInput.Value() + "\""
		}
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			dimStyle.Render(msg))
	}

	colName := 30
	colVer := 16

	header := colHeaderStyle.Render(
		fmt.Sprintf("  %-*s %-*s %s", colName, "NAME", colVer, "VERSION", "DESCRIPTION"),
	)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(header)
	sb.WriteByte('\n')

	start := clampScrollStart(app.viewScrollStart, app.extCur, len(exts), height-3)

	for i := start; i < len(exts) && i-start < height-3; i++ {
		e := exts[i]
		cursor := "  "
		if i == app.extCur {
			cursor = okStyle.Render("▶ ")
		}

		row := fmt.Sprintf("%s%-*s %-*s %s",
			cursor,
			colName, truncate(e.Name, colName),
			colVer, truncate(e.Version, colVer),
			truncate(e.Description, app.width-colName-colVer-6),
		)

		if i == app.extCur {
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
