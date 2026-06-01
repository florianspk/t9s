package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/florianspk/t9s/internal/config"
	"github.com/florianspk/t9s/internal/talos"
)

type AppState int

const (
	StateNodeList AppState = iota
	StateServices
	StateLogs
	StateMachineConfig
	StateExtensions
	StateExtCatalog
	StateDmesg
	StateMetrics
	StateUpgradeTalos
	StateUpgradeK8s
	StateContextSwitcher
	StateDisks
	StateProcesses
	StateContainers
	StateAddresses
	StateHealth
	StateHelp
)

const (
	headerBaseH = 3 // topBar + resourceLine + separator (without hints)
	footerH     = 2 // separator + status line
)

type App struct {
	width  int
	height int

	cfg      *config.TalosConfig
	cfgPath  string
	talosCtx string
	client   *talos.Client

	state AppState
	prev  AppState

	statusMsg  string
	clientVer  string // talosctl binary version
	serverVer  string // Talos server version (from first node)
	verMismatch string // non-empty when client/server versions diverge

	// Node list
	nodes       []talos.Node
	nodeCur     int
	nodeLoading bool

	// Selected node (for sub-views)
	selNode *talos.Node

	// Services
	services    []talos.Service
	svcCur      int
	svcLoading  bool

	// Logs
	logLines    []string
	logCh       chan string
	logCtx      context.Context
	logCancel   context.CancelFunc
	logVP       viewport.Model
	logService  string
	logStreaming bool

	// Machine config
	machConf     string // raw full YAML from talosctl
	machSection  string // extracted "machine:" section shown in UI
	machVP       viewport.Model
	machLoading  bool
	machEditFile string // temp file path while editing
	machEditMode bool   // waiting for apply confirmation

	// Extensions
	extensions []talos.Extension
	extCur     int
	extLoading bool

	// Extension catalog
	catalog        []talos.CatalogExtension
	catalogCur     int
	catalogLoading bool
	catalogVersion string

	// Dmesg
	dmesgLines    []string
	dmesgCh       chan string
	dmesgCtx      context.Context
	dmesgCancel   context.CancelFunc
	dmesgVP       viewport.Model
	dmesgStreaming bool

	// Metrics
	stats        []talos.StatsResult
	statsAt      time.Time // when current stats were loaded
	prevStats    []talos.StatsResult
	prevStatsAt  time.Time // when previous stats were loaded
	statsLoading bool

	// Upgrade
	upgradeInput   textinput.Model
	upgradeLines   []string
	upgradeCh      chan string
	upgradeCtx     context.Context
	upgradeCancel  context.CancelFunc
	upgradeVP      viewport.Model
	upgradeForK8s   bool
	upgradePreserve bool
	upgradeConfirm  bool
	upgradeRunning  bool

	// Context switcher
	contexts []string
	ctxCur   int

	// Search / filter
	searchInput  textinput.Model
	searchActive bool

	// Disks
	disks       []talos.DiskInfo
	diskLoading bool

	// Processes
	processes   []talos.ProcessInfo
	procLoading bool

	// Containers
	containers  []talos.ContainerInfo
	contCur     int
	contLoading bool

	// Addresses
	addresses   []talos.AddressInfo
	addrLoading bool

	// Health
	healthLines    []string
	healthCh       chan string
	healthCtx      context.Context
	healthCancel   context.CancelFunc
	healthVP       viewport.Model
	healthStreaming bool

	// Help
	helpVP viewport.Model

	// Pending confirmation (reboot / shutdown)
	pendingAction string // "reboot" or "shutdown"

	// Generic scroll offset for simple list views (disks, processes, addresses)
	listScroll int

	// Wrap mode: long lines wrap instead of being truncated
	wrapMode bool

	// Line cursors for streaming/static text views
	logCur    int
	dmesgCur  int
	healthCur int

	// Find in log/dmesg views (/ n N)
	findInput  textinput.Model
	findActive bool
	findQuery  string
}

