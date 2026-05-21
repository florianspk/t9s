package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleDisksKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		if app.listScroll > 0 {
			app.listScroll--
		}
	case "down", "j":
		if app.listScroll < len(app.disks)-1 {
			app.listScroll++
		}
	case "r":
		if app.selNode != nil {
			app.diskLoading = true
			app.listScroll = 0
			return app, app.loadDisks()
		}
	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderDisks(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Disks on %s\n", titleStyle.Render(node))

	if app.diskLoading && len(app.disks) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading disks…"))
	}
	if len(app.disks) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No disks found."))
	}

	const (
		colDev = 10
		colType = 6
		// overhead = cursor(2) + dev + sep(2) + type + sep(2) + sep_after_model(2) + sep_after_serial(2)
		overhead = 2 + colDev + 2 + colType + 2 + 2 + 2 // = 26
		sizeEst  = 10                                    // reserve for size value ("1.00 TB" etc.)
	)

	// Remaining space is shared between MODEL (≤20) and SERIAL (≤16).
	// MODEL gets 60% of available, SERIAL gets the rest, both capped.
	avail := app.width - overhead - sizeEst
	if avail < 14 {
		avail = 14
	}
	colModel := min(20, avail*6/10)
	if colModel < 8 {
		colModel = 8
	}
	colSerial := min(16, avail-colModel)
	if colSerial < 6 {
		colSerial = 6
	}

	hdr := colHeaderStyle.Render(
		"  " + col("DEV", colDev) + "  " + col("TYPE", colType) + "  " +
			col("MODEL", colModel) + "  " + col("SERIAL", colSerial) + "  SIZE",
	)

	maxRows := height - 3
	cur := app.listScroll
	start := computeScrollStart(cur, len(app.disks), maxRows)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	for i := start; i < len(app.disks) && i-start < maxRows; i++ {
		d := app.disks[i]
		selected := i == cur

		cursor := "  "
		if selected {
			cursor = okStyle.Render("▶ ")
		}

		// MODEL is a middle column — always truncate so SERIAL/SIZE stay aligned.
		row := cursor +
			col(truncate(d.Dev, colDev), colDev) + "  " +
			col(truncate(d.Type, colType), colType) + "  " +
			col(truncate(d.Model, colModel), colModel) + "  " +
			col(truncate(d.Serial, colSerial), colSerial) + "  " +
			infoStyle.Render(d.Size)

		if selected {
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			sb.WriteString(row)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
