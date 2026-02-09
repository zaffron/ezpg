package tui

import "github.com/zaffron/ezpg/internal/db"

// I want to define some of the messages that I have to show to the user

// Connection messages
type ConnectMsg struct {
	Name string
	Err  error
}

type DisconnectMsg struct {
	Name string
}

// Schema messages
type TablesLoadedMsg struct {
	ConnName string
	Tables   []db.TableInfo
	Err      error
}

type ColumnsLoadedMsg struct {
	ConnName string
	Schema   string
	Table    string
	Columns  []db.ColumnInfo
	Err      error
}

// Data messages
type TableDataMsg struct {
	ConnName string
	Schema   string
	Table    string
	Result   *db.QueryResult
	Err      error
}

type QueryResultMsg struct {
	Result *db.QueryResult
	Err    error
}

// UI messages
type StatusMsg struct {
	Text  string
	IsErr bool
}

type ClearStatusMsg struct{}

// CRUD messages
type RowDeletedMsg struct {
	Err error
}

type RowInsertedMsg struct {
	Err error
}

type RowUpdatedMsg struct {
	Err error
}

// Connection config management messages
type ConnectionSavedMsg struct {
	Name string
	Err  error
}

type ConnectionDeletedMsg struct {
	Name string
	Err  error
}

// Confirmation
type ConfirmMsg struct {
	Prompt    string
	OnConfirm func()
}

type ConfirmResultMsg struct {
	Confirmed bool
}

type LoadingMsg struct {
	Loading bool
	Text    string
}
