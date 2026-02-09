package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/config"
	"github.com/zaffron/ezpg/internal/db"
	"github.com/zaffron/ezpg/internal/tui/components/editor"
	"github.com/zaffron/ezpg/internal/tui/components/homescreen"
	"github.com/zaffron/ezpg/internal/tui/components/keyhints"
	"github.com/zaffron/ezpg/internal/tui/components/sidebar"
	"github.com/zaffron/ezpg/internal/tui/components/statusbar"
	"github.com/zaffron/ezpg/internal/tui/components/tableview"
)

const sidebarWidth = 30

type App struct {
	cfg    *config.Config
	mgr    *db.Manager
	screen AppScreen
	panel  Panel
	width  int
	height int

	// Input focus: true when a text field captures keys
	inputFocused bool

	sidebar    sidebar.Sidebar
	tableview  tableview.TableView
	editor     editor.Editor
	statusbar  statusbar.StatusBar
	homescreen homescreen.HomeScreen

	showEditor bool
	loading    bool

	// For confirming destructive actions
	confirming  bool
	confirmText string
	onConfirm   func() tea.Cmd

	// Primary key cache
	pkCache map[string][]string // "schema.table" -> pk column names

	// Active connection context
	activeConn string
}

func NewApp(cfg *config.Config) App {
	mgr := db.NewManager(cfg.Connections)

	sb := sidebar.New(cfg.Connections)
	tv := tableview.New()
	ed := editor.New()
	st := statusbar.New()
	hs := homescreen.New(cfg.Connections)

	return App{
		cfg:        cfg,
		mgr:        mgr,
		screen:     ScreenHome,
		panel:      PanelSidebar,
		sidebar:    sb,
		tableview:  tv,
		editor:     ed,
		statusbar:  st,
		homescreen: hs,
		pkCache:    make(map[string][]string),
	}
}

