package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/config"
	"github.com/zaffron/ezpg/internal/db"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

type item struct {
	connName  string
	tableName string
	schema    string
	isConn    bool
	expanded  bool
}

type Sidebar struct {
	items       []item
	connections []config.Connection
	cursor      int
	width       int
	height      int
	connected   map[string]bool
	filter      string
	filtering   bool
	filterInput textinput.Model
}

func New(connections []config.Connection) Sidebar {
	items := make([]item, 0, len(connections))
	for _, c := range connections {
		items = append(items, item{
			connName: c.Name,
			isConn:   true,
		})
	}

	fi := textinput.New()
	fi.Prompt = "/ "
	fi.PromptStyle = lipgloss.NewStyle().Foreground(shared.ColorWarning)
	fi.CharLimit = 128
	fi.Width = 20

	return Sidebar{
		items:       items,
		connections: connections,
		connected:   make(map[string]bool),
		filterInput: fi,
	}
}

func (s Sidebar) SelectedItem() (connName, schema, table string, isConn bool) {
	if s.cursor < 0 || s.cursor >= len(s.items) {
		return "", "", "", false
	}
	it := s.items[s.cursor]
	return it.connName, it.schema, it.tableName, it.isConn
}

func (s *Sidebar) SetSize(w, h int) {
	s.width = w
	s.height = h
	s.filterInput.Width = w - 6
}

func (s *Sidebar) SetConnected(name string, connected bool) {
	s.connected[name] = connected
}

func (s *Sidebar) SetFilter(f string) {
	s.filter = strings.ToLower(f)
}

func (s *Sidebar) IsFiltering() bool {
	return s.filtering
}

func (s *Sidebar) StartFilter() {
	s.filtering = true
	s.filterInput.SetValue("")
	s.filterInput.Focus()
}

func (s *Sidebar) StopFilter(apply bool) {
	s.filtering = false
	s.filterInput.Blur()
	if !apply {
		s.filter = ""
		s.filterInput.SetValue("")
	}
}

func (s *Sidebar) LoadTables(connName string, tables []db.TableInfo) {
	s.connected[connName] = true

	// Find the connection item
	connIdx := -1
	for i, it := range s.items {
		if it.isConn && it.connName == connName {
			connIdx = i
			break
		}
	}
	if connIdx == -1 {
		return
	}

	// Remove old table entries for this connection
	newItems := make([]item, 0, len(s.items))
	for i, it := range s.items {
		if i == connIdx {
			it.expanded = true
			newItems = append(newItems, it)
			// Insert new tables right after
			for _, t := range tables {
				newItems = append(newItems, item{
					connName:  connName,
					tableName: t.Name,
					schema:    t.Schema,
					isConn:    false,
				})
			}
		} else if !it.isConn && it.connName == connName {
			// skip old table entries for this connection
			continue
		} else {
			newItems = append(newItems, it)
		}
	}
	s.items = newItems
}

func (s *Sidebar) CollapseConnection(connName string) {
	newItems := make([]item, 0, len(s.items))
	for _, it := range s.items {
		if it.isConn && it.connName == connName {
			it.expanded = false
			newItems = append(newItems, it)
		} else if !it.isConn && it.connName == connName {
			continue
		} else {
			newItems = append(newItems, it)
		}
	}
	s.items = newItems
	if s.cursor >= len(s.items) {
		s.cursor = len(s.items) - 1
	}
}

func (s *Sidebar) RemoveTables(connName string) {
	s.connected[connName] = false
	s.CollapseConnection(connName)
}

func (s Sidebar) filteredIndices() []int {
	if s.filter == "" {
		indices := make([]int, len(s.items))
		for i := range s.items {
			indices[i] = i
		}
		return indices
	}
	var indices []int
	for i, it := range s.items {
		if it.isConn || strings.Contains(strings.ToLower(it.tableName), s.filter) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (s *Sidebar) Update(msg tea.KeyMsg) (Sidebar, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
		s.moveDown()
	case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
		s.moveUp()
	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		indices := s.filteredIndices()
		if len(indices) > 0 {
			s.cursor = indices[0]
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
		indices := s.filteredIndices()
		if len(indices) > 0 {
			s.cursor = indices[len(indices)-1]
		}
	}
	return *s, nil
}

func (s *Sidebar) UpdateFilter(msg tea.Msg) (Sidebar, tea.Cmd) {
	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)
	s.filter = strings.ToLower(s.filterInput.Value())
	return *s, cmd
}

func (s *Sidebar) moveDown() {
	indices := s.filteredIndices()
	for _, idx := range indices {
		if idx > s.cursor {
			s.cursor = idx
			return
		}
	}
}

func (s *Sidebar) moveUp() {
	indices := s.filteredIndices()
	for i := len(indices) - 1; i >= 0; i-- {
		if indices[i] < s.cursor {
			s.cursor = indices[i]
			return
		}
	}
}

func (s Sidebar) View(active bool) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(shared.ColorPrimary).Render("Connections")
	b.WriteString(title + "\n")

	// Show filter input when filtering
	if s.filtering {
		b.WriteString(s.filterInput.View() + "\n")
	}

	visible := s.filteredIndices()
	visibleSet := make(map[int]bool, len(visible))
	for _, i := range visible {
		visibleSet[i] = true
	}

	// Calculate scroll window
	maxLines := s.height - 3 // title + padding
	if s.filtering {
		maxLines-- // filter input takes a line
	}
	if maxLines < 1 {
		maxLines = 10
	}

	scrollStart := 0
	if s.cursor > scrollStart+maxLines-1 {
		scrollStart = s.cursor - maxLines + 1
	}

	lines := 0
	for i, it := range s.items {
		if !visibleSet[i] {
			continue
		}
		if i < scrollStart {
			continue
		}
		if lines >= maxLines {
			break
		}

		selected := i == s.cursor && active
		line := s.renderItem(it, selected)
		b.WriteString(line + "\n")
		lines++
	}

	// Pad remaining lines
	for lines < maxLines {
		b.WriteString("\n")
		lines++
	}

	return b.String()
}

func (s Sidebar) renderItem(it item, selected bool) string {
	var prefix, label string

	if it.isConn {
		if s.connected[it.connName] {
			if it.expanded {
				prefix = "● ▾ "
			} else {
				prefix = "● ▸ "
			}
		} else {
			prefix = "○   "
		}
		label = it.connName
	} else {
		prefix = "    "
		label = it.tableName
		if it.schema != "" && it.schema != "public" {
			label = fmt.Sprintf("%s.%s", it.schema, it.tableName)
		}
	}

	text := prefix + label

	// Truncate if needed
	if s.width > 4 && len(text) > s.width-4 {
		text = text[:s.width-7] + "..."
	}

	if selected {
		return lipgloss.NewStyle().Bold(true).Foreground(shared.ColorPrimary).Render(text)
	}
	if it.isConn && s.connected[it.connName] {
		return lipgloss.NewStyle().Foreground(shared.ColorSuccess).Render(text)
	}
	return text
}
