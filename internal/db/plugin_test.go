package db

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindDriverExecutable(t *testing.T) {
	// This test requires an actual driver binary to exist
	// Skip if not in a full test environment
	t.Skip("Requires actual driver binary - integration test")
}

func TestGetDriverExecutableName(t *testing.T) {
	tests := []struct {
		name       string
		driverName string
		expected   string
	}{
		{
			name:       "MySQL driver name on Windows",
			driverName: "mysql",
			expected:   getExpectedExeName("mysql"),
		},
		{
			name:       "PostgreSQL driver name",
			driverName: "postgres",
			expected:   getExpectedExeName("postgres"),
		},
		{
			name:       "SQLite driver name",
			driverName: "sqlite",
			expected:   getExpectedExeName("sqlite"),
		},
	}

	rm := &RegistryManager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rm.getDriverExecutableName(tt.driverName)
			if result != tt.expected {
				t.Errorf("Expected executable name '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func getExpectedExeName(driverName string) string {
	exeName := "dbc-driver-" + driverName
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	return exeName
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// File doesn't exist yet
	if fileExists(tmpFile) {
		t.Error("Expected file to not exist")
	}

	// Create the file
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_ = f.Close()

	// File should exist now
	if !fileExists(tmpFile) {
		t.Error("Expected file to exist")
	}
}

func TestPluginDriverInterface(_ *testing.T) {
	// Test that PluginDriver implements the Driver interface
	var _ Driver = (*PluginDriver)(nil)
}

func TestExtractParams(t *testing.T) {
	params := ExtractParams{
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Workers:  10,
	}

	if params.Host != "localhost" {
		t.Errorf("Expected Host 'localhost', got '%s'", params.Host)
	}

	if params.Port != 3306 {
		t.Errorf("Expected Port 3306, got %d", params.Port)
	}

	if params.Database != "testdb" {
		t.Errorf("Expected Database 'testdb', got '%s'", params.Database)
	}

	if params.Workers != 10 {
		t.Errorf("Expected Workers 10, got %d", params.Workers)
	}
}

func TestDriverFeatures(t *testing.T) {
	features := DriverFeatures{
		SupportsChecksums:   true,
		SupportsRowCounts:   true,
		SupportsIndexes:     true,
		SupportsForeignKeys: true,
		SupportsConstraints: true,
	}

	if !features.SupportsChecksums {
		t.Error("Expected SupportsChecksums true")
	}

	if !features.SupportsRowCounts {
		t.Error("Expected SupportsRowCounts true")
	}

	if !features.SupportsIndexes {
		t.Error("Expected SupportsIndexes true")
	}

	if !features.SupportsForeignKeys {
		t.Error("Expected SupportsForeignKeys true")
	}

	if !features.SupportsConstraints {
		t.Error("Expected SupportsConstraints true")
	}
}
