package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// viewAliases are the palette entries that map to a t9s screen. Kept in the
// order they are offered as completions; keep in sync with runCommand.
var viewAliases = []string{
	"nodes", "health", "disks", "lvm", "services", "extensions", "catalog",
	"machineconfig", "metrics", "processes", "containers", "addresses",
	"dmesg", "contexts", "df", "help", "quit",
}

// maxCmdMatches caps the completion strip so it stays on one line.
const maxCmdMatches = 8

// candidate is a completion with its ranking key.
type candidate struct {
	name string
	// alias marks a t9s view rather than a raw resource kind.
	alias bool
	// head is true when the candidate starts with the query's first letter —
	// the strongest signal that an abbreviation was meant for it ("mnt" is
	// meant for "mounts", not for "environment").
	head bool
	// span is the width of the subsequence match; 0 for exact/prefix hits.
	span int
}

// matchCommands ranks completion candidates for prefix. Exact hits come first,
// then prefix hits, then subsequence abbreviations ("mnt" → "mounts"). Within a
// tier: t9s views before raw resource kinds, tighter matches before loose ones,
// then shorter names, then alphabetical.
func matchCommands(prefix string, kinds []string) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return nil
	}
	var exact, pre, fuzzy []candidate
	consider := func(c string, alias bool) {
		head := c[0] == prefix[0]
		switch {
		case c == prefix:
			exact = append(exact, candidate{c, alias, head, 0})
		case strings.HasPrefix(c, prefix):
			pre = append(pre, candidate{c, alias, head, 0})
		default:
			if span, ok := subsequenceSpan(prefix, c); ok {
				fuzzy = append(fuzzy, candidate{c, alias, head, span})
			}
		}
	}
	seen := make(map[string]struct{}, len(viewAliases)+len(kinds))
	for _, a := range viewAliases {
		seen[a] = struct{}{}
		consider(a, true)
	}
	for _, k := range kinds {
		if _, dup := seen[k]; dup {
			continue
		}
		consider(k, false)
	}

	rank := func(c []candidate) {
		sort.SliceStable(c, func(i, j int) bool {
			a, b := c[i], c[j]
			if a.alias != b.alias {
				return a.alias
			}
			if a.head != b.head {
				return a.head
			}
			if a.span != b.span {
				return a.span < b.span
			}
			if len(a.name) != len(b.name) {
				return len(a.name) < len(b.name)
			}
			return a.name < b.name
		})
	}
	rank(exact)
	rank(pre)
	rank(fuzzy)

	out := make([]string, 0, maxCmdMatches)
	for _, tier := range [][]candidate{exact, pre, fuzzy} {
		for _, c := range tier {
			if len(out) == maxCmdMatches {
				return out
			}
			out = append(out, c.name)
		}
	}
	return out
}

// subsequenceSpan reports whether every byte of short appears in long in order,
// and how wide the greedy match is (smaller is tighter).
func subsequenceSpan(short, long string) (int, bool) {
	if short == "" {
		return 0, true
	}
	i, first, last := 0, -1, -1
	for j := 0; j < len(long) && i < len(short); j++ {
		if long[j] == short[i] {
			if first < 0 {
				first = j
			}
			last = j
			i++
		}
	}
	if i != len(short) {
		return 0, false
	}
	return last - first, true
}

// isSubsequence reports whether every byte of short appears in long in order.
func isSubsequence(short, long string) bool {
	_, ok := subsequenceSpan(short, long)
	return ok
}

func (app App) refreshCmdMatches() App {
	app.palette.matches = matchCommands(app.palette.input.Value(), app.palette.kinds)
	if app.palette.matchIdx >= len(app.palette.matches) {
		app.palette.matchIdx = 0
	}
	return app
}

// renderCmdMatches draws the completion strip under the ":" prompt.
func (app App) renderCmdMatches() string {
	if len(app.palette.matches) == 0 {
		return ""
	}
	sel := lipgloss.NewStyle().Background(colorBgSel).Foreground(colorWhite).Bold(true)
	parts := make([]string, 0, len(app.palette.matches))
	for i, m := range app.palette.matches {
		if i == app.palette.matchIdx {
			parts = append(parts, sel.Render(" "+m+" "))
		} else {
			parts = append(parts, dimStyle.Render(" "+m+" "))
		}
	}
	return strings.Join(parts, " ")
}

