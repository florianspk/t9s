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
	title := fmt.Sprintf("  Machine Config › %s  %s\n",
		titleStyle.Render(node),
		dimStyle.Render("(machine: section)"),
	)

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

// startMachineConfigEdit writes the machine: section to a temp file and
// opens $EDITOR. The edited file is a valid strategic merge patch that
// talosctl patch machineconfig can apply directly.
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

// applyMachineConfig runs talosctl patch machineconfig with the edited file.
// The file contains only the machine: section which is a valid strategic
// merge patch — only the modified fields are applied.
func (app App) applyMachineConfig() tea.Cmd {
	client := app.client
	node := app.selNode.IP
	file := app.machEditFile
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := client.PatchMachineConfig(ctx, node, file)
		return machineConfigAppliedMsg{err: err, file: file}
	}
}

// extractMachineSection parses the raw YAML from `talosctl get machineconfig`
// and returns only the machine: section formatted as a YAML document.
// This section is also a valid strategic merge patch for talosctl patch.
// Falls back to the raw content if parsing fails.
func extractMachineSection(raw string) string {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil || doc.Kind == 0 {
		return raw
	}

	// The talosctl resource wraps the config under "spec:".
	// Navigate: root → mapping → spec → mapping → machine
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	machineNode := yamlFindKey(root, "machine")
	if machineNode == nil {
		// Try spec.machine
		if spec := yamlFindKey(root, "spec"); spec != nil {
			machineNode = yamlFindKey(spec, "machine")
		}
	}
	if machineNode == nil {
		return raw
	}

	// Re-marshal as "machine:\n  ..." for readability and patch compatibility.
	out := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "machine"},
					machineNode,
				},
			},
		},
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return raw
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
