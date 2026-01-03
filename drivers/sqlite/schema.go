package main

import (
	"database/sql"
	"fmt"
	"time"
)

func extractSchema(connStr, database string, verifyData, verifyRowCounts bool) (map[string]interface{}, error) {
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if database == "" {
		database = connStr
	}

	tables, err := getTables(db, database, verifyData, verifyRowCounts)
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

func getTables(db *sql.DB, database string, verifyData, verifyRowCounts bool) ([]map[string]interface{}, error) {
	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type='table'
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
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
			if err != nil {
				return nil, fmt.Errorf("failed to count rows for table %s: %w", tableName, err)
			}
			table["row_count"] = rowCount
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
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"name":        name,
			"data_type":   dataType,
			"column_type": dataType,
			"is_nullable": notNull == 0,
			"key":         "",
		}

		if pk > 0 {
			column["key"] = "PRI"
		}

		if defaultValue.Valid {
			column["default"] = defaultValue.String
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func getIndexes(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []map[string]interface{}
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		if origin == "pk" {
			continue
		}

		indexCols, err := getIndexColumns(db, name)
		if err != nil {
			return nil, err
		}

		index := map[string]interface{}{
			"name":      name,
			"columns":   indexCols,
			"is_unique": unique == 1,
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func getIndexColumns(db *sql.DB, indexName string) ([]map[string]interface{}, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA index_info(%s)", indexName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var seqno, cid int
		var name sql.NullString

		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}

		if name.Valid {
			col := map[string]interface{}{
				"name":     name.String,
				"sequence": seqno,
			}
			columns = append(columns, col)
		}
	}

	return columns, nil
}

func getTableChecksum(db *sql.DB, tableName string) (string, error) {
	var count sql.NullInt64
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		return "", err
	}

	if !count.Valid || count.Int64 == 0 {
		return "0", nil
	}

	var hashSum sql.NullInt64
	hashQuery := fmt.Sprintf(`
		SELECT SUM(
			(julianday(COALESCE(CAST(rowid AS TEXT), '')) * 1000000)
		) FROM %s LIMIT 100
	`, tableName)

	err = db.QueryRow(hashQuery).Scan(&hashSum)
	if err != nil {
		return fmt.Sprintf("%d", count.Int64), nil
	}

	if hashSum.Valid {
		return fmt.Sprintf("%d-%d", count.Int64, int64(hashSum.Int64)), nil
	}

	return fmt.Sprintf("%d", count.Int64), nil
}

func getForeignKeys(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		fk := map[string]interface{}{
			"constraint_name":   fmt.Sprintf("fk_%s_%s_%d", tableName, table, id),
			"column_name":       from,
			"referenced_table":  table,
			"referenced_column": to,
			"update_rule":       onUpdate,
			"delete_rule":       onDelete,
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}
