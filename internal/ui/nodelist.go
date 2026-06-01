package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleNodeListKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		app.cleanup()
		return app, tea.Quit

	case "up", "k":
		if app.nodeCur > 0 {
			app.nodeCur--
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.nodeCur, len(app.filteredNodes()), app.mainHeight()-2)
		}

	case "down", "j":
		if app.nodeCur < len(app.filteredNodes())-1 {
			app.nodeCur++
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.nodeCur, len(app.filteredNodes()), app.mainHeight()-2)
		}

	case "enter", "s":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.services = nil
		app.svcLoading = true
		app.statusMsg = "Loading services..."
		app = app.goTo(StateServices)
		return app, app.loadServices()

	case "e":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.extensions = nil
		app.extLoading = true
		app.statusMsg = "Loading extensions..."
		app = app.goTo(StateExtensions)
		return app, app.loadExtensions()

	case "C":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.catalog = nil
		app.catalogCur = 0
		app.catalogLoading = true
		app.catalogVersion = n.Version
		app.statusMsg = "Loading extension catalog..."
		app = app.goTo(StateExtCatalog)
		return app, app.loadCatalog()

	case "m":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.machConf = ""
		app.machLoading = true
		app.statusMsg = "Loading machine config..."
		app = app.goTo(StateMachineConfig)
		return app, app.loadMachineConfig()

	case "d":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.dmesgLines = nil
		app.dmesgCur = 0
		app.dmesgStreaming = true
		app = app.goTo(StateDmesg)
		app.dmesgCh = make(chan string, 500)
		app.dmesgCtx, app.dmesgCancel = context.WithCancel(context.Background())
		client := app.client
		node := app.selNode.IP
		dmesgCh := app.dmesgCh
		dmesgCtx := app.dmesgCtx
		go func() {
			defer close(dmesgCh)
			client.StreamDmesg(dmesgCtx, node, dmesgCh)
		}()
		return app, waitForDmesgLine(app.dmesgCh)

	case "t":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.stats = nil
		app.statsLoading = true
		app.statusMsg = "Loading stats..."
		app = app.goTo(StateMetrics)
		return app, app.loadStats()

	case "U":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.upgradeForK8s = false
		app.upgradePreserve = true // default on: required for single-node etcd clusters
		app.upgradeConfirm = false
		app.upgradeRunning = false
		app.upgradeLines = nil
		app.upgradeVP.SetContent("")
		app.upgradeInput.Reset()
		// Pre-fill installer image with current node version
		app.upgradeInput.SetValue("ghcr.io/siderolabs/installer:" + n.Version)
		app.upgradeInput.Placeholder = "ghcr.io/siderolabs/installer:v1.7.0"
		app = app.goTo(StateUpgradeTalos)
		return app, app.upgradeInput.Focus()

	case "K":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.upgradeForK8s = true
		app.upgradeConfirm = false
		app.upgradeRunning = false
		app.upgradeLines = nil
		app.upgradeVP.SetContent("")
		app.upgradeInput.Reset()
		app.upgradeInput.Placeholder = "fetching version…"
		app = app.goTo(StateUpgradeK8s)
		return app, tea.Batch(app.upgradeInput.Focus(), app.loadKubeVersion())

	case "x":
		app.contexts = app.cfg.ContextNames()
		app.ctxCur = 0
		for i, c := range app.contexts {
			if c == app.talosCtx {
				app.ctxCur = i
				break
			}
		}
		app = app.goTo(StateContextSwitcher)

	case "p":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.processes = nil
		app.procLoading = true
		app = app.goTo(StateProcesses)
		return app, app.loadProcesses()

	case "c":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.containers = nil
		app.contLoading = true
		app = app.goTo(StateContainers)
		return app, app.loadContainers()

	case "a":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.addresses = nil
		app.addrLoading = true
		app = app.goTo(StateAddresses)
		return app, app.loadAddresses()

	case "i":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.disks = nil
		app.diskLoading = true
		app = app.goTo(StateDisks)
		app.volumes = nil
		return app, tea.Batch(app.loadDisks(), app.loadVolumes())

	case "H":
		app.selNode = app.selectedNode()
		app = app.goTo(StateHealth)
		var cmd tea.Cmd
		app, cmd = startHealth(app)
		return app, cmd

	case "R":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.pendingAction = "reboot"
		app.statusMsg = warnStyle.Render("Reboot " + n.Hostname + "? (y/N)")
		return app, nil

	case "S":
		n := app.selectedNode()
		if n == nil {
			return app, nil
		}
		app.selNode = n
		app.pendingAction = "shutdown"
		app.statusMsg = warnStyle.Render("Shutdown " + n.Hostname + "? (y/N)")
		return app, nil

	case "r":
		app.nodeLoading = true
		app.statusMsg = "Refreshing..."
		return app, app.loadNodes()
	}
	return app, nil
}

