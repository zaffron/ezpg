package keyhints

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

type Hint struct {
	Key  string
	Desc string
}

var (
	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(shared.ColorBg).
			Background(shared.ColorPrimary).
			Padding(0, 1)

	descStyle = lipgloss.NewStyle().
			Foreground(shared.ColorFg).
			Background(shared.ColorBgAlt).
			Padding(0, 1)
)

func View(hints []Hint, width int) string {
	if len(hints) == 0 {
		return lipgloss.NewStyle().Width(width).Background(shared.ColorBgAlt).Render("")
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.Key)+descStyle.Render(h.Desc))
	}

	line := strings.Join(parts, " ")
	return lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Background(shared.ColorBgAlt).
		Render(line)
}
