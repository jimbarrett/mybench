package store

import (
	"time"

	"github.com/google/uuid"
)

// ConnectionProfile represents a saved database connection.
type ConnectionProfile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	DefaultDB  string `json:"defaultDb"`
	UseSSL     bool   `json:"useSsl"`
	SSHEnabled bool   `json:"sshEnabled"`
	SSHHost    string `json:"sshHost"`
	SSHPort    int    `json:"sshPort"`
	SSHUser    string `json:"sshUser"`
	SSHAuth    string `json:"sshAuth"` // "key" or "password"
	SSHKeyPath string `json:"sshKeyPath"`
	SSHPass    string `json:"sshPassword"`
	SortOrder  int    `json:"sortOrder"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// ListConnections returns all saved connection profiles ordered by sort_order.
func (s *Store) ListConnections() ([]ConnectionProfile, error) {
	rows, err := s.db.Query(`
		SELECT id, name, host, port, username, password, default_db, use_ssl,
		       ssh_enabled, ssh_host, ssh_port, ssh_user, ssh_auth, ssh_key_path, ssh_password,
		       sort_order, created_at, updated_at
		FROM connections ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []ConnectionProfile
	for rows.Next() {
		var c ConnectionProfile
		var useSSL, sshEnabled int
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.Password, &c.DefaultDB, &useSSL,
			&sshEnabled, &c.SSHHost, &c.SSHPort, &c.SSHUser, &c.SSHAuth, &c.SSHKeyPath, &c.SSHPass,
			&c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		c.UseSSL = useSSL == 1
		c.SSHEnabled = sshEnabled == 1
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// GetConnection retrieves a single connection profile by ID.
func (s *Store) GetConnection(id string) (*ConnectionProfile, error) {
	var c ConnectionProfile
	var useSSL, sshEnabled int
	err := s.db.QueryRow(`
		SELECT id, name, host, port, username, password, default_db, use_ssl,
		       ssh_enabled, ssh_host, ssh_port, ssh_user, ssh_auth, ssh_key_path, ssh_password,
		       sort_order, created_at, updated_at
		FROM connections WHERE id = ?
	`, id).Scan(
		&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.Password, &c.DefaultDB, &useSSL,
		&sshEnabled, &c.SSHHost, &c.SSHPort, &c.SSHUser, &c.SSHAuth, &c.SSHKeyPath, &c.SSHPass,
		&c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.UseSSL = useSSL == 1
	c.SSHEnabled = sshEnabled == 1
	return &c, nil
}

// SaveConnection creates or updates a connection profile.
// Passwords should already be encrypted before calling this.
func (s *Store) SaveConnection(c *ConnectionProfile) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if c.ID == "" {
		c.ID = uuid.New().String()
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	useSSL := 0
	if c.UseSSL {
		useSSL = 1
	}
	sshEnabled := 0
	if c.SSHEnabled {
		sshEnabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO connections (id, name, host, port, username, password, default_db, use_ssl,
		                         ssh_enabled, ssh_host, ssh_port, ssh_user, ssh_auth, ssh_key_path, ssh_password,
		                         sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, host=excluded.host, port=excluded.port,
			username=excluded.username, password=excluded.password,
			default_db=excluded.default_db, use_ssl=excluded.use_ssl,
			ssh_enabled=excluded.ssh_enabled, ssh_host=excluded.ssh_host,
			ssh_port=excluded.ssh_port, ssh_user=excluded.ssh_user,
			ssh_auth=excluded.ssh_auth, ssh_key_path=excluded.ssh_key_path,
			ssh_password=excluded.ssh_password, sort_order=excluded.sort_order,
			updated_at=excluded.updated_at
	`,
		c.ID, c.Name, c.Host, c.Port, c.Username, c.Password, c.DefaultDB, useSSL,
		sshEnabled, c.SSHHost, c.SSHPort, c.SSHUser, c.SSHAuth, c.SSHKeyPath, c.SSHPass,
		c.SortOrder, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// DeleteConnection removes a connection profile by ID.
func (s *Store) DeleteConnection(id string) error {
	_, err := s.db.Exec("DELETE FROM connections WHERE id = ?", id)
	return err
}
