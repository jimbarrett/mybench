package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// UserInfo holds basic MySQL user metadata.
type UserInfo struct {
	User   string `json:"user"`
	Host   string `json:"host"`
	Plugin string `json:"plugin"`
}

// UserDetail holds full user info including grants.
type UserDetail struct {
	User   string   `json:"user"`
	Host   string   `json:"host"`
	Plugin string   `json:"plugin"`
	Grants []string `json:"grants"`
}

// ListUsers returns all MySQL users.
func ListUsers(db *sql.DB) ([]UserInfo, error) {
	rows, err := db.Query(`
		SELECT User, Host, IFNULL(plugin, '')
		FROM mysql.user
		ORDER BY User, Host
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.User, &u.Host, &u.Plugin); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserDetail returns a user's full info including grants.
func GetUserDetail(db *sql.DB, user, host string) (*UserDetail, error) {
	detail := &UserDetail{User: user, Host: host}

	// Get plugin
	err := db.QueryRow(
		"SELECT IFNULL(plugin, '') FROM mysql.user WHERE User = ? AND Host = ?",
		user, host,
	).Scan(&detail.Plugin)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get grants
	rows, err := db.Query(fmt.Sprintf("SHOW GRANTS FOR '%s'@'%s'", user, host))
	if err != nil {
		return nil, fmt.Errorf("failed to get grants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return nil, err
		}
		detail.Grants = append(detail.Grants, grant)
	}

	return detail, rows.Err()
}

// CreateUser creates a new MySQL user.
func CreateUser(db *sql.DB, user, host, password, plugin string) error {
	if host == "" {
		host = "%"
	}
	if plugin == "" {
		plugin = "caching_sha2_password"
	}

	query := fmt.Sprintf(
		"CREATE USER '%s'@'%s' IDENTIFIED WITH %s BY '%s'",
		escapeQuote(user), escapeQuote(host), plugin, escapeQuote(password),
	)
	_, err := db.Exec(query)
	return err
}

// DropUser drops a MySQL user.
func DropUser(db *sql.DB, user, host string) error {
	query := fmt.Sprintf("DROP USER '%s'@'%s'", escapeQuote(user), escapeQuote(host))
	_, err := db.Exec(query)
	return err
}

// ChangePassword changes a user's password.
func ChangePassword(db *sql.DB, user, host, newPassword string) error {
	query := fmt.Sprintf(
		"ALTER USER '%s'@'%s' IDENTIFIED BY '%s'",
		escapeQuote(user), escapeQuote(host), escapeQuote(newPassword),
	)
	_, err := db.Exec(query)
	return err
}

// GrantPrivileges grants privileges to a user.
func GrantPrivileges(db *sql.DB, user, host, privileges, on string) error {
	if on == "" {
		on = "*.*"
	}
	query := fmt.Sprintf(
		"GRANT %s ON %s TO '%s'@'%s'",
		privileges, on, escapeQuote(user), escapeQuote(host),
	)
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	_, err = db.Exec("FLUSH PRIVILEGES")
	return err
}

// RevokePrivileges revokes privileges from a user.
func RevokePrivileges(db *sql.DB, user, host, privileges, on string) error {
	if on == "" {
		on = "*.*"
	}
	query := fmt.Sprintf(
		"REVOKE %s ON %s FROM '%s'@'%s'",
		privileges, on, escapeQuote(user), escapeQuote(host),
	)
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	_, err = db.Exec("FLUSH PRIVILEGES")
	return err
}

func escapeQuote(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}
