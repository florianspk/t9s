package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleLVMKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.scrollList(msg.String(), len(app.lvmLines())) {
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
		if app.selNode != nil {
			app.lvm.load = true
			return app, app.loadLVM()
		}
	case "e":
		// LVM is declared in the machine config; edit it there.
		if app.selNode != nil {
			app.machConf, app.machLoading = "", true
			app.statusMsg = dimStyle.Render("LVM lives in the machine config — press e here to edit & apply")
			app = app.goTo(StateMachineConfig)
			return app, app.loadMachineConfig()
		}
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

	// Validation errors first — they explain why the desired state is not met.
	if len(app.lvm.errors) > 0 {
		out = append(out, errStyle.Bold(true).Render("VALIDATION ERRORS"))
		for _, e := range app.lvm.errors {
			out = append(out, row(col(truncate(e.VolumeGroup, 20), 20), errStyle.Render(e.Message)))
		}
		out = append(out, "")
	}

	out = append(out, colHeaderStyle.Render("VOLUME GROUPS"))
	if len(app.lvm.vgs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("NAME", 20), col("SIZE", 12), col("FREE", 12), col("PV", 4), "LV")))
		for _, v := range app.lvm.vgs {
			out = append(out, row(col(truncate(v.Name, 20), 20), col(dash(v.Size), 12), col(dash(v.Free), 12), col(dash(v.PVs), 4), dash(v.LVs)))
		}
	}

	out = append(out, "", colHeaderStyle.Render("LOGICAL VOLUMES"))
	if len(app.lvm.lvs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("PATH", 46), col("VG", 14), col("LAYOUT", 8), col("SIZE", 12), "ACTIVE")))
		for _, l := range app.lvm.lvs {
			out = append(out, row(col(truncate(l.Name, 46), 46), col(dash(l.VolumeGroup), 14), col(dash(l.Layout), 8), col(dash(l.Size), 12), dash(l.Active)))
		}
	}

	out = append(out, "", colHeaderStyle.Render("PHYSICAL VOLUMES"))
	if len(app.lvm.pvs) == 0 {
		out = append(out, dimStyle.Render("  none"))
	} else {
		out = append(out, dimStyle.Render(row(col("DEVICE", 22), col("VG", 16), col("SIZE", 12), "FREE")))
		for _, p := range app.lvm.pvs {
			out = append(out, row(col(truncate(p.Device, 22), 22), col(dash(p.VolumeGroup), 16), col(dash(p.Size), 12), dash(p.Free)))
		}
	}

	// Desired state, from the machine config's LVM* documents.
	if len(app.lvm.vgCfgs) > 0 || len(app.lvm.lvCfgs) > 0 {
		out = append(out, "", colHeaderStyle.Render("CONFIG")+dimStyle.Render("   (from the machine config — press e to open it)"))
		if len(app.lvm.vgCfgs) > 0 {
			out = append(out, dimStyle.Render(row(col("VOLUME GROUP", 22), "PHYSICAL VOLUMES")))
			for _, v := range app.lvm.vgCfgs {
				out = append(out, row(col(truncate(v.Name, 22), 22), dash(strings.Join(v.PhysicalVolumes, ", "))))
			}
		}
		if len(app.lvm.lvCfgs) > 0 {
			out = append(out, dimStyle.Render(row(col("LOGICAL VOLUME", 22), col("VG", 16), col("TYPE", 8), col("SIZE", 10), "MIRRORS/STRIPES")))
			for _, l := range app.lvm.lvCfgs {
				out = append(out, row(
					col(truncate(l.Name, 22), 22),
					col(dash(l.VolumeGroup), 16),
					col(dash(l.Type), 8),
					col(dash(l.Size), 10),
					fmt.Sprintf("%d/%d", l.Mirrors, l.Stripes),
				))
			}
		}
	}
	return out
}

func (app App) renderLVM(height int) string {
	title := renderTitleBar("LVM", len(app.lvm.vgs), 0, nodeName(app.selNode))

	if app.selNode == nil {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No node selected — pick a node first."))
	}
	if app.lvm.err != nil {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			errStyle.Render(app.lvm.err.Error()))
	}
	if app.lvm.load && len(app.lvm.vgs)+len(app.lvm.lvs)+len(app.lvm.pvs) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading LVM status…"))
	}
	if len(app.lvm.vgs)+len(app.lvm.lvs)+len(app.lvm.pvs) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			dimStyle.Render(fmt.Sprintf("No LVM volumes on %s.", nodeName(app.selNode))))
	}

	lines := app.lvmLines()
	maxRows := max(1, height-3)
	start := clampScrollStart(app.viewScrollStart, app.listScroll, len(lines), maxRows)
	end := min(len(lines), start+maxRows)
	return title + strings.Join(lines[start:end], "\n")
}
