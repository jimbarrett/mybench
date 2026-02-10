package database

import (
	"database/sql"
)

// GetCompletionSchema returns a schema map for editor autocomplete.
// Keys are table names (both "db.table" qualified and bare "table" forms).
// Values are column name slices for that table.
func GetCompletionSchema(db *sql.DB) (map[string][]string, error) {
	// Single query to get all databases, tables, and columns visible to this user.
	query := `
		SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schema := make(map[string][]string)

	for rows.Next() {
		var dbName, tableName, colName string
		if err := rows.Scan(&dbName, &tableName, &colName); err != nil {
			return nil, err
		}

		// Qualified form: "database.table" -> columns
		qualified := dbName + "." + tableName
		schema[qualified] = append(schema[qualified], colName)

		// Bare form: "table" -> columns (for convenience)
		schema[tableName] = append(schema[tableName], colName)
	}

	return schema, rows.Err()
}