func New(cfg *config.TalosConfig, cfgPath, talosCtx string) App {
	ti := textinput.New()
	ti.Placeholder = "ghcr.io/siderolabs/installer:v1.6.x"
	ti.CharLimit = 256

	si := textinput.New()
	si.Placeholder = "filter…"
	si.CharLimit = 100
	si.Prompt = ""

	fi := textinput.New()
	fi.Placeholder = "search…"
	fi.CharLimit = 100
	fi.Prompt = ""

	return App{
		cfg:          cfg,
		cfgPath:      cfgPath,
		talosCtx:     talosCtx,
		client:       talos.New(cfgPath, talosCtx),
		state:        StateNodeList,
		upgradeInput: ti,
		searchInput:  si,
		findInput:    fi,
		contexts:     cfg.ContextNames(),
		logVP:     viewport.New(80, 20),
		dmesgVP:   viewport.New(80, 20),
		machVP:    viewport.New(80, 20),
		upgradeVP: viewport.New(80, 20),
		healthVP:  viewport.New(80, 20),
		helpVP:    viewport.New(80, 20),
	}
}

func (app App) Init() tea.Cmd {
	return tea.Batch(app.loadNodes(), doTick(), loadClientVersion())
}

func doTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (app App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.width = msg.Width
		app.height = msg.Height
		mainH := app.mainHeight()
		app.logVP.Width = app.width
		app.logVP.Height = mainH
		app.dmesgVP.Width = app.width
		app.dmesgVP.Height = mainH
		app.machVP.Width = app.width
		app.machVP.Height = mainH
		app.upgradeVP.Width = app.width
		app.upgradeVP.Height = mainH
		app.healthVP.Width = app.width
		app.healthVP.Height = mainH
		app.helpVP.Width = app.width
		app.helpVP.Height = mainH
		return app, nil

	case tickMsg:
		var cmd tea.Cmd
		switch app.state {
		case StateNodeList:
			cmd = app.loadNodes()
		case StateMetrics:
			if app.selNode != nil {
				cmd = app.loadStats()
			}
		}
		return app, tea.Batch(cmd, doTick())

	case tea.KeyMsg:
		return app.handleKey(msg)

	case clientVersionMsg:
		app.clientVer = msg.version
		app.verMismatch = checkVersionMismatch(app.clientVer, app.serverVer)
		return app, nil

	case kubeVersionsMsg:
		for i := range app.nodes {
			if v, ok := msg.versions[app.nodes[i].IP]; ok {
				app.nodes[i].KubeVersion = v
			}
		}
		return app, nil

	case nodesLoadedMsg:
		app.nodeLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.nodes = msg.nodes
			if app.nodeCur >= len(app.nodes) {
				app.nodeCur = max(0, len(app.nodes)-1)
			}
			app.statusMsg = fmt.Sprintf("%d nodes", len(app.nodes))
			// Pick server version from first node to compare with client
			if len(msg.nodes) > 0 && app.serverVer == "" {
				app.serverVer = msg.nodes[0].Version
				app.verMismatch = checkVersionMismatch(app.clientVer, app.serverVer)
			}
		}
		return app, app.loadKubeVersions()

	case servicesLoadedMsg:
		app.svcLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.services = msg.services
			app.svcCur = 0
			app.statusMsg = fmt.Sprintf("%d services", len(msg.services))
		}
		return app, nil

	case extensionsLoadedMsg:
		app.extLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.extensions = msg.extensions
			app.extCur = 0
			app.statusMsg = fmt.Sprintf("%d extensions", len(msg.extensions))
		}
		return app, nil

	case machineConfigLoadedMsg:
		app.machLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.machConf = msg.content
			app.machSection = extractSpecContent(msg.content)
			app.machVP.SetContent(app.machSection)
			app.statusMsg = ""
		}
		return app, nil

	case editorDoneMsg:
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Editor error: " + msg.err.Error())
			os.Remove(app.machEditFile)
			app.machEditFile = ""
			return app, nil
		}
		content, err := os.ReadFile(app.machEditFile)
		if err != nil {
			app.statusMsg = errStyle.Render("Cannot read edited file: " + err.Error())
			os.Remove(app.machEditFile)
			app.machEditFile = ""
			return app, nil
		}
		if string(content) == app.machSection {
			app.statusMsg = dimStyle.Render("No changes.")
			os.Remove(app.machEditFile)
			app.machEditFile = ""
			return app, nil
		}
		app.machEditMode = true
		node := ""
		if app.selNode != nil {
			node = app.selNode.Hostname
		}
		app.statusMsg = warnStyle.Render("Apply config to " + node + "? (y/n)")
		return app, nil

	case machineConfigAppliedMsg:
		os.Remove(msg.file)
		app.machEditFile = ""
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Apply failed: " + msg.err.Error())
		} else {
			app.statusMsg = okStyle.Render("Config applied!")
			app.machConf = ""
			app.machSection = ""
			app.machLoading = true
			return app, app.loadMachineConfig()
		}
		return app, nil

	case statsLoadedMsg:
		app.statsLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.prevStats = app.stats
			app.prevStatsAt = app.statsAt
			app.stats = msg.stats
			app.statsAt = time.Now()
			app.statusMsg = ""
		}
		return app, nil

	case catalogLoadedMsg:
		app.catalogLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.catalog = msg.catalog
			app.catalogCur = 0
			app.statusMsg = fmt.Sprintf("%d extensions available", len(msg.catalog))
		}
		return app, nil

	case disksLoadedMsg:
		app.diskLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.disks = msg.disks
			app.statusMsg = fmt.Sprintf("%d disks", len(msg.disks))
		}
		return app, nil

	case processesLoadedMsg:
		app.procLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.processes = msg.processes
			app.statusMsg = fmt.Sprintf("%d processes", len(msg.processes))
		}
		return app, nil

	case containersLoadedMsg:
		app.contLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.containers = msg.containers
			app.contCur = 0
			app.statusMsg = fmt.Sprintf("%d containers", len(msg.containers))
		}
		return app, nil

	case addressesLoadedMsg:
		app.addrLoading = false
		if msg.err != nil {
			app.statusMsg = errStyle.Render("Error: " + msg.err.Error())
		} else {
			app.addresses = msg.addresses
			app.statusMsg = fmt.Sprintf("%d addresses", len(msg.addresses))
		}
		return app, nil

	case healthLineMsg:
		if app.healthStreaming {
			wasAtLast := len(app.healthLines) == 0 || app.healthCur >= len(app.healthLines)-1
			app.healthLines = append(app.healthLines, colorHealthLine(string(msg)))
			if wasAtLast {
				app.healthCur = len(app.healthLines) - 1
			}
			return app, waitForHealthLine(app.healthCh)
		}
		return app, nil

	case healthDoneMsg:
		app.healthStreaming = false
		return app, nil

	case actionDoneMsg:
		app.pendingAction = ""
		if msg.err != nil {
			app.statusMsg = errStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			app.statusMsg = okStyle.Render(msg.action + " sent successfully")
		}
		return app, nil

	case logLineMsg:
		if app.logStreaming {
			wasAtLast := len(app.logLines) == 0 || app.logCur >= len(app.logLines)-1
			app.logLines = append(app.logLines, string(msg))
			if wasAtLast {
				app.logCur = len(app.logLines) - 1
			}
			return app, waitForLine(app.logCh)
		}
		return app, nil

	case logDoneMsg:
		app.logStreaming = false
		return app, nil

	case dmesgLineMsg:
		if app.dmesgStreaming {
			wasAtLast := len(app.dmesgLines) == 0 || app.dmesgCur >= len(app.dmesgLines)-1
			app.dmesgLines = append(app.dmesgLines, string(msg))
			if wasAtLast {
				app.dmesgCur = len(app.dmesgLines) - 1
			}
			return app, waitForDmesgLine(app.dmesgCh)
		}
		return app, nil

	case dmesgDoneMsg:
		app.dmesgStreaming = false
		return app, nil

	case kubeVersionLoadedMsg:
		if msg.err == nil && msg.version != "" {
			app.upgradeInput.SetValue(msg.version)
		} else {
			// Leave placeholder if fetch failed
			app.upgradeInput.SetValue("")
		}
		return app, nil

	case upgradeLineMsg:
		app.upgradeLines = append(app.upgradeLines, string(msg))
		app.upgradeVP.SetContent(strings.Join(app.upgradeLines, "\n"))
		app.upgradeVP.GotoBottom()
		return app, waitForUpgradeLine(app.upgradeCh)

	case upgradeDoneMsg:
		app.upgradeRunning = false
		// Error may have been delivered as an "ERROR: ..." line via the channel
		// rather than through msg.err — check both.
		failed := msg.err != nil
		if !failed {
			for _, line := range app.upgradeLines {
				if strings.HasPrefix(line, "ERROR:") {
					failed = true
					break
				}
			}
		}
		if msg.err != nil {
			app.upgradeLines = append(app.upgradeLines, errStyle.Render("\nError: "+msg.err.Error()))
		} else if !failed {
			app.upgradeLines = append(app.upgradeLines, okStyle.Render("\nUpgrade completed successfully!"))
		}
		app.upgradeVP.SetContent(strings.Join(app.upgradeLines, "\n"))
		app.upgradeVP.GotoBottom()
		return app, nil
	}

	// Delegate viewport scroll to active view
	switch app.state {
	case StateMachineConfig:
		var cmd tea.Cmd
		app.machVP, cmd = app.machVP.Update(msg)
		return app, cmd
	case StateUpgradeTalos, StateUpgradeK8s:
		if app.upgradeRunning {
			var cmd tea.Cmd
			app.upgradeVP, cmd = app.upgradeVP.Update(msg)
			return app, cmd
		}
	}

	return app, nil
}

