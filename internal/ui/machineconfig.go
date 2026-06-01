package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		// Using Update ensures the viewport is properly initialized before
		// any scroll operation is applied.
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

	if app.machLoading && app.machConf == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			infoStyle.Render("Loading machine config..."))
	}

	if app.machConf == "" {
		return title + lipgloss.Place(app.width, height-2, lipgloss.Center, lipgloss.Center,
			warnStyle.Render("No machine config found."))
	}

	app.machVP.Height = height - 2
	app.machVP.Width = app.width

	return title + app.machVP.View()
}

// startMachineConfigEdit writes the current config to a temp file and opens
// $EDITOR. When the editor exits, editorDoneMsg is sent back.
func (app App) startMachineConfigEdit() (App, tea.Cmd) {
	f, err := os.CreateTemp("", "t9s-machconf-*.yaml")
	if err != nil {
		app.statusMsg = errStyle.Render("Cannot create temp file: " + err.Error())
		return app, nil
	}
	_, writeErr := f.WriteString(app.machConf)
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

// applyMachineConfig runs talosctl apply-config with the edited temp file.
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
