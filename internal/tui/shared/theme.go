package shared

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
)

var (
	mocha = catppuccin.Mocha

	// Colors
	// I am currently taking mocha as a reference because it is my favorite flavor, might add configurable theme later
	ColorPrimary   = lipgloss.Color(mocha.Lavender().Hex)
	ColorSecondary = lipgloss.Color(mocha.Blue().Hex)
	ColorSuccess   = lipgloss.Color(mocha.Green().Hex)
	ColorDanger    = lipgloss.Color(mocha.Red().Hex)
	ColorWarning   = lipgloss.Color(mocha.Yellow().Hex)
	ColorMuted     = lipgloss.Color(mocha.Subtext0().Hex)
	ColorBg        = lipgloss.Color(mocha.Base().Hex)
	ColorBgAlt     = lipgloss.Color(mocha.Surface0().Hex)
	ColorFg        = lipgloss.Color(mocha.Text().Hex)
	ColorBorder    = lipgloss.Color(mocha.Lavender().Hex)
	ColorSurface1  = lipgloss.Color(mocha.Surface1().Hex)
	ColorKeyHint   = lipgloss.Color(mocha.Mantle().Hex)

	StyleSidebarActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	StyleSidebarInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	StyleMainActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	StyleMainInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	StyleEditorActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0, 1)

	StyleEditorInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	StyleKeyHintKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorKeyHint).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleKeyHintDesc = lipgloss.NewStyle().
				Foreground(ColorFg).
				Background(ColorBgAlt).
				Padding(0, 1)
)
