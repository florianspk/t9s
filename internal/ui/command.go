package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (app App) handleCommandKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc":
		app.cmdActive = false
		app.cmdErr = ""
		app.cmdInput.Reset()
		return app, nil
	case "enter":
		raw := strings.TrimSpace(app.cmdInput.Value())
		app.cmdActive = false
		app.cmdErr = ""
		app.cmdInput.Reset()
		if raw == "" {
			return app, nil
		}
		return app.runCommand(raw)
	default:
		var cmd tea.Cmd
		app.cmdInput, cmd = app.cmdInput.Update(msg)
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
		app.lvmPVs, app.lvmVGs, app.lvmLVs, app.lvmErr = nil, nil, nil, nil
		app = app.goTo(StateDisks)
		return app, tea.Batch(app.loadDisks(), app.loadVolumes(), app.loadLVM())
	case "lvm", "pvs", "vgs", "lvs":
		app.lvmPVs, app.lvmVGs, app.lvmLVs, app.lvmErr, app.lvmLoad = nil, nil, nil, nil, true
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
	app.resBrowserKind = tok
	app.resBrowserLines, app.resBrowserErr, app.resBrowserLoad = nil, "", true
	app = app.goTo(StateResourceBrowser)
	return app, app.loadResourceTable(tok)
}
