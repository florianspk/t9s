package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/florianspk/t9s/internal/config"
	"github.com/florianspk/t9s/internal/talos"
)

func TestApp_LogStreamsNavigation_supportsPageAndBoundaryKeys(t *testing.T) {
	// Given
	streams := make([]string, 30)
	for i := range streams {
		streams[i] = "stream"
	}
	app := App{width: 100, height: 20, state: StateLogStreams, logStreams: streams}

	// When / Then
	app, _ = app.handleLogStreamsKey(tea.KeyMsg{Type: tea.KeyEnd})
	if app.logStreamCur != len(streams)-1 {
		t.Fatalf("end cursor = %d, want %d", app.logStreamCur, len(streams)-1)
	}
	app, _ = app.handleLogStreamsKey(tea.KeyMsg{Type: tea.KeyHome})
	if app.logStreamCur != 0 {
		t.Fatalf("home cursor = %d, want 0", app.logStreamCur)
	}
	app, _ = app.handleLogStreamsKey(tea.KeyMsg{Type: tea.KeyPgDown})
	if app.logStreamCur == 0 {
		t.Fatal("page down did not advance the cursor")
	}
	app, _ = app.handleLogStreamsKey(tea.KeyMsg{Type: tea.KeyPgUp})
	if app.logStreamCur != 0 {
		t.Fatalf("page up cursor = %d, want 0", app.logStreamCur)
	}
	app, _ = app.handleLogStreamsKey(logStreamsRuneKey('j'))
	app, _ = app.handleLogStreamsKey(logStreamsRuneKey('k'))
	if app.logStreamCur != 0 {
		t.Fatalf("j then k cursor = %d, want 0", app.logStreamCur)
	}
}

func TestApp_LogStreamsSearch_activatesThroughGlobalRouter(t *testing.T) {
	// Given
	app := App{state: StateLogStreams, searchInput: textinput.New()}

	// When
	got, cmd := app.handleKey(logStreamsRuneKey('/'))

	// Then
	if !got.searchActive {
		t.Fatal("global search did not activate for log streams")
	}
	if cmd == nil {
		t.Fatal("search activation returned no focus command")
	}
}

func TestApp_LogStreamsEnter_startsSelectedTargetWithoutRunningCommand(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{
		state:        StateLogStreams,
		selNode:      &node,
		logStreams:   []string{"apid", "kubelet", "controller-runtime"},
		logStreamCur: 0,
		logLines:     []string{"stale"},
		searchInput:  logStreamsTestInput("KUBE"),
	}

	// When
	got, cmd := app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Then
	if got.state != StateLogs || got.navTop() != StateLogStreams || got.logService != "kubelet" {
		t.Fatalf("state = %v, back = %v, target = %q", got.state, got.navTop(), got.logService)
	}
	if !got.logStreaming || got.logCh == nil || got.logCtx == nil || got.logCancel == nil {
		t.Fatal("log stream resources were not initialized")
	}
	if len(got.logLines) != 0 || got.logCur != 0 {
		t.Fatalf("log viewer was not reset: lines %v, cursor %d", got.logLines, got.logCur)
	}
	if cmd == nil {
		t.Fatal("starting logs returned no command")
	}
	got.stopLogs()
}

func TestApp_LogStreamsReload_marksLoadingAndReturnsCommandWithoutRunningIt(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{
		state:                StateLogStreams,
		selNode:              &node,
		logStreams:           []string{"apid"},
		logStreamRequestSeq:  7,
		logStreamRequestNode: node.IP,
		searchInput:          textinput.New(),
	}

	// When
	got, cmd := app.handleKey(logStreamsRuneKey('r'))

	// Then
	if !got.logStreamLoading || !strings.Contains(got.statusMsg, "Refreshing log streams") {
		t.Fatalf("loading = %v, status = %q", got.logStreamLoading, got.statusMsg)
	}
	if got.logStreamRequestSeq != 8 || got.logStreamRequestNode != node.IP {
		t.Fatalf("active request = node %q sequence %d, want %q sequence 8", got.logStreamRequestNode, got.logStreamRequestSeq, node.IP)
	}
	if cmd == nil {
		t.Fatal("reload returned no command")
	}
}

