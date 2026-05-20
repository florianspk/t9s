package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (app App) handleHelpKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q", "?":
		app = app.goBack()
		return app, nil
	default:
		var cmd tea.Cmd
		app.helpVP, cmd = app.helpVP.Update(msg)
		return app, cmd
	}
}

func buildHelpContent() string {
	k := keyStyle.Render
	d := dimStyle.Render
	h := titleStyle.Render

	section := func(title string, rows [][2]string) string {
		var sb strings.Builder
		sb.WriteString(h(title) + "\n")
		for _, r := range rows {
			sb.WriteString(fmt.Sprintf("  %-12s %s\n", k(r[0]), d(r[1])))
		}
		return sb.String()
	}

	var sb strings.Builder

	sb.WriteString(section("Global", [][2]string{
		{"?", "Toggle this help"},
		{"x", "Switch context"},
		{"/", "Search / filter list"},
		{"Esc", "Back / cancel"},
		{"q / ctrl+c", "Quit"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Node List", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"↵ / s", "Services"},
		{"e", "Extensions (installed)"},
		{"C", "Extension catalog"},
		{"m", "Machine config"},
		{"d", "Dmesg stream"},
		{"t", "Metrics (CPU/RAM)"},
		{"p", "Processes"},
		{"c", "Containers"},
		{"a", "Network addresses"},
		{"i", "Disks"},
		{"H", "Cluster health"},
		{"R", "Reboot node"},
		{"S", "Shutdown node"},
		{"U", "Upgrade Talos"},
		{"K", "Upgrade Kubernetes"},
		{"r", "Refresh nodes"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Services", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"↵ / l", "Stream logs"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Logs / Dmesg", [][2]string{
		{"↑↓", "Scroll"},
		{"g", "Go to top"},
		{"G", "Go to bottom"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Machine Config / Health", [][2]string{
		{"↑↓", "Scroll"},
		{"g", "Go to top"},
		{"G", "Go to bottom"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Extensions / Metrics / Processes / Containers / Disks / Addresses", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"r", "Refresh"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Extension Catalog", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Upgrade", [][2]string{
		{"type", "Enter image / version"},
		{"↵", "Confirm"},
		{"y / n", "Yes / No on confirm step"},
		{"Esc", "Abort / back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Context Switcher", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"↵", "Switch to context"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(dimStyle.Render("  Press Esc, q or ? to close"))

	return sb.String()
}

func (app App) renderHelpView(height int) string {
	app.helpVP.Height = height
	app.helpVP.Width = app.width
	if app.helpVP.TotalLineCount() == 0 {
		app.helpVP.SetContent(buildHelpContent())
	}
	return app.helpVP.View()
}
