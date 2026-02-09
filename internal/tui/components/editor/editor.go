package editor

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

func New() Editor {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.CharLimit = 0
	ta.ShowLineNumbers = true
	ta.SetWidth(80)
	ta.SetHeight(6)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(shared.ColorBgAlt)
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(shared.ColorFg)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(shared.ColorMuted)

	return Editor{
		textarea: ta,
		history:  NewHistory(),
	}
}

func (e *Editor) SetSize(w, h int) {
	e.width = w
	e.height = h
	e.textarea.SetWidth(w - 2) // -2 for the borders
	e.textarea.SetHeight(h - 2)
}

func (e *Editor) Focus() {
	e.focused = true
	e.textarea.Focus()
}

func (e *Editor) Blur() {
	e.focused = false
	e.textarea.Blur()
}

func (e *Editor) IsFocused() bool {
	return e.focused
}

func (e *Editor) Value() string {
	return e.textarea.Value()
}

func (e *Editor) SetValue(value string) {
	e.textarea.SetValue(value)
}

func (e *Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	var cmd tea.Cmd
	e.textarea, cmd = e.textarea.Update(msg)
	return *e, cmd
}

func (e *Editor) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(shared.ColorSecondary).
		Render("SQL Editor")

	return title + "\n" + e.textarea.View()
}

// History related
func (e *Editor) AddToHistory(query string) {
	e.history.Add(query)
}

func (e *Editor) HistoryPrev() {
	if q, ok := e.history.Prev(); ok {
		e.textarea.SetValue(q)
	}
}

func (e *Editor) HistoryNext() {
	if q, ok := e.history.Next(); ok {
		e.textarea.SetValue(q)
	}
}
