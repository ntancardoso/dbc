package main

import (
	"database/sql"
	"fmt"
)

func getTables(db *sql.DB, database string, verifyData, verifyRowCounts bool) ([]map[string]interface{}, error) {
	query := `
		SELECT
			table_name,
			engine,
			table_collation,
			table_rows,
			avg_row_length,
			data_length,
			create_time,
			update_time
		FROM information_schema.tables
		WHERE table_schema = ?
			AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []map[string]interface{}

	for rows.Next() {
		var tableName, engine string
		var collation sql.NullString
		var rowCount, avgRowLength, dataLength sql.NullInt64
		var createTime, updateTime sql.NullString

		err := rows.Scan(&tableName, &engine, &collation, &rowCount, &avgRowLength, &dataLength, &createTime, &updateTime)
		if err != nil {
			return nil, err
		}

		table := map[string]interface{}{
			"name":       tableName,
			"engine":     engine,
			"collation":  collation.String,
			"row_count":  rowCount.Int64,
		}

		if verifyRowCounts {
			exactCount, err := getExactRowCount(db, tableName)
			if err == nil {
				table["exact_row_count"] = exactCount
			}
		}

		if verifyData {
			checksum, err := getTableChecksum(db, tableName)
			if err == nil {
				table["checksum"] = checksum
			}
		}

		columns, err := getColumns(db, database, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		table["columns"] = columns

		indexes, err := getIndexes(db, database, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		table["indexes"] = indexes

		foreignKeys, err := getForeignKeys(db, database, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get foreign keys for table %s: %w", tableName, err)
		}
		table["foreign_keys"] = foreignKeys

		constraints, err := getConstraints(db, database, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get constraints for table %s: %w", tableName, err)
		}
		table["constraints"] = constraints

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func getColumns(db *sql.DB, database, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			column_name,
			ordinal_position,
			data_type,
			column_type,
			is_nullable,
			column_default,
			column_key,
			extra
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []map[string]interface{}

	for rows.Next() {
		var columnName, dataType, columnType, isNullable, columnKey, extra string
		var position int
		var defaultValue sql.NullString

		err := rows.Scan(&columnName, &position, &dataType, &columnType, &isNullable, &defaultValue, &columnKey, &extra)
		if err != nil {
			return nil, err
		}

		column := map[string]interface{}{
			"name":        columnName,
			"position":    position,
			"data_type":   dataType,
			"column_type": columnType,
			"is_nullable": isNullable == "YES",
			"key":         columnKey,
			"extra":       extra,
		}

		if defaultValue.Valid {
			column["default_value"] = defaultValue.String
		}

		columns = append(columns, column)
	}

	return columns, rows.Err()
}

func getIndexes(db *sql.DB, database, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			index_name,
			non_unique,
			seq_in_index,
			column_name,
			collation,
			index_type
		FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ?
		ORDER BY index_name, seq_in_index
	`

	rows, err := db.Query(query, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]map[string]interface{})
	indexColumns := make(map[string][]map[string]interface{})

	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique, seqInIndex int
		var collation sql.NullString

		err := rows.Scan(&indexName, &nonUnique, &seqInIndex, &columnName, &collation, &indexType)
		if err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = map[string]interface{}{
				"name":       indexName,
				"is_unique":  nonUnique == 0,
				"is_primary": indexName == "PRIMARY",
				"type":       indexType,
			}
			indexColumns[indexName] = []map[string]interface{}{}
		}

		indexColumns[indexName] = append(indexColumns[indexName], map[string]interface{}{
			"name":      columnName,
			"sequence":  seqInIndex,
			"collation": collation.String,
		})
	}

	var indexes []map[string]interface{}
	for indexName, index := range indexMap {
		index["columns"] = indexColumns[indexName]
		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}

func getForeignKeys(db *sql.DB, database, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			constraint_name,
			column_name,
			referenced_table_name,
			referenced_column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
			AND table_name = ?
			AND referenced_table_name IS NOT NULL
		ORDER BY constraint_name
	`

	rows, err := db.Query(query, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []map[string]interface{}

	for rows.Next() {
		var constraintName, columnName, referencedTable, referencedColumn string

		err := rows.Scan(&constraintName, &columnName, &referencedTable, &referencedColumn)
		if err != nil {
			return nil, err
		}

		foreignKey := map[string]interface{}{
			"name":              constraintName,
			"column":            columnName,
			"referenced_table":  referencedTable,
			"referenced_column": referencedColumn,
		}

		foreignKeys = append(foreignKeys, foreignKey)
	}

	return foreignKeys, rows.Err()
}

func getConstraints(db *sql.DB, database, tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			constraint_name,
			constraint_type
		FROM information_schema.table_constraints
		WHERE constraint_schema = ? AND table_name = ?
		ORDER BY constraint_type, constraint_name
	`

	rows, err := db.Query(query, database, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []map[string]interface{}

	for rows.Next() {
		var constraintName, constraintType string

		err := rows.Scan(&constraintName, &constraintType)
		if err != nil {
			return nil, err
		}

		constraint := map[string]interface{}{
			"name": constraintName,
			"type": constraintType,
		}

		constraints = append(constraints, constraint)
	}

	return constraints, rows.Err()
}

func getExactRowCount(db *sql.DB, tableName string) (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

func getTableChecksum(db *sql.DB, tableName string) (string, error) {
	var checksum sql.NullInt64
	query := fmt.Sprintf("CHECKSUM TABLE `%s`", tableName)

	rows, err := db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var table string
		if err := rows.Scan(&table, &checksum); err != nil {
			return "", err
		}
	}

	if checksum.Valid {
		return fmt.Sprintf("%d", checksum.Int64), nil
	}

	return "", nil
}
