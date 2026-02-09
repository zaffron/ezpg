package homescreen

import "github.com/zaffron/ezpg/internal/config"

func New(connections []config.Connection) HomeScreen {
	return HomeScreen{
		connections: connections,
	}
}

func (h *HomeScreen) SetSize(w, ht int) {
	h.width = w
	h.height = ht
}

func (h *HomeScreen) SetConnections(conns []config.Connection) {
	h.connections = conns
	if h.cursor >= len(conns) && len(conns) > 0 {
		h.cursor = len(conns) - 1
	}
	if len(conns) == 0 {
		h.cursor = 0
	}
}

func (h *HomeScreen) Connections() []config.Connection {
	return h.connections
}
