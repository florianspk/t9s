package ui

import (
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
		w := 12
		for _, r := range rows {
			if len(r[0]) > w {
				w = len(r[0])
			}
		}
		var sb strings.Builder
		sb.WriteString(h(title) + "\n")
		for _, r := range rows {
			// pad the raw key text (styling adds ANSI that breaks %-Ns width)
			sb.WriteString("  " + k(r[0]) + strings.Repeat(" ", w-len(r[0])+2) + d(r[1]) + "\n")
		}
		return sb.String()
	}

	var sb strings.Builder

	sb.WriteString(section("General", [][2]string{
		{"?", "Toggle this help"},
		{":", "Command palette (jump to any view / resource)"},
		{"/", "Search / filter list"},
		{"x", "Switch context"},
		{"w", "Toggle line wrap"},
		{"Esc", "Back / cancel (walks the breadcrumb stack)"},
		{"q / ctrl+c", "Quit"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Navigation", [][2]string{
		{"↑↓ / j k", "Move cursor"},
		{"PgUp / PgDn", "Half page"},
		{"g / G", "Top / bottom"},
		{"n / N", "Next / prev search match (text views)"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Command palette (:)", [][2]string{
		{":nodes", "Node list"},
		{":health", "Cluster health"},
		{":disks", "Disks (+ LVM)"},
		{":lvm", "LVM: PV / VG / LV"},
		{":svc :ext :mc", "Services / extensions / machine config"},
		{":metrics :procs", "Metrics / processes"},
		{":containers :addr", "Containers / addresses"},
		{":dmesg :ctx", "Dmesg / context switcher"},
		{":<resource>", "Any talosctl resource (e.g. :mounts, :routes, :members)"},
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

	sb.WriteString(section("LVM / Resource browser", [][2]string{
		{"↑↓ / j k", "Scroll"},
		{"g / G", "Top / bottom"},
		{"r", "Refresh"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Extension Catalog", [][2]string{
		{"↑↓ / j k", "Navigate"},
		{"Esc / q", "Back"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Upgrade Talos (U)", [][2]string{
		{"type", "Installer image, pre-filled with the node's version"},
		{"↵ then y", "Confirm and start; streams progress, node reboots"},
		{"Esc", "Back (upgrade keeps running if already started)"},
	}))
	sb.WriteByte('\n')

	sb.WriteString(section("Upgrade Kubernetes (K)", [][2]string{
		{"type", "Target version, pre-filled from the running control plane"},
		{"↵ then y", "Confirm; talosctl upgrade-k8s rolls every node"},
		{"Esc", "Back (upgrade keeps running if already started)"},
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
