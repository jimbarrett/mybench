# mybench

A personal MySQL database management tool. Go backend (Echo) + Vue 3 frontend, compiled into a single binary with `//go:embed`.

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

- **Backend:** Go, Echo v4, `database/sql` + `go-sql-driver/mysql`, SQLite for local storage (`modernc.org/sqlite`)
- **Frontend:** Vue 3 (Composition API, TypeScript), CodeMirror 6, scoped CSS (no UI framework)
- **Encryption:** AES-256-GCM with argon2id key derivation
- **Build:** Single binary with embedded frontend via `//go:embed`

## Building

Requires [Go 1.21+](https://go.dev/) and [Node.js 18+](https://nodejs.org/).

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Production build (builds frontend, then embeds into Go binary)
make build

# Run it
./bin/mybench
```

## Development

Run the backend and frontend in separate terminals:

```bash
# Terminal 1: Go backend on :8080
make dev-backend

# Terminal 2: Vite dev server on :5173 (hot reload, proxies /api to backend)
make dev-frontend
```

Open `http://localhost:5173` in your browser.

## Usage

```bash
# Start with default settings (opens browser, port 8080)
./bin/mybench

# Custom port
./bin/mybench 9090

# Don't open browser automatically
./bin/mybench --no-browser

# Check version / update
./bin/mybench version
./bin/mybench update
```

## Status

Work in progress. Core features are functional but this is a personal tool under active development. Contributions and issues welcome but no guarantees on stability or support.

## License

MIT
