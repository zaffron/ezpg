package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaffron/ezpg/internal/config"
)

func NewManager(connections []config.Connection) *Manager {
	m := &Manager{
		pools: make(map[string]*pgxpool.Pool),
		conns: make(map[string]*config.Connection),
	}

	for i := range connections {
		m.conns[connections[i].Name] = &connections[i]
	}

	return m
}

func (m *Manager) Connect(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.pools[name]; ok {
		return nil // already connected
	}

	conn, ok := m.conns[name]
	if !ok {
		return fmt.Errorf("unknown connection: %s", name)
	}

	pool, err := pgxpool.New(ctx, conn.DSN())
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", name, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("pinging %s: %w", name, err)
	}

	m.pools[name] = pool

	return nil
}

func (m *Manager) Disconnect(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, ok := m.pools[name]; ok {
		pool.Close()
		delete(m.pools, name)
	}
}

func (m *Manager) Pool(name string) (*pgxpool.Pool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.pools[name]
	if !ok {
		return nil, fmt.Errorf("not connected to %s", name)
	}

	return pool, nil
}

func (m *Manager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.pools[name]
	return ok
}

func (m *Manager) ConnectionConfig(name string) (*config.Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.conns[name]
	return conn, ok
}

func (m *Manager) AddConnection(conn config.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.conns[conn.Name] = &conn
}

func (m *Manager) RemoveConnection(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, ok := m.pools[name]; ok {
		pool.Close()
		delete(m.pools, name)
	}
	delete(m.conns, name)
}

func (m *Manager) UpdateConnection(oldName string, conn config.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close old pool if connected
	if pool, ok := m.pools[oldName]; ok {
		pool.Close()
		delete(m.pools, oldName)
	}
	delete(m.conns, oldName)
	m.conns[conn.Name] = &conn
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, pool := range m.pools {
		pool.Close()
		delete(m.pools, name)
	}
}
