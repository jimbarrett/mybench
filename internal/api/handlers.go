package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"

	"mybench/internal/crypto"
	"mybench/internal/database"
	"mybench/internal/store"

	"github.com/labstack/echo/v4"
)

// Handlers holds all HTTP handler methods and shared state.
type Handlers struct {
	Version string
	Store   *store.Store
	Vault   *crypto.Vault
	ConnMgr *database.Manager

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc

	// SSE: per-tab event channels
	sseMu    sync.Mutex
	sseChans map[string][]chan sseEvent
}

type sseEvent struct {
	Event string
	Data  interface{}
}

func NewHandlers(version string, s *store.Store, connMgr *database.Manager) *Handlers {
	return &Handlers{
		Version: version,
		Store:   s,
		ConnMgr: connMgr,
		cancels: make(map[string]context.CancelFunc),
		sseChans: make(map[string][]chan sseEvent),
	}
}

func (h *Handlers) Shutdown() {
	if h.ConnMgr != nil {
		h.ConnMgr.CloseAll()
	}
	if h.Store != nil {
		h.Store.Close()
	}
}

// emitEvent sends an SSE event to all listeners for a tab.
func (h *Handlers) emitEvent(tabID, event string, data interface{}) {
	h.sseMu.Lock()
	chans := h.sseChans[tabID]
	h.sseMu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- sseEvent{Event: event, Data: data}:
		default:
		}
	}
}

// --- Health ---

func (h *Handlers) ping(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"go":      runtime.Version(),
		"app":     "mybench",
		"version": h.Version,
	})
}

// --- Vault / Auth ---

func (h *Handlers) vaultStatus(c echo.Context) error {
	hash, err := h.Store.GetConfig("master_hash")
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"hasMasterPassword": hash != "",
		"isUnlocked":        h.Vault != nil,
	})
}

func (h *Handlers) vaultCreate(c echo.Context) error {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return jsonErr(c, err)
	}

	hash := crypto.HashPassword(body.Password, salt)
	saltB64 := crypto.EncodeSalt(salt)

	if err := h.Store.SetConfig("master_salt", saltB64); err != nil {
		return jsonErr(c, err)
	}
	if err := h.Store.SetConfig("master_hash", hash); err != nil {
		return jsonErr(c, err)
	}

	h.Vault = crypto.NewVault(body.Password, salt)
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) vaultUnlock(c echo.Context) error {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	saltB64, err := h.Store.GetConfig("master_salt")
	if err != nil {
		return jsonErr(c, err)
	}
	salt, err := crypto.DecodeSalt(saltB64)
	if err != nil {
		return jsonErr(c, err)
	}

	hash, err := h.Store.GetConfig("master_hash")
	if err != nil {
		return jsonErr(c, err)
	}

	if !crypto.VerifyPassword(body.Password, salt, hash) {
		return c.JSON(http.StatusOK, map[string]bool{"ok": false})
	}

	h.Vault = crypto.NewVault(body.Password, salt)
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- Connections ---

type connectionProfile struct {
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

func (h *Handlers) listConnections(c echo.Context) error {
	conns, err := h.Store.ListConnections()
	if err != nil {
		return jsonErr(c, err)
	}

	result := make([]connectionProfile, len(conns))
	for i, conn := range conns {
		pwd := conn.Password
		sshPwd := conn.SSHPass
		if h.Vault != nil {
			if dec, err := h.Vault.Decrypt(pwd); err == nil {
				pwd = dec
			}
			if dec, err := h.Vault.Decrypt(sshPwd); err == nil {
				sshPwd = dec
			}
		}
		result[i] = connectionProfile{
			ID:         conn.ID,
			Name:       conn.Name,
			Host:       conn.Host,
			Port:       conn.Port,
			Username:   conn.Username,
			Password:   pwd,
			DefaultDB:  conn.DefaultDB,
			UseSSL:     conn.UseSSL,
			SSHEnabled: conn.SSHEnabled,
			SSHHost:    conn.SSHHost,
			SSHPort:    conn.SSHPort,
			SSHUser:    conn.SSHUser,
			SSHAuth:    conn.SSHAuth,
			SSHKeyPath: conn.SSHKeyPath,
			SSHPass:    sshPwd,
			SortOrder:  conn.SortOrder,
		}
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handlers) saveConnection(c echo.Context) error {
	var cp connectionProfile
	if err := c.Bind(&cp); err != nil {
		return jsonErr(c, err)
	}
	id, err := h.saveConn(cp)
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"id": id})
}

func (h *Handlers) updateConnection(c echo.Context) error {
	var cp connectionProfile
	if err := c.Bind(&cp); err != nil {
		return jsonErr(c, err)
	}
	cp.ID = c.Param("id")
	id, err := h.saveConn(cp)
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"id": id})
}

