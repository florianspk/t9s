package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleExtCatalogKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "up", "k":
		if app.catalogCur > 0 {
			app.catalogCur--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.catalogCur, len(app.filteredCatalog()), app.mainHeight()-4)
		}

	case "down", "j":
		if app.catalogCur < len(app.filteredCatalog())-1 {
			app.catalogCur++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.catalogCur, len(app.filteredCatalog()), app.mainHeight()-4)
		}

	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderExtCatalog(height int) string {
	title := fmt.Sprintf("  Extension Catalog  %s\n",
		dimStyle.Render("ghcr.io/siderolabs/extensions:"+app.catalogVersion))

	if app.catalogLoading && len(app.catalog) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Pulling catalog from registry…"))
	}
	catalog := app.filteredCatalog()
	if len(catalog) == 0 {
		msg := "No extensions found for " + app.catalogVersion
		if app.searchInput.Value() != "" {
			msg = "No match for \"" + app.searchInput.Value() + "\""
		}
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render(msg))
	}

	const (
		colName   = 28
		colAuthor = 18
	)

	hdr := colHeaderStyle.Render(
		"  " + col("NAME", colName) + "  " + col("AUTHOR", colAuthor) + "  " + "DESCRIPTION",
	)

	// Reserve 3 rows: title + header + ref-panel at bottom
	listH := height - 4
	if listH < 1 {
		listH = 1
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	start := clampScrollStart(app.viewScrollStart, app.catalogCur, len(catalog), listH)

	for i := start; i < len(catalog) && i-start < listH; i++ {
		e := catalog[i]
		cursor := "  "
		if i == app.catalogCur {
			cursor = okStyle.Render("▶ ")
		}

		descAvail := app.width - colName - colAuthor - 8
		if descAvail < 10 {
			descAvail = 10
		}
		// First line of description only
		desc := e.Description
		if nl := strings.Index(desc, "\n"); nl >= 0 {
			desc = desc[:nl]
		}

		row := cursor +
			col(truncate(e.Name, colName), colName) + "  " +
			col(truncate(e.Author, colAuthor), colAuthor) + "  " +
			truncate(desc, descAvail)

		if i == app.catalogCur {
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteByte('\n')
	}

	// Detail panel: show image ref of selected extension
	if app.catalogCur < len(catalog) {
		sel := catalog[app.catalogCur]
		sep := strings.Repeat("─", app.width)
		refLine := "  " + dimStyle.Render("ref: ") + infoStyle.Render(sel.ImageRef)
		sb.WriteString(sep + "\n")
		sb.WriteString(refLine)
	}

	return sb.String()
}
