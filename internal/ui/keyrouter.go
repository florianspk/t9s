package ui

import tea "github.com/charmbracelet/bubbletea"

func (app App) handleKey(msg tea.KeyMsg) (App, tea.Cmd) {
	// Search mode: route all keys to search handler
	if app.searchActive {
		return app.handleSearchKey(msg)
	}

	// Confirmation dialog: reboot / shutdown
	if app.pendingAction != "" {
		switch msg.String() {
		case "y":
			action := app.pendingAction
			app.pendingAction = ""
			if action == "reboot" {
				return app, app.execReboot()
			}
			return app, app.execShutdown()
		default:
			app.pendingAction = ""
			app.statusMsg = dimStyle.Render("Cancelled.")
			return app, nil
		}
	}

	// Global help view via ?
	if msg.String() == "?" &&
		app.state != StateUpgradeTalos &&
		app.state != StateUpgradeK8s &&
		!app.upgradeRunning {
		app.helpVP.SetContent(buildHelpContent())
		app.helpVP.GotoTop()
		app = app.goTo(StateHelp)
		return app, nil
	}

	// Global search: activate on '/' for list-based views
	if msg.String() == "/" {
		switch app.state {
		case StateNodeList, StateServices, StateExtensions, StateExtCatalog,
			StateMetrics, StateContextSwitcher:
			app.searchActive = true
			app.searchInput.Reset()
			return app, app.searchInput.Focus()
		}
	}

	// Global wrap toggle for list views
	if msg.String() == "w" {
		switch app.state {
		case StateNodeList, StateServices, StateProcesses,
			StateContainers, StateAddresses, StateExtensions, StateExtCatalog,
			StateMetrics, StateContextSwitcher:
			app.wrapMode = !app.wrapMode
			return app, nil
		}
	}

	// Global context switcher from any view
	if msg.String() == "x" &&
		app.state != StateUpgradeTalos &&
		app.state != StateUpgradeK8s &&
		!app.upgradeRunning {
		app.contexts = app.cfg.ContextNames()
		app.ctxCur = 0
		for i, c := range app.contexts {
			if c == app.talosCtx || (app.talosCtx == "" && c == app.cfg.Context) {
				app.ctxCur = i
				break
			}
		}
		app = app.goTo(StateContextSwitcher)
		return app, nil
	}

	switch app.state {
	case StateNodeList:
		return app.handleNodeListKey(msg)
	case StateServices:
		return app.handleServicesKey(msg)
	case StateLogs:
		return app.handleLogsKey(msg)
	case StateMachineConfig:
		return app.handleMachineConfigKey(msg)
	case StateExtensions:
		return app.handleExtensionsKey(msg)
	case StateExtCatalog:
		return app.handleExtCatalogKey(msg)
	case StateDisks:
		return app.handleDisksKey(msg)
	case StateProcesses:
		return app.handleProcessesKey(msg)
	case StateContainers:
		return app.handleContainersKey(msg)
	case StateAddresses:
		return app.handleAddressesKey(msg)
	case StateHealth:
		return app.handleHealthKey(msg)
	case StateHelp:
		return app.handleHelpKey(msg)
	case StateDmesg:
		return app.handleDmesgKey(msg)
	case StateMetrics:
		return app.handleMetricsKey(msg)
	case StateUpgradeTalos, StateUpgradeK8s:
		return app.handleUpgradeKey(msg)
	case StateContextSwitcher:
		return app.handleContextsKey(msg)
	}
	return app, nil
}
