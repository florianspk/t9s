package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app App) startLogStreamsLoad() (App, tea.Cmd) {
	app.logStreamLoading = true
	app.logStreamRequestSeq++
	app.logStreamRequestNode = app.selNode.IP

	client := app.client
	nodeIP := app.logStreamRequestNode
	sequence := app.logStreamRequestSeq

	return app, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		streams, err := client.GetLogStreams(ctx, nodeIP)

		return logStreamsLoadedMsg{streams: streams, nodeIP: nodeIP, sequence: sequence, err: err}
	}
}

func (app App) filteredLogStreams() []string {
	query := app.searchQuery()
	if query == "" {
		return app.logStreams
	}

	var streams []string
	for _, stream := range app.logStreams {
		if matchSearch(query, stream) {
			streams = append(streams, stream)
		}
	}

	return streams
}

func (app App) handleLogStreamsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	streams := app.filteredLogStreams()
	app.logStreamCur = clamp(app.logStreamCur, 0, max(0, len(streams)-1))
	maxRows := max(1, app.mainHeight()-3)

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "up", "k":
		app.logStreamCur = max(0, app.logStreamCur-1)
	case "down", "j":
		app.logStreamCur = min(max(0, len(streams)-1), app.logStreamCur+1)
	case "pgup":
		app.logStreamCur = max(0, app.logStreamCur-maxRows)
	case "pgdown":
		app.logStreamCur = min(max(0, len(streams)-1), app.logStreamCur+maxRows)
	case "home", "g":
		app.logStreamCur = 0
	case "end", "G":
		app.logStreamCur = max(0, len(streams)-1)
	case "enter":
		if len(streams) == 0 || app.selNode == nil {
			return app, nil
		}
		return app.startLogStream(streams[app.logStreamCur])
	case "r":
		if app.selNode == nil {
			return app, nil
		}
		app.statusMsg = "Refreshing log streams..."
		return app.startLogStreamsLoad()
	case "esc", "q":
		return app.goBack(), nil
	}

	app.viewScrollStart = clampScrollStart(
		app.viewScrollStart,
		app.logStreamCur,
		len(streams),
		maxRows,
	)

	return app, nil
}

func waitForLine(ch <-chan string, sessionSeq uint64) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{sessionSeq: sessionSeq}
		}
		return logLineMsg{line: line, sessionSeq: sessionSeq}
	}
}

func (app App) startLogStream(target string) (App, tea.Cmd) {
	app.stopLogs()
	app.logSessionSeq++
	app.logService = target
	app.logLines = nil
	app.logCur = 0
	app.logStreaming = true
	app = app.goTo(StateLogs)
	app.logCh = make(chan string, 500)
	app.logCtx, app.logCancel = context.WithCancel(context.Background())

	node := app.selNode.IP
	logCh := app.logCh
	logCtx := app.logCtx
	logSessionSeq := app.logSessionSeq
	runLogStream := app.runLogStream

	return app, func() tea.Msg {
		go func() {
			defer close(logCh)
			runLogStream(logCtx, node, target, logCh)
		}()

		return waitForLine(logCh, logSessionSeq)()
	}
}

func (app App) renderLogStreams(height int) string {
	if app.width <= 0 || height <= 0 {
		return ""
	}

	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	widthStyle := lipgloss.NewStyle().MaxWidth(app.width)
	title := widthStyle.Render(fmt.Sprintf("  Log Streams on %s", titleStyle.Render(node)))
	if height == 1 {
		return title
	}

	streams := app.filteredLogStreams()
	renderTarget := func(target string) string {
		gutter := ""
		targetWidth := app.width
		if app.width >= 3 {
			gutter = "▶ "
			targetWidth -= 2
		}
		row := gutter + lipgloss.NewStyle().MaxWidth(targetWidth).Render(target)
		return selectedStyle.Width(app.width).MaxWidth(app.width).Render(row)
	}
	status := func() string {
		if app.logStreamLoading && len(app.logStreams) == 0 {
			return widthStyle.Render(infoStyle.Render("Loading log streams…"))
		}
		if len(streams) > 0 {
			return renderTarget(streams[app.logStreamCur])
		}
		message := "No log streams found."
		if app.searchInput.Value() != "" {
			message = "No match for \"" + app.searchInput.Value() + "\""
		}
		return widthStyle.Render(warnStyle.Render(message))
	}()
	if height == 2 {
		return title + "\n" + status
	}

	if app.logStreamLoading && len(app.logStreams) == 0 {
		return title + "\n" + lipgloss.Place(app.width, height-1, lipgloss.Center, lipgloss.Center, status)
	}

	if len(streams) == 0 {
		return title + "\n" + lipgloss.Place(app.width, height-1, lipgloss.Center, lipgloss.Center, status)
	}

	var output strings.Builder
	output.WriteString(title)
	output.WriteByte('\n')
	output.WriteString(widthStyle.Render(colHeaderStyle.Render("  TARGET")))
	output.WriteByte('\n')

	maxRows := height - 2
	start := clampScrollStart(app.viewScrollStart, app.logStreamCur, len(streams), maxRows)
	gutterWidth := 0
	if app.width >= 3 {
		gutterWidth = 2
	}
	targetWidth := app.width - gutterWidth
	targetStyle := lipgloss.NewStyle().MaxWidth(targetWidth)

	for index := start; index < len(streams) && index < start+maxRows; index++ {
		cursor := ""
		if gutterWidth > 0 {
			cursor = "  "
			if index == app.logStreamCur {
				cursor = "▶ "
			}
		}
		row := cursor + targetStyle.Render(streams[index])
		if index == app.logStreamCur {
			output.WriteString(selectedStyle.Width(app.width).MaxWidth(app.width).Render(row))
		} else {
			output.WriteString(row)
		}
		output.WriteByte('\n')
	}

	return output.String()
}
