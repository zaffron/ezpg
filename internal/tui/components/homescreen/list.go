package homescreen

import "github.com/zaffron/ezpg/internal/config"

func (h *HomeScreen) SelectedIndex() int {
	return h.cursor
}

func (h *HomeScreen) SelectedConnection() (config.Connection, bool) {
	if h.cursor < 0 || h.cursor >= len(h.connections) {
		return config.Connection{}, false
	}
	return h.connections[h.cursor], true
}

func (h *HomeScreen) MoveUp() {
	if h.cursor > 0 {
		h.cursor--
	}
}

func (h *HomeScreen) MoveDown() {
	if h.cursor < len(h.connections)-1 {
		h.cursor++
	}
}
