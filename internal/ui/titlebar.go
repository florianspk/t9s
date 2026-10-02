package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// renderTitleBar draws a k9s-style view title:  ╭ Disks(3)[1] · node1 ╮
// count < 0 hides the "(n)" part; idx < 1 hides the "[i]" cursor part.
func renderTitleBar(name string, count, idx int, subtitle string) string {
	label := titleBarStyle.Render(name)
	if count >= 0 {
		label += dimStyle.Render(fmt.Sprintf("(%d)", count))
	}
	if idx >= 1 {
		label += keyStyle.Render(fmt.Sprintf("[%d]", idx))
	}
	if subtitle != "" {
		label += dimStyle.Render("  ·  " + subtitle)
	}
	brace := lipgloss.NewStyle().Foreground(colorDimGray)
	return "  " + brace.Render("╭ ") + label + " " + brace.Render("╮") + "\n"
}
