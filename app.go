package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"sync"

	"mybench/internal/crypto"
	"mybench/internal/database"
	"mybench/internal/store"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct. All Wails-bound methods live here.
type App struct {
	ctx     context.Context
	store   *store.Store
	vault   *crypto.Vault
	connMgr *database.Manager

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc // per-tab cancel functions
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		connMgr: database.NewManager(),
		cancels: make(map[string]context.CancelFunc),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	s, err := store.New()
	if err != nil {
		fmt.Printf("failed to open store: %v\n", err)
		return
	}
	a.store = s
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.connMgr != nil {
		a.connMgr.CloseAll()
	}
	if a.store != nil {
		a.store.Close()
	}
}

// --- Master Password ---

// HasMasterPassword returns true if a master password has been set.
func (a *App) HasMasterPassword() (bool, error) {
	hash, err := a.store.GetConfig("master_hash")
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

// SetMasterPassword sets the master password for the first time.
func (a *App) SetMasterPassword(password string) error {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	hash := crypto.HashPassword(password, salt)
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	if err := a.store.SetConfig("master_salt", saltB64); err != nil {
		return err
	}
	if err := a.store.SetConfig("master_hash", hash); err != nil {
		return err
	}

	a.vault = crypto.NewVault(password, salt)
	return nil
}

// UnlockVault verifies the master password and unlocks encryption.
func (a *App) UnlockVault(password string) (bool, error) {
	saltB64, err := a.store.GetConfig("master_salt")
	if err != nil {
		return false, err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, err
	}

	hash, err := a.store.GetConfig("master_hash")
	if err != nil {
		return false, err
	}

	if !crypto.VerifyPassword(password, salt, hash) {
		return false, nil
	}

	a.vault = crypto.NewVault(password, salt)
	return true, nil
}

// IsUnlocked returns true if the vault has been unlocked.
func (a *App) IsUnlocked() bool {
	return a.vault != nil
}

// --- Connection Profiles ---

// ConnectionProfile mirrors the store type for Wails bindings.
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
	SSHAuth    string `json:"sshAuth"`
	SSHKeyPath string `json:"sshKeyPath"`
	SSHPass    string `json:"sshPassword"`
	SortOrder  int    `json:"sortOrder"`
}

// ListConnections returns all saved connection profiles with passwords decrypted.
func (a *App) ListConnections() ([]ConnectionProfile, error) {
	conns, err := a.store.ListConnections()
	if err != nil {
		return nil, err
	}

	result := make([]ConnectionProfile, len(conns))
	for i, c := range conns {
		pwd := c.Password
		sshPwd := c.SSHPass
		if a.vault != nil {
			if dec, err := a.vault.Decrypt(pwd); err == nil {
				pwd = dec
			}
			if dec, err := a.vault.Decrypt(sshPwd); err == nil {
				sshPwd = dec
			}
		}
		result[i] = ConnectionProfile{
			ID:         c.ID,
			Name:       c.Name,
			Host:       c.Host,
			Port:       c.Port,
			Username:   c.Username,
			Password:   pwd,
			DefaultDB:  c.DefaultDB,
			UseSSL:     c.UseSSL,
			SSHEnabled: c.SSHEnabled,
			SSHHost:    c.SSHHost,
			SSHPort:    c.SSHPort,
			SSHUser:    c.SSHUser,
			SSHAuth:    c.SSHAuth,
			SSHKeyPath: c.SSHKeyPath,
			SSHPass:    sshPwd,
			SortOrder:  c.SortOrder,
		}
	}
	return result, nil
}

// SaveConnection saves a connection profile, encrypting passwords.
func (a *App) SaveConnection(cp ConnectionProfile) (string, error) {
	pwd := cp.Password
	sshPwd := cp.SSHPass
	if a.vault != nil {
		if enc, err := a.vault.Encrypt(pwd); err == nil {
			pwd = enc
		}
		if enc, err := a.vault.Encrypt(sshPwd); err == nil {
			sshPwd = enc
		}
	}

	sc := &store.ConnectionProfile{
		ID:         cp.ID,
		Name:       cp.Name,
		Host:       cp.Host,
		Port:       cp.Port,
		Username:   cp.Username,
		Password:   pwd,
		DefaultDB:  cp.DefaultDB,
		UseSSL:     cp.UseSSL,
		SSHEnabled: cp.SSHEnabled,
		SSHHost:    cp.SSHHost,
		SSHPort:    cp.SSHPort,
		SSHUser:    cp.SSHUser,
		SSHAuth:    cp.SSHAuth,
		SSHKeyPath: cp.SSHKeyPath,
		SSHPass:    sshPwd,
		SortOrder:  cp.SortOrder,
	}

	if err := a.store.SaveConnection(sc); err != nil {
		return "", err
	}
	return sc.ID, nil
}

