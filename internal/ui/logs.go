package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wrap"
)

func waitForLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg(line)
	}
}

func (app App) handleLogsKey(msg tea.KeyMsg) (App, tea.Cmd) {
	if app.findActive {
		var cmd tea.Cmd
		app, app.logCur, cmd = app.handleFindKey(msg, app.logLines, app.logCur)
		return app, cmd
	}

	n := len(app.logLines)
	findBarH := 0
	if app.findActive || app.findQuery != "" {
		findBarH = 1
	}
	logMaxRows := max(1, app.mainHeight()-2-findBarH)
	// Approximate anchor boundary (exact value accounts for wrapped lines but
	// this is close enough for maintaining the scroll offset in the key handler).
	approxAnchor := max(0, n-logMaxRows)

	updateLogScroll := func() {
		if app.logCur >= approxAnchor {
			// Inside anchor window — reset so the transition out is smooth.
			app.viewScrollStart = approxAnchor
		} else {
			app.viewScrollStart = clampScrollStart(app.viewScrollStart, app.logCur, n, logMaxRows)
		}
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit
	case "esc", "q":
		if app.findQuery != "" {
			app.findQuery = ""
			return app, nil
		}
		app.stopLogs()
		app = app.goBack()
		return app, nil
	case "up", "k":
		if app.logCur > 0 {
			app.logCur--
		}
		updateLogScroll()
	case "down", "j":
		if app.logCur < n-1 {
			app.logCur++
		}
		updateLogScroll()
	case "pgup":
		app.logCur = max(0, app.logCur-app.mainHeight()/2)
		updateLogScroll()
	case "pgdown":
		app.logCur = min(max(0, n-1), app.logCur+app.mainHeight()/2)
		updateLogScroll()
	case "g":
		app.logCur = 0
		updateLogScroll()
	case "G":
		app.logCur = max(0, n-1)
		updateLogScroll()
	case "/":
		app.findActive = true
		app.findInput.SetValue("")
		return app, app.findInput.Focus()
	case "n":
		if app.findQuery != "" {
			if idx := findLineNext(app.logLines, app.logCur+1, app.findQuery); idx >= 0 {
				app.logCur = idx
			}
		}
	case "N":
		if app.findQuery != "" {
			if idx := findLinePrev(app.logLines, app.logCur-1, app.findQuery); idx >= 0 {
				app.logCur = idx
			}
		}
	}
	return app, nil
}

func (app App) renderLogs(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	streaming := ""
	if app.logStreaming {
		streaming = infoStyle.Render(" [streaming]")
	} else {
		streaming = dimStyle.Render(" [stopped]")
	}
	title := fmt.Sprintf("  Logs: %s on %s%s\n",
		titleStyle.Render(app.logService),
		titleStyle.Render(node),
		streaming,
	)

	if len(app.logLines) == 0 {
		return title + "  " + infoStyle.Render("Waiting for logs…")
	}

	findBarH := 0
	if app.findActive || app.findQuery != "" {
		findBarH = 1
	}
	content := renderLinesCursor(app.logLines, app.logCur, app.width, height-2-findBarH, app.viewScrollStart, app.findQuery)
	return title + content + app.renderFindBar(app.logLines)
}

// handleFindKey routes keypresses while the find bar is open.
// Returns (updated app, updated cursor, cmd).
func (app App) handleFindKey(msg tea.KeyMsg, lines []string, cur int) (App, int, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, cur, tea.Quit
	case "esc":
		app.findActive = false
		app.findInput.Blur()
		return app, cur, nil
	case "enter":
		q := strings.TrimSpace(app.findInput.Value())
		app.findActive = false
		app.findInput.Blur()
		if q != "" {
			app.findQuery = q
			if idx := findLineNext(lines, cur, q); idx >= 0 {
				cur = idx
			}
		}
		return app, cur, nil
	default:
		var cmd tea.Cmd
		app.findInput, cmd = app.findInput.Update(msg)
		return app, cur, cmd
	}
}

// renderFindBar returns a one-line bar shown at the bottom of log/dmesg views.
func (app App) renderFindBar(lines []string) string {
	if app.findActive {
		return keyStyle.Render("/") + " " + app.findInput.View() + "\n"
	}
	if app.findQuery != "" {
		n := countMatches(lines, app.findQuery)
		hint := dimStyle.Render(fmt.Sprintf("  /%s  %d match(es)  n/N: next/prev  Esc: clear", app.findQuery, n))
		return hint + "\n"
	}
	return ""
}

// findLineNext returns the index of the next line containing q, starting at
// from and wrapping around. Returns -1 when no match exists.
func findLineNext(lines []string, from int, q string) int {
	if q == "" || len(lines) == 0 {
		return -1
	}
	q = strings.ToLower(q)
	n := len(lines)
	for i := 0; i < n; i++ {
		idx := (from + i) % n
		if strings.Contains(strings.ToLower(lines[idx]), q) {
			return idx
		}
	}
	return -1
}