// fillLine pads a single pre-rendered line to exactly app.width cells.
func (app App) fillLine(line string) string {
	w := lipgloss.Width(line)
	if w >= app.width {
		return line
	}
	return line + strings.Repeat(" ", app.width-w)
}

func (app App) View() string {
	if app.width == 0 {
		return "Initializing t9s..."
	}

	header := app.renderHeader()
	mainH := app.mainHeight()
	main := app.renderMain(mainH)
	footer := app.renderFooter()

	// Pad main to fill the full height so the footer anchors to the bottom.
	blankRow := strings.Repeat(" ", app.width) + "\n"
	if n := strings.Count(main, "\n"); n < mainH {
		main += strings.Repeat(blankRow, mainH-n)
	}

	// Pad every line to full width so no transparent terminal-background
	// cells bleed through (k9s-style full coverage).
	full := header + "\n" + main + footer
	lines := strings.Split(full, "\n")
	for i, l := range lines {
		lines[i] = app.fillLine(l)
	}
	return strings.Join(lines, "\n")
}

func (app App) headerHeight() int {
	return headerBaseH + app.hintsHeight()
}

func (app App) mainHeight() int {
	h := app.height - app.headerHeight() - footerH
	if h < 1 {
		h = 1
	}
	return h
}

func (app App) renderHeader() string {
	ctx := app.talosCtx
	if ctx == "" {
		ctx = app.cfg.Context
	}

	// Left: logo + context
	logo := lipgloss.NewStyle().
		Background(colorBgHead).
		Foreground(colorCyan).
		Bold(true).
		Render(" t9s ")
	sep := headerSepStyle.Render("│")
	ctxPart := headerStyle.Render(" ctx: " + ctx + " ")

	left := logo + sep + ctxPart
	if app.selNode != nil {
		left += sep + headerStyle.Render(" "+app.selNode.Hostname+" ("+app.selNode.IP+") ")
	}

	// Right: current view name
	right := lipgloss.NewStyle().
		Background(colorBgHead).
		Foreground(colorYellow).
		Bold(true).
		Render(" " + viewTitle(app.state) + " ")

	// Fill the gap
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := app.width - leftW - rightW
	if gap < 0 {
		gap = 0
	}
	fill := lipgloss.NewStyle().Background(colorBgHead).Render(strings.Repeat(" ", gap))

	topBar := left + fill + right

	resource := dimStyle.Render("  " + resourceLine(app))
	hints := app.renderHintsPanel()
	sepLine := strings.Repeat("─", app.width)

	parts := topBar + "\n" + resource
	if hints != "" {
		parts += "\n" + hints
	}
	return parts + "\n" + sepLine
}

