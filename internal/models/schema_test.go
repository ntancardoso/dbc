package models

import (
	"testing"
	"time"
)

func TestSchemaSnapshot(t *testing.T) {
	snapshot := SchemaSnapshot{
		Key:       "test_snapshot",
		Timestamp: time.Now(),
		Database:  "testdb",
		Host:      "localhost",
		DBType:    "mysql",
		Tables:    []Table{},
		Metadata: Metadata{
			Version:         "1.0.0",
			VerifyData:      false,
			VerifyRowCounts: true,
			Workers:         10,
			Duration:        "5s",
		},
	}

	if snapshot.Key != "test_snapshot" {
		t.Errorf("Expected key 'test_snapshot', got '%s'", snapshot.Key)
	}

	if snapshot.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", snapshot.Database)
	}

	if snapshot.DBType != "mysql" {
		t.Errorf("Expected dbtype 'mysql', got '%s'", snapshot.DBType)
	}
}

func TestTable(t *testing.T) {
	table := Table{
		Name:     "users",
		Engine:   "InnoDB",
		RowCount: 1000,
		Columns: []Column{
			{
				Name:       "id",
				Position:   1,
				DataType:   "int",
				ColumnType: "int(11)",
				IsNullable: false,
				Key:        "PRI",
			},
			{
				Name:       "email",
				Position:   2,
				DataType:   "varchar",
				ColumnType: "varchar(255)",
				IsNullable: false,
				Key:        "UNI",
			},
		},
		Indexes: []Index{
			{
				Name:      "PRIMARY",
				IsUnique:  true,
				IsPrimary: true,
				Type:      "BTREE",
				Columns: []IndexColumn{
					{Name: "id", Sequence: 1},
				},
			},
		},
	}

	if table.Name != "users" {
		t.Errorf("Expected table name 'users', got '%s'", table.Name)
	}

	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.Columns))
	}

	if table.Columns[0].Name != "id" {
		t.Errorf("Expected first column 'id', got '%s'", table.Columns[0].Name)
	}

	if table.Columns[0].Key != "PRI" {
		t.Errorf("Expected primary key column, got '%s'", table.Columns[0].Key)
	}

	if len(table.Indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(table.Indexes))
	}

	if !table.Indexes[0].IsPrimary {
		t.Errorf("Expected primary index")
	}
}

func TestColumn(t *testing.T) {
	defaultValue := "CURRENT_TIMESTAMP"
	col := Column{
		Name:         "created_at",
		Position:     3,
		DataType:     "timestamp",
		ColumnType:   "timestamp",
		IsNullable:   false,
		DefaultValue: &defaultValue,
		Extra:        "on update CURRENT_TIMESTAMP",
	}

	if col.Name != "created_at" {
		t.Errorf("Expected column name 'created_at', got '%s'", col.Name)
	}

	if col.DefaultValue == nil {
		t.Error("Expected default value to be set")
	}

	if *col.DefaultValue != "CURRENT_TIMESTAMP" {
		t.Errorf("Expected default value 'CURRENT_TIMESTAMP', got '%s'", *col.DefaultValue)
	}
}

func TestForeignKey(t *testing.T) {
	fk := ForeignKey{
		Name:             "fk_user_id",
		Column:           "user_id",
		ReferencedTable:  "users",
		ReferencedColumn: "id",
		OnDelete:         "CASCADE",
		OnUpdate:         "RESTRICT",
	}

	if fk.Name != "fk_user_id" {
		t.Errorf("Expected FK name 'fk_user_id', got '%s'", fk.Name)
	}

	if fk.ReferencedTable != "users" {
		t.Errorf("Expected referenced table 'users', got '%s'", fk.ReferencedTable)
	}

	if fk.OnDelete != "CASCADE" {
		t.Errorf("Expected OnDelete 'CASCADE', got '%s'", fk.OnDelete)
	}
}

func TestChangeSet(t *testing.T) {
	changeSet := ChangeSet{
		Snapshot1Key: "baseline",
		Snapshot2Key: "v1.2.3",
		TablesAdded: []Table{
			{Name: "new_table"},
		},
		TablesRemoved: []Table{
			{Name: "old_table"},
		},
		Summary: ChangeSummary{
			TablesAdded:    1,
			TablesRemoved:  1,
			TablesModified: 0,
			HasChanges:     true,
		},
	}

	if !changeSet.Summary.HasChanges {
		t.Error("Expected HasChanges to be true")
	}

	if changeSet.Summary.TablesAdded != 1 {
		t.Errorf("Expected 1 table added, got %d", changeSet.Summary.TablesAdded)
	}

	if changeSet.Summary.TablesRemoved != 1 {
		t.Errorf("Expected 1 table removed, got %d", changeSet.Summary.TablesRemoved)
	}

	if len(changeSet.TablesAdded) != 1 {
		t.Errorf("Expected 1 table in TablesAdded, got %d", len(changeSet.TablesAdded))
	}

	if changeSet.TablesAdded[0].Name != "new_table" {
		t.Errorf("Expected added table 'new_table', got '%s'", changeSet.TablesAdded[0].Name)
	}
}

func TestTableDiff(t *testing.T) {
	tableDiff := TableDiff{
		Name: "users",
		ColumnsAdded: []Column{
			{Name: "new_column", DataType: "varchar"},
		},
		ColumnsRemoved: []Column{
			{Name: "old_column", DataType: "int"},
		},
		ColumnsModified: []ColumnDiff{
			{
				Name: "email",
				Before: Column{
					Name:       "email",
					ColumnType: "varchar(100)",
				},
				After: Column{
					Name:       "email",
					ColumnType: "varchar(255)",
				},
			},
		},
	}

	if tableDiff.Name != "users" {
		t.Errorf("Expected table name 'users', got '%s'", tableDiff.Name)
	}

	if len(tableDiff.ColumnsAdded) != 1 {
		t.Errorf("Expected 1 column added, got %d", len(tableDiff.ColumnsAdded))
	}

	if len(tableDiff.ColumnsRemoved) != 1 {
		t.Errorf("Expected 1 column removed, got %d", len(tableDiff.ColumnsRemoved))
	}

	if len(tableDiff.ColumnsModified) != 1 {
		t.Errorf("Expected 1 column modified, got %d", len(tableDiff.ColumnsModified))
	}

	if tableDiff.ColumnsModified[0].Before.ColumnType != "varchar(100)" {
		t.Errorf("Expected before type 'varchar(100)', got '%s'", tableDiff.ColumnsModified[0].Before.ColumnType)
	}

	if tableDiff.ColumnsModified[0].After.ColumnType != "varchar(255)" {
		t.Errorf("Expected after type 'varchar(255)', got '%s'", tableDiff.ColumnsModified[0].After.ColumnType)
	}
}