// findLinePrev returns the index of the previous line containing q, starting
// at from and wrapping around. Returns -1 when no match exists.
func findLinePrev(lines []string, from int, q string) int {
	if q == "" || len(lines) == 0 {
		return -1
	}
	q = strings.ToLower(q)
	n := len(lines)
	for i := 0; i < n; i++ {
		idx := ((from - i) % n + n) % n
		if strings.Contains(strings.ToLower(lines[idx]), q) {
			return idx
		}
	}
	return -1
}

// countMatches returns how many lines contain q (case-insensitive).
func countMatches(lines []string, q string) int {
	if q == "" {
		return 0
	}
	q = strings.ToLower(q)
	n := 0
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), q) {
			n++
		}
	}
	return n
}

// renderLinesCursor renders a scrollable, cursor-highlighted list of lines.
// findQuery, when non-empty, marks matching lines with a ▸ prefix.
// scrollStart is the caller's persisted scroll offset, used when the cursor
// is above the anchor window so the cursor moves within the view instead of
// being pinned to the top row.
func renderLinesCursor(lines []string, cur, width, maxRows, scrollStart int, findQuery string) string {
	if len(lines) == 0 || maxRows <= 0 {
		return ""
	}
	if cur < 0 {
		cur = 0
	}
	if cur >= len(lines) {
		cur = len(lines) - 1
	}

	w := width
	if w <= 0 {
		w = 80
	}

	fq := strings.ToLower(findQuery)

	// Anchor the view to the LAST line of the log buffer.
	// Walk backward from len(lines)-1 until we fill the screen.
	// The cursor then moves freely inside this window without
	// shifting the view — the latest log stays visible at the
	// bottom until the cursor scrolls above the top of the window.
	anchorStart := len(lines) - 1
	budget := maxRows - 1
	for anchorStart > 0 && budget > 0 {
		prev := anchorStart - 1
		n := strings.Count(wrap.String(lines[prev], max(1, w-2)), "\n") + 1
		if n > budget {
			break
		}
		budget -= n
		anchorStart = prev
	}

	// If the cursor is inside the anchor window, keep the anchor so the
	// last line stays pinned at the bottom.
	// If the cursor moved above the anchor window, apply the same
	// boundary-scroll logic as list views: cursor moves freely inside the
	// visible area and the window only scrolls when the cursor hits an edge.
	//
	// clampScrollStart counts logical items, not physical rows. For wrapped
	// lines the two differ, so we verify the cursor is reachable from the
	// chosen start and fall back to start=cur (cursor at top) if not.
	var start int
	if cur >= anchorStart {
		start = anchorStart
	} else {
		candidate := clampScrollStart(scrollStart, cur, len(lines), maxRows)
		rows := 0
		curReachable := false
		for i := candidate; i < len(lines) && rows < maxRows; i++ {
			n := strings.Count(wrap.String(lines[i], max(1, w-2)), "\n") + 1
			if n > maxRows-rows {
				n = maxRows - rows
			}
			rows += n
			if i == cur {
				curReachable = true
				break
			}
		}
		if curReachable {
			start = candidate
		} else {
			start = cur // fallback: cursor pinned to top
		}
	}

	var sb strings.Builder
	lineCount := 0

	for i := start; i < len(lines) && lineCount < maxRows; i++ {
		raw := lines[i]

		// Detect severity from the full line, then wrap raw text.
		// Applying the style per physical sub-line ensures continuation
		// lines keep the correct colour even after a wrap.
		applyColor := lineLogStyle(raw)
		wrapped := wrap.String(raw, max(1, w-2))
		physLines := strings.Split(wrapped, "\n")

		// Trim to fit remaining budget.
		remaining := maxRows - lineCount
		if len(physLines) > remaining {
			physLines = physLines[:remaining]
		}

		selected := i == cur
		isMatch := fq != "" && strings.Contains(strings.ToLower(raw), fq)

		for j, pl := range physLines {
			switch {
			case selected:
				// All physical lines of the selected item are highlighted.
				prefix := "  "
				if j == 0 {
					prefix = "▶ "
				}
				sb.WriteString(selectedStyle.Width(w).Render(prefix + pl))
			case isMatch && j == 0:
				sb.WriteString(warnStyle.Render("▸") + " " + applyColor(pl))
			default:
				sb.WriteString("  " + applyColor(pl))
			}
			sb.WriteByte('\n')
			lineCount++
		}
	}

	return sb.String()
}

// lineLogStyle returns a coloring function based on keywords in the full
// original log line. Called once per logical line so continuation physical
// lines share the same colour.
func lineLogStyle(fullLine string) func(string) string {
	switch {
	case containsAny(fullLine, "ERROR", "FATAL", "CRIT", "error", "fatal", "crit"):
		return func(s string) string { return errStyle.Render(s) }
	case containsAny(fullLine, "WARN", "WARNING", "warn"):
		return func(s string) string { return warnStyle.Render(s) }
	case containsAny(fullLine, "DEBUG", "debug", "TRACE", "trace"):
		return func(s string) string { return dimStyle.Render(s) }
	}
	return func(s string) string { return s }
}
