package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/florianspk/t9s/internal/talos"
)

func logStreamsRuneKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func logStreamsTestInput(value string) textinput.Model {
	input := textinput.New()
	input.SetValue(value)
	return input
}

func TestApp_NodeLogStreamsKey_entersPickerForFilteredSelectedNode(t *testing.T) {
	// Given
	app := App{
		width:           100,
		height:          30,
		state:           StateNodeList,
		nodes:           makeNodes(2),
		nodeCur:         0,
		logStreams:      []string{"stale"},
		logStreamCur:    4,
		searchInput:     logStreamsTestInput("node-01"),
		viewScrollStart: 3,
	}

	// When
	model, cmd := app.Update(logStreamsRuneKey('l'))
	got := model.(App)

	// Then
	if got.state != StateLogStreams || got.navTop() != StateNodeList {
		t.Fatalf("state = %v, back = %v; want LogStreams from NodeList", got.state, got.navTop())
	}
	if got.selNode == nil || got.selNode.Hostname != "talos-node-01.example.internal" {
		t.Fatalf("selected node = %#v, want filtered node 01", got.selNode)
	}
	if len(got.logStreams) != 0 || got.logStreamCur != 0 || !got.logStreamLoading {
		t.Fatalf("picker state = streams %v, cursor %d, loading %v", got.logStreams, got.logStreamCur, got.logStreamLoading)
	}
	if got.logStreamRequestSeq != 1 || got.logStreamRequestNode != got.selNode.IP {
		t.Fatalf("active request = node %q sequence %d, want selected node sequence 1", got.logStreamRequestNode, got.logStreamRequestSeq)
	}
	if got.viewScrollStart != 0 || got.searchInput.Value() != "" {
		t.Fatalf("list reset = scroll %d, query %q", got.viewScrollStart, got.searchInput.Value())
	}
	if !strings.Contains(got.statusMsg, "Loading log streams") {
		t.Fatalf("status = %q, want loading status", got.statusMsg)
	}
	if cmd == nil {
		t.Fatal("node log-stream transition returned no load command")
	}
}

func TestApp_NodeLogStreamsKey_doesNothingWithoutFilteredSelection(t *testing.T) {
	// Given
	app := App{
		state:       StateNodeList,
		nodes:       makeNodes(2),
		searchInput: logStreamsTestInput("no-such-node"),
	}

	// When
	model, cmd := app.Update(logStreamsRuneKey('l'))
	got := model.(App)

	// Then
	if got.state != StateNodeList || got.selNode != nil {
		t.Fatalf("state = %v, selected node = %#v; want unchanged node list", got.state, got.selNode)
	}
	if cmd != nil {
		t.Fatal("missing selection returned a command")
	}
}

func TestApp_LogStreamsLoaded_storesNamesAndClearsLoading(t *testing.T) {
	// Given
	node := makeNodes(1)[0]
	app := App{
		state:                StateLogStreams,
		selNode:              &node,
		logStreamLoading:     true,
		logStreamCur:         8,
		logStreamRequestNode: node.IP,
		logStreamRequestSeq:  4,
	}
	want := []string{"apid", "kubelet", "containerd"}

	// When
	model, cmd := app.Update(logStreamsLoadedMsg{streams: want, nodeIP: node.IP, sequence: 4})
	got := model.(App)

	// Then
	if !reflect.DeepEqual(got.logStreams, want) {
		t.Fatalf("log streams = %v, want %v", got.logStreams, want)
	}
	if got.logStreamLoading || got.logStreamCur != 0 {
		t.Fatalf("loading = %v, cursor = %d; want false, 0", got.logStreamLoading, got.logStreamCur)
	}
	if got.statusMsg != "3 log streams" {
		t.Fatalf("status = %q, want stream count", got.statusMsg)
	}
	if cmd != nil {
		t.Fatal("loaded message returned a command")
	}
}

func TestApp_LogStreamsLoaded_reportsErrorWithoutReplacingNames(t *testing.T) {
	// Given
	node := makeNodes(1)[0]
	app := App{
		state:                StateLogStreams,
		selNode:              &node,
		logStreams:           []string{"existing"},
		logStreamLoading:     true,
		logStreamRequestNode: node.IP,
		logStreamRequestSeq:  5,
	}

	// When
	model, _ := app.Update(logStreamsLoadedMsg{
		streams:  []string{"ignored"},
		err:      errors.New("permission denied"),
		nodeIP:   node.IP,
		sequence: 5,
	})
	got := model.(App)

	// Then
	if got.logStreamLoading {
		t.Fatal("loading remained active after an error")
	}
	if !reflect.DeepEqual(got.logStreams, []string{"existing"}) {
		t.Fatalf("log streams = %v, want existing data preserved", got.logStreams)
	}
	if !strings.Contains(got.statusMsg, "Error: permission denied") {
		t.Fatalf("status = %q, want rendered error", got.statusMsg)
	}
}

