package database

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ConnConfig holds the parameters needed to open a MySQL connection.
type ConnConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	Database  string
	UseSSL    bool
}

// Connection wraps a live MySQL connection with metadata.
type Connection struct {
	ID       string // matches the tab ID
	ProfileID string
	DB       *sql.DB
	Config   ConnConfig
}

// Manager tracks all active MySQL connections.
type Manager struct {
	mu    sync.RWMutex
	conns map[string]*Connection // keyed by tab ID
}

// NewManager creates a connection manager.
func NewManager() *Manager {
	return &Manager{
		conns: make(map[string]*Connection),
	}
}

// Connect opens a MySQL connection for a given tab.
func (m *Manager) Connect(tabID, profileID string, cfg ConnConfig) error {
	dsn, err := buildDSN(cfg)
	if err != nil {
		return err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to connect: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing connection for this tab if any.
	if old, ok := m.conns[tabID]; ok {
		old.DB.Close()
	}

	m.conns[tabID] = &Connection{
		ID:        tabID,
		ProfileID: profileID,
		DB:        db,
		Config:    cfg,
	}

	return nil
}

// Disconnect closes the connection for a tab.
func (m *Manager) Disconnect(tabID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.conns[tabID]
	if !ok {
		return nil
	}

	err := conn.DB.Close()
	delete(m.conns, tabID)
	return err
}

// Get returns the connection for a tab, or nil if not connected.
func (m *Manager) Get(tabID string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conns[tabID]
}

// Ping checks if a tab's connection is still alive.
func (m *Manager) Ping(tabID string) error {
	m.mu.RLock()
	conn, ok := m.conns[tabID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no connection for tab %s", tabID)
	}
	return conn.DB.Ping()
}

// CloseAll closes all connections. Called on app shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, conn := range m.conns {
		conn.DB.Close()
		delete(m.conns, id)
	}
}

// ActiveConnections returns a list of tab IDs with active connections.
func (m *Manager) ActiveConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.conns))
	for id := range m.conns {
		ids = append(ids, id)
	}
	return ids
}

func buildDSN(cfg ConnConfig) (string, error) {
	mc := mysql.NewConfig()
	mc.User = cfg.Username
	mc.Passwd = cfg.Password
	mc.Net = "tcp"
	mc.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mc.DBName = cfg.Database
	mc.Timeout = 10 * time.Second
	mc.ReadTimeout = 30 * time.Second
	mc.WriteTimeout = 30 * time.Second
	mc.ParseTime = true
	mc.InterpolateParams = true

	if cfg.UseSSL {
		mc.TLSConfig = "custom"
		err := mysql.RegisterTLSConfig("custom", &tls.Config{
			InsecureSkipVerify: true, // DO managed DBs use self-signed certs
		})
		if err != nil {
			return "", fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	return mc.FormatDSN(), nil
}
