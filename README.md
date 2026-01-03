# DBC - Database Comparison Tool

A powerful Go-based database comparison tool that captures database schema snapshots and compares them across different versions, environments, or database systems. Built with a plugin architecture for extensibility and multi-database support.

## Features

- **Multi-Database Support**: MySQL, PostgreSQL, SQL Server, Oracle, SQLite
- **Schema Change Detection**: Tracks tables, columns, indexes, foreign keys, constraints
- **Data Verification**: Optional data checksums to detect data modifications
- **Multiple Output Formats**: Text, JSON, and HTML reports
- **Flexible Comparison**: Compare any two snapshots - versions, environments, or database types
- **Plugin Architecture**: Modular driver system for database connectivity
- **Parallel Processing**: Configurable workers for fast schema extraction
- **Cross-Platform**: Works on Windows, Linux, macOS

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ntancardoso/dbc.git
cd dbc

# Build core binary
make build

# Build all drivers
make build-all

# Run tests
make test
```

The binaries will be available in `./bin/`:
- `dbc.exe` - Core application
- `dbc-driver-mysql.exe` - MySQL driver
- `dbc-driver-postgres.exe` - PostgreSQL driver
- `dbc-driver-sqlserver.exe` - SQL Server driver
- `dbc-driver-oracle.exe` - Oracle driver
- `dbc-driver-sqlite.exe` - SQLite driver

## Quick Start

### 1. Capture a Database Snapshot

```bash
# MySQL
./bin/dbc.exe capture -dbtype mysql -host localhost -port 3306 \
  -user root -password secret -database mydb

# PostgreSQL
./bin/dbc.exe capture -dbtype postgres -host localhost -port 5432 \
  -user postgres -password secret -database mydb

# SQL Server
./bin/dbc.exe capture -dbtype sqlserver -host localhost -port 1433 \
  -user sa -password secret -database mydb

# Oracle
./bin/dbc.exe capture -dbtype oracle -host localhost -port 1521 \
  -user system -password secret -database XE

# SQLite
./bin/dbc.exe capture -dbtype sqlite -database /path/to/database.db
```

### 2. List Captured Snapshots

```bash
./bin/dbc.exe list
```

Output:
```
Snapshots in ./db_snapshots:

KEY                  DATABASE        TIMESTAMP                 TABLES
--------------------------------------------------------------------------------
snapshot_20251231_202450 mydb            2025-12-31 20:24:50       15
snapshot_20251231_202114 mydb            2025-12-31 20:21:14       15
```

### 3. Compare Two Snapshots

```bash
# Text format (default)
./bin/dbc.exe compare snapshot_20251231_202114 snapshot_20251231_202450

# JSON format
./bin/dbc.exe compare snapshot_20251231_202114 snapshot_20251231_202450 -format json

# HTML format
./bin/dbc.exe compare snapshot_20251231_202114 snapshot_20251231_202450 -format html > report.html
```

## Command Reference

### capture - Capture Database Schema

```bash
dbc capture [flags]

Flags:
  -dbtype string          Database type (mysql, postgres, sqlserver, oracle, sqlite)
  -host string           Database host (default: localhost)
  -port int              Database port
  -user string           Database username
  -password string       Database password
  -database string       Database name (required)
  -output string         Output directory (default: ./db_snapshots)
  -workers int           Number of parallel workers (default: 10)
  -verify-data           Calculate data checksums (default: false)
  -verify-counts         Get exact row counts (default: true)
```

### compare - Compare Two Snapshots

```bash
dbc compare <snapshot1> <snapshot2> [flags]

Flags:
  -format string         Output format: text, json, html (default: text)
  -output string         Snapshot directory (default: ./db_snapshots)
```

### list - List All Snapshots

```bash
dbc list [flags]

Flags:
  -output string         Snapshot directory (default: ./db_snapshots)
```

## Schema Elements Captured

DBC captures comprehensive database schema information:

- **Tables**: Name, engine, collation, row count, creation time
- **Columns**: Name, data type, nullable, default value, key type, position, extra info
- **Indexes**: Name, uniqueness, primary key flag, type, columns
- **Foreign Keys**: Name, column, referenced table, referenced column, on delete/update actions
- **Constraints**: Primary keys, unique constraints, foreign keys
- **Row Counts**: Estimated and exact counts
- **Checksums**: Optional data checksums for detecting modifications

## Output Formats

### Text Format (Human-Readable)

```
=== Schema Comparison: baseline → latest ===

Summary:
  Tables Added:    1
  Tables Removed:  0
  Tables Modified: 2

Added Tables:
  + audit_log (5 columns, 0 rows)

Modified Tables:
  ~ users
    Added Columns:
      + phone (varchar(20))
    Added Indexes:
      + idx_users_phone