func (a App) Init() tea.Cmd {
	// TODO: load from config
	develop := true
	if develop {
		return tea.Batch(
			tea.SetWindowTitle("lazygres"),
		)
	}
	return tea.Batch(
		tea.EnterAltScreen,
		tea.SetWindowTitle("lazygres"),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layoutResize()
		a.updateHints()
		return a, nil

	case ConnectMsg:
		a.loading = false
		a.statusbar.SetLoading(false, "")
		if msg.Err != nil {
			a.statusbar.SetMessage("Connect failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.sidebar.SetConnected(msg.Name, true)
		a.activeConn = msg.Name
		a.statusbar.SetMessage("Connected to "+msg.Name, false)
		a.statusbar.SetContext(msg.Name, "")
		// Switch to browse screen
		a.screen = ScreenBrowse
		a.panel = PanelSidebar
		a.inputFocused = false
		a.layoutResize()
		a.updateHints()
		return a, tea.Batch(
			loadTablesCmd(a.mgr, msg.Name),
			statusTimeoutCmd(3*time.Second),
		)

	case DisconnectMsg:
		a.sidebar.RemoveTables(msg.Name)
		if a.activeConn == msg.Name {
			a.activeConn = ""
			a.statusbar.SetContext("", "")
		}
		a.statusbar.SetMessage("Disconnected from "+msg.Name, false)
		a.updateHints()
		return a, statusTimeoutCmd(3 * time.Second)

	case TablesLoadedMsg:
		if msg.Err != nil {
			a.statusbar.SetMessage("Load tables failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.sidebar.LoadTables(msg.ConnName, msg.Tables)
		a.updateHints()
		return a, nil

	case ColumnsLoadedMsg:
		if msg.Err != nil {
			return a, nil
		}
		var pks []string
		for _, c := range msg.Columns {
			if c.IsPrimary {
				pks = append(pks, c.Name)
			}
		}
		cacheKey := msg.Schema + "." + msg.Table
		a.pkCache[cacheKey] = pks
		return a, nil

	case TableDataMsg:
		a.loading = false
		a.statusbar.SetLoading(false, "")
		if msg.Err != nil {
			a.statusbar.SetMessage("Load data failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.tableview.SetData(msg.ConnName, msg.Schema, msg.Table, msg.Result)
		a.statusbar.SetContext(msg.ConnName, msg.Table)
		a.updateHints()
		return a, nil

	case QueryResultMsg:
		a.loading = false
		a.statusbar.SetLoading(false, "")
		if msg.Err != nil {
			a.statusbar.SetMessage("Query error: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		if msg.Result.Message != "" {
			a.statusbar.SetMessage(msg.Result.Message+fmt.Sprintf(" (%s)", msg.Result.ExecTime.Round(time.Millisecond)), false)
		} else {
			a.statusbar.SetMessage(fmt.Sprintf("%d rows (%s)", msg.Result.RowCount, msg.Result.ExecTime.Round(time.Millisecond)), false)
		}
		if len(msg.Result.Columns) > 0 {
			a.tableview.SetQueryResult(msg.Result)
		}
		a.updateHints()
		return a, statusTimeoutCmd(5 * time.Second)

	case RowDeletedMsg:
		if msg.Err != nil {
			a.statusbar.SetMessage("Delete failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.statusbar.SetMessage("Row deleted", false)
		a.updateHints()
		return a, tea.Batch(
			a.reloadTableData(),
			statusTimeoutCmd(3*time.Second),
		)

	case RowInsertedMsg:
		a.tableview.CancelInsert()
		if msg.Err != nil {
			a.statusbar.SetMessage("Insert failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.statusbar.SetMessage("Row inserted", false)
		a.inputFocused = false
		a.updateHints()
		return a, tea.Batch(
			a.reloadTableData(),
			statusTimeoutCmd(3*time.Second),
		)

	case RowUpdatedMsg:
		a.tableview.CancelEdit()
		if msg.Err != nil {
			a.statusbar.SetMessage("Update failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.statusbar.SetMessage("Row updated", false)
		a.inputFocused = false
		a.updateHints()
		return a, tea.Batch(
			a.reloadTableData(),
			statusTimeoutCmd(3*time.Second),
		)

	case ConnectionSavedMsg:
		if msg.Err != nil {
			a.statusbar.SetMessage("Save failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.statusbar.SetMessage("Connection saved", false)
		a.homescreen.SetConnections(a.cfg.Connections)
		a.sidebar = sidebar.New(a.cfg.Connections)
		a.updateHints()
		return a, statusTimeoutCmd(3 * time.Second)

	case ConnectionDeletedMsg:
		if msg.Err != nil {
			a.statusbar.SetMessage("Delete failed: "+msg.Err.Error(), true)
			a.updateHints()
			return a, statusTimeoutCmd(5 * time.Second)
		}
		a.statusbar.SetMessage("Connection deleted", false)
		a.homescreen.SetConnections(a.cfg.Connections)
		a.sidebar = sidebar.New(a.cfg.Connections)
		a.updateHints()
		return a, statusTimeoutCmd(3 * time.Second)

	case StatusMsg:
		a.statusbar.SetMessage(msg.Text, msg.IsErr)
		a.updateHints()
		return a, statusTimeoutCmd(5 * time.Second)

	case ClearStatusMsg:
		a.statusbar.ClearMessage()
		return a, nil

	case LoadingMsg:
		a.loading = msg.Loading
		a.statusbar.SetLoading(msg.Loading, msg.Text)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit on ctrl+c
	if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
		a.mgr.CloseAll()
		return a, tea.Quit
	}

	// Confirmation mode
	if a.confirming {
		return a.handleConfirmKey(msg)
	}

	switch a.screen {
	case ScreenHome:
		return a.handleHomeKey(msg)
	case ScreenBrowse:
		if a.inputFocused {
			return a.handleInputKey(msg)
		}
		return a.handleBrowseKey(msg)
	}

	return a, nil
}

func (a App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		a.confirming = false
		a.statusbar.ClearMessage()
		if a.onConfirm != nil {
			cmd := a.onConfirm()
			a.onConfirm = nil
			a.updateHints()
			return a, cmd
		}
	case "n", "N", "esc":
		a.confirming = false
		a.onConfirm = nil
		a.statusbar.SetMessage("Cancelled", false)
		a.updateHints()
		return a, statusTimeoutCmd(2 * time.Second)
	}
	return a, nil
}

// --- Home Screen Key Handling ---

func (a App) handleHomeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.homescreen.IsFormOpen() {
		return a.handleHomeFormKey(msg)
	}

	switch {
	case key.Matches(msg, Keys.Quit):
		a.mgr.CloseAll()
		return a, tea.Quit

	case key.Matches(msg, Keys.Up):
		a.homescreen.MoveUp()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.Down):
		a.homescreen.MoveDown()
		a.updateHints()
		return a, nil

	case msg.String() == "c":
		a.homescreen.StartCreate()
		a.updateHints()
		return a, nil

	case msg.String() == "e":
		if a.homescreen.StartEdit() {
			a.updateHints()
		}
		return a, nil

	case key.Matches(msg, Keys.Delete):
		return a.handleDeleteConnection()

	case key.Matches(msg, Keys.Enter):
		conn, ok := a.homescreen.SelectedConnection()
		if !ok {
			return a, nil
		}
		a.loading = true
		a.statusbar.SetLoading(true, "Connecting to "+conn.Name+"...")
		a.activeConn = conn.Name
		a.updateHints()
		return a, connectCmd(a.mgr, conn.Name)
	}

	return a, nil
}

func (a App) handleHomeFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Escape):
		a.homescreen.CancelForm()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.Enter):
		return a.saveConnection()

	case key.Matches(msg, Keys.Tab):
		a.homescreen.NextField()
		return a, nil

	case key.Matches(msg, Keys.ShiftTab):
		a.homescreen.PrevField()
		return a, nil
	}

	cmd := a.homescreen.UpdateField(msg)
	return a, cmd
}

func (a App) saveConnection() (tea.Model, tea.Cmd) {
	conn := a.homescreen.FormConnection()
	if conn.Name == "" {
		a.statusbar.SetMessage("Connection name is required", true)
		a.updateHints()
		return a, statusTimeoutCmd(3 * time.Second)
	}
	if conn.URL == "" && conn.Host == "" {
		a.statusbar.SetMessage("Either URL or host is required", true)
		a.updateHints()
		return a, statusTimeoutCmd(3 * time.Second)
	}

	if a.homescreen.IsCreating() {
		a.cfg.Connections = append(a.cfg.Connections, conn)
		a.mgr.AddConnection(conn)
	} else {
		idx := a.homescreen.EditIndex()
		oldName := a.cfg.Connections[idx].Name
		a.cfg.Connections[idx] = conn
		a.mgr.UpdateConnection(oldName, conn)
	}

	a.homescreen.CancelForm()

	return a, func() tea.Msg {
		err := a.cfg.Save()
		return ConnectionSavedMsg{Name: conn.Name, Err: err}
	}
}

func (a App) handleDeleteConnection() (tea.Model, tea.Cmd) {
	conn, ok := a.homescreen.SelectedConnection()
	if !ok {
		return a, nil
	}

	if a.cfg.Settings.ConfirmDestructive {
		a.confirming = true
		a.confirmText = fmt.Sprintf("Delete connection %q? (y/n)", conn.Name)
		a.statusbar.SetMessage(a.confirmText, true)
		a.updateHints()
		a.onConfirm = func() tea.Cmd {
			idx := a.homescreen.SelectedIndex()
			if idx < 0 || idx >= len(a.cfg.Connections) {
				return nil
			}
			name := a.cfg.Connections[idx].Name
			a.mgr.RemoveConnection(name)
			a.cfg.Connections = append(a.cfg.Connections[:idx], a.cfg.Connections[idx+1:]...)
			return func() tea.Msg {
				err := a.cfg.Save()
				return ConnectionDeletedMsg{Name: name, Err: err}
			}
		}
		return a, nil
	}

	idx := a.homescreen.SelectedIndex()
	name := a.cfg.Connections[idx].Name
	a.mgr.RemoveConnection(name)
	a.cfg.Connections = append(a.cfg.Connections[:idx], a.cfg.Connections[idx+1:]...)
	return a, func() tea.Msg {
		err := a.cfg.Save()
		return ConnectionDeletedMsg{Name: name, Err: err}
	}
}

// --- Browse Screen Key Handling ---

func (a App) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Quit), key.Matches(msg, Keys.Escape):
		// Back to home screen
		a.screen = ScreenHome
		a.inputFocused = false
		a.showEditor = false
		a.editor.Blur()
		a.layoutResize()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.Tab):
		a.cyclePanelForward()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.ShiftTab):
		a.cyclePanelBackward()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.ToggleEditor):
		a.showEditor = !a.showEditor
		if a.showEditor {
			a.panel = PanelEditor
			a.inputFocused = true
			a.editor.Focus()
		} else {
			a.editor.Blur()
			a.inputFocused = false
			if a.panel == PanelEditor {
				a.panel = PanelTable
			}
		}
		a.layoutResize()
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.Execute):
		return a.executeQuery()

	case key.Matches(msg, Keys.Enter):
		return a.handleEnter()

	case key.Matches(msg, Keys.Delete):
		return a.handleDeleteRow()

	case key.Matches(msg, Keys.Insert):
		return a.handleInsertRow()

	case key.Matches(msg, Keys.Search):
		if a.panel == PanelSidebar {
			a.sidebar.StartFilter()
			a.inputFocused = true
			a.updateHints()
			return a, nil
		}

	case key.Matches(msg, Keys.NextPage):
		if a.panel == PanelTable && a.tableview.HasData() {
			a.tableview.NextPage()
			return a, a.reloadTableData()
		}

	case key.Matches(msg, Keys.PrevPage):
		if a.panel == PanelTable && a.tableview.HasData() {
			a.tableview.PrevPage()
			return a, a.reloadTableData()
		}

	default:
		return a.delegateToPanel(msg)
	}

	return a, nil
}

func (a App) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Escape):
		// Unfocus and cancel
		a.inputFocused = false
		a.editor.Blur()
		if a.tableview.IsEditing() {
			a.tableview.CancelEdit()
		}
		if a.tableview.IsInserting() {
			a.tableview.CancelInsert()
		}
		if a.sidebar.IsFiltering() {
			a.sidebar.StopFilter(false)
		}
		a.updateHints()
		return a, nil

	case key.Matches(msg, Keys.Execute):
		return a.executeQuery()
	}

	// Sidebar filter mode
	if a.sidebar.IsFiltering() {
		switch {
		case key.Matches(msg, Keys.Enter):
			a.sidebar.StopFilter(true) // apply filter
			a.inputFocused = false
			a.updateHints()
			return a, nil
		}
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.UpdateFilter(msg)
		return a, cmd
	}

	// Editor focused
	if a.panel == PanelEditor && a.showEditor {
		var cmd tea.Cmd
		a.editor, cmd = a.editor.Update(msg)
		return a, cmd
	}

	// Cell editing
	if a.tableview.IsEditing() {
		return a.handleCellEditKey(msg)
	}

	// Row inserting
	if a.tableview.IsInserting() {
		return a.handleRowInsertKey(msg)
	}

	return a, nil
}

