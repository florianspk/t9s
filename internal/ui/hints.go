package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type hint struct {
	key  string
	desc string
}

func yamlHint(app App) string {
	if app.resBrowserYAML {
		return "Table"
	}
	return "All YAML"
}

func wrapHint(app App) string {
	if app.wrapMode {
		return "Wrap [ON]"
	}
	return "Wrap"
}

// stateHints returns the small set of hints shown in the header.
// Kept short on purpose — press ? for the full list.
func stateHints(app App) []hint {
	switch app.state {
	case StateNodeList:
		return []hint{
			{"↑↓", "Navigate"},
			{"↵/s", "Services"},
			{"i", "Disks"},
			{":", "Command"},
			{"?", "All shortcuts"},
			{"q", "Quit"},
		}
	case StateServices:
		return []hint{
			{"↑↓", "Navigate"},
			{"↵/l", "Logs"},
			{"?", "All shortcuts"},
			{"Esc/q", "Back"},
		}
	case StateLogs, StateDmesg:
		return []hint{
			{"↑↓", "Select line"},
			{"PgUp/Dn", "Half page"},
			{"g/G", "Top/Bottom"},
			{"/", "Find"},
			{"n/N", "Next/Prev"},
			{"Esc/q", "Back"},
		}
	case StateMachineConfig:
		return []hint{
			{"↑↓", "Scroll"},
			{"PgUp/Dn", "Half page"},
			{"g/G", "Top/Bottom"},
			{"/", "Find"},
			{"n/N", "Next/Prev"},
			{"e", "Edit & apply"},
			{"Esc/q", "Back"},
		}
	case StateExtensions:
		return []hint{
			{"↑↓", "Navigate"},
			{"C", "Catalog"},
			{"Esc/q", "Back"},
		}
	case StateExtCatalog:
		return []hint{
			{"↑↓", "Navigate"},
			{"Esc/q", "Back"},
		}
	case StateMetrics:
		return []hint{
			{"↑↓", "Navigate"},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateDisks:
		return []hint{
			{"↑↓", "Navigate"},
			{":lvm", "LVM view"},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateLVM:
		return []hint{
			{"↑↓", "Scroll"},
			{"g/G", "Top/Bottom"},
			{"e", "Config"},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateResourceBrowser:
		if app.detailOpen() {
			return []hint{
				{"↑↓", "Scroll"},
				{"g/G", "Top/Bottom"},
				{"r", "Refresh"},
				{"Esc/q", "Back to list"},
			}
		}
		return []hint{
			{"↑↓", "Scroll"},
			{"↵", "Show YAML"},
			{"y", yamlHint(app)},
			{"g/G", "Top/Bottom"},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateProcesses, StateAddresses:
		return []hint{
			{"↑↓", "Navigate"},
			{"w", wrapHint(app)},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateContainers:
		return []hint{
			{"↑↓", "Navigate"},
			{"w", wrapHint(app)},
			{"r", "Refresh"},
			{"Esc/q", "Back"},
		}
	case StateHealth:
		return []hint{
			{"↑↓", "Select line"},
			{"PgUp/Dn", "Half page"},
			{"g/G", "Top/Bottom"},
			{"Esc/q", "Back"},
		}
	case StateHelp:
		return []hint{
			{"↑↓", "Scroll"},
			{"Esc/q/?", "Close"},
		}
	case StateUpgradeTalos, StateUpgradeK8s:
		if app.upgradeRunning {
			return []hint{{"↑↓", "Scroll"}, {"Esc", "Back (keeps running)"}}
		}
		if app.upgradeConfirm {
			return []hint{{"y", "Confirm"}, {"n/Esc", "Cancel"}}
		}
		return []hint{{"↵", "Confirm"}, {"Esc/q", "Back"}}
	case StateContextSwitcher:
		return []hint{
			{"↑↓", "Navigate"},
			{"↵", "Switch"},
			{"Esc/q", "Back"},
		}
	}
	return nil
}

// hintRows packs the hint entries into as few lines as the terminal allows.
// renderHintsPanel and hintsHeight must agree, so both go through here.
func (app App) hintRows() [][]string {
	hints := stateHints(app)
	if len(hints) == 0 {
		return nil
	}

	const (
		indent = 2
		gutter = 3
	)
	avail := app.width - indent
	if avail < 20 {
		avail = 20
	}

	var (
		rows [][]string
		cur  []string
		used int
	)
	for _, h := range hints {
		e := keyStyle.Render("<"+h.key+">") + " " + dimStyle.Render(h.desc)
		w := lipgloss.Width(e)
		need := w
		if len(cur) > 0 {
			need += gutter
		}
		if len(cur) > 0 && used+need > avail {
			rows = append(rows, cur)
			cur, used = nil, 0
			need = w
		}
		cur = append(cur, e)
		used += need
	}
	if len(cur) > 0 {
		rows = append(rows, cur)
	}
	if len(rows) > 3 {
		rows = rows[:3]
	}
	return rows
}

func (app App) renderHintsPanel() string {
	rows := app.hintRows()
	if len(rows) == 0 {
		return ""
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = "  " + strings.Join(r, "   ")
	}
	return strings.Join(out, "\n")
}

func (app App) hintsHeight() int {
	return len(app.hintRows())
}
