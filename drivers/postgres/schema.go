package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

func extractSchema(connStr, database string, verifyData, verifyRowCounts bool) (map[string]interface{}, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	tables, err := getTables(db, verifyData, verifyRowCounts)
	if err != nil {
		return nil, err
	}

	snapshot := map[string]interface{}{
		"database":  database,
		"timestamp": time.Now().Format(time.RFC3339),
		"tables":    tables,
		"metadata": map[string]interface{}{
			"driver":           driverName,
			"driver_version":   driverVersion,
			"verify_data":      verifyData,
			"verify_row_count": verifyRowCounts,
		},
	}

	return snapshot, nil
}

func getTables(db *sql.DB, verifyData, verifyRowCounts bool) ([]map[string]interface{}, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []map[string]interface{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		table := map[string]interface{}{
			"name": tableName,
		}

		columns, err := getColumns(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		table["columns"] = columns

		indexes, err := getIndexes(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		table["indexes"] = indexes

		foreignKeys, err := getForeignKeys(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", tableName, err)
		}
		table["foreign_keys"] = foreignKeys

		if verifyRowCounts {
			var rowCount int64
			err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount)
			if err == nil {
				table["row_count"] = rowCount
			}
		}

		if verifyData {
			checksum, err := getTableChecksum(db, tableName)
			if err == nil && checksum != "" {
				table["checksum"] = checksum
			}
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func getColumns(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			column_name,
			data_type,
			udt_name,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
			AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var name, dataType, udtName, isNullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&name, &dataType, &udtName, &isNullable, &defaultValue); err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"name":        name,
			"data_type":   dataType,
			"column_type": udtName,
			"is_nullable": isNullable == "YES",
			"key":         "",
		}

		if defaultValue.Valid {
			column["default_value"] = defaultValue.String
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func getIndexes(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
			AND t.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
			AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique
		ORDER BY i.relname
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var indexName string
		var isUnique bool
		var columns []string

		if err := rows.Scan(&indexName, &isUnique, pq.Array(&columns)); err != nil {
			return nil, err
		}

		indexCols := make([]map[string]interface{}, len(columns))
		for i, col := range columns {
			indexCols[i] = map[string]interface{}{
				"name":     col,
				"sequence": i,
			}
		}

		index := map[string]interface{}{
			"name":       indexName,
			"is_unique":  isUnique,
			"is_primary": false,
			"columns":    indexCols,
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func getForeignKeys(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_name = $1
			AND tc.table_schema = 'public'
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}
	for rows.Next() {
		var constraintName, columnName, foreignTable, foreignColumn string

		if err := rows.Scan(&constraintName, &columnName, &foreignTable, &foreignColumn); err != nil {
			return nil, err
		}

		fk := map[string]interface{}{
			"name":              constraintName,
			"column":            columnName,
			"referenced_table":  foreignTable,
			"referenced_column": foreignColumn,
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

func getTableChecksum(db *sql.DB, tableName string) (string, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as row_count,
			COALESCE(SUM(pg_column_size(t.*)), 0) as total_size
		FROM %s t
	`, tableName)

	var count, totalSize sql.NullInt64
	err := db.QueryRow(query).Scan(&count, &totalSize)
	if err != nil {
		return "", err
	}

	if !count.Valid {
		return "0", nil
	}

	if totalSize.Valid {
		return fmt.Sprintf("%d-%d", count.Int64, totalSize.Int64), nil
	}

	return fmt.Sprintf("%d", count.Int64), nil
}