func (a App) handleCellEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return a.saveCellEdit()
	case "tab":
		a.tableview.NextEditCol()
	case "shift+tab":
		a.tableview.PrevEditCol()
	case "backspace":
		v := a.tableview.EditValue()
		if len(v) > 0 {
			a.tableview.SetEditValue(v[:len(v)-1])
		}
	default:
		if len(msg.String()) == 1 {
			a.tableview.SetEditValue(a.tableview.EditValue() + msg.String())
		}
	}
	return a, nil
}

func (a App) handleRowInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		done := a.tableview.NextInsertCol()
		if done {
			return a.submitInsertRow()
		}
	case "tab":
		a.tableview.NextInsertCol()
	case "shift+tab":
		a.tableview.PrevInsertCol()
	case "backspace":
		vals := a.tableview.InsertValues()
		col := a.tableview.InsertCol()
		if col < len(vals) && len(vals[col]) > 0 {
			a.tableview.SetInsertValue(vals[col][:len(vals[col])-1])
		}
	default:
		if len(msg.String()) == 1 {
			vals := a.tableview.InsertValues()
			col := a.tableview.InsertCol()
			if col < len(vals) {
				a.tableview.SetInsertValue(vals[col] + msg.String())
			}
		}
	}
	return a, nil
}

func (a *App) cyclePanelForward() {
	switch a.panel {
	case PanelSidebar:
		a.panel = PanelTable
	case PanelTable:
		if a.showEditor {
			a.panel = PanelEditor
			a.inputFocused = true
			a.editor.Focus()
		} else {
			a.panel = PanelSidebar
		}
	case PanelEditor:
		a.editor.Blur()
		a.inputFocused = false
		a.panel = PanelSidebar
	}
}

