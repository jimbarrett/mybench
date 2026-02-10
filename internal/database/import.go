package database

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// CSVPreview holds the header and first few rows of a CSV file for column mapping.
type CSVPreview struct {
	Headers    []string   `json:"headers"`
	SampleRows [][]string `json:"sampleRows"`
	TotalRows  int        `json:"totalRows"`
}

// PreviewCSV reads a CSV file and returns its headers and first N sample rows.
func PreviewCSV(filePath string, sampleSize int) (*CSVPreview, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true

	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var samples [][]string
	total := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		total++
		if len(samples) < sampleSize {
			samples = append(samples, record)
		}
	}

	return &CSVPreview{
		Headers:    headers,
		SampleRows: samples,
		TotalRows:  total,
	}, nil
}

// ColumnMapping maps a CSV column index to a database column name.
type ColumnMapping struct {
	CSVIndex   int    `json:"csvIndex"`
	ColumnName string `json:"columnName"`
}

// ImportCSV imports a CSV file into a table using the given column mappings.
func ImportCSV(ctx context.Context, db *sql.DB, dbName, tableName, filePath string, mappings []ColumnMapping, progress ProgressFunc) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true

	// Skip header row.
	if _, err := r.Read(); err != nil {
		return 0, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Build INSERT template.
	colNames := make([]string, len(mappings))
	placeholders := make([]string, len(mappings))
	for i, m := range mappings {
		colNames[i] = "`" + m.ColumnName + "`"
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		dbName, tableName,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	var imported int64
	for {
		if ctx.Err() != nil {
			return imported, ctx.Err()
		}

		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return imported, fmt.Errorf("CSV read error at row %d: %w", imported+1, err)
		}

		vals := make([]interface{}, len(mappings))
		for i, m := range mappings {
			if m.CSVIndex < len(record) {
				v := record[m.CSVIndex]
				if v == "" || strings.EqualFold(v, "NULL") {
					vals[i] = nil
				} else {
					vals[i] = v
				}
			} else {
				vals[i] = nil
			}
		}

		if _, err := db.ExecContext(ctx, insertSQL, vals...); err != nil {
			return imported, fmt.Errorf("insert error at row %d: %w", imported+1, err)
		}
		imported++

		if progress != nil && imported%500 == 0 {
			if !progress(imported, -1) {
				return imported, fmt.Errorf("cancelled")
			}
		}
	}

	if progress != nil {
		progress(imported, imported)
	}

	return imported, nil
}

// ImportSQLFile executes a SQL file against the database.
// It splits on semicolons and executes each statement.
func ImportSQLFile(ctx context.Context, db *sql.DB, filePath string, progress ProgressFunc) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Read statements separated by semicolons, handling quoted strings.
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0), 10*1024*1024) // 10MB max line

	var buf strings.Builder
	var executed int64
	inQuote := false
	quoteChar := byte(0)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return executed, ctx.Err()
		}

		line := scanner.Text()

		// Skip comment-only lines (outside of quotes).
		trimmed := strings.TrimSpace(line)
		if !inQuote && (strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#")) {
			continue
		}

		for i := 0; i < len(line); i++ {
			ch := line[i]

			if inQuote {
				buf.WriteByte(ch)
				if ch == '\\' && i+1 < len(line) {
					i++
					buf.WriteByte(line[i])
					continue
				}
				if ch == quoteChar {
					inQuote = false
				}
				continue
			}

			if ch == '\'' || ch == '"' || ch == '`' {
				inQuote = true
				quoteChar = ch
				buf.WriteByte(ch)
				continue
			}

			if ch == ';' {
				stmt := strings.TrimSpace(buf.String())
				buf.Reset()
				if stmt == "" {
					continue
				}
				if _, err := db.ExecContext(ctx, stmt); err != nil {
					return executed, fmt.Errorf("error at statement %d: %w", executed+1, err)
				}
				executed++
				if progress != nil && executed%100 == 0 {
					if !progress(executed, -1) {
						return executed, fmt.Errorf("cancelled")
					}
				}
				continue
			}

			buf.WriteByte(ch)
		}
		buf.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return executed, err
	}

	// Execute any remaining statement without trailing semicolon.
	remaining := strings.TrimSpace(buf.String())
	if remaining != "" {
		if _, err := db.ExecContext(ctx, remaining); err != nil {
			return executed, fmt.Errorf("error at statement %d: %w", executed+1, err)
		}
		executed++
	}

	if progress != nil {
		progress(executed, executed)
	}

	return executed, nil
}
