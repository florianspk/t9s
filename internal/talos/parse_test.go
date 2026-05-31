package talos

import (
	"strings"
	"testing"
)

// ── parseJSONStream ───────────────────────────────────────────────────────────

type testItem struct {
	Name string `json:"name"`
	Val  int    `json:"val"`
}

func TestParseJSONStreamNDJSON(t *testing.T) {
	input := []byte(`{"name":"foo","val":1}` + "\n" + `{"name":"bar","val":2}`)
	got, err := parseJSONStream[testItem](input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got[0].Name != "foo" || got[1].Name != "bar" {
		t.Errorf("unexpected items: %v", got)
	}
}

func TestParseJSONStreamPrettyPrinted(t *testing.T) {
	input := []byte("{\n  \"name\": \"baz\",\n  \"val\": 3\n}")
	got, err := parseJSONStream[testItem](input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "baz" || got[0].Val != 3 {
		t.Errorf("unexpected items: %v", got)
	}
}

func TestParseJSONStreamEmpty(t *testing.T) {
	got, err := parseJSONStream[testItem]([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %d items", len(got))
	}
}

func TestParseJSONStreamSkipsMalformedToken(t *testing.T) {
	// One valid object, one non-JSON line, one valid object.
	// The parser should skip the bad token and continue.
	input := []byte(`{"name":"ok","val":1}` + "\n" + `not-json` + "\n" + `{"name":"also-ok","val":2}`)
	got, err := parseJSONStream[testItem](input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Error("want at least 1 valid item after skipping malformed token")
	}
	// At least the first object must be present.
	found := false
	for _, item := range got {
		if item.Name == "ok" {
			found = true
		}
	}
	if !found {
		t.Error("first valid item 'ok' not found in results")
	}
}

func TestParseJSONStreamMultipleObjects(t *testing.T) {
	var lines []string
	for i := range 5 {
		lines = append(lines, `{"name":"item","val":`+string(rune('0'+i))+`}`)
	}
	got, err := parseJSONStream[testItem]([]byte(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("want 5, got %d", len(got))
	}
}

// ── parseServiceLines ─────────────────────────────────────────────────────────

func TestParseServicesSkipsHeader(t *testing.T) {
	// The GetServices function parses tabular output from talosctl services.
	// We test the parsing logic by constructing realistic input.
	input := "NODE        SERVICE   STATE    HEALTH\n" +
		"10.0.0.1    apid      Running  OK\n" +
		"10.0.0.1    etcd      Running  OK\n"
	svcs := parseServiceLines([]byte(input))
	if len(svcs) != 2 {
		t.Fatalf("want 2 services, got %d", len(svcs))
	}
	if svcs[0].ID != "apid" {
		t.Errorf("ID: want apid, got %q", svcs[0].ID)
	}
	if svcs[0].State != "Running" {
		t.Errorf("State: want Running, got %q", svcs[0].State)
	}
	if svcs[0].Healthy != "OK" {
		t.Errorf("Healthy: want OK, got %q", svcs[0].Healthy)
	}
}

func TestParseServicesSkipsBlankLines(t *testing.T) {
	input := "NODE  SERVICE  STATE  HEALTH\n" +
		"\n" +
		"10.0.0.1  apid  Running  OK\n"
	svcs := parseServiceLines([]byte(input))
	if len(svcs) != 1 {
		t.Fatalf("want 1, got %d", len(svcs))
	}
}

func TestParseServicesSkipsTooFewFields(t *testing.T) {
	input := "NODE  SERVICE  STATE  HEALTH\n" +
		"10.0.0.1  apid\n" // only 2 fields — malformed
	svcs := parseServiceLines([]byte(input))
	if len(svcs) != 0 {
		t.Fatalf("malformed line must be skipped, got %d", len(svcs))
	}
}

// ── parseContainerLines ───────────────────────────────────────────────────────

func TestParseContainerLinesSystemNoImage(t *testing.T) {
	// System containers (apid, trustd) often have no image reference.
	// The tabwriter leaves the IMAGE column empty, so strings.Fields yields
	// only 5 tokens instead of 6.  Regression: was silently dropped.
	input := []byte(
		"NODE        NAMESPACE  ID      IMAGE  PID   STATUS\n" +
			"10.5.0.2    system     apid          2445  RUNNING\n" +
			"10.5.0.2    system     trustd        2615  RUNNING\n",
	)
	got := parseContainerLines(input)
	if len(got) != 2 {
		t.Fatalf("want 2 containers, got %d", len(got))
	}
	if got[0].ID != "apid" {
		t.Errorf("ID: want apid, got %q", got[0].ID)
	}
	if got[0].Image != "" {
		t.Errorf("Image should be empty for no-image container, got %q", got[0].Image)
	}
	if got[0].PID != "2445" {
		t.Errorf("PID: want 2445, got %q", got[0].PID)
	}
	if got[0].Status != "RUNNING" {
		t.Errorf("Status: want RUNNING, got %q", got[0].Status)
	}
}

func TestParseContainerLinesSystemWithImage(t *testing.T) {
	input := []byte(
		"NODE       NAMESPACE  ID   IMAGE              PID  STATUS\n" +
			"10.5.0.2   system     foo  ghcr.io/bar:v1.0   123  RUNNING\n",
	)
	got := parseContainerLines(input)
	if len(got) != 1 {
		t.Fatalf("want 1 container, got %d", len(got))
	}
	if got[0].Image != "ghcr.io/bar:v1.0" {
		t.Errorf("Image: want ghcr.io/bar:v1.0, got %q", got[0].Image)
	}
}

func TestParseContainerLinesK8sSandbox(t *testing.T) {
	// Regular k8s sandbox — 6 fields.
	input := []byte(
		"NODE       NAMESPACE  ID                     IMAGE                      PID   STATUS\n" +
			"10.5.0.2   k8s.io     kube-system/coredns    registry.k8s.io/pause:3.8  1481  SANDBOX_READY\n",
	)
	got := parseContainerLines(input)
	if len(got) != 1 {
		t.Fatalf("want 1 container, got %d", len(got))
	}
	if got[0].Namespace != "k8s.io" {
		t.Errorf("Namespace: want k8s.io, got %q", got[0].Namespace)
	}
	if got[0].ID != "kube-system/coredns" {
		t.Errorf("ID: want kube-system/coredns, got %q", got[0].ID)
	}
	if got[0].Status != "READY" { // normalised from SANDBOX_READY
		t.Errorf("Status: want READY, got %q", got[0].Status)
	}
}

func TestParseContainerLinesK8sChildTreeMarker(t *testing.T) {
	// Child containers have "└─" as the ID column in talosctl output.
	// This shifts every subsequent field by one. Regression: image was parsed
	// as ID, PID as image, status as PID — everything wrong.
	input := []byte(
		"NODE       NAMESPACE  ID   FULL_ID                                      IMAGE                           PID   STATUS\n" +
			"10.5.0.2   k8s.io     └─   kube-system/coredns:coredns:abc123          registry.k8s.io/coredns:v1.11  1636  CONTAINER_RUNNING\n",
	)
	got := parseContainerLines(input)
	if len(got) != 1 {
		t.Fatalf("want 1 container, got %d", len(got))
	}
	if got[0].ID != "kube-system/coredns:coredns:abc123" {
		t.Errorf("ID: want pod ref, got %q", got[0].ID)
	}
	if got[0].Image != "registry.k8s.io/coredns:v1.11" {
		t.Errorf("Image: want coredns image, got %q", got[0].Image)
	}
	if got[0].PID != "1636" {
		t.Errorf("PID: want 1636, got %q", got[0].PID)
	}
	if got[0].Status != "RUNNING" { // normalised from CONTAINER_RUNNING
		t.Errorf("Status: want RUNNING, got %q", got[0].Status)
	}
}

func TestParseContainerLinesSkipsHeaderAndBlank(t *testing.T) {
	input := []byte(
		"NODE  NAMESPACE  ID  IMAGE  PID  STATUS\n" +
			"\n" +
			"10.5.0.2  system  apid    2445  RUNNING\n",
	)
	got := parseContainerLines(input)
	if len(got) != 1 {
		t.Fatalf("header and blank must be skipped, got %d containers", len(got))
	}
}

func TestParseContainerLinesSkipsTooFewFields(t *testing.T) {
	input := []byte(
		"NODE  NAMESPACE  ID  IMAGE  PID  STATUS\n" +
			"10.5.0.2  system\n", // only 2 fields — malformed
	)
	got := parseContainerLines(input)
	if len(got) != 0 {
		t.Fatalf("malformed line must be skipped, got %d containers", len(got))
	}
}

// ── normalizeContainerStatus ─────────────────────────────────────────────────

func TestNormalizeContainerStatus(t *testing.T) {
	cases := []struct{ in, want string }{
		{"CONTAINER_RUNNING", "RUNNING"},
		{"CONTAINER_STOPPED", "STOPPED"},
		{"CONTAINER_EXITED", "STOPPED"},
		{"SANDBOX_READY", "READY"},
		{"SANDBOX_NOTREADY", "NOT_READY"},
		{"RUNNING", "RUNNING"},   // already normalised
		{"STOPPED", "STOPPED"},   // already normalised
		{"UNKNOWN", "UNKNOWN"},   // pass-through
		{"", ""},                 // empty
	}
	for _, tc := range cases {
		if got := normalizeContainerStatus(tc.in); got != tc.want {
			t.Errorf("normalizeContainerStatus(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── kubelet version line parsing ──────────────────────────────────────────────

func TestParseKubeletVersionFromSpec(t *testing.T) {
	// Simulate a kubeletspec JSON line containing the image tag.
	lines := []string{
		`        "image": "ghcr.io/siderolabs/kubelet:v1.31.0",`,
		`        "image": "ghcr.io/siderolabs/kubelet:v1.32.0"`,  // no trailing comma
		`        "image": "ghcr.io/siderolabs/kubelet:v1.30.5",`, // patch version
	}
	wants := []string{"1.31.0", "1.32.0", "1.30.5"}
	const prefix = "ghcr.io/siderolabs/kubelet:"
	for i, line := range lines {
		idx := -1
		for j := range line {
			if line[j:] >= prefix && line[j:j+len(prefix)] == prefix {
				idx = j
				break
			}
		}
		if idx < 0 {
			t.Errorf("line %d: prefix not found in %q", i, line)
			continue
		}
		tag := line[idx+len(prefix):]
		for j, ch := range tag {
			if ch == '"' || ch == ',' || ch == ' ' {
				tag = tag[:j]
				break
			}
		}
		tag = stripLeadingV(tag)
		if tag != wants[i] {
			t.Errorf("line %d: got %q, want %q", i, tag, wants[i])
		}
	}
}

func stripLeadingV(s string) string {
	if len(s) > 0 && s[0] == 'v' {
		return s[1:]
	}
	return s
}