func (a *App) cyclePanelBackward() {
	switch a.panel {
	case PanelSidebar:
		if a.showEditor {
			a.panel = PanelEditor
			a.inputFocused = true
			a.editor.Focus()
		} else {
			a.panel = PanelTable
		}
	case PanelTable:
		a.panel = PanelSidebar
	case PanelEditor:
		a.editor.Blur()
		a.inputFocused = false
		a.panel = PanelTable
	}
}

func (a App) delegateToPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.panel {
	case PanelSidebar:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(msg)
		return a, cmd
	case PanelTable:
		var cmd tea.Cmd
		a.tableview, cmd = a.tableview.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) handleEnter() (tea.Model, tea.Cmd) {
	if a.panel == PanelSidebar {
		connName, schema, table, isConn := a.sidebar.SelectedItem()
		if connName == "" {
			return a, nil
		}
		if isConn {
			if a.mgr.IsConnected(connName) {
				a.sidebar.CollapseConnection(connName)
				a.sidebar.SetConnected(connName, true)
				a.loading = true
				a.statusbar.SetLoading(true, "Loading tables...")
				return a, loadTablesCmd(a.mgr, connName)
			}
			a.loading = true
			a.statusbar.SetLoading(true, "Connecting to "+connName+"...")
			a.activeConn = connName
			return a, connectCmd(a.mgr, connName)
		}
		// Selected a table
		a.panel = PanelTable
		a.activeConn = connName
		a.loading = true
		a.statusbar.SetLoading(true, "Loading "+table+"...")
		cacheKey := schema + "." + table
		var cmds []tea.Cmd
		cmds = append(cmds, loadTableDataCmd(a.mgr, connName, schema, table,
			a.cfg.Settings.DefaultLimit, 0, a.cfg.Settings.NullDisplay))
		if _, ok := a.pkCache[cacheKey]; !ok {
			cmds = append(cmds, loadColumnsCmd(a.mgr, connName, schema, table))
		}
		a.updateHints()
		return a, tea.Batch(cmds...)
	}

	if a.panel == PanelTable && a.tableview.HasData() {
		row, _, val := a.tableview.StartEdit()
		if row >= 0 {
			a.inputFocused = true
			a.updateHints()
			_ = val
		}
		return a, nil
	}

	if a.panel == PanelEditor && a.showEditor && !a.inputFocused {
		a.inputFocused = true
		a.editor.Focus()
		a.updateHints()
		return a, nil
	}

	return a, nil
}

