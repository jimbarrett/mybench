package database

import (
	"database/sql"
	"fmt"
)

// DatabaseInfo holds basic database metadata.
type DatabaseInfo struct {
	Name string `json:"name"`
}

// TableInfo holds basic table/view metadata.
type TableInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "BASE TABLE" or "VIEW"
	Engine    string `json:"engine"`
	RowCount  int64  `json:"rowCount"`
	DataSize  int64  `json:"dataSize"`
	Collation string `json:"collation"`
}

// ColumnInfo holds column metadata.
type ColumnInfo struct {
	Name         string  `json:"name"`
	Position     int     `json:"position"`
	Default      *string `json:"default"`
	Nullable     bool    `json:"nullable"`
	DataType     string  `json:"dataType"`
	ColumnType   string  `json:"columnType"`
	MaxLength    *int64  `json:"maxLength"`
	CharSet      *string `json:"charSet"`
	Collation    *string `json:"collation"`
	Key          string  `json:"key"` // PRI, UNI, MUL, or ""
	Extra        string  `json:"extra"`
	Comment      string  `json:"comment"`
}

// IndexInfo holds index metadata.
type IndexInfo struct {
	Name       string `json:"name"`
	Columns    string `json:"columns"`
	Unique     bool   `json:"unique"`
	Type       string `json:"type"` // BTREE, FULLTEXT, HASH, etc.
	Comment    string `json:"comment"`
}

// ForeignKeyInfo holds foreign key metadata.
type ForeignKeyInfo struct {
	Name             string `json:"name"`
	Column           string `json:"column"`
	RefTable         string `json:"refTable"`
	RefColumn        string `json:"refColumn"`
	UpdateRule       string `json:"updateRule"`
	DeleteRule       string `json:"deleteRule"`
}

// RoutineInfo holds stored procedure/function metadata.
type RoutineInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "PROCEDURE" or "FUNCTION"
	Created string `json:"created"`
}

// TriggerInfo holds trigger metadata.
type TriggerInfo struct {
	Name      string `json:"name"`
	Event     string `json:"event"` // INSERT, UPDATE, DELETE
	Timing    string `json:"timing"` // BEFORE, AFTER
	Table     string `json:"table"`
	Statement string `json:"statement"`
}

// TableDetail is the full detail view for a single table.
type TableDetail struct {
	Columns     []ColumnInfo     `json:"columns"`
	Indexes     []IndexInfo      `json:"indexes"`
	ForeignKeys []ForeignKeyInfo `json:"foreignKeys"`
	CreateSQL   string           `json:"createSql"`
}

// ListDatabases returns all databases visible to the connection.
func ListDatabases(db *sql.DB) ([]DatabaseInfo, error) {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []DatabaseInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, DatabaseInfo{Name: name})
	}
	return dbs, rows.Err()
}

// ListTables returns tables and views in a database.
func ListTables(db *sql.DB, database string) ([]TableInfo, error) {
	query := `
		SELECT TABLE_NAME, TABLE_TYPE, IFNULL(ENGINE, ''),
		       IFNULL(TABLE_ROWS, 0), IFNULL(DATA_LENGTH, 0),
		       IFNULL(TABLE_COLLATION, '')
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_TYPE, TABLE_NAME
	`
	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Type, &t.Engine, &t.RowCount, &t.DataSize, &t.Collation); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// GetTableDetail returns full details for a table: columns, indexes, FKs, DDL.
func GetTableDetail(db *sql.DB, database, table string) (*TableDetail, error) {
	detail := &TableDetail{}

	// Columns
	cols, err := listColumns(db, database, table)
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}
	detail.Columns = cols

	// Indexes
	indexes, err := listIndexes(db, database, table)
	if err != nil {
		return nil, fmt.Errorf("indexes: %w", err)
	}
	detail.Indexes = indexes

	// Foreign Keys
	fks, err := listForeignKeys(db, database, table)
	if err != nil {
		return nil, fmt.Errorf("foreign keys: %w", err)
	}
	detail.ForeignKeys = fks

	// DDL
	ddl, err := getCreateTable(db, database, table)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	detail.CreateSQL = ddl

	return detail, nil
}