func (app App) renderNodeList(height int) string {
	if app.nodeLoading && len(app.nodes) == 0 {
		return lipgloss.Place(app.width, height, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading nodes…"))
	}
	nodes := app.filteredNodes()
	if len(nodes) == 0 {
		msg := "No nodes found. Check talosconfig and connectivity."
		if app.searchInput.Value() != "" {
			msg = "No match for \"" + app.searchInput.Value() + "\""
		}
		return lipgloss.Place(app.width, height, lipgloss.Center, lipgloss.Center,
			warnStyle.Render(msg))
	}

	const (
		colIP   = 16
		colRole = 14
		colVer  = 12
		colK8s  = 10
	)
	// K8S column is shown only when the terminal is wide enough.
	// Fixed overhead with K8S:    cursor(2)+sep×5(10)+IP(16)+ROLE(14)+VER(12)+K8S(10)+"STATUS"(6) = 70
	// Fixed overhead without K8S: cursor(2)+sep×4(8)+IP(16)+ROLE(14)+VER(12)+"STATUS"(6)          = 58
	showK8s := app.width >= 90
	fixedW := 58
	if showK8s {
		fixedW = 70
	}
	colHost := app.width - fixedW
	if colHost < 1 {
		colHost = 1
	}

	// Column header row
	hdrCols := "  " +
		col(truncate("NAME", colHost), colHost) + "  " +
		col("IP", colIP) + "  " +
		col("ROLE", colRole) + "  " +
		col("VERSION", colVer) + "  "
	if showK8s {
		hdrCols += col("K8S", colK8s) + "  "
	}
	hdrCols += "STATUS"
	hdr := colHeaderStyle.Render(hdrCols)

	var sb strings.Builder
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	maxRows := height - 2
	start := clampScrollStart(app.viewScrollStart, app.nodeCur, len(nodes), maxRows)

	for i := start; i < len(nodes) && i < start+maxRows; i++ {
		n := nodes[i]

		// Build plain fields first, then colorise — keeps alignment correct.
		host := col(truncate(n.Hostname, colHost), colHost)
		ip   := col(truncate(n.DisplayIP, colIP), colIP)
		role := col(truncate(n.Role, colRole), colRole)
		ver  := col(truncate(n.Version, colVer), colVer)

		selected := i == app.nodeCur
		cursor := "  "
		if selected {
			cursor = "▶ "
		}

		// Apply semantic colors to plain-padded strings
		roleColored   := colorRole(role)
		verColored    := dimStyle.Render(ver)
		statusColored := colorNodeStatus(n.Status)
		if selected {
			roleColored   = role
			verColored    = ver
			statusColored = n.Status
		}

		row := cursor + host + "  " + ip + "  " + roleColored + "  " + verColored + "  "
		if showK8s {
			k8s := col(truncate(n.KubeVersion, colK8s), colK8s)
			k8sColored := infoStyle.Render(k8s)
			if selected {
				k8sColored = k8s
			}
			row += k8sColored + "  "
		}
		row += statusColored

		if selected {
			sb.WriteString(selectedStyle.Width(app.width).Render(row))
		} else {
			sb.WriteString("  " + row[2:]) // keep indent, drop cursor space
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
