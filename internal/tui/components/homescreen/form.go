package homescreen

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/config"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

func (h *HomeScreen) initForm(conn config.Connection) {
	h.fields = make([]textinput.Model, fieldCount)

	labels := []string{
		"Name",
		"Host",
		"Port",
		"User",
		"Password",
		"Database",
		"SSL Mode",
		"URL",
	}

	values := []string{
		conn.Name,
		conn.Host,
		fmt.Sprintf("%d", conn.Port),
		conn.User,
		conn.Password,
		conn.Database,
		conn.SSLMode,
		conn.URL,
	}

	if conn.Port == 0 {
		values[fieldPort] = "5432"
	}
	if conn.SSLMode == "" {
		values[fieldSSLMode] = "disable"
	}

	for i := range h.fields {
		ti := textinput.New()
		ti.Prompt = labels[i] + ": "
		ti.PromptStyle = lipgloss.NewStyle().
			Foreground(shared.ColorSecondary).
			Bold(true)
		ti.TextStyle = lipgloss.NewStyle().
			Foreground(shared.ColorFg)
		ti.CharLimit = 256
		ti.Width = 40
		ti.SetValue(values[i])

		if i == int(fieldPassword) {
			ti.EchoMode = textinput.EchoPassword
		}

		h.fields[i] = ti
	}

	h.activeField = fieldName
	h.fields[h.activeField].Focus()
}

func (h *HomeScreen) IsFormOpen() bool {
	return h.editing || h.creating
}

func (h *HomeScreen) IsCreating() bool {
	return h.creating
}

func (h *HomeScreen) StartCreate() {
	h.creating = true
	h.editing = false
	h.initForm(config.Connection{Port: 5432, SSLMode: "disable"})
}

func (h *HomeScreen) StartEdit() bool {
	if h.cursor < 0 || h.cursor >= len(h.connections) {
		return false
	}
	h.editing = true
	h.creating = false
	h.editIdx = h.cursor
	h.initForm(h.connections[h.cursor])
	return true
}

func (h *HomeScreen) EditIndex() int {
	return h.editIdx
}

func (h *HomeScreen) CancelForm() {
	h.editing = false
	h.creating = false
	h.fields = nil
}

func (h *HomeScreen) NextField() {
	h.fields[h.activeField].Blur()
	h.activeField++
	if h.activeField >= fieldCount {
		h.activeField = 0
	}
	h.fields[h.activeField].Focus()
}

func (h *HomeScreen) UpdateField(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.fields[h.activeField], cmd = h.fields[h.activeField].Update(msg)
	return cmd
}

func (h *HomeScreen) PrevField() {
	h.fields[h.activeField].Blur()
	h.activeField--
	if h.activeField < 0 {
		h.activeField = fieldCount - 1
	}
	h.fields[h.activeField].Focus()
}

func (h *HomeScreen) FormConnection() config.Connection {
	port := 5432
	if p := h.fields[fieldPort].Value(); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	return config.Connection{
		Name:     h.fields[fieldName].Value(),
		Host:     h.fields[fieldHost].Value(),
		Port:     port,
		User:     h.fields[fieldUser].Value(),
		Password: h.fields[fieldPassword].Value(),
		Database: h.fields[fieldDatabase].Value(),
		SSLMode:  h.fields[fieldSSLMode].Value(),
		URL:      h.fields[fieldURL].Value(),
	}
}
