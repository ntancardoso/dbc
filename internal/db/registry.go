package db

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	httpTimeout = 30 * time.Second
)

type DriverRegistry struct {
	Drivers map[string]DriverInfo `json:"drivers"`
}

type DriverInfo struct {
	Name        string                        `json:"name"`
	Version     string                        `json:"version"`
	Description string                        `json:"description"`
	Platforms   map[string]DriverPlatformInfo `json:"platforms"`
}

type DriverPlatformInfo struct {
	URL string `json:"url"`
}

type RegistryManager struct {
	registryURL string
	driversDir  string
	httpClient  *http.Client
}

func NewRegistryManager(registryURL string) (*RegistryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	driversDir := filepath.Join(homeDir, ".dbc", "drivers")

	return &RegistryManager{
		registryURL: registryURL,
		driversDir:  driversDir,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}, nil
}

func (rm *RegistryManager) FetchRegistry() (*DriverRegistry, error) {
	resp, err := rm.httpClient.Get(rm.registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry fetch failed with status: %d", resp.StatusCode)
	}

	var registry DriverRegistry
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &registry, nil
}

func (rm *RegistryManager) InstallDriver(driverName string, version string) error {
	registry, err := rm.FetchRegistry()
	if err != nil {
		return err
	}

	driverInfo, exists := registry.Drivers[driverName]
	if !exists {
		return fmt.Errorf("driver '%s' not found in registry", driverName)
	}

	platform := rm.getCurrentPlatform()
	platformInfo, exists := driverInfo.Platforms[platform]
	if !exists {
		return fmt.Errorf("driver '%s' not available for platform '%s'", driverName, platform)
	}

	downloadURL := platformInfo.URL
	downloadVersion := driverInfo.Version

	if version != "" {
		downloadURL = strings.Replace(platformInfo.URL, driverInfo.Version, version, 1)
		downloadVersion = version
	}

	driverDir := filepath.Join(rm.driversDir, driverName)
	if mkdirErr := os.MkdirAll(driverDir, 0755); mkdirErr != nil {
		return fmt.Errorf("failed to create driver directory: %w", mkdirErr)
	}

	exeName := rm.getDriverExecutableName(driverName)
	driverPath := filepath.Join(driverDir, exeName)

	fmt.Printf("Downloading %s driver %s for %s...\n", driverName, downloadVersion, platform)
	if downloadErr := rm.downloadFile(downloadURL, driverPath); downloadErr != nil {
		return fmt.Errorf("failed to download driver: %w", downloadErr)
	}

	fmt.Println("Fetching checksum from GitHub release...")
	urlFilename := filepath.Base(downloadURL)
	checksum, err := rm.fetchChecksumFromGitHub(downloadURL, urlFilename, downloadVersion)
	if err != nil {
		fmt.Printf("Warning: Could not verify checksum: %v\n", err)
	} else if checksum != "" {
		fmt.Println("Verifying checksum...")
		if err := rm.verifyChecksum(driverPath, checksum); err != nil {
			_ = os.Remove(driverPath)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(driverPath, 0755); err != nil {
			return fmt.Errorf("failed to make driver executable: %w", err)
		}
	}

	metadata := DriverMetadata{
		Name:        driverInfo.Name,
		Version:     downloadVersion,
		Description: driverInfo.Description,
		Path:        driverPath,
	}

	metadataPath := filepath.Join(driverDir, "metadata.json")
	if err := rm.saveMetadata(metadataPath, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Printf("Successfully installed %s driver %s\n", driverName, downloadVersion)
	return nil
}

func (rm *RegistryManager) UninstallDriver(driverName string) error {
	driverDir := filepath.Join(rm.driversDir, driverName)

	if _, err := os.Stat(driverDir); os.IsNotExist(err) {
		return fmt.Errorf("driver '%s' is not installed", driverName)
	}

	if err := os.RemoveAll(driverDir); err != nil {
		return fmt.Errorf("failed to uninstall driver: %w", err)
	}

	fmt.Printf("Successfully uninstalled %s driver\n", driverName)
	return nil
}

func (rm *RegistryManager) ListInstalledDrivers() ([]DriverMetadata, error) {
	var drivers []DriverMetadata

	if _, err := os.Stat(rm.driversDir); os.IsNotExist(err) {
		return drivers, nil // No drivers installed yet
	}

	entries, err := os.ReadDir(rm.driversDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read drivers directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(rm.driversDir, entry.Name(), "metadata.json")
		metadata, err := rm.loadMetadata(metadataPath)
		if err != nil {
			continue // Skip if metadata can't be read
		}

		drivers = append(drivers, metadata)
	}

	return drivers, nil
}

func (rm *RegistryManager) IsDriverInstalled(driverName string) bool {
	driverDir := filepath.Join(rm.driversDir, driverName)
	exeName := rm.getDriverExecutableName(driverName)
	driverPath := filepath.Join(driverDir, exeName)

	_, err := os.Stat(driverPath)
	return err == nil
}

func (rm *RegistryManager) fetchChecksumFromGitHub(downloadURL, filename, _ string) (string, error) {
	baseURL := strings.TrimSuffix(downloadURL, filename)
	checksumURL := baseURL + "checksums.txt"

	resp, err := rm.httpClient.Get(checksumURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksums: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("checksum for %s not found in checksums.txt", filename)
}

func (rm *RegistryManager) getCurrentPlatform() string {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}

func (rm *RegistryManager) getDriverExecutableName(driverName string) string {
	exeName := "dbc-driver-" + driverName
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	return exeName
}

func (rm *RegistryManager) downloadFile(url, filepath string) error {
	resp, err := rm.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

// verifyChecksum verifies the SHA256 checksum of a file
func (rm *RegistryManager) verifyChecksum(filepath, expectedChecksum string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))

	// Use strings.TrimPrefix instead of manual implementation
	expectedChecksum = strings.TrimPrefix(expectedChecksum, "sha256:")

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (rm *RegistryManager) saveMetadata(path string, metadata DriverMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (rm *RegistryManager) loadMetadata(path string) (DriverMetadata, error) {
	var metadata DriverMetadata

	data, err := os.ReadFile(path)
	if err != nil {
		return metadata, err
	}

	err = json.Unmarshal(data, &metadata)
	return metadata, err
}
