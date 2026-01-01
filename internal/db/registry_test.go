package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"strings"
)

func TestNewRegistryManager(t *testing.T) {
	rm, err := NewRegistryManager("https://example.com/registry.json")
	if err != nil {
		t.Fatalf("Failed to create registry manager: %v", err)
	}

	if rm.registryURL != "https://example.com/registry.json" {
		t.Errorf("Expected registry URL 'https://example.com/registry.json', got '%s'", rm.registryURL)
	}

	if rm.driversDir == "" {
		t.Error("Expected driversDir to be set")
	}
}

func TestGetCurrentPlatform(t *testing.T) {
	rm := &RegistryManager{}
	platform := rm.getCurrentPlatform()

	expectedPlatform := runtime.GOOS + "-" + runtime.GOARCH
	if platform != expectedPlatform {
		t.Errorf("Expected platform '%s', got '%s'", expectedPlatform, platform)
	}
}

func TestGetDriverExecutableNameWindows(t *testing.T) {
	rm := &RegistryManager{}

	tests := []struct {
		driverName string
		wantSuffix string
	}{
		{"mysql", "dbc-driver-mysql"},
		{"postgres", "dbc-driver-postgres"},
		{"sqlite", "dbc-driver-sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.driverName, func(t *testing.T) {
			result := rm.getDriverExecutableName(tt.driverName)

			if runtime.GOOS == "windows" {
				expected := tt.wantSuffix + ".exe"
				if result != expected {
					t.Errorf("Expected '%s', got '%s'", expected, result)
				}
			} else if result != tt.wantSuffix {
				t.Errorf("Expected '%s', got '%s'", tt.wantSuffix, result)
			}
		})
	}
}

func TestFetchRegistry(t *testing.T) {
	// Create a mock HTTP server
	registry := DriverRegistry{
		Drivers: map[string]DriverInfo{
			"mysql": {
				Name:        "mysql",
				Version:     "1.0.0",
				Description: "MySQL driver",
				Platforms: map[string]DriverPlatformInfo{
					"linux-amd64": {
						URL:      "https://example.com/mysql-linux-amd64",
						Checksum: "sha256:abc123",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(registry)
	}))
	defer server.Close()

	rm, err := NewRegistryManager(server.URL)
	if err != nil {
		t.Fatalf("Failed to create registry manager: %v", err)
	}

	fetchedRegistry, err := rm.FetchRegistry()
	if err != nil {
		t.Fatalf("Failed to fetch registry: %v", err)
	}

	if len(fetchedRegistry.Drivers) != 1 {
		t.Errorf("Expected 1 driver, got %d", len(fetchedRegistry.Drivers))
	}

	mysqlDriver, exists := fetchedRegistry.Drivers["mysql"]
	if !exists {
		t.Fatal("Expected mysql driver in registry")
	}

	if mysqlDriver.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", mysqlDriver.Version)
	}
}

func TestIsDriverInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	rm := &RegistryManager{
		driversDir: tmpDir,
	}

	// Driver not installed yet
	if rm.IsDriverInstalled("mysql") {
		t.Error("Expected driver to not be installed")
	}

	// Create driver directory and executable
	driverDir := filepath.Join(tmpDir, "mysql")
	if err := os.MkdirAll(driverDir, 0755); err != nil {
		t.Fatalf("Failed to create driver directory: %v", err)
	}

	exeName := rm.getDriverExecutableName("mysql")
	driverPath := filepath.Join(driverDir, exeName)

	f, err := os.Create(driverPath)
	if err != nil {
		t.Fatalf("Failed to create driver executable: %v", err)
	}
	_ = f.Close()

	// Driver should be installed now
	if !rm.IsDriverInstalled("mysql") {
		t.Error("Expected driver to be installed")
	}
}