func (a App) handleDeleteRow() (tea.Model, tea.Cmd) {
	if a.panel != PanelTable || !a.tableview.HasData() {
		return a, nil
	}

	connName := a.tableview.ConnName()
	schema := a.tableview.Schema()
	tableName := a.tableview.TableName()
	if schema == "" || tableName == "" || tableName == "query result" {
		a.statusbar.SetMessage("Cannot delete from query results", true)
		return a, statusTimeoutCmd(3 * time.Second)
	}

	if connCfg, ok := a.mgr.ConnectionConfig(connName); ok && connCfg.ReadOnly {
		a.statusbar.SetMessage("Connection is read-only", true)
		return a, statusTimeoutCmd(3 * time.Second)
	}

	row := a.tableview.SelectedRow()
	if row == nil {
		return a, nil
	}

	columns := a.tableview.Columns()
	cacheKey := schema + "." + tableName
	pkCols := a.pkCache[cacheKey]

	if a.cfg.Settings.ConfirmDestructive {
		a.confirming = true
		a.confirmText = "Delete this row? (y/n)"
		a.statusbar.SetMessage(a.confirmText, true)
		a.updateHints()
		a.onConfirm = func() tea.Cmd {
			return deleteRowCmd(a.mgr, connName, schema, tableName, columns, pkCols, row, a.cfg.Settings.NullDisplay)
		}
		return a, nil
	}

	return a, deleteRowCmd(a.mgr, connName, schema, tableName, columns, pkCols, row, a.cfg.Settings.NullDisplay)
}

