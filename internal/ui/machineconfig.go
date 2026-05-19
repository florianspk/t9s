package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleMachineConfigKey(msg tea.KeyMsg) (App, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "esc", "q":
		app = app.goBack()
		return app, nil

	case "g":
		app.machVP.GotoTop()
	case "G":
		app.machVP.GotoBottom()

	default:
		var cmd tea.Cmd
		app.machVP, cmd = app.machVP.Update(msg)
		return app, cmd
	}
	return app, nil
}

func (app App) renderMachineConfig(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Machine Config: %s\n", titleStyle.Render(node))

	if app.machLoading && app.machConf == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading machine config..."))
	}

	if app.machConf == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No machine config found."))
	}

	app.machVP.Height = height - 2
	app.machVP.Width = app.width

	return title + app.machVP.View()
}