// ListRoutines returns stored procedures and functions in a database.
func ListRoutines(db *sql.DB, database string) ([]RoutineInfo, error) {
	query := `
		SELECT ROUTINE_NAME, ROUTINE_TYPE, CREATED
		FROM INFORMATION_SCHEMA.ROUTINES
		WHERE ROUTINE_SCHEMA = ?
		ORDER BY ROUTINE_TYPE, ROUTINE_NAME
	`
	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routines []RoutineInfo
	for rows.Next() {
		var r RoutineInfo
		if err := rows.Scan(&r.Name, &r.Type, &r.Created); err != nil {
			return nil, err
		}
		routines = append(routines, r)
	}
	return routines, rows.Err()
}

// ListTriggers returns triggers in a database.
func ListTriggers(db *sql.DB, database string) ([]TriggerInfo, error) {
	query := `
		SELECT TRIGGER_NAME, EVENT_MANIPULATION, ACTION_TIMING,
		       EVENT_OBJECT_TABLE, ACTION_STATEMENT
		FROM INFORMATION_SCHEMA.TRIGGERS
		WHERE TRIGGER_SCHEMA = ?
		ORDER BY EVENT_OBJECT_TABLE, TRIGGER_NAME
	`
	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []TriggerInfo
	for rows.Next() {
		var t TriggerInfo
		if err := rows.Scan(&t.Name, &t.Event, &t.Timing, &t.Table, &t.Statement); err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}
	return triggers, rows.Err()
}

func listColumns(db *sql.DB, database, table string) ([]ColumnInfo, error) {
	query := `
		SELECT COLUMN_NAME, ORDINAL_POSITION, COLUMN_DEFAULT, IS_NULLABLE,
		       DATA_TYPE, COLUMN_TYPE, CHARACTER_MAXIMUM_LENGTH,
		       CHARACTER_SET_NAME, COLLATION_NAME, COLUMN_KEY, EXTRA, COLUMN_COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		if err := rows.Scan(
			&c.Name, &c.Position, &c.Default, &nullable,
			&c.DataType, &c.ColumnType, &c.MaxLength,
			&c.CharSet, &c.Collation, &c.Key, &c.Extra, &c.Comment,
		); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func listIndexes(db *sql.DB, database, table string) ([]IndexInfo, error) {
	// Group columns per index since STATISTICS has one row per column.
	query := `
		SELECT INDEX_NAME, GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX),
		       CASE WHEN NON_UNIQUE = 0 THEN 1 ELSE 0 END,
		       INDEX_TYPE, IFNULL(INDEX_COMMENT, '')
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE, INDEX_TYPE, INDEX_COMMENT
		ORDER BY INDEX_NAME
	`
	rows, err := db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var unique int
		if err := rows.Scan(&idx.Name, &idx.Columns, &unique, &idx.Type, &idx.Comment); err != nil {
			return nil, err
		}
		idx.Unique = unique == 1
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func listForeignKeys(db *sql.DB, database, table string) ([]ForeignKeyInfo, error) {
	query := `
		SELECT kcu.CONSTRAINT_NAME, kcu.COLUMN_NAME,
		       kcu.REFERENCED_TABLE_NAME, kcu.REFERENCED_COLUMN_NAME,
		       rc.UPDATE_RULE, rc.DELETE_RULE
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
		JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
		  ON rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
		  AND rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
		WHERE kcu.TABLE_SCHEMA = ? AND kcu.TABLE_NAME = ?
		  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
	`
	rows, err := db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKeyInfo
	for rows.Next() {
		var fk ForeignKeyInfo
		if err := rows.Scan(&fk.Name, &fk.Column, &fk.RefTable, &fk.RefColumn, &fk.UpdateRule, &fk.DeleteRule); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func getCreateTable(db *sql.DB, database, table string) (string, error) {
	var tbl, ddl string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table)
	err := db.QueryRow(query).Scan(&tbl, &ddl)
	if err != nil {
		return "", err
	}
	return ddl, nil
}