func (app App) handleCommandKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "esc":
		app.palette.active = false
		app.palette.err = ""
		app.palette.matches, app.palette.matchIdx = nil, 0
		app.palette.input.Reset()
		return app, nil

	case "tab":
		if len(app.palette.matches) > 0 {
			app.palette.input.SetValue(app.palette.matches[app.palette.matchIdx])
			app.palette.input.CursorEnd()
			app = app.refreshCmdMatches()
		}
		return app, nil

	// ↑↓ / ctrl+p ctrl+n cycle completions; ←→ stay with the text cursor.
	case "down", "ctrl+n":
		if n := len(app.palette.matches); n > 0 {
			app.palette.matchIdx = (app.palette.matchIdx + 1) % n
		}
		return app, nil

	case "up", "ctrl+p":
		if n := len(app.palette.matches); n > 0 {
			app.palette.matchIdx = (app.palette.matchIdx - 1 + n) % n
		}
		return app, nil

	case "enter":
		raw := strings.TrimSpace(app.palette.input.Value())
		// Run the highlighted completion so ":mnt" resolves to "mounts".
		if len(app.palette.matches) > 0 {
			raw = app.palette.matches[app.palette.matchIdx]
		}
		app.palette.active = false
		app.palette.err = ""
		app.palette.matches, app.palette.matchIdx = nil, 0
		app.palette.input.Reset()
		if raw == "" {
			return app, nil
		}
		return app.runCommand(raw)

	default:
		var cmd tea.Cmd
		app.palette.input, cmd = app.palette.input.Update(msg)
		app.palette.matchIdx = 0
		app = app.refreshCmdMatches()
		return app, cmd
	}
}

// runCommand resolves a ":" palette entry: a known view alias jumps to that
// view; anything else is treated as a Talos resource kind for the generic
// browser.
func (app App) runCommand(raw string) (App, tea.Cmd) {
	tok := strings.ToLower(strings.Fields(raw)[0])

	// --- cluster-scoped / special targets ---
	switch tok {
	case "q", "quit", "exit":
		app.cleanup()
		return app, tea.Quit
	case "help", "?", "h":
		app.helpVP.SetContent(buildHelpContent())
		app.helpVP.GotoTop()
		return app.goTo(StateHelp), nil
	case "nodes", "node", "members", "member":
		app = app.goTo(StateNodeList)
		app.nodeLoading = true
		return app, app.loadNodes()
	case "health", "checks":
		app.selNode = nil
		app = app.goTo(StateHealth)
		return startHealth(app)
	case "ctx", "context", "contexts":
		app.contexts = app.cfg.ContextNames()
		app.ctxCur = 0
		for i, c := range app.contexts {
			if c == app.talosCtx || (app.talosCtx == "" && c == app.cfg.Context) {
				app.ctxCur = i
				break
			}
		}
		return app.goTo(StateContextSwitcher), nil
	}

	// --- node-scoped targets ---
	node := app.selNode
	if node == nil {
		node = app.selectedNode()
	}
	if node == nil && len(app.nodes) > 0 {
		n := app.nodes[0]
		node = &n
	}
	if node == nil {
		app.statusMsg = warnStyle.Render("select a node first (" + tok + ")")
		return app, nil
	}
	app.selNode = node

	switch tok {
	case "svc", "service", "services":
		app.services, app.svcLoading = nil, true
		app = app.goTo(StateServices)
		return app, app.loadServices()
	case "ext", "extension", "extensions":
		app.extensions, app.extLoading = nil, true
		app = app.goTo(StateExtensions)
		return app, app.loadExtensions()
	case "catalog", "extcatalog":
		app.catalog, app.catalogCur, app.catalogLoading = nil, 0, true
		app.catalogVersion = node.Version
		app = app.goTo(StateExtCatalog)
		return app, app.loadCatalog()
	case "mc", "machineconfig", "config":
		app.machConf, app.machLoading = "", true
		app = app.goTo(StateMachineConfig)
		return app, app.loadMachineConfig()
	case "disks", "disk":
		app.disks, app.diskLoading, app.volumes = nil, true, nil
		app.lvm.pvs, app.lvm.vgs, app.lvm.lvs, app.lvm.err = nil, nil, nil, nil
		app = app.goTo(StateDisks)
		return app, tea.Batch(app.loadDisks(), app.loadVolumes(), app.loadLVM())
	case "lvm", "pvs", "vgs", "lvs":
		app.lvm.pvs, app.lvm.vgs, app.lvm.lvs, app.lvm.err, app.lvm.load = nil, nil, nil, nil, true
		app = app.goTo(StateLVM)
		return app, app.loadLVM()
	case "metrics", "stats", "top":
		app.stats, app.statsLoading = nil, true
		app = app.goTo(StateMetrics)
		return app, app.loadStats()
	case "procs", "ps", "processes", "process":
		app.processes, app.procLoading = nil, true
		app = app.goTo(StateProcesses)
		return app, app.loadProcesses()
	case "containers", "container", "pods":
		app.containers, app.contLoading = nil, true
		app = app.goTo(StateContainers)
		return app, app.loadContainers()
	case "addr", "addrs", "addresses", "ip":
		app.addresses, app.addrLoading = nil, true
		app = app.goTo(StateAddresses)
		return app, app.loadAddresses()
	case "dmesg", "kernel":
		return startDmesg(app)
	}

	// --- fallback: generic Talos resource browser ---
	app.browser.kind = tok
	app.browser.lines, app.browser.err, app.browser.load = nil, "", true
	app.browser.yaml = false
	app.browser.detail, app.browser.detailID = nil, ""
	app.browser.detailErr, app.browser.detailLoad = "", false
	app = app.goTo(StateResourceBrowser)
	return app, app.loadResourceTable(tok)
}
