package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zaffron/ezpg/internal/db"
)

func connectCmd(mgr *db.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := mgr.Connect(ctx, name)
		return ConnectMsg{Name: name, Err: err}
	}
}

// func disconnectCmd(mgr *db.Manager, name string) tea.Cmd {
// 	return func() tea.Msg {
// 		mgr.Disconnect(name)
// 		return DisconnectMsg{Name: name}
// 	}
// }

func loadTablesCmd(mgr *db.Manager, connName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tables, err := mgr.ListTables(ctx, connName)
		return TablesLoadedMsg{ConnName: connName, Tables: tables, Err: err}
	}
}

func loadColumnsCmd(mgr *db.Manager, connName, schema, table string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cols, err := mgr.ListColumns(ctx, connName, schema, table)
		return ColumnsLoadedMsg{ConnName: connName, Schema: schema, Table: table, Columns: cols, Err: err}
	}
}

func loadTableDataCmd(mgr *db.Manager, connName, schema, table string, limit, offset int, nullDisplay string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := mgr.QueryTableData(ctx, connName, schema, table, limit, offset, nullDisplay)
		return TableDataMsg{ConnName: connName, Schema: schema, Table: table, Result: result, Err: err}
	}
}

func execQueryCmd(mgr *db.Manager, connName, query, nullDisplay string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := mgr.ExecQuery(ctx, connName, query, nullDisplay)
		return QueryResultMsg{Result: result, Err: err}
	}
}

func deleteRowCmd(mgr *db.Manager, connName, schema, table string, columns []string, pkCols []string, rowValues []string, nullDisplay string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Build WHERE clause from primary keys or all columns
		whereCols := pkCols
		if len(whereCols) == 0 {
			whereCols = columns
		}

		where := ""
		args := []any{}
		argIdx := 1
		for _, col := range whereCols {
			for i, c := range columns {
				if c == col {
					if where != "" {
						where += " AND "
					}
					if rowValues[i] == nullDisplay {
						where += fmt.Sprintf("%q IS NULL", col)
					} else {
						where += fmt.Sprintf("%q = $%d", col, argIdx)
						args = append(args, rowValues[i])
						argIdx++
					}
					break
				}
			}
		}

		query := fmt.Sprintf(
			`DELETE FROM %q.%q WHERE ctid = (SELECT ctid FROM %q.%q WHERE %s LIMIT 1)`,
			schema, table, schema, table, where,
		)

		pool, err := mgr.Pool(connName)
		if err != nil {
			return RowDeletedMsg{Err: err}
		}

		_, err = pool.Exec(ctx, query, args...)
		return RowDeletedMsg{Err: err}
	}
}

func insertRowCmd(mgr *db.Manager, connName, schema, table string, columns []string, values []string, nullDisplay string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cols := ""
		placeholders := ""
		args := []any{}
		argIdx := 1
		for i, col := range columns {
			if values[i] == "" || values[i] == nullDisplay {
				continue
			}
			if cols != "" {
				cols += ", "
				placeholders = placeholders + ", "
			}
			cols += fmt.Sprintf("%q", col)
			placeholders += fmt.Sprintf("$%d", argIdx)
			args = append(args, values[i])
			argIdx++
		}

		if cols == "" {
			return RowInsertedMsg{Err: fmt.Errorf("no values provided")}
		}

		query := fmt.Sprintf(`INSERT INTO %q.%q (%s) VALUES (%s)`, schema, table, cols, placeholders)

		pool, err := mgr.Pool(connName)
		if err != nil {
			return RowInsertedMsg{Err: err}
		}

		_, err = pool.Exec(ctx, query, args...)
		return RowInsertedMsg{Err: err}
	}
}

func updateCellCmd(mgr *db.Manager, connName, schema, table string, columns []string, pkCols []string, rowValues []string, colIdx int, newValue, nullDisplay string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Build WHERE clause
		whereCols := pkCols
		if len(whereCols) == 0 {
			whereCols = columns
		}

		where := ""
		args := []any{}
		argIdx := 1

		// First arg is the new value
		var setClause string
		if newValue == nullDisplay {
			setClause = fmt.Sprintf("%q = NULL", columns[colIdx])
		} else {
			setClause = fmt.Sprintf("%q = $%d", columns[colIdx], argIdx)
			args = append(args, newValue)
			argIdx++
		}

		for _, col := range whereCols {
			for i, c := range columns {
				if c == col {
					if where != "" {
						where += " AND "
					}
					if rowValues[i] == nullDisplay {
						where += fmt.Sprintf("%q IS NULL", col)
					} else {
						where += fmt.Sprintf("%q = $%d", col, argIdx)
						args = append(args, rowValues[i])
						argIdx++
					}
					break
				}
			}
		}

		query := fmt.Sprintf(`UPDATE %q.%q SET %s WHERE %s`, schema, table, setClause, where)

		pool, err := mgr.Pool(connName)
		if err != nil {
			return RowUpdatedMsg{Err: err}
		}

		_, err = pool.Exec(ctx, query, args...)
		return RowUpdatedMsg{Err: err}
	}
}

func statusTimeoutCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}
