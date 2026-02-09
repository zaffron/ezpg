package tableview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/db"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

type TableView struct {
	table     table.Model
	connName  string
	schema    string
	tableName string
	columns   []string
	rows      [][]string
	width     int
	height    int
	page      int
	pageSize  int
	totalRows int
	hasData   bool
	// For cell editing
	editingRow int
	editingCol int
	editing    bool
	editValue  string
	// For new row insertion
	inserting    bool
	insertValues []string
	insertCol    int
}

func New() TableView {
	t := table.New(
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(shared.ColorSurface1).
		BorderBottom(true).
		Bold(true).
		Foreground(shared.ColorPrimary)
	s.Selected = s.Selected.
		Foreground(shared.ColorFg).
		Background(shared.ColorBgAlt).
		Bold(false)
	t.SetStyles(s)

	return TableView{
		table:    t,
		pageSize: 100,
	}
}

func (tv *TableView) SetSize(w, h int) {
	tv.width = w
	tv.height = h
	tv.table.SetWidth(w)
	tv.table.SetHeight(h - 3) // leave room for info line
	tv.recalcColumns()
}

func (tv *TableView) SetData(connName, schema, tableName string, result *db.QueryResult) {
	tv.connName = connName
	tv.schema = schema
	tv.tableName = tableName
	tv.columns = result.Columns
	tv.rows = result.Rows
	tv.totalRows = result.RowCount
	tv.hasData = true
	tv.editing = false
	tv.inserting = false

	tv.recalcColumns()

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = r
	}
	tv.table.SetRows(rows)
	tv.table.GotoTop()
}

func (tv *TableView) SetQueryResult(result *db.QueryResult) {
	tv.columns = result.Columns
	tv.rows = result.Rows
	tv.totalRows = result.RowCount
	tv.hasData = true
	tv.schema = ""
	tv.tableName = "query result"
	tv.editing = false
	tv.inserting = false

	tv.recalcColumns()

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = r
	}
	tv.table.SetRows(rows)
	tv.table.GotoTop()
}

