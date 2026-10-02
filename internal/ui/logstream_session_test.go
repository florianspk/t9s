package ui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/florianspk/t9s/internal/config"
	"github.com/florianspk/t9s/internal/talos"
)

type capturedLogStreamRun struct {
	ctx    context.Context
	node   string
	target string
}

func TestNew_initializesLogStreamRunner(t *testing.T) {
	// Given
	cfg := &config.TalosConfig{
		Context:  "default",
		Contexts: map[string]config.Context{"default": {}},
	}

	// When
	app := New(cfg, "", cfg.Context)

	// Then
	if app.runLogStream == nil {
		t.Fatal("New returned an app without a live-log runner")
	}
}

func TestApp_StartLogStream_forwardsCapturedRunnerLifecycle(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	runs := make(chan capturedLogStreamRun, 1)
	app := App{
		state:         StateLogStreams,
		selNode:       &node,
		logSessionSeq: 41,
		searchInput:   textinput.New(),
		runLogStream: func(ctx context.Context, node, target string, lines chan<- string) {
			runs <- capturedLogStreamRun{ctx: ctx, node: node, target: target}
			lines <- "first line"
		},
	}

	// When
	started, startCmd := app.startLogStream("kubelet")
	firstMsg := startCmd()
	captured := <-runs

	// Then
	if captured.node != node.IP || captured.target != "kubelet" {
		t.Fatalf("runner captured node %q target %q; want %q and kubelet", captured.node, captured.target, node.IP)
	}
	firstLine, ok := firstMsg.(logLineMsg)
	if !ok {
		t.Fatalf("first message = %T, want logLineMsg", firstMsg)
	}
	if firstLine.line != "first line" || firstLine.sessionSeq != 42 || started.logSessionSeq != 42 {
		t.Fatalf("first line = %#v, app session = %d; want line payload and session 42", firstLine, started.logSessionSeq)
	}

	model, nextCmd := started.Update(firstLine)
	afterLine := model.(App)
	if !reflect.DeepEqual(afterLine.logLines, []string{"first line"}) || nextCmd == nil {
		t.Fatalf("forwarded lines = %v, next command nil = %v", afterLine.logLines, nextCmd == nil)
	}
	done, ok := nextCmd().(logDoneMsg)
	if !ok {
		t.Fatal("closed runner channel did not emit logDoneMsg")
	}
	if done.sessionSeq != started.logSessionSeq {
		t.Fatalf("done session = %d, want %d", done.sessionSeq, started.logSessionSeq)
	}
	model, cmd := afterLine.Update(done)
	finished := model.(App)
	if finished.logStreaming || cmd != nil {
		t.Fatalf("finished stream = streaming %v, command nil %v; want false, true", finished.logStreaming, cmd == nil)
	}
	finished.stopLogs()
}

func TestApp_LogLineMsg_ignoresStaleSession(t *testing.T) {
	// Given
	app := App{
		logLines:      []string{"current"},
		logCur:        0,
		logStreaming:  true,
		logCh:         make(chan string, 1),
		logSessionSeq: 8,
	}

	// When
	model, cmd := app.Update(logLineMsg{line: "stale", sessionSeq: 7})
	got := model.(App)

	// Then
	if !reflect.DeepEqual(got.logLines, app.logLines) || got.logCur != app.logCur || !got.logStreaming {
		t.Fatalf("stale line changed current stream: lines=%v cursor=%d streaming=%v", got.logLines, got.logCur, got.logStreaming)
	}
	if cmd != nil {
		t.Fatal("stale line scheduled another waiter")
	}
}

func TestApp_LogDoneMsg_ignoresStaleSession(t *testing.T) {
	// Given
	app := App{logStreaming: true, logSessionSeq: 8}

	// When
	model, cmd := app.Update(logDoneMsg{sessionSeq: 7})
	got := model.(App)

	// Then
	if !got.logStreaming {
		t.Fatal("stale done message stopped the current stream")
	}
	if cmd != nil {
		t.Fatal("stale done message returned a command")
	}
}

func TestApp_StartLogStream_cancelsReplacedSession(t *testing.T) {
	// Given
	node := talos.Node{Hostname: "node-a", IP: "10.0.0.2"}
	runs := make(chan capturedLogStreamRun, 2)
	releaseOld := make(chan struct{})
	app := App{
		state:       StateServices,
		selNode:     &node,
		searchInput: textinput.New(),
		runLogStream: func(ctx context.Context, node, target string, _ chan<- string) {
			runs <- capturedLogStreamRun{ctx: ctx, node: node, target: target}
			if target == "old" {
				<-releaseOld
			}
		},
	}
	oldSession, oldCmd := app.startLogStream("old")
	oldResult := make(chan tea.Msg, 1)
	go func() {
		oldResult <- oldCmd()
	}()
	oldRun := <-runs

	// When
	replacement, replacementCmd := oldSession.startLogStream("new")

	// Then
	if !errors.Is(oldRun.ctx.Err(), context.Canceled) {
		close(releaseOld)
		t.Fatalf("replaced runner context error = %v, want context canceled", oldRun.ctx.Err())
	}
	if oldSession.logSessionSeq != 1 || replacement.logSessionSeq != 2 {
		close(releaseOld)
		t.Fatalf("session sequence = old %d replacement %d; want 1 and 2", oldSession.logSessionSeq, replacement.logSessionSeq)
	}
	close(releaseOld)
	staleDone, ok := (<-oldResult).(logDoneMsg)
	if !ok {
		t.Fatal("canceled runner did not close with logDoneMsg")
	}
	currentDone, ok := replacementCmd().(logDoneMsg)
	if !ok {
		t.Fatal("replacement runner did not close with logDoneMsg")
	}
	newRun := <-runs
	if newRun.node != node.IP || newRun.target != "new" || currentDone.sessionSeq != replacement.logSessionSeq {
		t.Fatalf("replacement capture = node %q target %q session %d", newRun.node, newRun.target, currentDone.sessionSeq)
	}

	model, cmd := replacement.Update(staleDone)
	got := model.(App)
	if !got.logStreaming || cmd != nil {
		t.Fatalf("stale canceled-session done changed replacement: streaming=%v command nil=%v", got.logStreaming, cmd == nil)
	}
	got.stopLogs()
}
