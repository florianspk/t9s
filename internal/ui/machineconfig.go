package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// ── key handler ───────────────────────────────────────────────────────────────

func (app App) handleMachineConfigKey(msg tea.KeyMsg) (App, tea.Cmd) {
	// Find bar is open: route all typing to the find input.
	if app.findActive {
		switch msg.String() {
		case "ctrl+c":
			app.cleanup()
			return app, tea.Quit
		case "esc":
			app.findActive = false
			app.findInput.Blur()
			return app, nil
		case "enter":
			q := strings.TrimSpace(app.findInput.Value())
			app.findActive = false
			app.findInput.Blur()
			if q != "" {
				app.machFindQuery = q
				app = app.machComputeMatches()
				app = app.machJumpTo(0)
			}
			return app, nil
		default:
			var cmd tea.Cmd
			app.findInput, cmd = app.findInput.Update(msg)
			return app, cmd
		}
	}

	// Confirm applying the edited config.
	if app.machEditMode {
		switch msg.String() {
		case "y":
			app.machEditMode = false
			return app, app.applyMachineConfig()
		case "n", "esc", "q":
			app.machEditMode = false
			app.statusMsg = dimStyle.Render("Cancelled.")
			os.Remove(app.machEditFile)
			app.machEditFile = ""
		}
		return app, nil
	}

	switch msg.String() {
	case "ctrl+c":
		app.cleanup()
		return app, tea.Quit

	case "esc", "q":
		if app.machFindQuery != "" {
			// First Esc clears the search highlight.
			app.machFindQuery = ""
			app.machFindLines = nil
			app.machVP.SetContent(app.machSection)
			app.statusMsg = ""
			return app, nil
		}
		app = app.goBack()
		return app, nil

	case "/":
		app.findActive = true
		app.findInput.SetValue("")
		return app, app.findInput.Focus()

	case "n":
		if len(app.machFindLines) > 0 {
			app = app.machJumpTo((app.machFindIdx + 1) % len(app.machFindLines))
		}

	case "N":
		if len(app.machFindLines) > 0 {
			n := len(app.machFindLines)
			app = app.machJumpTo((app.machFindIdx - 1 + n) % n)
		}

	case "e":
		return app.startMachineConfigEdit()

	case "g":
		app.machVP.GotoTop()

	case "G":
		app.machVP.GotoBottom()

	default:
		var cmd tea.Cmd
		app.machVP, cmd = app.machVP.Update(msg)
		return app, cmd
	}
	return app, nil
}

// ── find helpers ──────────────────────────────────────────────────────────────

// machComputeMatches finds all lines in machSection that contain machFindQuery
// and re-renders the viewport content with matches highlighted.
func (app App) machComputeMatches() App {
	q := strings.ToLower(app.machFindQuery)
	lines := strings.Split(app.machSection, "\n")
	var matchLines []int
	for i, l := range lines {
		if strings.Contains(strings.ToLower(l), q) {
			matchLines = append(matchLines, i)
		}
	}
	app.machFindLines = matchLines
	app.machFindIdx = 0

	// Re-render content with highlighted matches.
	app.machVP.SetContent(machHighlightMatches(app.machSection, app.machFindQuery))
	return app
}

// machJumpTo scrolls the viewport to the idx-th match and updates statusMsg.
func (app App) machJumpTo(idx int) App {
	if len(app.machFindLines) == 0 {
		app.statusMsg = warnStyle.Render(fmt.Sprintf("/%s  no match", app.machFindQuery))
		return app
	}
	app.machFindIdx = idx
	line := app.machFindLines[idx]
	app.machVP.GotoTop()
	app.machVP.LineDown(line)
	n := len(app.machFindLines)
	app.statusMsg = dimStyle.Render(
		fmt.Sprintf("  /%s  [%d/%d]  n/N: next/prev  Esc: clear", app.machFindQuery, idx+1, n),
	)
	return app
}