func (a App) handleInsertRow() (tea.Model, tea.Cmd) {
	if a.panel != PanelTable || !a.tableview.HasData() {
		return a, nil
	}

	schema := a.tableview.Schema()
	tableName := a.tableview.TableName()
	if schema == "" || tableName == "" || tableName == "query result" {
		a.statusbar.SetMessage("Cannot insert into query results", true)
		return a, statusTimeoutCmd(3 * time.Second)
	}

	connName := a.tableview.ConnName()
	if connCfg, ok := a.mgr.ConnectionConfig(connName); ok && connCfg.ReadOnly {
		a.statusbar.SetMessage("Connection is read-only", true)
		return a, statusTimeoutCmd(3 * time.Second)
	}

	a.tableview.StartInsert()
	a.inputFocused = true
	a.updateHints()
	return a, nil
}

func (a App) submitInsertRow() (tea.Model, tea.Cmd) {
	connName := a.tableview.ConnName()
	schema := a.tableview.Schema()
	tableName := a.tableview.TableName()
	columns := a.tableview.Columns()
	values := a.tableview.InsertValues()

	return a, insertRowCmd(a.mgr, connName, schema, tableName, columns, values, a.cfg.Settings.NullDisplay)
}

func (a App) saveCellEdit() (tea.Model, tea.Cmd) {
	connName := a.tableview.ConnName()
	schema := a.tableview.Schema()
	tableName := a.tableview.TableName()
	columns := a.tableview.Columns()
	colIdx := a.tableview.EditingCol()
	newValue := a.tableview.EditValue()
	row := a.tableview.SelectedRow()

	cacheKey := schema + "." + tableName
	pkCols := a.pkCache[cacheKey]

	return a, updateCellCmd(a.mgr, connName, schema, tableName, columns, pkCols, row, colIdx, newValue, a.cfg.Settings.NullDisplay)
}

func (a App) executeQuery() (tea.Model, tea.Cmd) {
	query := strings.TrimSpace(a.editor.Value())
	if query == "" {
		return a, nil
	}
	if a.activeConn == "" {
		a.statusbar.SetMessage("No active connection", true)
		return a, statusTimeoutCmd(3 * time.Second)
	}

	a.editor.AddToHistory(query)
	a.loading = true
	a.statusbar.SetLoading(true, "Executing query...")
	return a, execQueryCmd(a.mgr, a.activeConn, query, a.cfg.Settings.NullDisplay)
}

