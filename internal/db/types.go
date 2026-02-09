package db

import (
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaffron/ezpg/internal/config"
)

/**
* =============================================================================
* Manager Related Types
* =============================================================================
 */
type Manager struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool
	conns map[string]*config.Connection
}

/**
* =============================================================================
* Query Related Types
* =============================================================================
 */
type QueryResult struct {
	Columns  []string
	Rows     [][]string
	RowCount int
	ExecTime time.Duration
	Message  string // for non-SELECT (e.g. INSERT 0 1)
}

type TableInfo struct {
	Schema string
	Name   string
}

type ColumnInfo struct {
	Name       string
	DataType   string
	IsNullable bool
	IsPrimary  bool
}

func (t TableInfo) FullName() string {
	if t.Schema == "public" {
		return t.Name
	}

	return t.Schema + "." + t.Name
}
