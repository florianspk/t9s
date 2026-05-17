package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/florianspk/t9s/internal/config"
	"github.com/florianspk/t9s/internal/ui"
)

var version = "0.1.0"

func main() {
	var (
		cfgPath  string
		talosCtx string
		showVer  bool
	)

	flag.StringVar(&cfgPath, "talosconfig", "", "Path to talosconfig (default: $TALOSCONFIG or ~/.talos/config)")
	flag.StringVar(&talosCtx, "context", "", "Talos context to use")
	flag.BoolVar(&showVer, "version", false, "Print version and exit")
	flag.Parse()

	if showVer {
		fmt.Printf("t9s v%s\n", version)
		os.Exit(0)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading talosconfig: %v\n\nMake sure talosctl is configured and ~/.talos/config exists.\n", err)
		os.Exit(1)
	}

	if talosCtx == "" {
		talosCtx = cfg.Context
	}

	app := ui.New(cfg, cfgPath, talosCtx)

	p := tea.NewProgram(app,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running t9s: %v\n", err)
		os.Exit(1)
	}
}
