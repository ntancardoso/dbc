package core

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DBType != "mysql" {
		t.Errorf("Expected default DBType 'mysql', got '%s'", cfg.DBType)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Expected default Host 'localhost', got '%s'", cfg.Host)
	}

	if cfg.Port != 3306 {
		t.Errorf("Expected default Port 3306, got %d", cfg.Port)
	}

	if cfg.OutputDir != "./db_snapshots" {
		t.Errorf("Expected default OutputDir './db_snapshots', got '%s'", cfg.OutputDir)
	}

	if cfg.Workers != 10 {
		t.Errorf("Expected default Workers 10, got %d", cfg.Workers)
	}

	if !cfg.AutoInstall {
		t.Error("Expected default AutoInstall true")
	}

	if cfg.VerifyData {
		t.Error("Expected default VerifyData false")
	}

	if !cfg.VerifyRowCounts {
		t.Error("Expected default VerifyRowCounts true")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DBC_OUTPUT_DIR", "/tmp/snapshots")
	os.Setenv("DBC_WORKERS", "20")
	os.Setenv("DBC_VERIFY_DATA", "true")
	os.Setenv("DBC_VERIFY_COUNTS", "false")
	os.Setenv("DBC_AUTO_INSTALL", "false")

	defer func() {
		// Clean up
		os.Unsetenv("DB_TYPE")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DBC_OUTPUT_DIR")
		os.Unsetenv("DBC_WORKERS")
		os.Unsetenv("DBC_VERIFY_DATA")
		os.Unsetenv("DBC_VERIFY_COUNTS")
		os.Unsetenv("DBC_AUTO_INSTALL")
	}()

	cfg := DefaultConfig()
	cfg.LoadFromEnv()

	if cfg.DBType != "postgres" {
		t.Errorf("Expected DBType 'postgres' from env, got '%s'", cfg.DBType)
	}

	if cfg.Host != "db.example.com" {
		t.Errorf("Expected Host 'db.example.com' from env, got '%s'", cfg.Host)
	}

	if cfg.Port != 5432 {
		t.Errorf("Expected Port 5432 from env, got %d", cfg.Port)
	}

	if cfg.User != "testuser" {
		t.Errorf("Expected User 'testuser' from env, got '%s'", cfg.User)
	}

	if cfg.Password != "testpass" {
		t.Errorf("Expected Password 'testpass' from env, got '%s'", cfg.Password)
	}

	if cfg.Database != "testdb" {
		t.Errorf("Expected Database 'testdb' from env, got '%s'", cfg.Database)
	}

	if cfg.OutputDir != "/tmp/snapshots" {
		t.Errorf("Expected OutputDir '/tmp/snapshots' from env, got '%s'", cfg.OutputDir)
	}

	if cfg.Workers != 20 {
		t.Errorf("Expected Workers 20 from env, got %d", cfg.Workers)
	}

	if !cfg.VerifyData {
		t.Error("Expected VerifyData true from env")
	}

	if cfg.VerifyRowCounts {
		t.Error("Expected VerifyRowCounts false from env")
	}

	if cfg.AutoInstall {
		t.Error("Expected AutoInstall false from env")
	}
}

func TestGetConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "MySQL connection string",
			config: Config{
				DBType:   "mysql",
				Host:     "localhost",
				Port:     3306,
				User:     "root",
				Password: "secret",
				Database: "testdb",
			},
			expected: "root:secret@tcp(localhost:3306)/testdb",
		},
		{
			name: "MySQL without password",
			config: Config{
				DBType:   "mysql",
				Host:     "localhost",
				Port:     3306,
				User:     "root",
				Password: "",
				Database: "testdb",
			},
			expected: "root@tcp(localhost:3306)/testdb",
		},
		{
			name: "PostgreSQL connection string",
			config: Config{
				DBType:   "postgres",
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				Database: "testdb",
			},
			expected: "postgres://postgres:secret@localhost:5432/testdb?sslmode=disable",
		},
		{
			name: "SQL Server connection string",
			config: Config{
				DBType:   "sqlserver",
				Host:     "localhost",
				Port:     1433,
				User:     "sa",
				Password: "secret",
				Database: "testdb",
			},
			expected: "sqlserver://sa:secret@localhost:1433?database=testdb",
		},
		{
			name: "SQLite connection string",
			config: Config{
				DBType:   "sqlite",
				Database: "/path/to/database.db",
			},
			expected: "/path/to/database.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetConnectionString()
			if result != tt.expected {
				t.Errorf("Expected connection string '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()

	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}