func resourceLine(app App) string {
	switch app.state {
	case StateNodeList:
		if len(app.nodes) > 0 {
			return fmt.Sprintf("Members (%d)", len(app.nodes))
		}
		return "Members"
	case StateServices:
		if app.selNode != nil {
			return fmt.Sprintf("Services › %s (%d)", app.selNode.Hostname, len(app.services))
		}
	case StateLogs:
		if app.selNode != nil {
			return fmt.Sprintf("Logs › %s › %s", app.selNode.Hostname, app.logService)
		}
	case StateMachineConfig:
		if app.selNode != nil {
			return fmt.Sprintf("MachineConfig › %s", app.selNode.Hostname)
		}
	case StateExtensions:
		if app.selNode != nil {
			return fmt.Sprintf("Extensions › %s (%d)", app.selNode.Hostname, len(app.extensions))
		}
	case StateExtCatalog:
		return fmt.Sprintf("Extension Catalog › %s (%d available)", app.catalogVersion, len(app.catalog))
	case StateDisks:
		if app.selNode != nil {
			return fmt.Sprintf("Disks › %s", app.selNode.Hostname)
		}
	case StateProcesses:
		if app.selNode != nil {
			return fmt.Sprintf("Processes › %s (%d)", app.selNode.Hostname, len(app.processes))
		}
	case StateContainers:
		if app.selNode != nil {
			return fmt.Sprintf("Containers › %s (%d)", app.selNode.Hostname, len(app.containers))
		}
	case StateAddresses:
		if app.selNode != nil {
			return fmt.Sprintf("Addresses › %s (%d)", app.selNode.Hostname, len(app.addresses))
		}
	case StateHealth:
		return "Cluster Health"
	case StateHelp:
		return "Help"
	case StateDmesg:
		if app.selNode != nil {
			return fmt.Sprintf("Dmesg › %s", app.selNode.Hostname)
		}
	case StateMetrics:
		if app.selNode != nil {
			return fmt.Sprintf("Stats › %s", app.selNode.Hostname)
		}
	case StateUpgradeTalos:
		if app.selNode != nil {
			return fmt.Sprintf("Upgrade Talos › %s", app.selNode.Hostname)
		}
	case StateUpgradeK8s:
		return "Upgrade Kubernetes"
	case StateContextSwitcher:
		return fmt.Sprintf("Contexts (%d)", len(app.contexts))
	}
	return ""
}

