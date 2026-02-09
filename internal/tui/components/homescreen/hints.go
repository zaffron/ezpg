package homescreen

import "github.com/zaffron/ezpg/internal/tui/components/keyhints"

func (h HomeScreen) Hints() []keyhints.Hint {
	if h.editing || h.creating {
		return []keyhints.Hint{
			{Key: "tab", Desc: "next field"},
			{Key: "shift+tab", Desc: "prev field"},
			{Key: "enter", Desc: "save"},
			{Key: "esc", Desc: "cancel"},
		}
	}

	hints := []keyhints.Hint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "enter", Desc: "connect"},
		{Key: "c", Desc: "create"},
	}

	if len(h.connections) > 0 {
		hints = append(hints,
			keyhints.Hint{Key: "e", Desc: "edit"},
			keyhints.Hint{Key: "d", Desc: "delete"},
		)
	}

	return append(hints, keyhints.Hint{Key: "q", Desc: "quit"})
}
