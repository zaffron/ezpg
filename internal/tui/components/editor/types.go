package editor

import "github.com/charmbracelet/bubbles/textarea"

const maxHistory = 100

type History struct {
	entries []string
	pos     int
}

type Editor struct {
	textarea textarea.Model
	history  *History
	width    int
	height   int
	focused  bool
}

func (h *History) Add(query string) {
	if query == "" {
		return
	}

	// Don't add duplicate of most recent
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == query {
		return
	}
	h.entries = append(h.entries, query)
	if len(h.entries) > maxHistory {
		h.entries = h.entries[1:]
	}
	h.pos = len(h.entries)
}

func (h *History) Prev() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}

	if h.pos > 0 {
		h.pos--
	}
	return h.entries[h.pos], true
}

func (h *History) Next() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}

	if h.pos < len(h.entries)-1 {
		h.pos++
		return h.entries[h.pos], true
	}

	h.pos = len(h.entries)
	return "", true
}

func (h *History) Reset() {
	h.pos = len(h.entries)
}