func viewTitle(s AppState) string {
	switch s {
	case StateNodeList:
		return "[ Nodes ]"
	case StateServices:
		return "[ Services ]"
	case StateLogs:
		return "[ Logs ]"
	case StateMachineConfig:
		return "[ Machine Config ]"
	case StateExtensions:
		return "[ Extensions ]"
	case StateExtCatalog:
		return "[ Ext Catalog ]"
	case StateDisks:
		return "[ Disks ]"
	case StateProcesses:
		return "[ Processes ]"
	case StateContainers:
		return "[ Containers ]"
	case StateAddresses:
		return "[ Addresses ]"
	case StateHealth:
		return "[ Health ]"
	case StateHelp:
		return "[ Help ]"
	case StateDmesg:
		return "[ Dmesg ]"
	case StateMetrics:
		return "[ Metrics ]"
	case StateUpgradeTalos:
		return "[ Upgrade Talos ]"
	case StateUpgradeK8s:
		return "[ Upgrade K8s ]"
	case StateContextSwitcher:
		return "[ Contexts ]"
	}
	return ""
}

func (app App) renderFooter() string {
	sepLine := strings.Repeat("─", app.width)
	if app.searchActive {
		bar := dimStyle.Render("/") + app.searchInput.View()
		return sepLine + "\n  " + bar
	}
	status := app.statusMsg
	if status == "" {
		status = okStyle.Render("✓ ready")
	}
	q := app.searchInput.Value()
	if q != "" {
		status = dimStyle.Render("filter: "+q+"  ") + status
	}
	if app.wrapMode {
		status = warnStyle.Render("[WRAP]") + "  " + status
	}
	if app.verMismatch != "" {
		status = warnStyle.Render(app.verMismatch) + "  " + status
	}
	return sepLine + "\n  " + status
}


