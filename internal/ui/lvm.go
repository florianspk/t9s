package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleLVMKey(msg tea.KeyMsg) (App, tea.Cmd) {
	lines := app.lvmLines()
	n := len(lines)
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
		if app.selNode != nil {
			app.lvmLoad = true
			return app, app.loadLVM()
		}
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

// lvmLines builds the full rendered body once so the key handler and the
// renderer agree on line count for scrolling.
func (app App) lvmLines() []string {
	var out []string
	dash := func(s string) string {
		if s == "" {
			return "—"
		}
		return s
	}

	row := func(cells ...string) string { return "  " + strings.Join(cells, " ") }

	out = append(out, colHeaderStyle.Render("VOLUME GROUPS"))
	if len(app.lvmVGs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("NAME", 20), col("SIZE", 12), col("FREE", 12), col("PV", 4), "LV")))
		for _, v := range app.lvmVGs {
			out = append(out, row(col(truncate(v.Name, 20), 20), col(dash(v.Size), 12), col(dash(v.Free), 12), col(dash(v.PVs), 4), dash(v.LVs)))
		}
	}

	out = append(out, "", colHeaderStyle.Render("LOGICAL VOLUMES"))
	if len(app.lvmLVs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("PATH", 46), col("VG", 14), col("LAYOUT", 8), col("SIZE", 12), "ACTIVE")))
		for _, l := range app.lvmLVs {
			out = append(out, row(col(truncate(l.Name, 46), 46), col(dash(l.VolumeGroup), 14), col(dash(l.Layout), 8), col(dash(l.Size), 12), dash(l.Active)))
		}
	}

	out = append(out, "", colHeaderStyle.Render("PHYSICAL VOLUMES"))
	if len(app.lvmPVs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("DEVICE", 22), col("VG", 16), col("SIZE", 12), "FREE")))
		for _, p := range app.lvmPVs {
			out = append(out, row(col(truncate(p.Device, 22), 22), col(dash(p.VolumeGroup), 16), col(dash(p.Size), 12), dash(p.Free)))
		}
	}
	return out
}

func (app App) renderLVM(height int) string {
	title := renderTitleBar("LVM", len(app.lvmVGs), 0, nodeName(app.selNode))

	if app.selNode == nil {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No node selected — pick a node first."))
	}
	if app.lvmErr != nil {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.lvmErr.Error()))
	}
	if app.lvmLoad && len(app.lvmVGs)+len(app.lvmLVs)+len(app.lvmPVs) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading LVM status…"))
	}
	if len(app.lvmVGs)+len(app.lvmLVs)+len(app.lvmPVs) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			dimStyle.Render(fmt.Sprintf("No LVM volumes on %s.", nodeName(app.selNode))))
	}

	lines := app.lvmLines()
	maxRows := max(1, height-3)
	start := clampScrollStart(app.viewScrollStart, app.listScroll, len(lines), maxRows)
	end := min(len(lines), start+maxRows)
	return title + strings.Join(lines[start:end], "\n")
}
