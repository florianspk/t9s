package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func waitForDmesgLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return dmesgDoneMsg{}
		}
		return dmesgLineMsg(line)
	}
}

func (app App) handleDmesgKey(msg tea.KeyMsg) (App, tea.Cmd) {
	n := len(app.dmesgLines)
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app.stopDmesg()
		app = app.goBack()
		return app, nil
	case "up", "k":
		if app.dmesgCur > 0 {
			app.dmesgCur--
		}
	case "down", "j":
		if app.dmesgCur < n-1 {
			app.dmesgCur++
		}
	case "pgup":
		app.dmesgCur = max(0, app.dmesgCur-app.mainHeight()/2)
	case "pgdown":
		app.dmesgCur = min(max(0, n-1), app.dmesgCur+app.mainHeight()/2)
	case "g":
		app.dmesgCur = 0
	case "G":
		app.dmesgCur = max(0, n-1)
	}
	return app, nil
}

func (app App) renderDmesg(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	streaming := ""
	if app.dmesgStreaming {
		streaming = infoStyle.Render(" [streaming]")
	} else {
		streaming = dimStyle.Render(" [stopped]")
	}
	title := fmt.Sprintf("  Dmesg: %s%s\n", titleStyle.Render(node), streaming)

	if len(app.dmesgLines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Waiting for dmesg…"))
	}

	return title + renderLinesCursor(app.dmesgLines, app.dmesgCur, app.width, height-2)
}
