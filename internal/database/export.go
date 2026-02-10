package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ExportResultCSV writes query result data (columns + rows) to a CSV writer.
func ExportResultCSV(w io.Writer, columns []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write(columns); err != nil {
		return err
	}
	for _, row := range rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

// ExportResultSQL writes query result data as SQL INSERT statements.
// tableName is used in the INSERT INTO clause.
func ExportResultSQL(w io.Writer, tableName string, columns []string, rows [][]string) error {
	for _, row := range rows {
		vals := make([]string, len(row))
		for i, v := range row {
			if v == "NULL" {
				vals[i] = "NULL"
			} else {
				vals[i] = "'" + strings.ReplaceAll(v, "'", "\\'") + "'"
			}
		}
		line := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES (%s);\n",
			tableName,
			strings.Join(columns, "`, `"),
			strings.Join(vals, ", "),
		)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

// ProgressFunc is called with (current, total) to report progress.
// Return false to cancel the operation.
type ProgressFunc func(current, total int64) bool

// ExportTableCSV streams an entire table to CSV.
func ExportTableCSV(ctx context.Context, db *sql.DB, dbName, tableName string, w io.Writer, progress ProgressFunc) error {
	// Get row count for progress reporting.
	var totalRows int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", dbName, tableName)
	if err := db.QueryRowContext(ctx, countQuery).Scan(&totalRows); err != nil {
		totalRows = -1 // unknown, continue anyway
	}

	query := fmt.Sprintf("SELECT * FROM `%s`.`%s`", dbName, tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write(cols); err != nil {
		return err
	}

	scanVals := make([]interface{}, len(cols))
	scanPtrs := make([]interface{}, len(cols))
	for i := range scanVals {
		scanPtrs[i] = &scanVals[i]
	}

	var written int64
	for rows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return err
		}

		record := make([]string, len(cols))
		for i, v := range scanVals {
			if v == nil {
				record[i] = "NULL"
			} else {
				record[i] = fmt.Sprintf("%s", v)
			}
		}
		if err := cw.Write(record); err != nil {
			return err
		}

		written++
		if progress != nil && written%500 == 0 {
			if !progress(written, totalRows) {
				return fmt.Errorf("cancelled")
			}
		}
	}

	// Final progress update.
	if progress != nil {
		progress(written, totalRows)
	}

	return rows.Err()
}

// ExportTableSQL streams an entire table as SQL INSERT statements.
func ExportTableSQL(ctx context.Context, db *sql.DB, dbName, tableName string, w io.Writer, progress ProgressFunc) error {
	var totalRows int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", dbName, tableName)
	if err := db.QueryRowContext(ctx, countQuery).Scan(&totalRows); err != nil {
		totalRows = -1
	}

	query := fmt.Sprintf("SELECT * FROM `%s`.`%s`", dbName, tableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	scanVals := make([]interface{}, len(cols))
	scanPtrs := make([]interface{}, len(cols))
	for i := range scanVals {
		scanPtrs[i] = &scanVals[i]
	}

	var written int64
	for rows.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return err
		}

		vals := make([]string, len(cols))
		for i, v := range scanVals {
			if v == nil {
				vals[i] = "NULL"
			} else {
				s := fmt.Sprintf("%s", v)
				vals[i] = "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
			}
		}

		line := fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES (%s);\n",
			tableName,
			strings.Join(cols, "`, `"),
			strings.Join(vals, ", "),
		)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}

		written++
		if progress != nil && written%500 == 0 {
			if !progress(written, totalRows) {
				return fmt.Errorf("cancelled")
			}
		}
	}

	if progress != nil {
		progress(written, totalRows)
	}

	return rows.Err()
}