func TestApp_LogStreamsLoaded_ignoresReplyOutsidePicker(t *testing.T) {
	// Given
	app := App{
		state:            StateNodeList,
		logStreams:       []string{"current"},
		logStreamCur:     3,
		logStreamLoading: true,
		statusMsg:        "Refreshing log streams...",
	}

	// When
	model, cmd := app.Update(logStreamsLoadedMsg{streams: []string{"stale"}})
	got := model.(App)

	// Then
	if !reflect.DeepEqual(got.logStreams, app.logStreams) ||
		got.logStreamCur != app.logStreamCur ||
		got.logStreamLoading != app.logStreamLoading ||
		got.statusMsg != app.statusMsg {
		t.Fatalf("stale reply changed picker: streams=%v cursor=%d loading=%v status=%q", got.logStreams, got.logStreamCur, got.logStreamLoading, got.statusMsg)
	}
	if cmd != nil {
		t.Fatal("stale reply returned a command")
	}
}

func TestApp_LogStreamsLoaded_appliesActiveReplyWhilePickerOverlayIsOpen(t *testing.T) {
	node := makeNodes(1)[0]
	for _, overlay := range []AppState{StateHelp, StateContextSwitcher} {
		t.Run(viewTitle(overlay), func(t *testing.T) {
			// Given
			app := App{
				state:                overlay,
				navStack:             []navEntry{{state: StateLogStreams, node: &node}},
				selNode:              &node,
				logStreamLoading:     true,
				logStreamRequestNode: node.IP,
				logStreamRequestSeq:  9,
			}

			// When
			model, cmd := app.Update(logStreamsLoadedMsg{
				streams:  []string{"kubelet"},
				nodeIP:   node.IP,
				sequence: 9,
			})
			got := model.(App)

			// Then
			if !reflect.DeepEqual(got.logStreams, []string{"kubelet"}) || got.logStreamLoading {
				t.Fatalf("active overlay reply = streams %v, loading %v; want applied result", got.logStreams, got.logStreamLoading)
			}
			if got.state != overlay {
				t.Fatalf("state = %v, want overlay %v retained", got.state, overlay)
			}
			if cmd != nil {
				t.Fatal("loaded message returned a command")
			}
		})
	}
}

func TestApp_LogStreamsLoaded_ignoresReplyWithoutActiveRequestIdentity(t *testing.T) {
	selectedNode := makeNodes(1)[0]
	otherNode := selectedNode
	otherNode.IP = "10.0.0.99"
	tests := []struct {
		name      string
		selected  *talos.Node
		messageIP string
		sequence  uint64
	}{
		{name: "message node differs", selected: &selectedNode, messageIP: otherNode.IP, sequence: 7},
		{name: "message sequence differs", selected: &selectedNode, messageIP: selectedNode.IP, sequence: 6},
		{name: "selected node changed", selected: &otherNode, messageIP: selectedNode.IP, sequence: 7},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			app := App{
				state:                StateLogStreams,
				selNode:              test.selected,
				logStreams:           []string{"current"},
				logStreamCur:         3,
				logStreamLoading:     true,
				logStreamRequestNode: selectedNode.IP,
				logStreamRequestSeq:  7,
				statusMsg:            "Refreshing log streams...",
			}

			// When
			model, cmd := app.Update(logStreamsLoadedMsg{
				streams:  []string{"stale"},
				nodeIP:   test.messageIP,
				sequence: test.sequence,
			})
			got := model.(App)

			// Then
			if !reflect.DeepEqual(got.logStreams, app.logStreams) ||
				got.logStreamCur != app.logStreamCur ||
				got.logStreamLoading != app.logStreamLoading ||
				got.statusMsg != app.statusMsg {
				t.Fatalf("stale reply changed picker: streams=%v cursor=%d loading=%v status=%q", got.logStreams, got.logStreamCur, got.logStreamLoading, got.statusMsg)
			}
			if cmd != nil {
				t.Fatal("stale reply returned a command")
			}
		})
	}
}

func TestApp_FilteredLogStreams_matchesCaseInsensitively(t *testing.T) {
	// Given
	app := App{
		logStreams:  []string{"APID", "kubelet", "apid-controller"},
		searchInput: logStreamsTestInput("ApiD"),
	}

	// When
	got := app.filteredLogStreams()

	// Then
	want := []string{"APID", "apid-controller"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered log streams = %v, want %v", got, want)
	}
}

func TestApp_ClampCursor_clampsLogStreamCursorAndScrollToFilteredBounds(t *testing.T) {
	// Given
	app := App{
		width:           100,
		height:          30,
		state:           StateLogStreams,
		logStreams:      []string{"apid", "kubelet", "containerd"},
		logStreamCur:    2,
		searchInput:     logStreamsTestInput("kube"),
		viewScrollStart: 2,
	}

	// When
	app.clampCursor()

	// Then
	if app.logStreamCur != 0 || app.viewScrollStart != 0 {
		t.Fatalf("cursor = %d, scroll = %d; want both clamped to 0", app.logStreamCur, app.viewScrollStart)
	}
}
