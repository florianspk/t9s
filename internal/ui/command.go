package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/florianspk/t9s/internal/talos"
)

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

// command is one palette entry. The first alias is the canonical name shown in
// completions; run performs the jump (goTo plus whatever load it needs).
type command struct {
	aliases []string
	node    bool // needs a target node
	run     func(app App, node *talos.Node) (App, tea.Cmd)
}

// commands is the single source of truth for the palette: both runCommand and
// the completion list are derived from it, so they cannot drift apart.
var commands = []command{
	{aliases: []string{"quit", "q", "exit"}, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.cleanup()
		return app, tea.Quit
	}},
	{aliases: []string{"help", "?", "h"}, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.helpVP.SetContent(buildHelpContent())
		app.helpVP.GotoTop()
		return app.goTo(StateHelp), nil
	}},
	{aliases: []string{"nodes", "node", "members", "member"}, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app = app.goTo(StateNodeList)
		app.nodeLoading = true
		return app, app.loadNodes()
	}},
	{aliases: []string{"health", "checks"}, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.selNode = nil
		app = app.goTo(StateHealth)
		return startHealth(app)
	}},
	{aliases: []string{"contexts", "ctx", "context"}, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.contexts = app.cfg.ContextNames()
		app.ctxCur = 0
		for i, c := range app.contexts {
			if c == app.talosCtx || (app.talosCtx == "" && c == app.cfg.Context) {
				app.ctxCur = i
				break
			}
		}
		return app.goTo(StateContextSwitcher), nil
	}},
	{aliases: []string{"services", "svc", "service"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.services, app.svcLoading = nil, true
		app = app.goTo(StateServices)
		return app, app.loadServices()
	}},
	{aliases: []string{"extensions", "ext", "extension"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.extensions, app.extLoading = nil, true
		app = app.goTo(StateExtensions)
		return app, app.loadExtensions()
	}},
	{aliases: []string{"catalog", "extcatalog"}, node: true, run: func(app App, node *talos.Node) (App, tea.Cmd) {
		app.catalog, app.catalogCur, app.catalogLoading = nil, 0, true
		app.catalogVersion = node.Version
		app = app.goTo(StateExtCatalog)
		return app, app.loadCatalog()
	}},
	{aliases: []string{"machineconfig", "mc", "config"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.machConf, app.machLoading = "", true
		app = app.goTo(StateMachineConfig)
		return app, app.loadMachineConfig()
	}},
	{aliases: []string{"disks", "disk"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.disks, app.diskLoading, app.volumes = nil, true, nil
		app.lvm = lvmState{load: true}
		app = app.goTo(StateDisks)
		return app, tea.Batch(app.loadDisks(), app.loadVolumes(), app.loadLVM())
	}},
	{aliases: []string{"lvm", "pvs", "vgs", "lvs"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.lvm = lvmState{load: true}
		app = app.goTo(StateLVM)
		return app, app.loadLVM()
	}},
	{aliases: []string{"metrics", "stats", "top"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.stats, app.statsLoading = nil, true
		app = app.goTo(StateMetrics)
		return app, app.loadStats()
	}},
	{aliases: []string{"processes", "procs", "ps", "process"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.processes, app.procLoading = nil, true
		app = app.goTo(StateProcesses)
		return app, app.loadProcesses()
	}},
	{aliases: []string{"containers", "container", "pods"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.containers, app.contLoading = nil, true
		app = app.goTo(StateContainers)
		return app, app.loadContainers()
	}},
	{aliases: []string{"addresses", "addr", "addrs", "ip"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		app.addresses, app.addrLoading = nil, true
		app = app.goTo(StateAddresses)
		return app, app.loadAddresses()
	}},
	{aliases: []string{"dmesg", "kernel"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		return startDmesg(app)
	}},
	// Not a COSI resource — see pseudoKinds.
	{aliases: []string{"df"}, node: true, run: func(app App, _ *talos.Node) (App, tea.Cmd) {
		return app.openResourceBrowser("df")
	}},
}

// commandIndex maps every alias to its command; built once at init.
var commandIndex = func() map[string]*command {
	m := make(map[string]*command, len(commands)*3)
	for i := range commands {
		for _, a := range commands[i].aliases {
			m[a] = &commands[i]
		}
	}
	return m
}()

// viewAliases are the canonical names offered as completions.
var viewAliases = func() []string {
	out := make([]string, 0, len(commands))
	for _, c := range commands {
		out = append(out, c.aliases[0])
	}
	return out
}()

// runCommand resolves a ":" palette entry: a known alias jumps to that view,
// anything else is treated as a Talos resource kind for the generic browser.
func (app App) runCommand(raw string) (App, tea.Cmd) {
	tok := strings.ToLower(strings.Fields(raw)[0])

	cmd, known := commandIndex[tok]
	if !known {
		return app.openResourceBrowser(tok)
	}

	var node *talos.Node
	if cmd.node {
		node = app.targetNode()
		if node == nil {
			app.statusMsg = warnStyle.Render("select a node first (" + tok + ")")
			return app, nil
		}
		app.selNode = node
	}
	return cmd.run(app, node)
}

// targetNode is the node a node-scoped command acts on: the one already
// selected, else the highlighted row, else the first known node.
func (app App) targetNode() *talos.Node {
	if app.selNode != nil {
		return app.selNode
	}
	if n := app.selectedNode(); n != nil {
		return n
	}
	if len(app.nodes) > 0 {
		n := app.nodes[0]
		return &n
	}
	return nil
}

// openResourceBrowser shows `talosctl get <kind>` (or a pseudo-kind) in the
// generic browser.
func (app App) openResourceBrowser(kind string) (App, tea.Cmd) {
	app.browser = resourceBrowser{kind: kind, load: true}
	app = app.goTo(StateResourceBrowser)
	return app, app.loadResourceTable(kind)
}
