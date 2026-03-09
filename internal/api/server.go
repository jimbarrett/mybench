package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"mybench/frontend"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func StartServer(ctx context.Context, h *Handlers, port string) error {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS for development (Vite dev server on a different port)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	api := e.Group("/api")

	// Health
	api.GET("/ping", h.ping)

	// Vault / Auth
	api.GET("/vault/status", h.vaultStatus)
	api.POST("/vault/create", h.vaultCreate)
	api.POST("/vault/unlock", h.vaultUnlock)

	// Connections
	api.GET("/connections", h.listConnections)
	api.POST("/connections", h.saveConnection)
	api.PUT("/connections/:id", h.updateConnection)
	api.DELETE("/connections/:id", h.deleteConnection)
	api.POST("/connections/:id/test", h.testConnection)

	// Tabs / Active Connections
	api.POST("/tabs/:id/connect", h.connect)
	api.POST("/tabs/:id/disconnect", h.disconnect)
	api.GET("/tabs/:id/ping", h.pingConnection)

	// Schema
	api.GET("/tabs/:id/databases", h.getDatabases)
	api.GET("/tabs/:id/databases/:db/tables", h.getTables)
	api.GET("/tabs/:id/databases/:db/tables/:table", h.getTableDetail)
	api.GET("/tabs/:id/databases/:db/tables/:table/columns", h.getTableColumns)
	api.GET("/tabs/:id/databases/:db/routines", h.getRoutines)
	api.GET("/tabs/:id/databases/:db/triggers", h.getTriggers)
	api.GET("/tabs/:id/completions", h.getSchemaCompletions)

	// Queries
	api.POST("/tabs/:id/query", h.executeQuery)
	api.POST("/tabs/:id/explain", h.explainQuery)
	api.POST("/tabs/:id/cancel", h.cancelQuery)

	// Users
	api.GET("/tabs/:id/users", h.listUsers)
	api.GET("/tabs/:id/users/:user/:host", h.getUserDetail)
	api.POST("/tabs/:id/users", h.createUser)
	api.DELETE("/tabs/:id/users/:user/:host", h.dropUser)
	api.PUT("/tabs/:id/users/:user/:host/password", h.changeUserPassword)
	api.POST("/tabs/:id/users/:user/:host/grant", h.grantPrivileges)
	api.POST("/tabs/:id/users/:user/:host/revoke", h.revokePrivileges)

	// Export
	api.GET("/tabs/:id/export/csv", h.exportTableCSV)
	api.GET("/tabs/:id/export/sql", h.exportTableSQL)
	api.POST("/tabs/:id/export/results/csv", h.exportResultsCSV)
	api.POST("/tabs/:id/export/results/sql", h.exportResultsSQL)

	// Import
	api.POST("/tabs/:id/import/csv/preview", h.importCSVPreview)
	api.POST("/tabs/:id/import/csv", h.importCSV)
	api.POST("/tabs/:id/import/sql", h.importSQL)
	api.POST("/tabs/:id/import-export/cancel", h.cancelImportExport)

	// SSE events
	api.GET("/tabs/:id/events", h.events)

	// Serve embedded frontend
	serveFrontend(e)

	fmt.Printf("mybench running at http://localhost:%s\n", port)

	errCh := make(chan error, 1)
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5e9)
		defer cancel()
		return e.Shutdown(shutdownCtx)
	}
}

func serveFrontend(e *echo.Echo) {
	distFS, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		e.GET("/*", func(c echo.Context) error {
			return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html><head><title>mybench</title></head>
<body><h1>mybench</h1><p>Frontend assets not found. Rebuild with <code>make build</code>.</p></body>
</html>`)
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		// SPA fallback: serve index.html for unknown paths
		if _, err := fs.Stat(distFS, path); err != nil {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})))
}
