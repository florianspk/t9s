package ui

import "github.com/charmbracelet/lipgloss"

// k9s-inspired palette
var (
	colorBg      = lipgloss.Color("#0d1117")
	colorCyan    = lipgloss.Color("#00b4d8")
	colorGreen   = lipgloss.Color("#3ddc84")
	colorRed     = lipgloss.Color("#ff6b6b")
	colorOrange  = lipgloss.Color("#ffb347")
	colorBlue    = lipgloss.Color("#74b9ff")
	colorYellow  = lipgloss.Color("#fdcb6e")
	colorMagenta = lipgloss.Color("#a29bfe")
	colorGray    = lipgloss.Color("#636e72")
	colorDimGray = lipgloss.Color("#444c56")
	colorWhite   = lipgloss.Color("#dfe6e9")
	colorBgSel   = lipgloss.Color("#1e3a5f")
	colorBgHead  = lipgloss.Color("#161b22")
)

// Header bar
var headerStyle = lipgloss.NewStyle().
	Background(colorBgHead).
	Foreground(colorCyan).
	Bold(true).
	Padding(0, 1)

var headerDimStyle = lipgloss.NewStyle().
	Background(colorBgHead).
	Foreground(colorGray)

var headerSepStyle = lipgloss.NewStyle().
	Background(colorBgHead).
	Foreground(colorDimGray)

// Resource title bar (row below header)
var titleBarStyle = lipgloss.NewStyle().
	Foreground(colorYellow).
	Bold(true)

// Table
var colHeaderStyle = lipgloss.NewStyle().
	Foreground(colorCyan).
	Bold(true)

var selectedStyle = lipgloss.NewStyle().
	Background(colorBgSel).
	Foreground(colorWhite).
	Bold(true)

// Text styles
var (
	titleStyle = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(colorGray)
	okStyle    = lipgloss.NewStyle().Foreground(colorGreen)
	errStyle   = lipgloss.NewStyle().Foreground(colorRed)
	warnStyle  = lipgloss.NewStyle().Foreground(colorOrange)
	infoStyle  = lipgloss.NewStyle().Foreground(colorBlue)
	keyStyle   = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
)

// Help overlay
var helpBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorCyan).
	Padding(1, 3).
	Background(colorBgHead)

// Semantic coloring helpers
func colorRole(role string) string {
	switch role {
	case "controlplane":
		return lipgloss.NewStyle().Foreground(colorMagenta).Bold(true).Render(role)
	case "worker":
		return lipgloss.NewStyle().Foreground(colorBlue).Render(role)
	}
	return role
}

func colorHealth(h string) string {
	switch h {
	case "OK", "healthy":
		return okStyle.Render(h)
	case "unhealthy":
		return errStyle.Render(h)
	case "?":
		return dimStyle.Render(h)
	}
	return dimStyle.Render(h)
}

func colorState(s string) string {
	switch s {
	case "Running":
		return okStyle.Render(s)
	case "Stopped", "Finished":
		return dimStyle.Render(s)
	case "Failed":
		return errStyle.Render(s)
	}
	return s
}

func colorLogLine(line string) string {
	switch {
	case containsAny(line, "ERROR", "FATAL", "CRIT", "error", "fatal", "crit"):
		return errStyle.Render(line)
	case containsAny(line, "WARN", "WARNING", "warn"):
		return warnStyle.Render(line)
	case containsAny(line, "DEBUG", "debug", "TRACE", "trace"):
		return dimStyle.Render(line)
	}
	return line
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > len(s) {
			continue
		}
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
