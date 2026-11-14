package ui

import (
	"reflect"
	"testing"
)

func TestApp_LogStreamsLoaded_ignoresMatchingReplyWithoutPickerOwnership(t *testing.T) {
	node := makeNodes(1)[0]
	tests := []struct {
		name  string
		state AppState
		prev  AppState
	}{
		{name: "services", state: StateServices},
		{name: "logs", state: StateLogs},
		{name: "node list", state: StateNodeList},
		{name: "help from services", state: StateHelp, prev: StateServices},
		{name: "context switcher from services", state: StateContextSwitcher, prev: StateServices},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			app := App{
				state:                test.state,
				navStack:             []navEntry{{state: test.prev}},
				selNode:              &node,
				logStreams:           []string{"current"},
				logStreamCur:         3,
				logStreamLoading:     true,
				logStreamRequestNode: node.IP,
				logStreamRequestSeq:  7,
				statusMsg:            "Refreshing log streams...",
				viewScrollStart:      2,
			}

			// When
			model, cmd := app.Update(logStreamsLoadedMsg{
				streams:  []string{"replacement"},
				nodeIP:   node.IP,
				sequence: 7,
			})
			got := model.(App)

			// Then
			if !reflect.DeepEqual(got.logStreams, app.logStreams) ||
				got.logStreamCur != app.logStreamCur ||
				got.logStreamLoading != app.logStreamLoading ||
				got.statusMsg != app.statusMsg ||
				got.viewScrollStart != app.viewScrollStart {
				t.Fatalf(
					"unowned reply changed picker: streams=%v cursor=%d loading=%v status=%q scroll=%d",
					got.logStreams,
					got.logStreamCur,
					got.logStreamLoading,
					got.statusMsg,
					got.viewScrollStart,
				)
			}
			if cmd != nil {
				t.Fatal("unowned reply returned a command")
			}
		})
	}
}
