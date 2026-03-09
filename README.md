# mybench

A personal MySQL database management tool. Single binary — no external dependencies beyond MySQL itself.

## Features

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

## Installation

Download the latest binary for your platform from the [Releases](https://github.com/jimbarrett/mybench/releases) page.

### Linux

```bash
mkdir -p ~/.local/bin
chmod +x mybench_*
mv mybench_* ~/.local/bin/mybench
```

If `~/.local/bin` is not on your PATH, add this to your `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Updating

```bash
mybench update
```

This checks GitHub for the latest release, downloads it, and replaces the binary in place.

## Usage

```bash
mybench              # Start in background, open browser (port auto-selected starting at 10200)
mybench start 9090   # Start on a specific port
mybench stop         # Stop the background process
mybench version      # Show version and check for updates
mybench update       # Update to the latest version
```

Runs as a background daemon — no terminal window needed. Data and logs are stored in `~/.config/mybench/`.

## Building from Source

Requires [Go 1.24+](https://go.dev/) and [Node.js 18+](https://nodejs.org/).

```bash
git clone git@github.com:jimbarrett/mybench.git
cd mybench
cd frontend && npm install && cd ..
make build
./bin/mybench
```

### Make Targets

| Target | Description |
|---|---|
| `make build` | Build frontend + Go binary |
| `make build-frontend` | Build only the Vue frontend |
| `make build-backend` | Build only the Go binary |
| `make dev-backend` | Run Go server in foreground (for development) |
| `make dev-frontend` | Run Vite dev server with hot-reload |
| `make clean` | Remove build artifacts |

### Development

Run the backend and frontend dev servers in separate terminals:

```bash
make dev-backend    # Go server (foreground, auto-selects port)
make dev-frontend   # Vite dev server on :5173 (proxies /api to backend)
```

Open `http://localhost:5173` in your browser.

## Tech Stack

- **Backend:** Go, Echo v4, `database/sql` + `go-sql-driver/mysql`, SQLite for local storage (`modernc.org/sqlite`)
- **Frontend:** Vue 3 (Composition API, TypeScript), CodeMirror 6, scoped CSS (no UI framework)
- **Encryption:** AES-256-GCM with argon2id key derivation
- **Build:** Single binary with embedded frontend via `//go:embed`

## License

MIT