func (tv *TableView) recalcColumns() {
	if len(tv.columns) == 0 || tv.width == 0 {
		return
	}

	// Calculate column widths
	numCols := len(tv.columns)
	available := tv.width - numCols - 1 // account for separators
	available = max(available, numCols)

	// Start with header widths
	widths := make([]int, numCols)
	for i, c := range tv.columns {
		widths[i] = len(c)
	}

	// Check data widths (sample first 50 rows)
	sampleSize := 50
	sampleSize = min(sampleSize, len(tv.rows))
	for _, row := range tv.rows[:sampleSize] {
		for i, val := range row {
			if i < numCols && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Cap and distribute
	maxColWidth := available / numCols
	maxColWidth = max(maxColWidth, 8)
	maxColWidth = min(maxColWidth, 50)

	totalUsed := 0
	for i := range widths {
		if widths[i] > maxColWidth {
			widths[i] = maxColWidth
		}
		if widths[i] < 4 {
			widths[i] = 4
		}
		totalUsed += widths[i]
	}

	// Distribute remaining space
	if totalUsed < available {
		extra := available - totalUsed
		perCol := extra / numCols
		for i := range widths {
			widths[i] += perCol
		}
	}

	cols := make([]table.Column, numCols)
	for i, c := range tv.columns {
		cols[i] = table.Column{Title: c, Width: widths[i]}
	}
	tv.table.SetColumns(cols)
}

func (tv *TableView) Page() int         { return tv.page }
func (tv *TableView) PageSize() int     { return tv.pageSize }
func (tv *TableView) HasData() bool     { return tv.hasData }
func (tv *TableView) ConnName() string  { return tv.connName }
func (tv *TableView) Schema() string    { return tv.schema }
func (tv *TableView) TableName() string { return tv.tableName }
func (tv *TableView) Columns() []string { return tv.columns }
func (tv *TableView) IsEditing() bool   { return tv.editing }
func (tv *TableView) IsInserting() bool { return tv.inserting }
func (tv *TableView) Cursor() int       { return tv.table.Cursor() }

func (tv *TableView) SelectedRow() []string {
	cursor := tv.table.Cursor()
	if cursor < 0 || cursor >= len(tv.rows) {
		return nil
	}
	return tv.rows[cursor]
}

func (tv *TableView) NextPage() {
	tv.page++
}

func (tv *TableView) PrevPage() {
	if tv.page > 0 {
		tv.page--
	}
}

// StartEdit begins editing the selected cell
func (tv *TableView) StartEdit() (int, int, string) {
	cursor := tv.table.Cursor()
	if cursor < 0 || cursor >= len(tv.rows) {
		return -1, -1, ""
	}
	tv.editing = true
	tv.editingRow = cursor
	tv.editingCol = 0
	tv.editValue = tv.rows[cursor][0]
	return cursor, 0, tv.editValue
}

func (tv *TableView) CancelEdit() {
	tv.editing = false
}

func (tv *TableView) EditingCol() int   { return tv.editingCol }
func (tv *TableView) EditingRow() int   { return tv.editingRow }
func (tv *TableView) EditValue() string { return tv.editValue }

func (tv *TableView) SetEditValue(v string) {
	tv.editValue = v
}

func (tv *TableView) NextEditCol() {
	if tv.editingCol < len(tv.columns)-1 {
		tv.editingCol++
		tv.editValue = tv.rows[tv.editingRow][tv.editingCol]
	}
}

func (tv *TableView) PrevEditCol() {
	if tv.editingCol > 0 {
		tv.editingCol--
		tv.editValue = tv.rows[tv.editingRow][tv.editingCol]
	}
}

// StartInsert begins inserting a new row
func (tv *TableView) StartInsert() {
	tv.inserting = true
	tv.insertValues = make([]string, len(tv.columns))
	tv.insertCol = 0
}

func (tv *TableView) CancelInsert() {
	tv.inserting = false
	tv.insertValues = nil
}

func (tv *TableView) InsertValues() []string { return tv.insertValues }
func (tv *TableView) InsertCol() int         { return tv.insertCol }

func (tv *TableView) SetInsertValue(v string) {
	if tv.insertCol < len(tv.insertValues) {
		tv.insertValues[tv.insertCol] = v
	}
}

func (tv *TableView) NextInsertCol() bool {
	if tv.insertCol < len(tv.columns)-1 {
		tv.insertCol++
		return false
	}
	return true // all columns filled
}

func (tv *TableView) PrevInsertCol() {
	if tv.insertCol > 0 {
		tv.insertCol--
	}
}

func (tv *TableView) Update(msg tea.KeyMsg) (TableView, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
		tv.table.MoveDown(1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
		tv.table.MoveUp(1)
	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		tv.table.GotoTop()
	case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
		tv.table.GotoBottom()
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+d"))):
		h := tv.height / 2
		tv.table.MoveDown(h)
	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+u"))):
		h := tv.height / 2
		tv.table.MoveUp(h)
	}
	return *tv, nil
}

func (tv TableView) View(active bool) string {
	if !tv.hasData {
		placeholder := lipgloss.NewStyle().
			Foreground(shared.ColorMuted).
			Width(tv.width).
			Height(tv.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Select a table to view data\nor press 'e' for SQL editor")
		return placeholder
	}

	var b strings.Builder

	// Table
	b.WriteString(tv.table.View())
	b.WriteString("\n")

	// Insert row indicator
	if tv.inserting {
		insertLine := lipgloss.NewStyle().Foreground(shared.ColorSuccess).
			Render(fmt.Sprintf("  INSERT: column %d/%d [%s]",
				tv.insertCol+1, len(tv.columns), tv.columns[tv.insertCol]))
		b.WriteString(insertLine + "\n")
	}

	// Info line
	info := fmt.Sprintf(" %d rows | page %d", tv.totalRows, tv.page+1)
	if tv.tableName != "" && tv.tableName != "query result" {
		tname := tv.tableName
		if tv.schema != "" && tv.schema != "public" {
			tname = tv.schema + "." + tname
		}
		info = " " + tname + " |" + info
	}
	infoStyle := lipgloss.NewStyle().Foreground(shared.ColorMuted)
	b.WriteString(infoStyle.Render(info))

	return b.String()
}
