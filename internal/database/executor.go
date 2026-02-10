package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// QueryResult holds the result of a single query execution.
type QueryResult struct {
	Columns      []string   `json:"columns"`
	Rows         [][]string `json:"rows"`
	RowCount     int        `json:"rowCount"`
	AffectedRows int64      `json:"affectedRows"`
	Duration     string     `json:"duration"`
	IsSelect     bool       `json:"isSelect"`
	Error        string     `json:"error"`
}

// ExecuteQuery runs a SQL query on the given connection and returns results.
func ExecuteQuery(ctx context.Context, db *sql.DB, query string) *QueryResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return &QueryResult{Error: "empty query"}
	}

	start := time.Now()
	isSelect := isSelectQuery(query)

	if isSelect {
		return executeSelect(ctx, db, query, start)
	}
	return executeExec(ctx, db, query, start)
}

// ExecuteMulti splits SQL by semicolons and executes each statement.
// Returns results for each statement.
func ExecuteMulti(ctx context.Context, db *sql.DB, queries string) []QueryResult {
	stmts := splitStatements(queries)
	results := make([]QueryResult, 0, len(stmts))

	for _, stmt := range stmts {
		if ctx.Err() != nil {
			results = append(results, QueryResult{Error: "cancelled"})
			break
		}
		result := ExecuteQuery(ctx, db, stmt)
		results = append(results, *result)
		if result.Error != "" {
			break
		}
	}

	return results
}

// ExplainQuery runs EXPLAIN on the given query.
func ExplainQuery(ctx context.Context, db *sql.DB, query string) *QueryResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return &QueryResult{Error: "empty query"}
	}
	explainSQL := "EXPLAIN " + query
	start := time.Now()
	return executeSelect(ctx, db, explainSQL, start)
}

// GetConnectionID returns the MySQL connection ID for KILL QUERY support.
func GetConnectionID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT CONNECTION_ID()").Scan(&id)
	return id, err
}

// KillQuery kills a running query by connection ID.
func KillQuery(db *sql.DB, connID int64) error {
	_, err := db.Exec(fmt.Sprintf("KILL QUERY %d", connID))
	return err
}

func executeSelect(ctx context.Context, db *sql.DB, query string, start time.Time) *QueryResult {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return &QueryResult{
			Error:    err.Error(),
			Duration: time.Since(start).String(),
			IsSelect: true,
		}
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return &QueryResult{
			Error:    err.Error(),
			Duration: time.Since(start).String(),
			IsSelect: true,
		}
	}

	// Detect binary columns via column types.
	colTypes, _ := rows.ColumnTypes()
	isBinary := make([]bool, len(cols))
	for i, ct := range colTypes {
		if ct != nil {
			typeName := strings.ToUpper(ct.DatabaseTypeName())
			isBinary[i] = strings.Contains(typeName, "BLOB") ||
				strings.Contains(typeName, "BINARY") ||
				typeName == "GEOMETRY"
		}
	}

	var resultRows [][]string
	scanArgs := make([]interface{}, len(cols))
	for i := range scanArgs {
		if isBinary[i] {
			scanArgs[i] = &sql.RawBytes{}
		} else {
			scanArgs[i] = &sql.NullString{}
		}
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return &QueryResult{
				Columns:  cols,
				Rows:     resultRows,
				RowCount: len(resultRows),
				Error:    err.Error(),
				Duration: time.Since(start).String(),
				IsSelect: true,
			}
		}

		row := make([]string, len(cols))
		for i := range cols {
			if isBinary[i] {
				raw := scanArgs[i].(*sql.RawBytes)
				if *raw == nil {
					row[i] = "NULL"
				} else if len(*raw) == 0 {
					row[i] = "(empty)"
				} else {
					row[i] = fmt.Sprintf("(binary %d bytes)", len(*raw))
				}
			} else {
				ns := scanArgs[i].(*sql.NullString)
				if ns.Valid {
					row[i] = ns.String
				} else {
					row[i] = "NULL"
				}
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{
			Columns:  cols,
			Rows:     resultRows,
			RowCount: len(resultRows),
			Error:    err.Error(),
			Duration: time.Since(start).String(),
			IsSelect: true,
		}
	}

	return &QueryResult{
		Columns:  cols,
		Rows:     resultRows,
		RowCount: len(resultRows),
		Duration: time.Since(start).String(),
		IsSelect: true,
	}
}

func executeExec(ctx context.Context, db *sql.DB, query string, start time.Time) *QueryResult {
	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return &QueryResult{
			Error:    err.Error(),
			Duration: time.Since(start).String(),
		}
	}

	affected, _ := result.RowsAffected()

	return &QueryResult{
		AffectedRows: affected,
		Duration:     time.Since(start).String(),
	}
}

func isSelectQuery(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "DESCRIBE") ||
		strings.HasPrefix(upper, "DESC") ||
		strings.HasPrefix(upper, "EXPLAIN")
}

func splitStatements(sql string) []string {
	var stmts []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := 0; i < len(sql); i++ {
		c := sql[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			current.WriteByte(c)
			escaped = true
			continue
		}

		if (c == '\'' || c == '"') && !inString {
			inString = true
			stringChar = c
			current.WriteByte(c)
			continue
		}

		if inString && c == stringChar {
			inString = false
			current.WriteByte(c)
			continue
		}

		if c == ';' && !inString {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	// Last statement without trailing semicolon.
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		stmts = append(stmts, stmt)
	}

	return stmts
}