func (app App) renderMain(height int) string {
	switch app.state {
	case StateNodeList:
		return app.renderNodeList(height)
	case StateServices:
		return app.renderServices(height)
	case StateLogs:
		return app.renderLogs(height)
	case StateMachineConfig:
		return app.renderMachineConfig(height)
	case StateExtensions:
		return app.renderExtensions(height)
	case StateExtCatalog:
		return app.renderExtCatalog(height)
	case StateDisks:
		return app.renderDisks(height)
	case StateProcesses:
		return app.renderProcesses(height)
	case StateContainers:
		return app.renderContainers(height)
	case StateAddresses:
		return app.renderAddresses(height)
	case StateHealth:
		return app.renderHealth(height)
	case StateHelp:
		return app.renderHelpView(height)
	case StateDmesg:
		return app.renderDmesg(height)
	case StateMetrics:
		return app.renderMetrics(height)
	case StateUpgradeTalos, StateUpgradeK8s:
		return app.renderUpgrade(height)
	case StateContextSwitcher:
		return app.renderContextSwitcher(height)
	}
	return ""
}


// --- Data loaders ---

func (app App) loadNodes() tea.Cmd {
	client := app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		nodes, err := client.GetNodes(ctx)
		return nodesLoadedMsg{nodes: nodes, err: err}
	}
}

func (app App) loadServices() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		svcs, err := client.GetServices(ctx, node)
		return servicesLoadedMsg{services: svcs, err: err}
	}
}

func (app App) loadExtensions() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		exts, err := client.GetExtensions(ctx, node)
		return extensionsLoadedMsg{extensions: exts, err: err}
	}
}

func (app App) loadMachineConfig() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conf, err := client.GetMachineConfig(ctx, node)
		return machineConfigLoadedMsg{content: conf, err: err}
	}
}

func (app App) loadStats() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		stats, err := client.GetStats(ctx, node)
		return statsLoadedMsg{stats: stats, err: err}
	}
}

func (app App) loadCatalog() tea.Cmd {
	client := app.client
	version := app.catalogVersion
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		catalog, err := client.GetExtensionCatalog(ctx, version)
		return catalogLoadedMsg{catalog: catalog, err: err}
	}
}

func (app App) loadDisks() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		disks, err := client.GetDisks(ctx, node)
		return disksLoadedMsg{disks: disks, err: err}
	}
}

func (app App) loadProcesses() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		procs, err := client.GetProcesses(ctx, node)
		return processesLoadedMsg{processes: procs, err: err}
	}
}

func (app App) loadContainers() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		containers, err := client.GetContainers(ctx, node)
		return containersLoadedMsg{containers: containers, err: err}
	}
}

func (app App) loadAddresses() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		addrs, err := client.GetAddresses(ctx, node)
		return addressesLoadedMsg{addresses: addrs, err: err}
	}
}

func (app App) loadKubeVersions() tea.Cmd {
	client := app.client
	nodes := app.nodes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return kubeVersionsMsg{versions: client.GetKubeVersions(ctx, nodes)}
	}
}

func loadClientVersion() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return clientVersionMsg{version: talos.GetClientVersion(ctx)}
	}
}

