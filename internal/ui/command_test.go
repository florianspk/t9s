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

func TestResourceRowKey(t *testing.T) {
	const hdr = "NODE           NAMESPACE   TYPE          ID          VERSION   SOURCE"
	node, id, ok := resourceRowKey(hdr, "10.17.84.213   runtime     MountStatus   EPHEMERAL   1         /dev/sda4")
	if !ok || node != "10.17.84.213" || id != "EPHEMERAL" {
		t.Errorf("got (%q, %q, %v)", node, id, ok)
	}
	// Not a `talosctl get` table (e.g. `talosctl mounts`, or YAML mode).
	if _, _, ok := resourceRowKey("NODE  FILESYSTEM  SIZE(GB)", "10.0.0.1  none  4.08"); ok {
		t.Error("non-resource table must not yield a key")
	}
	if _, _, ok := resourceRowKey(hdr, "10.17.84.213   runtime"); ok {
		t.Error("short row must not yield a key")
	}
}

func TestMatchCommands(t *testing.T) {
	// "environment" is a tighter m-n-t subsequence than "mounts", but only
	// "mounts" starts with the query's first letter — that must win.
	kinds := []string{"environment", "mounts", "mountstatus", "ms", "routes", "members", "disks"}
	cases := []struct {
		in   string
		want string // expected first match
	}{
		{"nodes", "nodes"},  // exact view alias
		{"disk", "disks"},   // prefix, view alias wins
		{"mount", "mounts"}, // prefix on a resource kind
		{"mnt", "mounts"},   // subsequence abbreviation
		{"hea", "health"},   // prefix on a view alias
		{"route", "routes"}, // singular -> plural by prefix
	}
	for _, tc := range cases {
		got := matchCommands(tc.in, kinds)
		if len(got) == 0 {
			t.Errorf("matchCommands(%q) = no matches", tc.in)
			continue
		}
		if got[0] != tc.want {
			t.Errorf("matchCommands(%q)[0] = %q, want %q (all: %v)", tc.in, got[0], tc.want, got)
		}
	}
	if got := matchCommands("", kinds); got != nil {
		t.Errorf("empty prefix must not match, got %v", got)
	}
	if got := matchCommands("zzzqqq", kinds); len(got) != 0 {
		t.Errorf("nonsense prefix matched %v", got)
	}
	if got := matchCommands("s", kinds); len(got) > maxCmdMatches {
		t.Errorf("matches not capped: %d", len(got))
	}
}

func TestIsSubsequence(t *testing.T) {
	for _, tc := range []struct {
		short, long string
		want        bool
	}{
		{"mnt", "mounts", true},
		{"ms", "mountstatus", true},
		{"tsm", "mounts", false},
		{"", "mounts", true},
		{"mounts", "mnt", false},
	} {
		if got := isSubsequence(tc.short, tc.long); got != tc.want {
			t.Errorf("isSubsequence(%q, %q) = %v, want %v", tc.short, tc.long, got, tc.want)
		}
	}
}
