package backup

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SafeCopySQLiteDatabase performs a safe online backup of a SQLite database
// using SQL commands that work even when the database is locked by another process
func SafeCopySQLiteDatabase(sourcePath, targetPath string) error {
	// Try VACUUM INTO method first (SQLite 3.27.0+, 2019)
	// This is the fastest and most atomic method
	err := vacuumIntoBackup(sourcePath, targetPath)
	if err == nil {
		log.Printf("Successfully completed online SQLite backup using VACUUM INTO: %s -> %s", sourcePath, targetPath)
		return nil
	}

	log.Printf("VACUUM INTO not available, falling back to table-by-table copy: %v", err)

	// Fallback: table-by-table copy
	err = copyDatabaseTableByTable(sourcePath, targetPath)
	if err != nil {
		return fmt.Errorf("failed to backup SQLite database: %v", err)
	}

	log.Printf("Successfully completed online SQLite backup using table copy: %s -> %s", sourcePath, targetPath)
	return nil
}

// vacuumIntoBackup uses VACUUM INTO command (atomic, safe, fast)
func vacuumIntoBackup(sourcePath, targetPath string) error {
	// Open source database in read-only mode
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", sourcePath))
	if err != nil {
		return fmt.Errorf("failed to open source database: %v", err)
	}
	defer db.Close()

	// VACUUM INTO creates a complete copy of the database
	// It works even when the database is being used by other processes
	_, err = db.Exec("VACUUM INTO ?", targetPath)
	if err != nil {
		return fmt.Errorf("VACUUM INTO failed: %v", err)
	}

	return nil
}

// copyDatabaseTableByTable copies each table individually (fallback method)
func copyDatabaseTableByTable(sourcePath, targetPath string) error {
	ctx := context.Background()

	// Open source database in read-only mode
	sourceDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", sourcePath))
	if err != nil {
		return fmt.Errorf("failed to open source database: %v", err)
	}
	defer sourceDB.Close()

	// Verify source database connection
	if err := sourceDB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to source database: %v", err)
	}

	// Open or create target database
	targetDB, err := sql.Open("sqlite3", targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target database: %v", err)
	}
	defer targetDB.Close()

	// Get list of all tables, indexes, views, and triggers
	schemas, err := getAllSchemas(ctx, sourceDB)
	if err != nil {
		return fmt.Errorf("failed to get schemas: %v", err)
	}

	// Create all schema objects (tables, indexes, views, triggers)
	for _, schema := range schemas {
		if schema.SQL == "" {
			continue // Skip internal objects
		}
		_, err = targetDB.ExecContext(ctx, schema.SQL)
		if err != nil {
			return fmt.Errorf("failed to create %s %s: %v", schema.Type, schema.Name, err)
		}
	}

	// Copy data for all tables
	for _, schema := range schemas {
		if schema.Type == "table" && !strings.HasPrefix(schema.Name, "sqlite_") {
			if err := copyTableData(ctx, sourceDB, targetDB, schema.Name); err != nil {
				return fmt.Errorf("failed to copy table %s: %v", schema.Name, err)
			}
			log.Printf("  Copied table: %s", schema.Name)
		}
	}

	return nil
}

// schemaObject represents a database schema object
type schemaObject struct {
	Type string
	Name string
	SQL  string
}

// getAllSchemas retrieves all schema objects from the database
func getAllSchemas(ctx context.Context, db *sql.DB) ([]schemaObject, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT type, name, sql FROM sqlite_master
		WHERE sql NOT NULL
		ORDER BY CASE type
			WHEN 'table' THEN 1
			WHEN 'index' THEN 2
			WHEN 'view' THEN 3
			WHEN 'trigger' THEN 4
			ELSE 5
		END, name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	var schemas []schemaObject
	for rows.Next() {
		var obj schemaObject
		if err := rows.Scan(&obj.Type, &obj.Name, &obj.SQL); err != nil {
			return nil, fmt.Errorf("failed to scan schema object: %v", err)
		}
		schemas = append(schemas, obj)
	}

	return schemas, rows.Err()
}

// copyTableData copies all data from a table
func copyTableData(ctx context.Context, srcDB, dstDB *sql.DB, tableName string) error {
	// Use a transaction for better performance
	tx, err := dstDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get all rows from source table
	rows, err := srcDB.QueryContext(ctx, fmt.Sprintf("SELECT * FROM \"%s\"", tableName))
	if err != nil {
		return fmt.Errorf("failed to query source table: %v", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %v", err)
	}

	if len(columns) == 0 {
		// Empty table, just commit
		return tx.Commit()
	}

	// Prepare insert statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertSQL := fmt.Sprintf("INSERT INTO \"%s\" VALUES (%s)",
		tableName,
		strings.Join(placeholders, ", "))

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	// Copy rows
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return fmt.Errorf("failed to insert row: %v", err)
		}
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	if rowCount > 0 {
		log.Printf("    Copied %d rows", rowCount)
	}

	return nil
}