// machHighlightMatches returns the content with every occurrence of query
// wrapped in a highlight style (case-insensitive).
func machHighlightMatches(content, query string) string {
	if query == "" {
		return content
	}
	hl := lipgloss.NewStyle().
		Background(lipgloss.Color("#fdcb6e")).
		Foreground(lipgloss.Color("#0d1117")).
		Bold(true)

	q := strings.ToLower(query)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, q) {
			continue
		}
		var sb strings.Builder
		rem := line
		remLow := lower
		for {
			idx := strings.Index(remLow, q)
			if idx < 0 {
				sb.WriteString(rem)
				break
			}
			sb.WriteString(rem[:idx])
			sb.WriteString(hl.Render(rem[idx : idx+len(q)]))
			rem = rem[idx+len(q):]
			remLow = remLow[idx+len(q):]
		}
		lines[i] = sb.String()
	}
	return strings.Join(lines, "\n")
}

// ── render ────────────────────────────────────────────────────────────────────

func (app App) renderMachineConfig(height int) string {
	node := ""
	if app.selNode != nil {
		node = app.selNode.Hostname
	}
	title := fmt.Sprintf("  Machine Config: %s\n", titleStyle.Render(node))

	if app.machLoading && app.machSection == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading machine config..."))
	}
	if app.machSection == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No machine config found."))
	}

	findBarH := 0
	if app.findActive {
		findBarH = 1
	}
	app.machVP.Height = height - 2 - findBarH
	app.machVP.Width = app.width

	out := title + app.machVP.View()
	if app.findActive {
		out += "\n" + keyStyle.Render("/") + " " + app.findInput.View()
	}
	return out
}

// ── edit & apply ──────────────────────────────────────────────────────────────

func (app App) startMachineConfigEdit() (App, tea.Cmd) {
	f, err := os.CreateTemp("", "t9s-machconf-*.yaml")
	if err != nil {
		app.statusMsg = errStyle.Render("Cannot create temp file: " + err.Error())
		return app, nil
	}
	_, writeErr := f.WriteString(app.machSection)
	f.Close()
	if writeErr != nil {
		app.statusMsg = errStyle.Render("Cannot write temp file: " + writeErr.Error())
		os.Remove(f.Name())
		return app, nil
	}
	app.machEditFile = f.Name()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	// editor comes from $EDITOR (falls back to vi) — user-controlled by design.
	c := exec.Command(editor, app.machEditFile) // #nosec G702 -- editor is $EDITOR, intentional
	return app, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{err: err}
	})
}

func (app App) applyMachineConfig() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	file := app.machEditFile
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := client.ApplyConfig(ctx, node, file)
		return machineConfigAppliedMsg{err: err, file: file}
	}
}

// ── YAML extraction ───────────────────────────────────────────────────────────

// extractSpecContent extracts and reformats the machine config from the raw
// YAML returned by `talosctl get machineconfig -o yaml`.
//
// In that output spec: is a scalar string containing the full machine config
// YAML with literal \n characters. This function parses the string and
// re-marshals it with proper yaml.v3 indentation.
func extractSpecContent(raw string) string {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil || doc.Kind == 0 {
		return raw
	}

	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	specNode := yamlFindKey(root, "spec")
	if specNode == nil {
		return raw
	}

	switch specNode.Kind {
	case yaml.ScalarNode:
		// Most common: spec is a quoted string with \n-escaped YAML.
		if specNode.Value == "" {
			return raw
		}
		var inner yaml.Node
		if err := yaml.Unmarshal([]byte(specNode.Value), &inner); err != nil {
			return specNode.Value
		}
		b, err := yaml.Marshal(&inner)
		if err != nil {
			return specNode.Value
		}
		return string(b)

	case yaml.MappingNode:
		b, err := yaml.Marshal(specNode)
		if err != nil {
			return raw
		}
		return string(b)
	}
	return raw
}

// yamlFindKey returns the value node for the given key in a mapping node.
func yamlFindKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
