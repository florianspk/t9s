package ui

import (
	"strings"
)

type hint struct {
	key  string
	desc string
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
			{"e", "Extensions"},
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
			{"Esc/q", "Back"},
		}
	case StateMachineConfig:
		return []hint{
			{"↑↓", "Scroll"},
			{"g/G", "Top/Bottom"},
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
		if app.state == StateUpgradeTalos {
			return []hint{{"↵", "Confirm"}, {"p", "--preserve"}, {"Esc/q", "Back"}}
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

func (app App) renderHintsPanel() string {
	hints := stateHints(app)
	if len(hints) == 0 {
		return ""
	}

	w := app.width
	if w < 40 {
		w = 40
	}
	numCols := max(2, min(5, w/20))
	colW := w / numCols

	numRows := (len(hints) + numCols - 1) / numCols
	if numRows > 3 {
		numRows = 3
	}

	var rows []string
	for row := 0; row < numRows; row++ {
		var sb strings.Builder
		sb.WriteString("  ")
		for col := 0; col < numCols; col++ {
			idx := row*numCols + col
			if idx < len(hints) {
				h := hints[idx]
				entry := keyStyle.Render("<"+h.key+">") + " " + dimStyle.Render(h.desc)
				if col < numCols-1 {
					entry = padRight(entry, colW-2)
				}
				sb.WriteString(entry)
			}
		}
		rows = append(rows, sb.String())
	}
	return strings.Join(rows, "\n")
}

func (app App) hintsHeight() int {
	hints := stateHints(app)
	if len(hints) == 0 {
		return 0
	}
	w := app.width
	if w < 40 {
		w = 40
	}
	numCols := max(2, min(5, w/20))
	rows := (len(hints) + numCols - 1) / numCols
	if rows > 3 {
		rows = 3
	}
	return rows
}
