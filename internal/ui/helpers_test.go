package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ── truncate ──────────────────────────────────────────────────────────────────

func TestTruncateShortString(t *testing.T) {
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("want hi, got %q", got)
	}
}

func TestTruncateExact(t *testing.T) {
	if got := truncate("hello", 5); got != "hello" {
		t.Errorf("want hello, got %q", got)
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	got := truncate("hello world", 8)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("want ellipsis suffix, got %q", got)
	}
	if len(got) != 8 {
		t.Errorf("want len 8, got %d (%q)", len(got), got)
	}
}

func TestTruncateMaxThreeOrLess(t *testing.T) {
	// max ≤ 3: cut without ellipsis
	got := truncate("hello", 2)
	if len(got) != 2 {
		t.Errorf("want len 2, got %d (%q)", len(got), got)
	}
	if strings.Contains(got, "...") {
		t.Errorf("no ellipsis expected, got %q", got)
	}
}

func TestTruncateEmpty(t *testing.T) {
	if got := truncate("", 5); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestTruncateMax4(t *testing.T) {
	// max = 4 with a long string → 1 char + "..."
	got := truncate("hello", 4)
	if got != "h..." {
		t.Errorf("want h..., got %q", got)
	}
}

// ── col ───────────────────────────────────────────────────────────────────────

func TestColPadsToWidth(t *testing.T) {
	got := col("hi", 5)
	if len(got) != 5 {
		t.Errorf("want len 5, got %d", len(got))
	}
	if !strings.HasPrefix(got, "hi") {
		t.Errorf("want hi prefix, got %q", got)
	}
}

func TestColExactWidth(t *testing.T) {
	if got := col("hello", 5); got != "hello" {
		t.Errorf("want hello, got %q", got)
	}
}

func TestColDoesNotTruncate(t *testing.T) {
	// col pads but never truncates
	got := col("toolong", 3)
	if got != "toolong" {
		t.Errorf("col must not truncate, got %q", got)
	}
}

func TestColEmptyString(t *testing.T) {
	got := col("", 4)
	if len(got) != 4 {
		t.Errorf("want len 4, got %d", len(got))
	}
	if strings.TrimSpace(got) != "" {
		t.Errorf("want all spaces, got %q", got)
	}
}

// ── clamp ─────────────────────────────────────────────────────────────────────

func TestClampBelow(t *testing.T) {
	if got := clamp(-5, 0, 10); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestClampAbove(t *testing.T) {
	if got := clamp(15, 0, 10); got != 10 {
		t.Errorf("want 10, got %d", got)
	}
}

func TestClampInRange(t *testing.T) {
	if got := clamp(5, 0, 10); got != 5 {
		t.Errorf("want 5, got %d", got)
	}
}

func TestClampAtBoundaries(t *testing.T) {
	if got := clamp(0, 0, 10); got != 0 {
		t.Errorf("want 0 at lo boundary, got %d", got)
	}
	if got := clamp(10, 0, 10); got != 10 {
		t.Errorf("want 10 at hi boundary, got %d", got)
	}
}

// ── containsAny ───────────────────────────────────────────────────────────────

func TestContainsAnyHit(t *testing.T) {
	if !containsAny("hello world", "world") {
		t.Error("want true")
	}
}

func TestContainsAnyMiss(t *testing.T) {
	if containsAny("hello world", "xyz") {
		t.Error("want false")
	}
}

func TestContainsAnyEmptyHaystack(t *testing.T) {
	if containsAny("", "abc") {
		t.Error("want false on empty haystack")
	}
}

func TestContainsAnyFirstMatch(t *testing.T) {
	if !containsAny("error: disk full", "ERROR", "error", "fatal") {
		t.Error("want true for 'error'")
	}
}

func TestContainsAnySubstringLongerThanHaystack(t *testing.T) {
	// Should not panic when a substring is longer than the haystack.
	if containsAny("hi", "verylongsubstring") {
		t.Error("want false")
	}
}

// ── matchSearch ───────────────────────────────────────────────────────────────

func TestMatchSearchEmptyQuery(t *testing.T) {
	if !matchSearch("", "anything", "at all") {
		t.Error("empty query must always match")
	}
}

func TestMatchSearchHit(t *testing.T) {
	if !matchSearch("talos", "talos-node-01") {
		t.Error("want match")
	}
}

func TestMatchSearchMiss(t *testing.T) {
	if matchSearch("xyz", "talos-node-01") {
		t.Error("want no match")
	}
}

func TestMatchSearchCaseInsensitive(t *testing.T) {
	// matchSearch lowercases the field but NOT the query — the caller
	// (searchQuery) is responsible for lowercasing the query first.
	if !matchSearch("talos", "TALOS-NODE-01") {
		t.Error("want case-insensitive match (field upper)")
	}
	// Uppercase query does NOT match (by design — searchQuery() pre-lowercases).
	if matchSearch("TALOS", "talos-node-01") {
		t.Error("uppercase query must not match (caller must lowercase first)")
	}
}

func TestMatchSearchMultipleFieldsHitOnThird(t *testing.T) {
	if !matchSearch("worker", "node-01", "10.0.0.1", "worker") {
		t.Error("want match on third field")
	}
}

func TestMatchSearchNoFields(t *testing.T) {
	// Non-empty query with zero fields: must not match.
	if matchSearch("hello") {
		t.Error("non-empty query with no fields must return false")
	}
}

// ── colorLogLine ─────────────────────────────────────────────────────────────

func TestColorLogLineErrorPreservesContent(t *testing.T) {
	line := "ERROR: connection refused"
	out := colorLogLine(line)
	// Content must be preserved regardless of terminal color support.
	if !strings.Contains(out, line) {
		t.Errorf("output must contain original text, got %q", out)
	}
}

func TestColorLogLineWarnPreservesContent(t *testing.T) {
	line := "WARN: disk almost full"
	if out := colorLogLine(line); !strings.Contains(out, line) {
		t.Errorf("output must contain original text, got %q", out)
	}
}

func TestColorLogLineDebugPreservesContent(t *testing.T) {
	line := "DEBUG: loading config"
	if out := colorLogLine(line); !strings.Contains(out, line) {
		t.Errorf("output must contain original text, got %q", out)
	}
}

func TestColorLogLineFatalPreservesContent(t *testing.T) {
	line := "fatal: out of memory"
	if out := colorLogLine(line); !strings.Contains(out, line) {
		t.Errorf("output must contain original text, got %q", out)
	}
}

func TestColorLogLineNormalUnchanged(t *testing.T) {
	// Lines without a severity keyword are returned as-is.
	line := "starting server on :8080"
	if got := colorLogLine(line); got != line {
		t.Errorf("normal line must be unchanged, got %q", got)
	}
}

func TestColorLogLineKeywordRouting(t *testing.T) {
	// Ensure each severity keyword triggers styling (visible width = len).
	for _, line := range []string{
		"ERROR: x", "FATAL: x", "CRIT: x",
		"error: x", "fatal: x", "crit: x",
		"WARN: x", "WARNING: x", "warn: x",
		"DEBUG: x", "debug: x", "TRACE: x", "trace: x",
	} {
		out := colorLogLine(line)
		if lipgloss.Width(out) != len(line) {
			t.Errorf("colorLogLine(%q): visible width %d ≠ len %d", line, lipgloss.Width(out), len(line))
		}
	}
}

// ── semantic color helpers: visible width == label width ───────────────────────

func TestColorRoleVisibleWidth(t *testing.T) {
	for _, role := range []string{"controlplane", "worker", "unknown"} {
		out := colorRole(role)
		if got := lipgloss.Width(out); got != len(role) {
			t.Errorf("colorRole(%q): visible width %d, want %d", role, got, len(role))
		}
	}
}

func TestColorStateVisibleWidth(t *testing.T) {
	for _, state := range []string{"Running", "Stopped", "Finished", "Failed", "unknown"} {
		out := colorState(state)
		if got := lipgloss.Width(out); got != len(state) {
			t.Errorf("colorState(%q): visible width %d, want %d", state, got, len(state))
		}
	}
}

func TestColorHealthVisibleWidth(t *testing.T) {
	for _, h := range []string{"OK", "healthy", "unhealthy", "?", "unknown"} {
		out := colorHealth(h)
		if got := lipgloss.Width(out); got != len(h) {
			t.Errorf("colorHealth(%q): visible width %d, want %d", h, got, len(h))
		}
	}
}

// ── filteredNodes ─────────────────────────────────────────────────────────────

func TestFilteredNodesNoQuery(t *testing.T) {
	app := newTestApp(120, 30)
	app.nodes = makeNodes(5)
	if got := app.filteredNodes(); len(got) != 5 {
		t.Errorf("no query: want 5, got %d", len(got))
	}
}

func TestFilteredNodesQueryHitRole(t *testing.T) {
	app := newTestApp(120, 30)
	app.nodes = makeNodes(5)
	// makeNodes: nodes[0].Role = "controlplane", rest = "worker"
	si := textinput.New()
	si.SetValue("controlplane")
	app.searchInput = si
	if got := app.filteredNodes(); len(got) != 1 {
		t.Errorf("query 'controlplane': want 1, got %d", len(got))
	}
}

func TestFilteredNodesQueryHitHostname(t *testing.T) {
	app := newTestApp(120, 30)
	app.nodes = makeNodes(3)
	// hostnames: talos-node-00.example.internal, talos-node-01..., talos-node-02...
	si := textinput.New()
	si.SetValue("node-01")
	app.searchInput = si
	if got := app.filteredNodes(); len(got) != 1 {
		t.Errorf("query 'node-01': want 1, got %d", len(got))
	}
}

func TestFilteredNodesQueryMiss(t *testing.T) {
	app := newTestApp(120, 30)
	app.nodes = makeNodes(3)
	si := textinput.New()
	si.SetValue("zzzzz-no-match")
	app.searchInput = si
	if got := app.filteredNodes(); len(got) != 0 {
		t.Errorf("miss: want 0, got %d", len(got))
	}
}

func TestFilteredNodesEmptyList(t *testing.T) {
	app := newTestApp(120, 30)
	// no nodes set
	if got := app.filteredNodes(); len(got) != 0 {
		t.Errorf("empty nodes: want 0, got %d", len(got))
	}
}

// ── filteredServices ─────────────────────────────────────────────────────────

func TestFilteredServicesNoQuery(t *testing.T) {
	app := newTestApp(120, 30)
	app.services = makeServices(7)
	if got := app.filteredServices(); len(got) != 7 {
		t.Errorf("no query: want 7, got %d", len(got))
	}
}

func TestFilteredServicesQueryHit(t *testing.T) {
	app := newTestApp(120, 30)
	app.services = makeServices(3)
	// makeServices: IDs are "service-00", "service-01", "service-02"
	si := textinput.New()
	si.SetValue("service-01")
	app.searchInput = si
	if got := app.filteredServices(); len(got) != 1 {
		t.Errorf("query 'service-01': want 1, got %d", len(got))
	}
}

func TestFilteredServicesQueryMiss(t *testing.T) {
	app := newTestApp(120, 30)
	app.services = makeServices(3)
	si := textinput.New()
	si.SetValue("zzzzz-no-match")
	app.searchInput = si
	if got := app.filteredServices(); len(got) != 0 {
		t.Errorf("miss: want 0, got %d", len(got))
	}
}

func TestFilteredServicesEmptyList(t *testing.T) {
	app := newTestApp(120, 30)
	if got := app.filteredServices(); len(got) != 0 {
		t.Errorf("empty services: want 0, got %d", len(got))
	}
}

// ── extractMachineSection ─────────────────────────────────────────────────────

func TestExtractMachineSectionFromSpec(t *testing.T) {
	raw := `node: 10.0.0.1
metadata:
    namespace: v1alpha1
spec:
    machine:
        type: controlplane
        install:
            disk: /dev/sda
    cluster:
        id: test-cluster
`
	got := extractMachineSection(raw)
	if !strings.Contains(got, "machine:") {
		t.Error("output must contain machine: key")
	}
	if strings.Contains(got, "cluster:") {
		t.Error("cluster: section must not appear in extracted output")
	}
	if strings.Contains(got, "metadata:") {
		t.Error("metadata must not appear in extracted output")
	}
	if !strings.Contains(got, "controlplane") {
		t.Error("machine type must be preserved")
	}
}

func TestExtractMachineSectionDirectKey(t *testing.T) {
	raw := `machine:
    type: worker
    network:
        hostname: talos-worker-01
cluster:
    id: mycluster
`
	got := extractMachineSection(raw)
	if !strings.Contains(got, "machine:") {
		t.Error("output must contain machine: key")
	}
	if strings.Contains(got, "cluster:") {
		t.Error("cluster: must not appear")
	}
}

func TestExtractMachineSectionFallbackOnInvalidYAML(t *testing.T) {
	raw := "not: valid: yaml: ::::"
	got := extractMachineSection(raw)
	if got != raw {
		t.Error("invalid YAML must return raw content unchanged")
	}
}
