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

// cellExtraWidth is the per-cell horizontal overhead beyond content width:
// Padding(0, 1) = 2 chars + BorderRight = 1 char = 3 total.
const cellExtraWidth = 3

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

	// Horizontal scroll
	colOffset   int // first visible column index
	visibleCols int // number of currently visible columns

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

	s.Cell = s.Cell.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(shared.ColorSurface1).
		BorderRight(true)

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(shared.ColorSurface1).
		BorderBottom(true).
		BorderRight(true).
		Foreground(shared.ColorPrimary)

	// Selected wraps the entire already-rendered row (not per-cell),
	// so it must only set colors — no padding or border.
	s.Selected = lipgloss.NewStyle().
		Foreground(shared.ColorFg).
		Background(shared.ColorBgAlt)

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
	tv.rebuildTable()
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
	tv.colOffset = 0

	tv.rebuildTable()
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
	tv.colOffset = 0

	tv.rebuildTable()
	tv.table.GotoTop()
}

// idealColWidth computes the ideal width for a column based on header and data.
func (tv *TableView) idealColWidth(colIdx int) int {
	w := len(tv.columns[colIdx])

	sample := min(50, len(tv.rows))
	for _, row := range tv.rows[:sample] {
		if colIdx < len(row) && len(row[colIdx]) > w {
			w = len(row[colIdx])
		}
	}

	w = max(w, 4)
	w = min(w, 40)
	return w
}

// rebuildTable reconstructs the bubbles/table with only the visible column window,
// accounting for cell padding and border overhead so columns never exceed the width.
func (tv *TableView) rebuildTable() {
	if len(tv.columns) == 0 || tv.width == 0 {
		return
	}

	if tv.colOffset < 0 {
		tv.colOffset = 0
	}
	if tv.colOffset >= len(tv.columns) {
		tv.colOffset = len(tv.columns) - 1
	}

	numCols := len(tv.columns)
	available := tv.width

	// Determine how many columns fit starting from colOffset,
	// accounting for cell padding + border overhead per column.
	used := 0
	end := tv.colOffset
	for i := tv.colOffset; i < numCols; i++ {
		w := tv.idealColWidth(i)
		needed := w + cellExtraWidth
		if used+needed > available && end > tv.colOffset {
			break
		}
		used += needed
		end = i + 1
	}

	tv.visibleCols = end - tv.colOffset
	if tv.visibleCols < 1 {
		tv.visibleCols = 1
		end = tv.colOffset + 1
	}

	// Build visible column definitions
	visCols := make([]table.Column, tv.visibleCols)
	totalUsed := 0
	for i := 0; i < tv.visibleCols; i++ {
		colIdx := tv.colOffset + i
		w := tv.idealColWidth(colIdx)
		visCols[i] = table.Column{Title: tv.columns[colIdx], Width: w}
		totalUsed += w + cellExtraWidth
	}

	// Distribute remaining space to visible columns
	remaining := available - totalUsed
	if remaining > 0 && tv.visibleCols > 0 {
		perCol := remaining / tv.visibleCols
		for i := range visCols {
			visCols[i].Width += perCol
		}
	}

	// Build visible rows before touching the table to avoid panics:
	// SetColumns triggers a re-render that accesses row data, so the
	// row width must already match the new column count.
	rows := make([]table.Row, len(tv.rows))
	for i, row := range tv.rows {
		vRow := make(table.Row, tv.visibleCols)
		for j := 0; j < tv.visibleCols; j++ {
			colIdx := tv.colOffset + j
			if colIdx < len(row) {
				vRow[j] = row[colIdx]
			}
		}
		rows[i] = vRow
	}

	// Clear rows first so SetColumns re-render finds no rows to iterate,
	// then set columns, then set the correctly-sized rows.
	tv.table.SetRows([]table.Row{})
	tv.table.SetColumns(visCols)
	tv.table.SetRows(rows)
}

func (tv *TableView) ScrollRight() {
	if tv.colOffset < len(tv.columns)-1 {
		cursor := tv.table.Cursor()
		tv.colOffset++
		tv.rebuildTable()
		if cursor >= 0 && cursor < len(tv.rows) {
			tv.table.SetCursor(cursor)
		}
	}
}

func (tv *TableView) ScrollLeft() {
	if tv.colOffset > 0 {
		cursor := tv.table.Cursor()
		tv.colOffset--
		tv.rebuildTable()
		if cursor >= 0 && cursor < len(tv.rows) {
			tv.table.SetCursor(cursor)
		}
	}
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
	case key.Matches(msg, key.NewBinding(key.WithKeys("h", "left"))):
		tv.ScrollLeft()
	case key.Matches(msg, key.NewBinding(key.WithKeys("l", "right"))):
		tv.ScrollRight()
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

	// Column scroll indicator
	if len(tv.columns) > tv.visibleCols {
		info += fmt.Sprintf(" | cols %d-%d/%d (h/l)",
			tv.colOffset+1, tv.colOffset+tv.visibleCols, len(tv.columns))
	}

	infoStyle := lipgloss.NewStyle().Foreground(shared.ColorMuted)
	b.WriteString(infoStyle.Render(info))

	return b.String()
}
