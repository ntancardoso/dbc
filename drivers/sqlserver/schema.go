package main

import (
	"database/sql"
	"fmt"
	"time"
)

func extractSchema(connStr, database string, verifyData, verifyRowCounts bool) (map[string]interface{}, error) {
	db, err := sql.Open("sqlserver", connStr)
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
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
			AND TABLE_SCHEMA = 'dbo'
		ORDER BY TABLE_NAME
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
			err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM [%s]", tableName)).Scan(&rowCount)
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
			c.COLUMN_NAME,
			c.DATA_TYPE,
			CASE
				WHEN c.DATA_TYPE IN ('varchar', 'nvarchar', 'char', 'nchar') THEN c.DATA_TYPE + '(' + CAST(c.CHARACTER_MAXIMUM_LENGTH AS VARCHAR) + ')'
				WHEN c.DATA_TYPE IN ('decimal', 'numeric') THEN c.DATA_TYPE + '(' + CAST(c.NUMERIC_PRECISION AS VARCHAR) + ',' + CAST(c.NUMERIC_SCALE AS VARCHAR) + ')'
				ELSE c.DATA_TYPE
			END as column_type,
			c.IS_NULLABLE,
			c.COLUMN_DEFAULT
		FROM INFORMATION_SCHEMA.COLUMNS c
		WHERE c.TABLE_NAME = @p1
			AND c.TABLE_SCHEMA = 'dbo'
		ORDER BY c.ORDINAL_POSITION
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var name, dataType, columnType, isNullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&name, &dataType, &columnType, &isNullable, &defaultValue); err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"name":        name,
			"data_type":   dataType,
			"column_type": columnType,
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
			i.name as index_name,
			i.is_unique,
			i.is_primary_key,
			COL_NAME(ic.object_id, ic.column_id) as column_name,
			ic.key_ordinal
		FROM sys.indexes i
		INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		WHERE i.object_id = OBJECT_ID(@p1)
			AND i.type > 0
			AND i.is_primary_key = 0
		ORDER BY i.name, ic.key_ordinal
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*struct {
		name      string
		isUnique  bool
		isPrimary bool
		columns   []map[string]interface{}
	})

	for rows.Next() {
		var indexName, columnName string
		var isUnique, isPrimary bool
		var keyOrdinal int

		if err := rows.Scan(&indexName, &isUnique, &isPrimary, &columnName, &keyOrdinal); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &struct {
				name      string
				isUnique  bool
				isPrimary bool
				columns   []map[string]interface{}
			}{
				name:      indexName,
				isUnique:  isUnique,
				isPrimary: isPrimary,
				columns:   []map[string]interface{}{},
			}
		}

		indexMap[indexName].columns = append(indexMap[indexName].columns, map[string]interface{}{
			"name":     columnName,
			"sequence": keyOrdinal - 1,
		})
	}

	var indexes []map[string]interface{}
	for _, idx := range indexMap {
		indexes = append(indexes, map[string]interface{}{
			"name":       idx.name,
			"is_unique":  idx.isUnique,
			"is_primary": idx.isPrimary,
			"columns":    idx.columns,
		})
	}

	return indexes, nil
}

func getForeignKeys(db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			fk.name as constraint_name,
			COL_NAME(fkc.parent_object_id, fkc.parent_column_id) as column_name,
			OBJECT_NAME(fkc.referenced_object_id) as referenced_table,
			COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) as referenced_column
		FROM sys.foreign_keys fk
		INNER JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		WHERE fk.parent_object_id = OBJECT_ID(@p1)
		ORDER BY fk.name, fkc.constraint_column_id
	`

	rows, err := db.Query(query, tableName)
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

func getTableChecksum(db *sql.DB, tableName string) (string, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as row_count,
			COALESCE(SUM(CAST(CHECKSUM(*) AS BIGINT)), 0) as checksum_value
		FROM [%s]
	`, tableName)

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
