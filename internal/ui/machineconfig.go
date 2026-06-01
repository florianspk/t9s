package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

func (app App) handleMachineConfigKey(msg tea.KeyMsg) (App, tea.Cmd) {
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
		app = app.goBack()
		return app, nil
	case "e":
		return app.startMachineConfigEdit()
	case "g":
		app.machVP.GotoTop()
	case "G":
		app.machVP.GotoBottom()
	default:
		// Delegate all navigation to the viewport (up/down/j/k/pgup/pgdn).
		var cmd tea.Cmd
		app.machVP, cmd = app.machVP.Update(msg)
		return app, cmd
	}
	return app, nil
}

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

	app.machVP.Height = height - 2
	app.machVP.Width = app.width

	return title + app.machVP.View()
}

// startMachineConfigEdit writes the spec content to a temp file and opens
// $EDITOR. The file is the raw machine config YAML (version, machine, cluster…)
// and can be applied directly with talosctl apply-config.
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

	c := exec.Command(editor, app.machEditFile) //nolint:gosec
	return app, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{err: err}
	})
}

// applyMachineConfig applies the edited spec file using talosctl apply-config.
// The file contains the full machine config YAML (version: v1alpha1, machine:, cluster:…).
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

// extractSpecContent extracts and reformats the machine config from the raw
// YAML returned by `talosctl get machineconfig -o yaml`.
//
// In that output the `spec:` field is a scalar string containing the actual
// machine config YAML with literal \n characters. This function:
//  1. Parses the outer document to find spec:
//  2. Treats spec as a string, parses its content as YAML
//  3. Re-marshals with proper indentation (yaml.v3 default = 4 spaces)
//
// Falls back to the raw input on any parse error so nothing is ever lost.
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

	// Determine the raw spec content depending on how talosctl serialised it.
	var specContent string
	switch specNode.Kind {
	case yaml.ScalarNode:
		// Most common case: spec is a quoted string with \n-escaped YAML.
		specContent = specNode.Value
	case yaml.MappingNode:
		// Occasionally returned as an inline map — marshal it directly.
		b, err := yaml.Marshal(specNode)
		if err != nil {
			return raw
		}
		return string(b)
	default:
		return raw
	}

	if specContent == "" {
		return raw
	}

	// Parse the inner YAML string and re-emit with clean indentation.
	var inner yaml.Node
	if err := yaml.Unmarshal([]byte(specContent), &inner); err != nil {
		return specContent
	}
	b, err := yaml.Marshal(&inner)
	if err != nil {
		return specContent
	}
	return string(b)
}

// yamlFindKey returns the value node for the given key in a mapping node,
// or nil if not found.
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