// DeleteConnection removes a saved connection profile.
func (a *App) DeleteConnection(id string) error {
	return a.store.DeleteConnection(id)
}

// --- Live Connections ---

// Connect opens a MySQL connection for a tab using a saved profile.
func (a *App) Connect(tabID, profileID string) error {
	conns, err := a.store.ListConnections()
	if err != nil {
		return err
	}

	var profile *store.ConnectionProfile
	for _, c := range conns {
		if c.ID == profileID {
			profile = &c
			break
		}
	}
	if profile == nil {
		return fmt.Errorf("connection profile not found: %s", profileID)
	}

	pwd := profile.Password
	if a.vault != nil {
		if dec, err := a.vault.Decrypt(pwd); err == nil {
			pwd = dec
		}
	}

	cfg := database.ConnConfig{
		Host:     profile.Host,
		Port:     profile.Port,
		Username: profile.Username,
		Password: pwd,
		Database: profile.DefaultDB,
		UseSSL:   profile.UseSSL,
	}

	return a.connMgr.Connect(tabID, profileID, cfg)
}

// Disconnect closes the connection for a tab.
func (a *App) Disconnect(tabID string) error {
	return a.connMgr.Disconnect(tabID)
}

// TestConnection attempts to connect with the given parameters without saving.
func (a *App) TestConnection(cp ConnectionProfile) error {
	cfg := database.ConnConfig{
		Host:     cp.Host,
		Port:     cp.Port,
		Username: cp.Username,
		Password: cp.Password,
		Database: cp.DefaultDB,
		UseSSL:   cp.UseSSL,
	}

	// Use a temporary tab ID for testing.
	err := a.connMgr.Connect("__test__", "", cfg)
	if err != nil {
		return err
	}
	a.connMgr.Disconnect("__test__")
	return nil
}

// PingConnection checks if a tab's connection is alive.
func (a *App) PingConnection(tabID string) error {
	return a.connMgr.Ping(tabID)
}

// --- Schema Introspection ---

// GetDatabases returns all databases visible on a connection.
func (a *App) GetDatabases(tabID string) ([]database.DatabaseInfo, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ListDatabases(conn.DB)
}

// GetTables returns tables and views in a database.
func (a *App) GetTables(tabID, dbName string) ([]database.TableInfo, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ListTables(conn.DB, dbName)
}

// GetTableDetail returns columns, indexes, foreign keys, and DDL for a table.
func (a *App) GetTableDetail(tabID, dbName, tableName string) (*database.TableDetail, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.GetTableDetail(conn.DB, dbName, tableName)
}

// GetRoutines returns stored procedures and functions in a database.
func (a *App) GetRoutines(tabID, dbName string) ([]database.RoutineInfo, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ListRoutines(conn.DB, dbName)
}

// GetTriggers returns triggers in a database.
func (a *App) GetTriggers(tabID, dbName string) ([]database.TriggerInfo, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ListTriggers(conn.DB, dbName)
}

// --- User Management ---

// ListUsers returns all MySQL users on a connection.
func (a *App) ListUsers(tabID string) ([]database.UserInfo, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ListUsers(conn.DB)
}

// GetUserDetail returns a user's full info including grants.
func (a *App) GetUserDetail(tabID, user, host string) (*database.UserDetail, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.GetUserDetail(conn.DB, user, host)
}

// CreateUser creates a new MySQL user.
func (a *App) CreateUser(tabID, user, host, password, plugin string) error {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.CreateUser(conn.DB, user, host, password, plugin)
}

// DropUser drops a MySQL user.
func (a *App) DropUser(tabID, user, host string) error {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.DropUser(conn.DB, user, host)
}

// ChangeUserPassword changes a MySQL user's password.
func (a *App) ChangeUserPassword(tabID, user, host, newPassword string) error {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.ChangePassword(conn.DB, user, host, newPassword)
}

