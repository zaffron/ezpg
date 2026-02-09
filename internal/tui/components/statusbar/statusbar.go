package statusbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/tui/components/keyhints"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

type StatusBar struct {
	message string
	isErr   bool
	width   int
	conn    string
	table   string
	loading bool
	loadMsg string
	hints   []keyhints.Hint
}

func New() StatusBar {
	return StatusBar{}
}

func (s *StatusBar) SetMessage(msg string, isErr bool) {
	s.message = msg
	s.isErr = isErr
}

func (s *StatusBar) ClearMessage() {
	s.message = ""
	s.isErr = false
}

func (s *StatusBar) SetSize(w int) {
	s.width = w
}

func (s *StatusBar) SetContext(conn, table string) {
	s.conn = conn
	s.table = table
}

func (s *StatusBar) SetLoading(loading bool, msg string) {
	s.loading = loading
	s.loadMsg = msg
}

func (s *StatusBar) SetHints(hints []keyhints.Hint) {
	s.hints = hints
}

func (s StatusBar) View() string {
	// Line 1: info bar
	infoLine := s.infoView()
	// Line 2: hints bar
	hintsLine := keyhints.View(s.hints, s.width)
	return infoLine + "\n" + hintsLine
}

func (s StatusBar) infoView() string {
	var ctx string
	if s.conn != "" {
		ctx = lipgloss.NewStyle().Foreground(shared.ColorSecondary).Render(s.conn)
		if s.table != "" {
			ctx += lipgloss.NewStyle().Foreground(shared.ColorMuted).Render(" > ") +
				lipgloss.NewStyle().Foreground(shared.ColorFg).Render(s.table)
		}
	}

	var msg string
	if s.loading {
		msg = lipgloss.NewStyle().Foreground(shared.ColorWarning).Render("⏳ " + s.loadMsg)
	} else if s.message != "" {
		if s.isErr {
			msg = lipgloss.NewStyle().Foreground(shared.ColorDanger).Render(s.message)
		} else {
			msg = lipgloss.NewStyle().Foreground(shared.ColorSuccess).Render(s.message)
		}
	}

	left := ctx
	if left == "" {
		left = lipgloss.NewStyle().Foreground(shared.ColorMuted).Render("lazygres")
	}

	gapWidth := s.width - lipgloss.Width(left) - lipgloss.Width(msg) - 4
	if gapWidth < 1 {
		gapWidth = 1
	}

	gap := lipgloss.NewStyle().Width(gapWidth).Render("")

	return lipgloss.NewStyle().
		Width(s.width).
		MaxWidth(s.width).
		Background(shared.ColorBgAlt).
		Render(left + "  " + msg + gap)
}