```

### JSON Format (Machine-Readable)

```json
{
  "baseline_key": "snapshot_20251231_202114",
  "target_key": "snapshot_20251231_202450",
  "summary": {
    "tables_added": 1,
    "tables_removed": 0,
    "tables_modified": 2
  },
  "changes": {
    "tables_added": [...],
    "tables_removed": [...],
    "tables_modified": [...]
  }
}
```

### HTML Format (Visual Reports)

Beautiful HTML reports with:
- Gradient headers
- Color-coded changes (green for additions, red for removals, yellow for modifications)
- Summary statistics
- Detailed change breakdowns
- Responsive design

## Example Workflows

### Version-Based Workflow

```bash
# Capture baseline
./bin/dbc.exe capture -dbtype mysql -database myapp -host localhost -user root -password secret

# Apply migrations
# ... run your migration scripts ...

# Capture after migration
./bin/dbc.exe capture -dbtype mysql -database myapp -host localhost -user root -password secret

# Compare to see what changed
./bin/dbc.exe compare snapshot_20251231_100000 snapshot_20251231_110000 -format html > migration_report.html
```

### Environment Comparison

```bash
# Capture development database
./bin/dbc.exe capture -dbtype postgres -database myapp -host dev.example.com -user devuser -password devpass

# Capture production database
./bin/dbc.exe capture -dbtype postgres -database myapp -host prod.example.com -user produser -password prodpass

# Compare to find drift
./bin/dbc.exe compare snapshot_dev snapshot_prod
```

### Data Verification

```bash
# Capture with checksums
./bin/dbc.exe capture -dbtype mysql -database myapp -verify-data

# Make data changes
# ... update some records ...

# Capture again with checksums
./bin/dbc.exe capture -dbtype mysql -database myapp -verify-data

# Compare - will detect data changes even if schema is identical
./bin/dbc.exe compare snapshot_before snapshot_after
```

Output will show:
```
Modified Tables:
  ~ users
    ⚠ Data Checksum Changed (data modified)
```

## Plugin Architecture

DBC uses a plugin-based architecture for database drivers:

### How It Works

1. **Core Application**: Lightweight CLI that handles snapshot management and comparison
2. **Driver Plugins**: Separate executables for each database type
3. **JSON-RPC Communication**: Drivers communicate via stdin/stdout using JSON-RPC protocol
4. **Driver Discovery**: Core finds drivers in `./bin/` directory or same directory as executable

### Driver Location Priority

The core searches for drivers in this order:
1. `./bin/dbc-driver-<name>.exe` (local development)
2. Same directory as dbc executable
3. `~/.dbc/drivers/<name>/dbc-driver-<name>.exe` (user installed)
4. Current working directory
5. System PATH

### Why Plugin Architecture?

- **Modularity**: Easy to add new database support
- **Cross-Platform**: Pure Go, no CGO dependencies
- **Maintainability**: Each driver is independent
- **Flexibility**: Mix and match driver versions
- **Security**: Drivers run in separate processes with timeouts

### Timeout Configuration

- HTTP operations: 30 seconds
- Driver operations: 5 minutes
- Prevents hanging on slow databases or network issues

## Project Structure

```
dbc/
├── cmd/dbc/                   # CLI entry point
├── internal/
│   ├── core/                  # CLI commands and routing
│   ├── db/                    # Driver interface and plugin system
│   ├── models/                # Data models for schema representation
│   └── projectpath/           # Path utilities
├── drivers/                   # Database driver implementations
│   ├── mysql/                 # MySQL driver
│   ├── postgres/              # PostgreSQL driver
│   ├── sqlserver/             # SQL Server driver
│   ├── oracle/                # Oracle driver
│   └── sqlite/                # SQLite driver
├── bin/                       # Build output directory
└── Makefile                   # Build automation
```

## Development

### Build Commands

```bash
# Build core only
make build

# Build all drivers
make build-drivers

# Build everything
make build-all

# Run tests
make test

# Clean build artifacts
make clean
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/core
```

### Adding a New Driver

1. Create driver directory: `drivers/newdb/`
2. Implement JSON-RPC interface in `main.go`
3. Support these methods:
   - `get_version` - Return driver version
   - `get_features` - Return supported features
   - `extract_schema` - Extract database schema
4. Add build target to Makefile
5. Update README with examples

## Testing Results

All database drivers have been tested and verified:

- ✅ MySQL - Fully functional
- ✅ PostgreSQL - Fully functional
- ✅ SQL Server - Fully functional
- ✅ Oracle - Fully functional
- ✅ SQLite - Fully functional

Features tested:
- Schema extraction
- Column detection
- Index detection
- Foreign key relationships
- Row count tracking
- Data checksum verification
- Text, JSON, and HTML output formats

## Roadmap

- [x] Core plugin architecture
- [x] MySQL driver
- [x] PostgreSQL driver
- [x] SQL Server driver
- [x] SQLite driver
- [x] Oracle driver
- [x] Schema comparison engine
- [x] Text output format
- [x] JSON output format
- [x] HTML output format
- [x] Data checksum verification
- [ ] Driver registry and auto-download
- [ ] Snapshot versioning and tagging
- [ ] CI/CD integration examples
- [ ] Docker container support
- [ ] Migration script generation
- [ ] Snapshot diff visualization (web UI)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details