// GrantPrivileges grants privileges to a MySQL user.
func (a *App) GrantPrivileges(tabID, user, host, privileges, on string) error {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.GrantPrivileges(conn.DB, user, host, privileges, on)
}

// RevokePrivileges revokes privileges from a MySQL user.
func (a *App) RevokePrivileges(tabID, user, host, privileges, on string) error {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.RevokePrivileges(conn.DB, user, host, privileges, on)
}

// --- Autocomplete ---

// GetSchemaCompletions returns schema metadata formatted for the editor's autocomplete.
// Returns a map of "database.table" -> []column_name, plus bare table -> []column_name
// for the default database.
func (a *App) GetSchemaCompletions(tabID string) (map[string][]string, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return database.GetCompletionSchema(conn.DB)
}

// --- Query Execution ---

// ExecuteQuery runs a SQL query (or multiple semicolon-separated statements) on a tab's connection.
func (a *App) ExecuteQuery(tabID, sql string) []database.QueryResult {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return []database.QueryResult{{Error: fmt.Sprintf("not connected on tab %s", tabID)}}
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelMu.Lock()
	a.cancels[tabID] = cancel
	a.cancelMu.Unlock()

	defer func() {
		cancel()
		a.cancelMu.Lock()
		delete(a.cancels, tabID)
		a.cancelMu.Unlock()
	}()

	return database.ExecuteMulti(ctx, conn.DB, sql)
}

// ExplainQuery runs EXPLAIN on a query.
func (a *App) ExplainQuery(tabID, sql string) *database.QueryResult {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return &database.QueryResult{Error: fmt.Sprintf("not connected on tab %s", tabID)}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return database.ExplainQuery(ctx, conn.DB, sql)
}

// CancelQuery cancels the currently running query on a tab.
func (a *App) CancelQuery(tabID string) error {
	// Cancel the context first.
	a.cancelMu.Lock()
	if cancel, ok := a.cancels[tabID]; ok {
		cancel()
	}
	a.cancelMu.Unlock()

	// Also send KILL QUERY to MySQL for immediate effect.
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil
	}
	connID, err := database.GetConnectionID(conn.DB)
	if err != nil {
		return err
	}
	return database.KillQuery(conn.DB, connID)
}

// --- Export ---

// ExportResultsCSV exports the given query results to a CSV file via save dialog.
func (a *App) ExportResultsCSV(columns []string, rows [][]string) (string, error) {
	filepath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Results to CSV",
		DefaultFilename: "results.csv",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if filepath == "" {
		return "", nil // cancelled
	}

	f, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := database.ExportResultCSV(f, columns, rows); err != nil {
		return "", err
	}
	return filepath, nil
}

// ExportResultsSQL exports the given query results as SQL INSERT statements via save dialog.
func (a *App) ExportResultsSQL(tableName string, columns []string, rows [][]string) (string, error) {
	filepath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Results to SQL",
		DefaultFilename: tableName + ".sql",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "SQL Files", Pattern: "*.sql"},
		},
	})
	if err != nil {
		return "", err
	}
	if filepath == "" {
		return "", nil
	}

	f, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := database.ExportResultSQL(f, tableName, columns, rows); err != nil {
		return "", err
	}
	return filepath, nil
}

// ExportTableCSV exports an entire table to CSV via save dialog (streamed).
func (a *App) ExportTableCSV(tabID, dbName, tableName string) (string, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return "", fmt.Errorf("not connected on tab %s", tabID)
	}

	filepath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Table to CSV",
		DefaultFilename: tableName + ".csv",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if filepath == "" {
		return "", nil
	}

	f, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelMu.Lock()
	a.cancels[tabID+"_export"] = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		delete(a.cancels, tabID+"_export")
		a.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		wailsRuntime.EventsEmit(a.ctx, "export-progress", map[string]interface{}{
			"current": current,
			"total":   total,
		})
		return ctx.Err() == nil
	}

	if err := database.ExportTableCSV(ctx, conn.DB, dbName, tableName, f, progress); err != nil {
		return "", err
	}
	return filepath, nil
}

