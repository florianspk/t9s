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
	if app.findActive {
		var cmd tea.Cmd
		app, app.dmesgCur, cmd = app.handleFindKey(msg, app.dmesgLines, app.dmesgCur)
		return app, cmd
	}

	n := len(app.dmesgLines)
	dmesgFindBarH := 0
	if app.findActive || app.findQuery != "" {
		dmesgFindBarH = 1
	}
	dmesgMaxRows := max(1, app.mainHeight()-2-dmesgFindBarH)
	approxDmesgAnchor := max(0, n-dmesgMaxRows)

	updateDmesgScroll := func() {
		if app.dmesgCur >= approxDmesgAnchor {
			app.viewScrollStart = approxDmesgAnchor
		} else {
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.dmesgCur, n, dmesgMaxRows)
		}
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		if app.findQuery != "" {
			app.findQuery = ""
			return app, nil
		}
		app.stopDmesg()
		app = app.goBack()
		return app, nil
	case "up", "k":
		if app.dmesgCur > 0 {
			app.dmesgCur--
		}
		updateDmesgScroll()
	case "down", "j":
		if app.dmesgCur < n-1 {
			app.dmesgCur++
		}
		updateDmesgScroll()
	case "pgup":
		app.dmesgCur = max(0, app.dmesgCur-app.mainHeight()/2)
		updateDmesgScroll()
	case "pgdown":
		app.dmesgCur = min(max(0, n-1), app.dmesgCur+app.mainHeight()/2)
		updateDmesgScroll()
	case "g":
		app.dmesgCur = 0
		updateDmesgScroll()
	case "G":
		app.dmesgCur = max(0, n-1)
		updateDmesgScroll()
	case "/":
		app.findActive = true
		app.findInput.SetValue("")
		return app, app.findInput.Focus()
	case "n":
		if app.findQuery != "" {
			if idx := findLineNext(app.dmesgLines, app.dmesgCur+1, app.findQuery); idx >= 0 {
				app.dmesgCur = idx
			}
		}
	case "N":
		if app.findQuery != "" {
			if idx := findLinePrev(app.dmesgLines, app.dmesgCur-1, app.findQuery); idx >= 0 {
				app.dmesgCur = idx
			}
		}
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

	findBarH := 0
	if app.findActive || app.findQuery != "" {
		findBarH = 1
	}
	content := renderLinesCursor(app.dmesgLines, app.dmesgCur, app.width, height-2-findBarH, app.viewScrollStart, app.findQuery)
	return title + content + app.renderFindBar(app.dmesgLines)
}