func (a *App) reloadTableData() tea.Cmd {
	connName := a.tableview.ConnName()
	schema := a.tableview.Schema()
	tableName := a.tableview.TableName()
	if schema == "" || tableName == "" || tableName == "query result" {
		return nil
	}
	offset := a.tableview.Page() * a.tableview.PageSize()
	return loadTableDataCmd(a.mgr, connName, schema, tableName,
		a.cfg.Settings.DefaultLimit, offset, a.cfg.Settings.NullDisplay)
}

func (a *App) updateHints() {
	var hints []keyhints.Hint

	if a.confirming {
		hints = []keyhints.Hint{
			{Key: "y", Desc: "confirm"},
			{Key: "n", Desc: "cancel"},
		}
		a.statusbar.SetHints(hints)
		return
	}

	switch a.screen {
	case ScreenHome:
		hints = a.homescreen.Hints()
	case ScreenBrowse:
		hints = a.browseHints()
	}

	a.statusbar.SetHints(hints)
}

func (a App) browseHints() []keyhints.Hint {
	if a.inputFocused {
		if a.sidebar.IsFiltering() {
			return []keyhints.Hint{
				{Key: "enter", Desc: "apply filter"},
				{Key: "esc", Desc: "cancel"},
			}
		}
		if a.tableview.IsEditing() {
			return []keyhints.Hint{
				{Key: "enter", Desc: "save"},
				{Key: "tab", Desc: "next col"},
				{Key: "esc", Desc: "cancel"},
			}
		}
		if a.tableview.IsInserting() {
			return []keyhints.Hint{
				{Key: "enter", Desc: "next/save"},
				{Key: "tab", Desc: "next col"},
				{Key: "esc", Desc: "cancel"},
			}
		}
		// Editor focused
		return []keyhints.Hint{
			{Key: "ctrl+e", Desc: "execute"},
			{Key: "esc", Desc: "unfocus"},
		}
	}

	hints := []keyhints.Hint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "tab", Desc: "switch panel"},
	}

	switch a.panel {
	case PanelSidebar:
		hints = append(hints,
			keyhints.Hint{Key: "enter", Desc: "select"},
			keyhints.Hint{Key: "/", Desc: "filter"},
		)
	case PanelTable:
		if a.tableview.HasData() {
			hints = append(hints,
				keyhints.Hint{Key: "enter", Desc: "edit cell"},
				keyhints.Hint{Key: "h/l", Desc: "scroll cols"},
				keyhints.Hint{Key: "d", Desc: "delete"},
				keyhints.Hint{Key: "o", Desc: "insert"},
				keyhints.Hint{Key: "n/p", Desc: "page"},
			)
		}
	case PanelEditor:
		hints = append(hints,
			keyhints.Hint{Key: "enter", Desc: "focus editor"},
		)
	}

	hints = append(hints,
		keyhints.Hint{Key: "e", Desc: "editor"},
		keyhints.Hint{Key: "q", Desc: "home"},
	)

	return hints
}

