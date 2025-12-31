package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func extractSchema(connStr, database string, verifyData, verifyRowCounts bool) (map[string]interface{}, error) {
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	var currentUser string
	err = db.QueryRow("SELECT USER FROM DUAL").Scan(&currentUser)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if database == "" {
		database = currentUser
	}

	tables, err := getTables(db, currentUser, verifyData, verifyRowCounts)
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

func getTables(db *sql.DB, owner string, verifyData, verifyRowCounts bool) ([]map[string]interface{}, error) {
	query := `
		SELECT table_name
		FROM all_tables
		WHERE owner = :1
		ORDER BY table_name
	`

	rows, err := db.Query(query, strings.ToUpper(owner))
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

		columns, err := getColumns(db, owner, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		table["columns"] = columns

		indexes, err := getIndexes(db, owner, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		table["indexes"] = indexes

		foreignKeys, err := getForeignKeys(db, owner, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", tableName, err)
		}
		table["foreign_keys"] = foreignKeys

		if verifyRowCounts {
			var rowCount int64
			err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", owner, tableName)).Scan(&rowCount)
			if err == nil {
				table["row_count"] = rowCount
			}
		}

		if verifyData {
			checksum, err := getTableChecksum(db, owner, tableName)
			if err == nil && checksum != "" {
				table["checksum"] = checksum
			}
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func getColumns(db *sql.DB, owner, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			column_name,
			data_type,
			CASE
				WHEN data_type IN ('VARCHAR2', 'CHAR', 'NVARCHAR2', 'NCHAR') THEN data_type || '(' || data_length || ')'
				WHEN data_type IN ('NUMBER') AND data_precision IS NOT NULL THEN
					CASE
						WHEN data_scale > 0 THEN data_type || '(' || data_precision || ',' || data_scale || ')'
						ELSE data_type || '(' || data_precision || ')'
					END
				ELSE data_type
			END as column_type,
			nullable,
			data_default
		FROM all_tab_columns
		WHERE owner = :1
			AND table_name = :2
		ORDER BY column_id
	`

	rows, err := db.Query(query, strings.ToUpper(owner), strings.ToUpper(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var name, dataType, columnType, nullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&name, &dataType, &columnType, &nullable, &defaultValue); err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"name":        name,
			"data_type":   dataType,
			"column_type": columnType,
			"is_nullable": nullable == "Y",
			"key":         "",
		}

		if defaultValue.Valid {
			column["default_value"] = strings.TrimSpace(defaultValue.String)
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func getIndexes(db *sql.DB, owner, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			i.index_name,
			i.uniqueness,
			ic.column_name,
			ic.column_position
		FROM all_indexes i
		JOIN all_ind_columns ic ON i.owner = ic.index_owner AND i.index_name = ic.index_name
		WHERE i.owner = :1
			AND i.table_name = :2
			AND i.index_name NOT IN (
				SELECT constraint_name
				FROM all_constraints
				WHERE owner = :1
					AND table_name = :2
					AND constraint_type = 'P'
			)
		ORDER BY i.index_name, ic.column_position
	`

	rows, err := db.Query(query, strings.ToUpper(owner), strings.ToUpper(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*struct {
		name     string
		isUnique bool
		columns  []map[string]interface{}
	})

	for rows.Next() {
		var indexName, uniqueness, columnName string
		var columnPosition int

		if err := rows.Scan(&indexName, &uniqueness, &columnName, &columnPosition); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &struct {
				name     string
				isUnique bool
				columns  []map[string]interface{}
			}{
				name:     indexName,
				isUnique: uniqueness == "UNIQUE",
				columns:  []map[string]interface{}{},
			}
		}

		indexMap[indexName].columns = append(indexMap[indexName].columns, map[string]interface{}{
			"name":     columnName,
			"sequence": columnPosition - 1,
		})
	}

	var indexes []map[string]interface{}
	for _, idx := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":       idx.name,
			"is_unique":  idx.isUnique,
			"is_primary": false,
			"columns":    idx.columns,
		})
	}

	return indexes, nil
}

func getForeignKeys(db *sql.DB, owner, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			c.constraint_name,
			cc.column_name,
			rc.table_name as referenced_table,
			rcc.column_name as referenced_column
		FROM all_constraints c
		JOIN all_cons_columns cc ON c.owner = cc.owner AND c.constraint_name = cc.constraint_name
		JOIN all_constraints rc ON c.r_owner = rc.owner AND c.r_constraint_name = rc.constraint_name
		JOIN all_cons_columns rcc ON rc.owner = rcc.owner AND rc.constraint_name = rcc.constraint_name
			AND cc.position = rcc.position
		WHERE c.owner = :1
			AND c.table_name = :2
			AND c.constraint_type = 'R'
		ORDER BY c.constraint_name, cc.position
	`

	rows, err := db.Query(query, strings.ToUpper(owner), strings.ToUpper(tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}
	for rows.Next() {
		var constraintName, columnName, referencedTable, referencedColumn string

		if err := rows.Scan(&constraintName, &columnName, &referencedTable, &referencedColumn); err != nil {
			return nil, err
		}

		fk := map[string]interface{}{
			"name":              constraintName,
			"column":            columnName,
			"referenced_table":  referencedTable,
			"referenced_column": referencedColumn,
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

func getTableChecksum(db *sql.DB, owner, tableName string) (string, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as row_count,
			COALESCE(SUM(ORA_HASH(ROWID)), 0) as checksum_value
		FROM %s.%s
	`, owner, tableName)

	var count, checksumValue sql.NullInt64
	err := db.QueryRow(query).Scan(&count, &checksumValue)
	if err != nil {
		return "", err
	}

	if !count.Valid {
		return "0", nil
	}

	if checksumValue.Valid {
		return fmt.Sprintf("%d-%d", count.Int64, checksumValue.Int64), nil
	}

	return fmt.Sprintf("%d", count.Int64), nil
}
