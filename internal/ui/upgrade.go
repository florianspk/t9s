package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func waitForUpgradeLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return upgradeDoneMsg{}
		}
		return upgradeLineMsg(line)
	}
}

func (app App) handleUpgradeKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.upgradeRunning {
		switch msg.String() {
		case "ctrl+c":
			app.cleanup()
			return app, tea.Quit
		case "esc":
			// Navigate back — upgrade keeps running in background.
			app = app.goBack()
		case "up", "k":
			app.upgradeVP.LineUp(1)
		case "down", "j":
			app.upgradeVP.LineDown(1)
		case "pgup":
			app.upgradeVP.HalfViewUp()
		case "pgdown":
			app.upgradeVP.HalfViewDown()
		case "g":
			app.upgradeVP.GotoTop()
		case "G":
			app.upgradeVP.GotoBottom()
		}
		return app, nil
	}

	if app.upgradeConfirm {
		switch msg.String() {
		case "ctrl+c":
			app.cleanup()
			return app, tea.Quit
		case "y":
			return app.startUpgrade()
		case "n", "esc":
			app.upgradeConfirm = false
		}
		return app, nil
	}

	// Input phase
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "esc":
		app = app.goBack()
		return app, nil

	case "p":
		app.upgradePreserve = !app.upgradePreserve
		return app, nil

	case "enter":
		if strings.TrimSpace(app.upgradeInput.Value()) == "" {
			app.statusMsg = warnStyle.Render("Please enter a value")
			return app, nil
		}
		app.upgradeConfirm = true
		return app, nil
	}

	var cmd tea.Cmd
	app.upgradeInput, cmd = app.upgradeInput.Update(msg)
	return app, cmd
}

func (app App) startUpgrade() (App, tea.Cmd) {
	app.upgradeConfirm = false
	app.upgradeRunning = true
	app.upgradeLines = nil
	app.upgradeVP.SetContent("")

	app.upgradeCh = make(chan string, 500)
	app.upgradeCtx, app.upgradeCancel = context.WithCancel(context.Background())

	val := strings.TrimSpace(app.upgradeInput.Value())
	client := app.client
	node := ""
	if app.selNode != nil {
		node = app.selNode.IP
	}
	forK8s := app.upgradeForK8s
	preserve := app.upgradePreserve

	upgradeCh := app.upgradeCh
	upgradeCtx := app.upgradeCtx
	go func() {
		defer close(upgradeCh)
		var err error
		if forK8s {
			err = client.UpgradeK8s(upgradeCtx, val, upgradeCh)
		} else {
			err = client.UpgradeTalos(upgradeCtx, node, val, preserve, upgradeCh)
		}
		if err != nil && upgradeCtx.Err() == nil {
			upgradeCh <- fmt.Sprintf("ERROR: %v", err)
		}
	}()

	return app, waitForUpgradeLine(app.upgradeCh)
}

func (app App) renderUpgrade(height int) string {
	isK8s := app.upgradeForK8s
	kind := "Talos"
	if isK8s {
		kind = "Kubernetes"
	}

	node := ""
	if app.selNode != nil && !isK8s {
		node = fmt.Sprintf(" on %s", titleStyle.Render(app.selNode.Hostname))
	}
	title := fmt.Sprintf("  Upgrade %s%s\n", titleStyle.Render(kind), node)

	if app.upgradeRunning {
		app.upgradeVP.Height = height - 2
		app.upgradeVP.Width = app.width
		return title + app.upgradeVP.View()
	}

	if app.upgradeConfirm {
		val := app.upgradeInput.Value()
		var msg string
		if isK8s {
			msg = fmt.Sprintf("Upgrade Kubernetes to %s?", okStyle.Render(val))
		} else {
			preserveNote := dimStyle.Render("--preserve=false")
			if app.upgradePreserve {
				preserveNote = okStyle.Render("--preserve=true")
			}
			msg = fmt.Sprintf("Upgrade Talos on %s to image %s  %s",
				titleStyle.Render(app.selNode.Hostname),
				okStyle.Render(val),
				preserveNote)
		}
		confirm := fmt.Sprintf("\n  %s\n\n  %s  %s",
			msg,
			keyStyle.Render("[y]")+" confirm",
			keyStyle.Render("[n/Esc]")+" cancel",
		)
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center, confirm)
	}

	// Show final output if upgrade completed
	if len(app.upgradeLines) > 0 {
		app.upgradeVP.Height = height - 2
		app.upgradeVP.Width = app.width
		return title + app.upgradeVP.View()
	}

	// Input phase
	label := "Image (e.g. ghcr.io/siderolabs/installer:v1.6.x):"
	if isK8s {
		label = "Target Kubernetes version (e.g. 1.29.0):"
	}
	preserveLine := ""
	if !isK8s {
		preserveVal := dimStyle.Render("off")
		if app.upgradePreserve {
			preserveVal = okStyle.Render("on")
		}
		preserveLine = fmt.Sprintf("\n  %s --preserve %s  %s\n",
			keyStyle.Render("[p]"),
			preserveVal,
			dimStyle.Render("(required for single-node etcd clusters)"),
		)
	}
	inputView := fmt.Sprintf("\n  %s\n\n  %s\n%s",
		dimStyle.Render(label),
		app.upgradeInput.View(),
		preserveLine,
	)
	return title + lipgloss.Place(app.width, height-2, lipgloss.Left, lipgloss.Center, inputView)
}
