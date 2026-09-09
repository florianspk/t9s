package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/florianspk/t9s/internal/talos"
)

func TestRunCommandAliases(t *testing.T) {
	cases := []struct {
		in   string
		want AppState
	}{
		{"nodes", StateNodeList},
		{"members", StateNodeList},
		{"disks", StateDisks},
		{"lvm", StateLVM},
		{"svc", StateServices},
		{"services", StateServices},
		{"ext", StateExtensions},
		{"mc", StateMachineConfig},
		{"metrics", StateMetrics},
		{"ps", StateProcesses},
		{"containers", StateContainers},
		{"addr", StateAddresses},
	}
	for _, tc := range cases {
		app := newTestApp(120, 40)
		app.cmdInput = textinput.New()
		app.selNode = &talos.Node{Hostname: "n1", IP: "10.0.0.1", Version: "v1.14.0"}
		app.nodes = []talos.Node{*app.selNode}

		got, _ := app.runCommand(tc.in)
		if got.state != tc.want {
			t.Errorf("runCommand(%q) → state %v, want %v", tc.in, got.state, tc.want)
		}
	}
}

func TestRunCommandUnknownResourceGoesToBrowser(t *testing.T) {
	app := newTestApp(120, 40)
	app.cmdInput = textinput.New()
	app.selNode = &talos.Node{Hostname: "n1", IP: "10.0.0.1"}
	app.nodes = []talos.Node{*app.selNode}

	got, cmd := app.runCommand("mounts")
	if got.state != StateResourceBrowser {
		t.Fatalf("state = %v, want StateResourceBrowser", got.state)
	}
	if got.resBrowserKind != "mounts" {
		t.Errorf("resBrowserKind = %q, want mounts", got.resBrowserKind)
	}
	if cmd == nil {
		t.Errorf("expected a load command")
	}
}

func TestRunCommandNodeScopedWithoutNode(t *testing.T) {
	app := newTestApp(120, 40)
	app.cmdInput = textinput.New()
	// no nodes, no selNode
	got, _ := app.runCommand("disks")
	if got.state == StateDisks {
		t.Errorf("should not switch to Disks without a node")
	}
}
