package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/florianspk/t9s/internal/talos"
)

// visibleDisks hides device-mapper nodes that are just an LVM logical volume —
// those are already listed, with context, in the LVM section below.
func (app App) visibleDisks() []talos.DiskInfo {
	dm := make(map[string]struct{}, len(app.lvm.lvs))
	for _, lv := range app.lvm.lvs {
		if lv.DMDevice != "" {
			dm[lv.DMDevice] = struct{}{}
		}
	}
	if len(dm) == 0 {
		return app.disks
	}
	out := make([]talos.DiskInfo, 0, len(app.disks))
	for _, d := range app.disks {
		if _, hidden := dm[strings.TrimPrefix(d.Dev, "/dev/")]; hidden {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (app App) handleDisksKey(msg tea.KeyMsg) (App, tea.Cmd) {
	nDisks := len(app.visibleDisks())
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		if app.listScroll > 0 {
			app.listScroll--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, nDisks, app.mainHeight()-3)
		}
	case "down", "j":
		if app.listScroll < nDisks-1 {
			app.listScroll++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.listScroll, nDisks, app.mainHeight()-3)
		}
	case "r":
		if app.selNode != nil {
			app.diskLoading = true
			app.listScroll = 0
			app.volumes = nil
			app.lvm.pvs, app.lvm.vgs, app.lvm.lvs, app.lvm.err = nil, nil, nil, nil
			return app, tea.Batch(app.loadDisks(), app.loadVolumes(), app.loadLVM())
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
	disks := app.visibleDisks()
	title := renderTitleBar("Disks", len(disks), 0, node)

	if app.diskLoading && len(app.disks) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading disks…"))
	}
	if len(app.disks) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No disks found."))
	}

	const (
		colDev  = 10
		colType = 6
		// overhead = cursor(2) + dev + sep(2) + type + sep(2) + sep_after_model(2) + sep_after_serial(2)
		overhead = 2 + colDev + 2 + colType + 2 + 2 + 2
		sizeEst  = 10
	)

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

	// Build volume index keyed by disk device name (e.g. "sda").
	volsByDisk := make(map[string][]talos.VolumeInfo)
	for _, v := range app.volumes {
		if v.DiskID != "" {
			volsByDisk[v.DiskID] = append(volsByDisk[v.DiskID], v)
		}
	}

	// Index LVM physical volumes by their device basename (e.g. "sdb", "sdb1")
	// so we can annotate the disk (or one of its partitions) that backs a VG.
	pvByDev := make(map[string]talos.LVMPhysicalVolume)
	for _, pv := range app.lvm.pvs {
		pvByDev[strings.TrimPrefix(pv.Device, "/dev/")] = pv
	}

	maxRows := height - 3
	cur := app.listScroll
	start := clampScrollStart(app.viewScrollStart, cur, len(disks), maxRows)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	rowsLeft := maxRows
	for i := start; i < len(disks) && rowsLeft > 0; i++ {
		d := disks[i]
		selected := i == cur

		cursor := "  "
		if selected {
			cursor = okStyle.Render("▶ ")
		}

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
		rowsLeft--

		// Strip "/dev/" prefix to match volumestatus diskID field.
		diskKey := strings.TrimPrefix(d.Dev, "/dev/")
		for _, v := range volsByDisk[diskKey] {
			if rowsLeft <= 0 {
				break
			}
			sb.WriteString(renderVolumeRow(v, app.width))
			sb.WriteByte('\n')
			rowsLeft--
		}

		// LVM physical-volume markers: the whole disk, or any of its partitions.
		for dev, pv := range pvByDev {
			if rowsLeft <= 0 {
				break
			}
			if dev != diskKey && !strings.HasPrefix(dev, diskKey) {
				continue
			}
			vg := pv.VolumeGroup
			if vg == "" {
				vg = "(unassigned)"
			}
			line := dimStyle.Render(fmt.Sprintf("    └─ PV %s → VG %s", dev, vg))
			if pv.Free != "" {
				line += dimStyle.Render(fmt.Sprintf("   %s free / %s", pv.Free, pv.Size))
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
			rowsLeft--
		}
	}

	// LVM section: volume groups and their logical volumes.
	if rowsLeft > 0 && (len(app.lvm.vgs) > 0 || len(app.lvm.lvs) > 0) {
		sb.WriteString(dimStyle.Render(strings.Repeat("─", min(app.width, 40))))
		sb.WriteByte('\n')
		rowsLeft--

		lvsByVG := make(map[string][]talos.LVMLogicalVolume)
		for _, lv := range app.lvm.lvs {
			lvsByVG[lv.VolumeGroup] = append(lvsByVG[lv.VolumeGroup], lv)
		}

		for _, vg := range app.lvm.vgs {
			if rowsLeft <= 0 {
				break
			}
			sb.WriteString(colHeaderStyle.Render("  VG ") +
				titleStyle.Render(truncate(vg.Name, max(8, app.width/3))) +
				dimStyle.Render(fmt.Sprintf("   %s PV · %s free / %s", nz(vg.PVs), nz(vg.Free), nz(vg.Size))))
			sb.WriteByte('\n')
			rowsLeft--

			lvName := min(48, max(16, app.width/2))
			for _, lv := range lvsByVG[vg.Name] {
				if rowsLeft <= 0 {
					break
				}
				active := okStyle.Render("●")
				if strings.EqualFold(lv.Active, "false") || lv.Active == "" {
					active = dimStyle.Render("○")
				}
				// The VG is already the section header — drop the "<vg>/" prefix.
				name := strings.TrimPrefix(lv.Name, vg.Name+"/")
				sb.WriteString(fmt.Sprintf("    %s %s  %s  %s",
					active,
					col(truncate(name, lvName), lvName),
					col(nz(lv.Layout), 8),
					infoStyle.Render(lv.Size)))
				sb.WriteByte('\n')
				rowsLeft--
			}
		}
	}
	return sb.String()
}

// nz returns "—" for an empty string.
func nz(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// renderVolumeRow renders one volume as an indented sub-row.
// When Available is known, shows a usage bar; otherwise shows the partition size.
func renderVolumeRow(v talos.VolumeInfo, width int) string {
	mount := v.Mount
	if mount == "" {
		mount = v.ID
	}
	fs := v.FS
	if fs == "" {
		fs = "—"
	}

	label := dimStyle.Render(fmt.Sprintf("    %-20s %-6s", truncate(mount, 20), fs))

	var detail string
	if v.Available > 0 {
		used := v.Size - v.Available
		pct := int(float64(used) * 100 / float64(v.Size))
		bar := usageBar(pct, 12)
		detail = fmt.Sprintf("%s  %s / %s (%d%%)",
			bar,
			talos.FormatBytes(used),
			talos.FormatBytes(v.Size),
			pct,
		)
	} else {
		detail = infoStyle.Render(talos.FormatBytes(v.Size))
	}

	row := label + "  " + detail
	if width > 0 && lipgloss.Width(row) > width {
		row = label + "  " + detail[:max(0, width-lipgloss.Width(label)-2)]
	}
	return row
}

// usageBar returns a compact block-character progress bar of given width.
func usageBar(pct, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * width / 100

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	var style lipgloss.Style
	switch {
	case pct >= 90:
		style = errStyle
	case pct >= 70:
		style = warnStyle
	default:
		style = okStyle
	}
	return style.Render(bar)
}