func (a *App) layoutResize() {
	if a.width == 0 || a.height == 0 {
		return
	}

	statusHeight := 2 // info line + hints line

	switch a.screen {
	case ScreenHome:
		contentHeight := a.height - statusHeight
		a.homescreen.SetSize(a.width, contentHeight)
		a.statusbar.SetSize(a.width)

	case ScreenBrowse:
		// Each panel has border + padding that adds to rendered size.
		// Compute frame overhead so content fits within the terminal.
		frameH := StyleSidebarActive.GetHorizontalFrameSize()
		frameV := StyleSidebarActive.GetVerticalFrameSize()

		sideW := sidebarWidth
		sideW = min(sideW, a.width/3)

		// Two panels side by side, each with horizontal frame overhead
		mainW := max(a.width-sideW-2*frameH, 0)

		// Sidebar spans full content height (one frame)
		sideContentH := max(a.height-statusHeight-frameV, 0)
		a.sidebar.SetSize(sideW, sideContentH)

		if a.showEditor {
			// Table + editor stacked, each with vertical frame overhead
			availH := max(a.height-statusHeight-2*frameV, 0)
			editorH := availH / 3
			tableH := availH - editorH
			a.tableview.SetSize(mainW, tableH)
			a.editor.SetSize(mainW, editorH)
		} else {
			tableH := max(a.height-statusHeight-frameV, 0)
			a.tableview.SetSize(mainW, tableH)
		}

		a.statusbar.SetSize(a.width)
	}
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	switch a.screen {
	case ScreenHome:
		return a.homeView()
	case ScreenBrowse:
		return a.browseView()
	}

	return ""
}

func (a App) homeView() string {
	content := a.homescreen.View()
	status := a.statusbar.View()
	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func (a App) browseView() string {
	frameH := StyleSidebarActive.GetHorizontalFrameSize()
	frameV := StyleSidebarActive.GetVerticalFrameSize()

	sideW := sidebarWidth
	sideW = min(sideW, a.width/3)

	statusHeight := 2
	sideContentH := max(a.height-statusHeight-frameV, 0)

	var sideStyle lipgloss.Style
	if a.panel == PanelSidebar {
		sideStyle = StyleSidebarActive.Width(sideW).MaxWidth(sideW).Height(sideContentH).MaxHeight(sideContentH)
	} else {
		sideStyle = StyleSidebarInactive.Width(sideW).MaxWidth(sideW).Height(sideContentH).MaxHeight(sideContentH)
	}
	sideView := sideStyle.Render(a.sidebar.View(a.panel == PanelSidebar))

	// Two panels side by side, each with horizontal frame overhead
	mainW := max(a.width-sideW-2*frameH, 0)

	var mainView string
	if a.showEditor {
		// Table + editor stacked vertically, each with vertical frame
		availH := max(a.height-statusHeight-2*frameV, 0)
		editorContentH := availH / 3
		tableContentH := availH - editorContentH

		var tableStyle lipgloss.Style
		if a.panel == PanelTable {
			tableStyle = StyleMainActive.Width(mainW).MaxWidth(mainW).Height(tableContentH).MaxHeight(tableContentH)
		} else {
			tableStyle = StyleMainInactive.Width(mainW).MaxWidth(mainW).Height(tableContentH).MaxHeight(tableContentH)
		}

		var edStyle lipgloss.Style
		if a.panel == PanelEditor {
			edStyle = StyleEditorActive.Width(mainW).MaxWidth(mainW).Height(editorContentH).MaxHeight(editorContentH)
		} else {
			edStyle = StyleEditorInactive.Width(mainW).MaxWidth(mainW).Height(editorContentH).MaxHeight(editorContentH)
		}

		tableSection := tableStyle.Render(a.tableview.View(a.panel == PanelTable))
		editorSection := edStyle.Render(a.editor.View())
		mainView = lipgloss.JoinVertical(lipgloss.Left, tableSection, editorSection)
	} else {
		tableContentH := max(a.height-statusHeight-frameV, 0)

		var tableStyle lipgloss.Style
		if a.panel == PanelTable {
			tableStyle = StyleMainActive.Width(mainW).MaxWidth(mainW).Height(tableContentH).MaxHeight(tableContentH)
		} else {
			tableStyle = StyleMainInactive.Width(mainW).MaxWidth(mainW).Height(tableContentH).MaxHeight(tableContentH)
		}
		mainView = tableStyle.Render(a.tableview.View(a.panel == PanelTable))
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)
	status := a.statusbar.View()

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}