func (h *Handlers) saveConn(cp connectionProfile) (string, error) {
	pwd := cp.Password
	sshPwd := cp.SSHPass
	if h.Vault != nil {
		if enc, err := h.Vault.Encrypt(pwd); err == nil {
			pwd = enc
		}
		if enc, err := h.Vault.Encrypt(sshPwd); err == nil {
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

	if err := h.Store.SaveConnection(sc); err != nil {
		return "", err
	}
	return sc.ID, nil
}

func (h *Handlers) deleteConnection(c echo.Context) error {
	id := c.Param("id")
	if err := h.Store.DeleteConnection(id); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) testConnection(c echo.Context) error {
	var cp connectionProfile
	if err := c.Bind(&cp); err != nil {
		return jsonErr(c, err)
	}

	cfg := database.ConnConfig{
		Host:     cp.Host,
		Port:     cp.Port,
		Username: cp.Username,
		Password: cp.Password,
		Database: cp.DefaultDB,
		UseSSL:   cp.UseSSL,
	}

	err := h.ConnMgr.Connect("__test__", "", cfg)
	if err != nil {
		return jsonErr(c, err)
	}
	h.ConnMgr.Disconnect("__test__")
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- Tabs / Active Connections ---

func (h *Handlers) connect(c echo.Context) error {
	tabID := c.Param("id")
	var body struct {
		ProfileID string `json:"profileId"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	conns, err := h.Store.ListConnections()
	if err != nil {
		return jsonErr(c, err)
	}

	var profile *store.ConnectionProfile
	for _, conn := range conns {
		if conn.ID == body.ProfileID {
			profile = &conn
			break
		}
	}
	if profile == nil {
		return jsonErr(c, fmt.Errorf("connection profile not found: %s", body.ProfileID))
	}

	pwd := profile.Password
	if h.Vault != nil {
		if dec, err := h.Vault.Decrypt(pwd); err == nil {
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

	if err := h.ConnMgr.Connect(tabID, body.ProfileID, cfg); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) disconnect(c echo.Context) error {
	tabID := c.Param("id")
	if err := h.ConnMgr.Disconnect(tabID); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) pingConnection(c echo.Context) error {
	tabID := c.Param("id")
	if err := h.ConnMgr.Ping(tabID); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- Schema ---

func (h *Handlers) getConn(c echo.Context) (*database.Connection, error) {
	tabID := c.Param("id")
	conn := h.ConnMgr.Get(tabID)
	if conn == nil {
		return nil, fmt.Errorf("not connected on tab %s", tabID)
	}
	return conn, nil
}

func (h *Handlers) getDatabases(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	dbs, err := database.ListDatabases(conn.DB)
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, dbs)
}

func (h *Handlers) getTables(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	tables, err := database.ListTables(conn.DB, c.Param("db"))
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, tables)
}

func (h *Handlers) getTableDetail(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	detail, err := database.GetTableDetail(conn.DB, c.Param("db"), c.Param("table"))
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, detail)
}

func (h *Handlers) getTableColumns(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	detail, err := database.GetTableDetail(conn.DB, c.Param("db"), c.Param("table"))
	if err != nil {
		return jsonErr(c, err)
	}
	cols := make([]string, len(detail.Columns))
	for i, col := range detail.Columns {
		cols[i] = col.Name
	}
	return c.JSON(http.StatusOK, cols)
}

func (h *Handlers) getRoutines(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	routines, err := database.ListRoutines(conn.DB, c.Param("db"))
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, routines)
}

func (h *Handlers) getTriggers(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	triggers, err := database.ListTriggers(conn.DB, c.Param("db"))
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, triggers)
}

func (h *Handlers) getSchemaCompletions(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	schema, err := database.GetCompletionSchema(conn.DB)
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, schema)
}

// --- Queries ---

func (h *Handlers) executeQuery(c echo.Context) error {
	tabID := c.Param("id")
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	var body struct {
		SQL string `json:"sql"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancels[tabID] = cancel
	h.cancelMu.Unlock()

	defer func() {
		cancel()
		h.cancelMu.Lock()
		delete(h.cancels, tabID)
		h.cancelMu.Unlock()
	}()

	results := database.ExecuteMulti(ctx, conn.DB, body.SQL)
	return c.JSON(http.StatusOK, results)
}

func (h *Handlers) explainQuery(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	var body struct {
		SQL string `json:"sql"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := database.ExplainQuery(ctx, conn.DB, body.SQL)
	return c.JSON(http.StatusOK, result)
}

func (h *Handlers) cancelQuery(c echo.Context) error {
	tabID := c.Param("id")

	h.cancelMu.Lock()
	if cancel, ok := h.cancels[tabID]; ok {
		cancel()
	}
	h.cancelMu.Unlock()

	conn := h.ConnMgr.Get(tabID)
	if conn != nil {
		connID, err := database.GetConnectionID(conn.DB)
		if err == nil {
			database.KillQuery(conn.DB, connID)
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- Users ---

func (h *Handlers) listUsers(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	users, err := database.ListUsers(conn.DB)
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handlers) getUserDetail(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	detail, err := database.GetUserDetail(conn.DB, c.Param("user"), c.Param("host"))
	if err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, detail)
}

func (h *Handlers) createUser(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	var body struct {
		User     string `json:"user"`
		Host     string `json:"host"`
		Password string `json:"password"`
		Plugin   string `json:"plugin"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	if err := database.CreateUser(conn.DB, body.User, body.Host, body.Password, body.Plugin); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) dropUser(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	if err := database.DropUser(conn.DB, c.Param("user"), c.Param("host")); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) changeUserPassword(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}
	if err := database.ChangePassword(conn.DB, c.Param("user"), c.Param("host"), body.Password); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) grantPrivileges(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	var body struct {
		Privileges string `json:"privileges"`
		On         string `json:"on"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}
	if err := database.GrantPrivileges(conn.DB, c.Param("user"), c.Param("host"), body.Privileges, body.On); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) revokePrivileges(c echo.Context) error {
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}
	var body struct {
		Privileges string `json:"privileges"`
		On         string `json:"on"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}
	if err := database.RevokePrivileges(conn.DB, c.Param("user"), c.Param("host"), body.Privileges, body.On); err != nil {
		return jsonErr(c, err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- Export ---

func (h *Handlers) exportTableCSV(c echo.Context) error {
	tabID := c.Param("id")
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	dbName := c.QueryParam("db")
	tableName := c.QueryParam("table")

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancels[tabID+"_export"] = cancel
	h.cancelMu.Unlock()
	defer func() {
		cancel()
		h.cancelMu.Lock()
		delete(h.cancels, tabID+"_export")
		h.cancelMu.Unlock()
	}()

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, tableName))

	progress := func(current, total int64) bool {
		h.emitEvent(tabID, "export-progress", map[string]int64{"current": current, "total": total})
		return ctx.Err() == nil
	}

	return database.ExportTableCSV(ctx, conn.DB, dbName, tableName, c.Response(), progress)
}

func (h *Handlers) exportTableSQL(c echo.Context) error {
	tabID := c.Param("id")
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	dbName := c.QueryParam("db")
	tableName := c.QueryParam("table")

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancels[tabID+"_export"] = cancel
	h.cancelMu.Unlock()
	defer func() {
		cancel()
		h.cancelMu.Lock()
		delete(h.cancels, tabID+"_export")
		h.cancelMu.Unlock()
	}()

	c.Response().Header().Set("Content-Type", "application/sql")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.sql"`, tableName))

	progress := func(current, total int64) bool {
		h.emitEvent(tabID, "export-progress", map[string]int64{"current": current, "total": total})
		return ctx.Err() == nil
	}

	return database.ExportTableSQL(ctx, conn.DB, dbName, tableName, c.Response(), progress)
}

func (h *Handlers) exportResultsCSV(c echo.Context) error {
	var body struct {
		Columns []string   `json:"columns"`
		Rows    [][]string `json:"rows"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="results.csv"`)

	return database.ExportResultCSV(c.Response(), body.Columns, body.Rows)
}

func (h *Handlers) exportResultsSQL(c echo.Context) error {
	var body struct {
		TableName string     `json:"tableName"`
		Columns   []string   `json:"columns"`
		Rows      [][]string `json:"rows"`
	}
	if err := c.Bind(&body); err != nil {
		return jsonErr(c, err)
	}

	c.Response().Header().Set("Content-Type", "application/sql")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.sql"`, body.TableName))

	return database.ExportResultSQL(c.Response(), body.TableName, body.Columns, body.Rows)
}

// --- Import ---

func (h *Handlers) importCSVPreview(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return jsonErr(c, fmt.Errorf("no file uploaded: %w", err))
	}

	src, err := file.Open()
	if err != nil {
		return jsonErr(c, err)
	}
	defer src.Close()

	// Save to temp file for later import
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "mybench-csv-*.csv")
	if err != nil {
		return jsonErr(c, err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return jsonErr(c, err)
	}
	tmpFile.Close()

	preview, err := database.PreviewCSV(tmpPath, 5)
	if err != nil {
		os.Remove(tmpPath)
		return jsonErr(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"filePath":   tmpPath,
		"headers":    preview.Headers,
		"sampleRows": preview.SampleRows,
		"totalRows":  preview.TotalRows,
	})
}

func (h *Handlers) importCSV(c echo.Context) error {
	tabID := c.Param("id")
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	// Check if file is uploaded as multipart or using saved temp path
	var filePath string
	var dbName, tableName, mappingsJSON string

	file, fileErr := c.FormFile("file")
	if fileErr == nil {
		// File uploaded directly
		src, err := file.Open()
		if err != nil {
			return jsonErr(c, err)
		}
		defer src.Close()

		tmpFile, err := os.CreateTemp(os.TempDir(), "mybench-csv-*.csv")
		if err != nil {
			return jsonErr(c, err)
		}
		if _, err := io.Copy(tmpFile, src); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return jsonErr(c, err)
		}
		tmpFile.Close()
		filePath = tmpFile.Name()
		defer os.Remove(filePath)
	} else {
		filePath = c.FormValue("filePath")
	}

	dbName = c.FormValue("db")
	tableName = c.FormValue("table")
	mappingsJSON = c.FormValue("mappings")

	var mappings []database.ColumnMapping
	if err := json.Unmarshal([]byte(mappingsJSON), &mappings); err != nil {
		return jsonErr(c, fmt.Errorf("invalid mappings: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancels[tabID+"_import"] = cancel
	h.cancelMu.Unlock()
	defer func() {
		cancel()
		h.cancelMu.Lock()
		delete(h.cancels, tabID+"_import")
		h.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		h.emitEvent(tabID, "import-progress", map[string]int64{"current": current, "total": total})
		return ctx.Err() == nil
	}

	rows, err := database.ImportCSV(ctx, conn.DB, dbName, tableName, filePath, mappings, progress)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"rows": rows, "error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"rows": rows})
}

func (h *Handlers) importSQL(c echo.Context) error {
	tabID := c.Param("id")
	conn, err := h.getConn(c)
	if err != nil {
		return jsonErr(c, err)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return jsonErr(c, fmt.Errorf("no file uploaded: %w", err))
	}

	src, err := file.Open()
	if err != nil {
		return jsonErr(c, err)
	}
	defer src.Close()

	// Save to temp file
	tmpFile, err := os.CreateTemp(os.TempDir(), "mybench-sql-*.sql")
	if err != nil {
		return jsonErr(c, err)
	}
	if _, err := io.Copy(tmpFile, src); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return jsonErr(c, err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancels[tabID+"_import"] = cancel
	h.cancelMu.Unlock()
	defer func() {
		cancel()
		h.cancelMu.Lock()
		delete(h.cancels, tabID+"_import")
		h.cancelMu.Unlock()
	}()

	progress := func(current, total int64) bool {
		h.emitEvent(tabID, "import-progress", map[string]int64{"current": current, "total": total})
		return ctx.Err() == nil
	}

	executed, err := database.ImportSQLFile(ctx, conn.DB, tmpFile.Name(), progress)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"statements": executed, "error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"statements": executed})
}

func (h *Handlers) cancelImportExport(c echo.Context) error {
	tabID := c.Param("id")
	h.cancelMu.Lock()
	defer h.cancelMu.Unlock()
	for _, suffix := range []string{"_import", "_export"} {
		if cancel, ok := h.cancels[tabID+suffix]; ok {
			cancel()
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// --- SSE Events ---

func (h *Handlers) events(c echo.Context) error {
	tabID := c.Param("id")

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	ch := make(chan sseEvent, 16)

	h.sseMu.Lock()
	h.sseChans[tabID] = append(h.sseChans[tabID], ch)
	h.sseMu.Unlock()

	defer func() {
		h.sseMu.Lock()
		chans := h.sseChans[tabID]
		for i, c := range chans {
			if c == ch {
				h.sseChans[tabID] = append(chans[:i], chans[i+1:]...)
				break
			}
		}
		h.sseMu.Unlock()
		close(ch)
	}()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case evt := <-ch:
			data, _ := json.Marshal(evt.Data)
			fmt.Fprintf(c.Response(), "event: %s\ndata: %s\n\n", evt.Event, data)
			c.Response().Flush()
		}
	}
}

// --- Helpers ---

func jsonErr(c echo.Context, err error) error {
	return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
}