func TestListInstalledDrivers(t *testing.T) {
	tmpDir := t.TempDir()

	rm := &RegistryManager{
		driversDir: tmpDir,
	}

	// No drivers installed
	drivers, err := rm.ListInstalledDrivers()
	if err != nil {
		t.Fatalf("Failed to list drivers: %v", err)
	}

	if len(drivers) != 0 {
		t.Errorf("Expected 0 drivers, got %d", len(drivers))
	}

	// Install a driver
	driverDir := filepath.Join(tmpDir, "mysql")
	if mkdirErr := os.MkdirAll(driverDir, 0755); mkdirErr != nil {
		t.Fatalf("Failed to create driver directory: %v", mkdirErr)
	}

	metadata := DriverMetadata{
		Name:        "mysql",
		Version:     "1.0.0",
		Description: "MySQL driver",
		Path:        filepath.Join(driverDir, "dbc-driver-mysql"),
	}

	metadataPath := filepath.Join(driverDir, "metadata.json")
	if saveErr := rm.saveMetadata(metadataPath, metadata); saveErr != nil {
		t.Fatalf("Failed to save metadata: %v", saveErr)
	}

	// List drivers again
	drivers, err = rm.ListInstalledDrivers()
	if err != nil {
		t.Fatalf("Failed to list drivers: %v", err)
	}

	if len(drivers) != 1 {
		t.Errorf("Expected 1 driver, got %d", len(drivers))
	}

	if drivers[0].Name != "mysql" {
		t.Errorf("Expected driver name 'mysql', got '%s'", drivers[0].Name)
	}

	if drivers[0].Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", drivers[0].Version)
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	rm := &RegistryManager{
		driversDir: tmpDir,
	}

	metadata := DriverMetadata{
		Name:        "mysql",
		Version:     "1.0.0",
		Description: "MySQL driver",
		Path:        "/path/to/driver",
	}

	metadataPath := filepath.Join(tmpDir, "metadata.json")

	// Save metadata
	if err := rm.saveMetadata(metadataPath, metadata); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Load metadata
	loaded, err := rm.loadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loaded.Name != metadata.Name {
		t.Errorf("Expected name '%s', got '%s'", metadata.Name, loaded.Name)
	}

	if loaded.Version != metadata.Version {
		t.Errorf("Expected version '%s', got '%s'", metadata.Version, loaded.Version)
	}

	if loaded.Description != metadata.Description {
		t.Errorf("Expected description '%s', got '%s'", metadata.Description, loaded.Description)
	}

	if loaded.Path != metadata.Path {
		t.Errorf("Expected path '%s', got '%s'", metadata.Path, loaded.Path)
	}
}

func TestTrimPrefix(t *testing.T) {
	tests := []struct {
		input    string
		prefix   string
		expected string
	}{
		{"sha256:abc123", "sha256:", "abc123"},
		{"abc123", "sha256:", "abc123"},
		{"", "sha256:", ""},
		{"sha256:", "sha256:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strings.TrimPrefix(tt.input, tt.prefix)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestUninstallDriver(t *testing.T) {
	tmpDir := t.TempDir()

	rm := &RegistryManager{
		driversDir: tmpDir,
	}

	// Create driver directory
	driverDir := filepath.Join(tmpDir, "mysql")
	if err := os.MkdirAll(driverDir, 0755); err != nil {
		t.Fatalf("Failed to create driver directory: %v", err)
	}

	// Create a file in the driver directory
	testFile := filepath.Join(driverDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Uninstall driver
	if err := rm.UninstallDriver("mysql"); err != nil {
		t.Fatalf("Failed to uninstall driver: %v", err)
	}

	// Driver directory should be removed
	if _, err := os.Stat(driverDir); !os.IsNotExist(err) {
		t.Error("Expected driver directory to be removed")
	}

	// Uninstalling non-existent driver should error
	if err := rm.UninstallDriver("nonexistent"); err == nil {
		t.Error("Expected error when uninstalling non-existent driver")
	}
}