// ExportTableSQL exports an entire table as SQL INSERT statements via save dialog (streamed).
func (a *App) ExportTableSQL(tabID, dbName, tableName string) (string, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return "", fmt.Errorf("not connected on tab %s", tabID)
	}

	filepath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Table to SQL",
		DefaultFilename: tableName + ".sql",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "SQL Files", Pattern: "*.sql"},
		},
	})
	if err != nil {
		return "", err
	}
	if filepath == "" {
		return "", nil
	}

	f, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelMu.Lock()
	a.cancels[tabID+"_export"] = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		delete(a.cancels, tabID+"_export")
		a.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		wailsRuntime.EventsEmit(a.ctx, "export-progress", map[string]interface{}{
			"current": current,
			"total":   total,
		})
		return ctx.Err() == nil
	}

	if err := database.ExportTableSQL(ctx, conn.DB, dbName, tableName, f, progress); err != nil {
		return "", err
	}
	return filepath, nil
}

// --- Import ---

// CSVImportPreview wraps a CSV preview with the selected file path.
type CSVImportPreview struct {
	FilePath   string             `json:"filePath"`
	Headers    []string           `json:"headers"`
	SampleRows [][]string         `json:"sampleRows"`
	TotalRows  int                `json:"totalRows"`
}

// ImportOpenCSV opens a file dialog for CSV selection and returns a preview.
func (a *App) ImportOpenCSV() (*CSVImportPreview, error) {
	filepath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Import CSV File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return nil, err
	}
	if filepath == "" {
		return nil, nil
	}

	preview, err := database.PreviewCSV(filepath, 5)
	if err != nil {
		return nil, err
	}
	return &CSVImportPreview{
		FilePath:   filepath,
		Headers:    preview.Headers,
		SampleRows: preview.SampleRows,
		TotalRows:  preview.TotalRows,
	}, nil
}

// ImportCSV imports a CSV file into a table with given column mappings.
func (a *App) ImportCSV(tabID, dbName, tableName, filePath string, mappings []database.ColumnMapping) (int64, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return 0, fmt.Errorf("not connected on tab %s", tabID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelMu.Lock()
	a.cancels[tabID+"_import"] = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		delete(a.cancels, tabID+"_import")
		a.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		wailsRuntime.EventsEmit(a.ctx, "import-progress", map[string]interface{}{
			"current": current,
			"total":   total,
		})
		return ctx.Err() == nil
	}

	return database.ImportCSV(ctx, conn.DB, dbName, tableName, filePath, mappings, progress)
}

// ImportOpenSQL opens a file dialog for SQL file selection and executes it.
func (a *App) ImportSQL(tabID string) (int64, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return 0, fmt.Errorf("not connected on tab %s", tabID)
	}

	filepath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Import SQL File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "SQL Files", Pattern: "*.sql"},
		},
	})
	if err != nil {
		return 0, err
	}
	if filepath == "" {
		return 0, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelMu.Lock()
	a.cancels[tabID+"_import"] = cancel
	a.cancelMu.Unlock()
	defer func() {
		cancel()
		a.cancelMu.Lock()
		delete(a.cancels, tabID+"_import")
		a.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		wailsRuntime.EventsEmit(a.ctx, "import-progress", map[string]interface{}{
			"current": current,
			"total":   total,
		})
		return ctx.Err() == nil
	}

	return database.ImportSQLFile(ctx, conn.DB, filepath, progress)
}

// CancelImportExport cancels a running import or export operation.
func (a *App) CancelImportExport(tabID string) {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	for _, suffix := range []string{"_import", "_export"} {
		if cancel, ok := a.cancels[tabID+suffix]; ok {
			cancel()
		}
	}
}

// GetTableColumns returns just the column names for a table (used for import column mapping).
func (a *App) GetTableColumns(tabID, dbName, tableName string) ([]string, error) {
	conn := a.connMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}

	detail, err := database.GetTableDetail(conn.DB, dbName, tableName)
	if err != nil {
		return nil, err
	}

	cols := make([]string, len(detail.Columns))
	for i, c := range detail.Columns {
		cols[i] = c.Name
	}
	return cols, nil
}

// --- Utility ---

// Ping verifies the Go backend is reachable from the frontend.
func (a *App) Ping() map[string]string {
	return map[string]string{
		"status":  "ok",
		"go":      runtime.Version(),
		"app":     "mybench",
		"version": "0.1.0",
	}
}
