# mybench

A personal MySQL database management tool. Built as a native desktop app with [Wails](https://wails.io/) (Go backend + Vue 3 frontend).

## What it does

- Connect to multiple MySQL servers simultaneously (localhost, remote, DigitalOcean managed clusters with SSL)
- Browse schemas: databases, tables, views, stored procedures, triggers
- Inspect table structure: columns, indexes, foreign keys, DDL
- Query editor with MySQL syntax highlighting, autocomplete, and vim-friendly keybindings
- Execute queries, view results, EXPLAIN plans, cancel running queries
- Multi-statement execution (split by `;`, results per statement)
- Quick-query tables directly from the schema tree
- User management: list, create, drop users, view/grant/revoke privileges
- Import CSV files with column mapping, import/execute SQL files
- Export query results or full tables to CSV or SQL INSERT statements
- Master password encryption for stored credentials (AES-256-GCM, argon2 key derivation)
- Tokyo Night dark theme, resizable panels, keyboard-driven workflow

## Tech stack

- **Backend:** Go, Wails v2, `database/sql` + `go-sql-driver/mysql`, SQLite for local storage (`modernc.org/sqlite`)
- **Frontend:** Vue 3 (Composition API, TypeScript), CodeMirror 6, scoped CSS (no UI framework)
- **Encryption:** AES-256-GCM with argon2id key derivation
- **Build:** Single native binary via Wails

## Building

Requires [Go 1.21+](https://go.dev/), [Node.js 18+](https://nodejs.org/), and [Wails CLI](https://wails.io/docs/gettingstarted/installation).

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Development (hot reload)
wails dev -tags webkit2_41

# Production build
wails build -tags webkit2_41
```

> The `-tags webkit2_41` flag is required on Ubuntu 24.04+ (ships webkit2gtk-4.1 instead of 4.0).

## Status

Work in progress. Core features are functional but this is a personal tool under active development. Contributions and issues welcome but no guarantees on stability or support.

## License

MIT