// checkVersionMismatch returns a warning string when client and server
// Talos versions differ by more than one minor version.
func checkVersionMismatch(client, server string) string {
	if client == "" || server == "" {
		return ""
	}
	if client == server {
		return ""
	}
	parseMinor := func(v string) int {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		if len(parts) < 2 {
			return -1
		}
		n := 0
		for _, c := range parts[1] {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		return n
	}
	cm, sm := parseMinor(client), parseMinor(server)
	if cm < 0 || sm < 0 {
		return ""
	}
	diff := sm - cm
	if diff < 0 {
		diff = -diff
	}
	if diff > 1 {
		return fmt.Sprintf("⚠ talosctl client %s ≠ server %s — update: curl -sL https://github.com/siderolabs/talos/releases/download/%s/talosctl-linux-amd64 -o ~/bin/talosctl", client, server, server)
	}
	if diff == 1 {
		return fmt.Sprintf("⚠ talosctl client %s / server %s", client, server)
	}
	return ""
}

func (app App) loadKubeVersion() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		v, err := client.GetKubernetesVersion(ctx, node)
		return kubeVersionLoadedMsg{version: v, err: err}
	}
}

func waitForHealthLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return healthDoneMsg{}
		}
		return healthLineMsg(line)
	}
}

func (app App) execReboot() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.Reboot(ctx, node)
		return actionDoneMsg{action: "Reboot", err: err}
	}
}

func (app App) execShutdown() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.Shutdown(ctx, node)
		return actionDoneMsg{action: "Shutdown", err: err}
	}
}

// --- Cleanup ---

func (app *App) stopLogs() {
	if app.logCancel != nil {
		app.logCancel()
		app.logCancel = nil
	}
	app.logStreaming = false
}

func (app *App) stopDmesg() {
	if app.dmesgCancel != nil {
		app.dmesgCancel()
		app.dmesgCancel = nil
	}
	app.dmesgStreaming = false
}

func (app *App) stopUpgrade() {
	if app.upgradeCancel != nil {
		app.upgradeCancel()
		app.upgradeCancel = nil
	}
	app.upgradeRunning = false
}

func (app *App) stopHealth() {
	if app.healthCancel != nil {
		app.healthCancel()
		app.healthCancel = nil
	}
	app.healthStreaming = false
}

func (app *App) cleanup() {
	app.stopLogs()
	app.stopDmesg()
	app.stopUpgrade()
	app.stopHealth()
}

// --- Helpers ---

// computeScrollStart returns the first visible item index such that cur
// appears near the center of a window of height maxRows (1 item per line).
func computeScrollStart(cur, total, maxRows int) int {
	if total <= maxRows {
		return 0
	}
	start := cur - maxRows/2
	if start < 0 {
		return 0
	}
	if start > total-maxRows {
		return total - maxRows
	}
	return start
}

// col pads plain ASCII text to exactly w chars (must be called before colorising).
func col(s string, w int) string {
	l := len(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

func padRight(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max > 3 {
		return s[:max-3] + "..."
	}
	return s[:max]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// --- Navigation helpers ---

func (app App) goTo(state AppState) App {
	app.prev = app.state
	app.state = state
	app.searchActive = false
	app.searchInput.Reset()
	app.listScroll = 0
	return app
}

func (app App) goBack() App {
	app.stopLogs()
	app.stopDmesg()
	app.stopHealth()
	app.searchActive = false
	app.searchInput.Reset()
	switch app.state {
	case StateLogs:
		app.state = StateServices
	case StateServices:
		app.state = StateNodeList
		app.selNode = nil
	case StateExtCatalog:
		app.state = app.prev
	case StateHelp:
		app.state = app.prev
	default:
		app.state = StateNodeList
		app.selNode = nil
	}
	return app
}

func (app App) selectedNode() *talos.Node {
	nodes := app.filteredNodes()
	if len(nodes) == 0 {
		return nil
	}
	n := nodes[app.nodeCur]
	return &n
}
