package models

import "time"

type SchemaSnapshot struct {
	Key       string    `json:"key"`       // User-provided or auto-generated identifier
	Timestamp time.Time `json:"timestamp"` // When the snapshot was captured
	Database  string    `json:"database"`  // Database name
	Host      string    `json:"host"`      // Database host
	DBType    string    `json:"db_type"`   // Database type (mysql, postgres, etc.)
	Tables    []Table   `json:"tables"`
	Metadata  Metadata  `json:"metadata"`
}

type Metadata struct {
	Version         string `json:"version"`           // dbc version
	VerifyData      bool   `json:"verify_data"`       // Whether data checksums were captured
	VerifyRowCounts bool   `json:"verify_row_counts"` // Whether exact row counts were captured
	Workers         int    `json:"workers"`           // Number of workers used
	Duration        string `json:"duration"`          // Time taken to capture
}

type Table struct {
	Name          string       `json:"name"`
	Engine        string       `json:"engine,omitempty"` // MySQL specific
	Collation     string       `json:"collation,omitempty"`
	RowCount      int64        `json:"row_count"`                 // Estimated
	ExactRowCount *int64       `json:"exact_row_count,omitempty"` // Optional exact count
	DataLength    int64        `json:"data_length,omitempty"`
	AvgRowLength  int64        `json:"avg_row_length,omitempty"`
	CreateTime    *time.Time   `json:"create_time,omitempty"`
	UpdateTime    *time.Time   `json:"update_time,omitempty"`
	Checksum      string       `json:"checksum,omitempty"` // Optional data checksum
	Columns       []Column     `json:"columns"`
	Indexes       []Index      `json:"indexes"`
	ForeignKeys   []ForeignKey `json:"foreign_keys"`
	Constraints   []Constraint `json:"constraints"`
}

type Column struct {
	Name         string  `json:"name"`
	Position     int     `json:"position"` // Ordinal position
	DataType     string  `json:"data_type"`
	ColumnType   string  `json:"column_type"` // Full type definition (e.g., "varchar(255)")
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value,omitempty"`
	Key          string  `json:"key,omitempty"`   // PRI, UNI, MUL
	Extra        string  `json:"extra,omitempty"` // auto_increment, etc.
}

type Index struct {
	Name      string        `json:"name"`
	IsUnique  bool          `json:"is_unique"`
	IsPrimary bool          `json:"is_primary"`
	Type      string        `json:"type,omitempty"` // BTREE, HASH, etc.
	Columns   []IndexColumn `json:"columns"`
}

type IndexColumn struct {
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`            // Position in index
	Collation string `json:"collation,omitempty"` // ASC, DESC
}

type ForeignKey struct {
	Name             string `json:"name"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete,omitempty"`
	OnUpdate         string `json:"on_update,omitempty"`
}

type Constraint struct {
	Name string `json:"name"`
	Type string `json:"type"` // PRIMARY KEY, UNIQUE, CHECK, etc.
}

type ChangeSet struct {
	Snapshot1Key   string        `json:"snapshot1_key"`
	Snapshot2Key   string        `json:"snapshot2_key"`
	TablesAdded    []Table       `json:"tables_added"`
	TablesRemoved  []Table       `json:"tables_removed"`
	TablesModified []TableDiff   `json:"tables_modified"`
	Summary        ChangeSummary `json:"summary"`
}

type TableDiff struct {
	Name               string           `json:"name"`
	ColumnsAdded       []Column         `json:"columns_added,omitempty"`
	ColumnsRemoved     []Column         `json:"columns_removed,omitempty"`
	ColumnsModified    []ColumnDiff     `json:"columns_modified,omitempty"`
	IndexesAdded       []Index          `json:"indexes_added,omitempty"`
	IndexesRemoved     []Index          `json:"indexes_removed,omitempty"`
	IndexesModified    []IndexDiff      `json:"indexes_modified,omitempty"`
	FKAdded            []ForeignKey     `json:"foreign_keys_added,omitempty"`
	FKRemoved          []ForeignKey     `json:"foreign_keys_removed,omitempty"`
	FKModified         []ForeignKeyDiff `json:"foreign_keys_modified,omitempty"`
	ConstraintsAdded   []Constraint     `json:"constraints_added,omitempty"`
	ConstraintsRemoved []Constraint     `json:"constraints_removed,omitempty"`
	RowCountChange     *int64           `json:"row_count_change,omitempty"`
	ChecksumChanged    bool             `json:"checksum_changed"`
}

type ColumnDiff struct {
	Name   string `json:"name"`
	Before Column `json:"before"`
	After  Column `json:"after"`
}

type IndexDiff struct {
	Name   string `json:"name"`
	Before Index  `json:"before"`
	After  Index  `json:"after"`
}

type ForeignKeyDiff struct {
	Name   string     `json:"name"`
	Before ForeignKey `json:"before"`
	After  ForeignKey `json:"after"`
}

type ChangeSummary struct {
	TablesAdded         int  `json:"tables_added"`
	TablesRemoved       int  `json:"tables_removed"`
	TablesModified      int  `json:"tables_modified"`
	ColumnsAdded        int  `json:"columns_added"`
	ColumnsRemoved      int  `json:"columns_removed"`
	ColumnsModified     int  `json:"columns_modified"`
	IndexesAdded        int  `json:"indexes_added"`
	IndexesRemoved      int  `json:"indexes_removed"`
	IndexesModified     int  `json:"indexes_modified"`
	ForeignKeysAdded    int  `json:"foreign_keys_added"`
	ForeignKeysRemoved  int  `json:"foreign_keys_removed"`
	ForeignKeysModified int  `json:"foreign_keys_modified"`
	HasChanges          bool `json:"has_changes"`
}