func TestApp_ServicesEnter_stillStartsServiceLogsWithoutRunningCommand(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{
		state:       StateServices,
		selNode:     &node,
		services:    []talos.Service{{ID: "etcd", State: "Running", Healthy: "OK"}},
		searchInput: textinput.New(),
	}

	// When
	got, cmd := app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Then
	if got.state != StateLogs || got.navTop() != StateServices || got.logService != "etcd" {
		t.Fatalf("state = %v, back = %v, service = %q", got.state, got.navTop(), got.logService)
	}
	if !got.logStreaming || cmd == nil {
		t.Fatal("service log behavior did not initialize streaming")
	}
	got.stopLogs()
}

func TestApp_LogsBack_returnsToItsServiceOrStreamSource(t *testing.T) {
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	tests := []struct {
		name string
		prev AppState
		want AppState
	}{
		{name: "service logs", prev: StateServices, want: StateServices},
		{name: "dynamic stream logs", prev: StateLogStreams, want: StateLogStreams},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			app := App{state: StateLogs, navStack: []navEntry{{state: StateNodeList}, {state: test.prev, node: &node}}, selNode: &node, searchInput: textinput.New()}

			// When
			got, cmd := app.handleLogsKey(tea.KeyMsg{Type: tea.KeyEsc})

			// Then
			if got.state != test.want || got.selNode == nil {
				t.Fatalf("state = %v, selected node = %#v; want %v with node retained", got.state, got.selNode, test.want)
			}
			if cmd != nil {
				t.Fatal("back navigation returned a command")
			}
		})
	}
}

func TestApp_LogsBack_returnsToOriginAfterOverlayRoundTrip(t *testing.T) {
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	cfg := &config.TalosConfig{
		Context:  "default",
		Contexts: map[string]config.Context{"default": {}},
	}
	tests := []struct {
		name    string
		origin  AppState
		overlay rune
	}{
		{name: "log streams after help", origin: StateLogStreams, overlay: '?'},
		{name: "log streams after context switcher", origin: StateLogStreams, overlay: 'x'},
		{name: "services after help", origin: StateServices, overlay: '?'},
		{name: "services after context switcher", origin: StateServices, overlay: 'x'},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			app := App{
				state:       test.origin,
				selNode:     &node,
				cfg:         cfg,
				searchInput: textinput.New(),
			}
			logs, streamCmd := app.startLogStream("kubelet")
			if streamCmd == nil {
				t.Fatal("starting logs returned no command")
			}

			// When
			overlay, _ := logs.handleKey(logStreamsRuneKey(test.overlay))
			logsAgain, _ := overlay.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
			got, backCmd := logsAgain.handleLogsKey(tea.KeyMsg{Type: tea.KeyEsc})

			// Then
			if got.state != test.origin {
				t.Fatalf("state after %q overlay = %v, want log origin %v", test.overlay, got.state, test.origin)
			}
			if backCmd != nil {
				t.Fatal("logs back returned a command")
			}
			got.stopLogs()
		})
	}
}

func TestApp_LogStreamsBack_returnsToNodeListAndClearsSelection(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{state: StateLogStreams, navStack: []navEntry{{state: StateNodeList}}, selNode: &node, searchInput: textinput.New()}

	// When
	got, cmd := app.handleLogStreamsKey(tea.KeyMsg{Type: tea.KeyEsc})

	// Then
	if got.state != StateNodeList || got.selNode != nil {
		t.Fatalf("state = %v, selected node = %#v; want node list with no selection", got.state, got.selNode)
	}
	if cmd != nil {
		t.Fatal("back navigation returned a command")
	}
}
