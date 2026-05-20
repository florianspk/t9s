package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) handleHealthKey(msg tea.KeyMsg) (App, tea.Cmd) {
	n := len(app.healthLines)
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		app = app.goBack()
		return app, nil
	case "up", "k":
		if app.healthCur > 0 {
			app.healthCur--
		}
	case "down", "j":
		if app.healthCur < n-1 {
			app.healthCur++
		}
	case "pgup":
		app.healthCur = max(0, app.healthCur-app.mainHeight()/2)
	case "pgdown":
		app.healthCur = min(max(0, n-1), app.healthCur+app.mainHeight()/2)
	case "g":
		app.healthCur = 0
	case "G":
		app.healthCur = max(0, n-1)
	}
	return app, nil
}

func (app App) renderHealth(height int) string {
	title := "  Cluster Health\n"

	if app.healthStreaming && len(app.healthLines) == 0 {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Running health check…"))
	}

	return title + renderLinesCursor(app.healthLines, app.healthCur, app.width, height-2)
}

func colorHealthLine(line string) string {
	l := strings.ToLower(line)
	switch {
	case strings.HasPrefix(l, "ok") || strings.Contains(l, "healthy") || strings.Contains(l, "running"):
		return okStyle.Render(line)
	case strings.Contains(l, "error") || strings.Contains(l, "fail") || strings.Contains(l, "unhealthy"):
		return errStyle.Render(line)
	case strings.HasPrefix(l, "waiting") || strings.Contains(l, "warn"):
		return warnStyle.Render(line)
	default:
		return dimStyle.Render(line)
	}
}

func startHealth(app App) (App, tea.Cmd) {
	app.stopHealth()
	app.healthLines = nil
	app.healthCur = 0
	app.healthStreaming = true
	app.healthCh = make(chan string, 200)
	app.healthCtx, app.healthCancel = context.WithCancel(context.Background())
	client := app.client
	ch := app.healthCh
	ctx := app.healthCtx
	go func() {
		defer close(ch)
		client.StreamHealth(ctx, ch)
	}()
	return app, waitForHealthLine(app.healthCh)
}
