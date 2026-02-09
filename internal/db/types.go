package db

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaffron/ezpg/internal/config"
)

type Manager struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool
	conns map[string]*config.Connection
}
