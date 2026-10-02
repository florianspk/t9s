package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/florianspk/t9s/internal/talos"
)

func TestApp_RenderLogStreams_showsLoadingState(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{width: 80, height: 24, selNode: &node, logStreamLoading: true}

	// When
	out := app.renderLogStreams(18)

	// Then
	if !strings.Contains(out, "Log Streams on") || !strings.Contains(out, "Loading log streams") {
		t.Fatalf("loading view missing title or status:\n%s", out)
	}
}

func TestApp_RenderLogStreams_showsValidEmptyState(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{width: 80, height: 24, selNode: &node}

	// When
	out := app.renderLogStreams(18)

	// Then
	if !strings.Contains(out, "No log streams found.") {
		t.Fatalf("empty view missing valid empty state:\n%s", out)
	}
}

func TestApp_RenderLogStreams_showsFilteredEmptyState(t *testing.T) {
	// Given
	app := App{
		width:       80,
		height:      24,
		logStreams:  []string{"apid", "kubelet"},
		searchInput: logStreamsTestInput("scheduler"),
	}

	// When
	out := app.renderLogStreams(18)

	// Then
	if !strings.Contains(out, `No match for "scheduler"`) {
		t.Fatalf("filtered view missing no-match state:\n%s", out)
	}
}

func TestApp_RenderLogStreams_rendersDynamicTargetsWithVisibleCursorWithinBounds(t *testing.T) {
	// Given
	streams := []string{
		"apid",
		"kubelet",
		"runtime-" + strings.Repeat("very-long-target-", 8),
	}
	app := App{
		width:        48,
		height:       20,
		logStreams:   streams,
		logStreamCur: 2,
	}

	// When
	out := app.renderLogStreams(12)

	// Then
	if !strings.Contains(out, "apid") || !strings.Contains(out, "kubelet") || !strings.Contains(out, "▶") {
		t.Fatalf("populated view missing targets or cursor:\n%s", out)
	}
	if got := maxLineWidth(out); got > app.width {
		t.Fatalf("maximum line width = %d, terminal width = %d\n%s", got, app.width, out)
	}
	if got := lineCount(out); got > 12 {
		t.Fatalf("line count = %d, height budget = 12\n%s", got, out)
	}
}

func TestApp_RenderLogStreams_keepsEveryStateWithinNarrowWidth(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "long-node-name.example.internal", IP: "10.0.0.2"}
	apps := []App{
		{width: 16, height: 20, selNode: &node, logStreamLoading: true},
		{width: 16, height: 20, selNode: &node},
		{width: 16, height: 20, selNode: &node, logStreams: []string{strings.Repeat("target-", 12)}},
	}

	for _, app := range apps {
		// When
		out := app.renderLogStreams(10)

		// Then
		if got := maxLineWidth(out); got > app.width {
			t.Fatalf("maximum line width = %d, terminal width = %d\n%s", got, app.width, out)
		}
	}
}

func TestApp_RenderLogStreams_clipsWideTargetsWithoutBreakingUTF8(t *testing.T) {
	// Given
	app := App{
		width:       12,
		height:      20,
		logStreams:  []string{strings.Repeat("日志", 12)},
		searchInput: textinput.New(),
	}

	// When
	out := app.renderLogStreams(10)

	// Then
	if !utf8.ValidString(out) {
		t.Fatalf("rendered output contains invalid UTF-8: %q", out)
	}
	if got := maxLineWidth(out); got > app.width {
		t.Fatalf("maximum line width = %d, terminal width = %d\n%s", got, app.width, out)
	}
}

func TestApp_RenderLogStreams_respectsTinyDimensionBudgets(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "non-positive width", width: 0, height: 3},
		{name: "non-positive height", width: 3, height: 0},
		{name: "negative height", width: 3, height: -1},
		{name: "one by one", width: 1, height: 1},
		{name: "one by two", width: 1, height: 2},
		{name: "one by three", width: 1, height: 3},
		{name: "two by one", width: 2, height: 1},
		{name: "two by two", width: 2, height: 2},
		{name: "two by three", width: 2, height: 3},
		{name: "three by one", width: 3, height: 1},
		{name: "three by two", width: 3, height: 2},
		{name: "three by three", width: 3, height: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			app := App{
				width:        test.width,
				logStreams:   []string{strings.Repeat("日志", 4)},
				logStreamCur: 0,
				searchInput:  textinput.New(),
			}

			// When
			out := app.renderLogStreams(test.height)

			// Then
			if test.width <= 0 || test.height <= 0 {
				if out != "" {
					t.Fatalf("render for %dx%d = %q, want empty", test.width, test.height, out)
				}
				return
			}
			if !utf8.ValidString(out) {
				t.Fatalf("render for %dx%d contains invalid UTF-8: %q", test.width, test.height, out)
			}
			if got := maxLineWidth(out); got > test.width {
				t.Fatalf("maximum line width for %dx%d = %d\n%q", test.width, test.height, got, out)
			}
			lines := strings.Count(strings.TrimSuffix(out, "\n"), "\n") + 1
			if got := lines; got > test.height {
				t.Fatalf("line count for %dx%d = %d\n%q", test.width, test.height, got, out)
			}
			wantCursor := test.width >= 3 && test.height >= 2
			if gotCursor := strings.Contains(out, "▶"); gotCursor != wantCursor {
				t.Fatalf("cursor visibility for %dx%d = %v, want %v\n%q", test.width, test.height, gotCursor, wantCursor, out)
			}
		})
	}
}

func TestApp_RenderLogStreams_keepsScrolledSelectionVisible(t *testing.T) {
	// Given
	streams := make([]string, 30)
	for i := range streams {
		streams[i] = "target-" + strings.Repeat("x", i)
	}
	app := App{
		width:           60,
		height:          20,
		logStreams:      streams,
		logStreamCur:    len(streams) - 1,
		viewScrollStart: 20,
	}

	// When
	out := app.renderLogStreams(8)

	// Then
	if !strings.Contains(out, "▶") || !strings.Contains(out, streams[len(streams)-1]) {
		t.Fatalf("selected stream is not visible:\n%s", out)
	}
	if got := lineCount(out); got > 8 {
		t.Fatalf("line count = %d, height budget = 8", got)
	}
}

func TestApp_LogStreamsView_integratesTitleResourceLineAndRenderDispatch(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	app := App{
		width:       80,
		height:      24,
		state:       StateLogStreams,
		selNode:     &node,
		logStreams:  []string{"apid"},
		searchInput: textinput.New(),
	}

	// When
	title := viewTitle(app.state)
	resource := resourceLine(app)
	main := app.renderMain(12)

	// Then
	if title != "[ Log Streams ]" {
		t.Fatalf("title = %q, want Log Streams", title)
	}
	if !strings.Contains(resource, "Log Streams") || !strings.Contains(resource, "node-a") {
		t.Fatalf("resource line = %q, want stream and node context", resource)
	}
	if !strings.Contains(main, "apid") {
		t.Fatalf("render dispatch omitted dynamic target:\n%s", main)
	}
}
