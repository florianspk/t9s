package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

func (app App) handleAddressesKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		if app.listScroll > 0 {
			app.listScroll--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, len(app.addresses), app.mainHeight()-3)
		}
	case "down", "j":
		if app.listScroll < len(app.addresses)-1 {
			app.listScroll++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, len(app.addresses), app.mainHeight()-3)
		}
	case "r":
		if app.selNode != nil {
			app.addrLoading = true
			app.listScroll = 0
			return app, app.loadAddresses()
		}
	case "esc", "q":
		app = app.goBack()
	}
	return app, nil
}

func (app App) renderAddresses(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Network Addresses: %s\n", titleStyle.Render(node))

	if app.addrLoading && len(app.addresses) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading addresses…"))
	}
	if len(app.addresses) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No addresses found."))
	}

	const (
		colIface  = 14
		colFamily = 8
	)
	// ADDRESS expands with terminal width; SCOPE gets the rest after fixed cols
	// fixed overhead: cursor/indent(2) + colIface(14) + sep(2) + sep(2) + colFamily(8) + sep(2) = 30
	colAddr := app.width/3
	if colAddr < 22 {
		colAddr = 22
	}
	if colAddr > 40 {
		colAddr = 40
	}
	const fixedW = 2 + colIface + 2 + 2 + colFamily + 2 // without colAddr
	scopeW := app.width - fixedW - colAddr
	if scopeW < 6 {
		scopeW = 6
	}

	hdr := colHeaderStyle.Render(
		"  " + col("INTERFACE", colIface) + "  " + col("ADDRESS", colAddr) + "  " +
			col("FAMILY", colFamily) + "  SCOPE",
	)

	maxRows := height - 3
	cur := app.listScroll
	var start int
	if app.wrapMode {
		start = cur
		budget := maxRows / 3
		for start > 0 && budget > 0 {
			prev := start - 1
			n := strings.Count(wrap.String(app.addresses[prev].Scope, max(1, scopeW)), "\n") + 1
			if n > budget {
				break
			}
			budget -= n
			start = prev
		}
	} else {
		start = clampScrollStart(app.viewScrollStart, cur, len(app.addresses), maxRows)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	lineCount := 0
	indent := strings.Repeat(" ", fixedW)

	for i := start; i < len(app.addresses) && lineCount < maxRows; i++ {
		a := app.addresses[i]
		selected := i == cur

		cursor := "  "
		if selected {
			cursor = okStyle.Render("▶ ")
		}

		fixedPart := cursor +
			col(truncate(a.Interface, colIface), colIface) + "  " +
			infoStyle.Render(col(truncate(a.Address, colAddr), colAddr)) + "  " +
			col(truncate(a.Family, colFamily), colFamily) + "  "

		var scopeLines []string
		if app.wrapMode {
			scopeLines = strings.Split(wrap.String(a.Scope, max(1, scopeW)), "\n")
		} else {
			scopeLines = []string{truncate(a.Scope, scopeW)}
		}

		if lineCount+len(scopeLines) > maxRows {
			scopeLines = scopeLines[:maxRows-lineCount]
		}
		if len(scopeLines) == 0 {
			break
		}

		firstLine := fixedPart + dimStyle.Render(scopeLines[0])
		if selected {
			sb.WriteString(selectedStyle.Width(app.width).Render(firstLine))
		} else {
			sb.WriteString(firstLine)
		}
		sb.WriteByte('\n')
		lineCount++

		for _, cont := range scopeLines[1:] {
			sb.WriteString(indent + dimStyle.Render(cont))
			sb.WriteByte('\n')
			lineCount++
		}
	}
	return sb.String()
}
