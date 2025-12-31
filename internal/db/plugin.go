package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ntancardoso/dbc/internal/models"
)

const (
	// driverTimeout is the maximum time a driver operation can take
	driverTimeout = 5 * time.Minute
)

type PluginDriver struct {
	name     string
	version  string
	path     string
	features DriverFeatures
}

func NewPluginDriver(driverName string) (*PluginDriver, error) {
	driverPath, err := findDriverExecutable(driverName)
	if err != nil {
		return nil, fmt.Errorf("driver not found: %w", err)
	}

	pd := &PluginDriver{
		name: driverName,
		path: driverPath,
	}

	if err := pd.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize driver: %w", err)
	}

	return pd, nil
}

// initialize queries the driver for its version and features
func (pd *PluginDriver) initialize() error {
	versionResp, err := pd.execute(MethodGetVersion, nil)
	if err != nil {
		return fmt.Errorf("failed to get driver version: %w", err)
	}

	var versionData GetVersionResponse
	if err := json.Unmarshal(versionResp.Data, &versionData); err != nil {
		return fmt.Errorf("failed to parse version response: %w", err)
	}
	pd.version = versionData.Version

	featuresResp, err := pd.execute(MethodGetFeatures, nil)
	if err != nil {
		return fmt.Errorf("failed to get driver features: %w", err)
	}

	var featuresData GetFeaturesResponse
	if err := json.Unmarshal(featuresResp.Data, &featuresData); err != nil {
		return fmt.Errorf("failed to parse features response: %w", err)
	}
	pd.features = featuresData.Features

	return nil
}

func (pd *PluginDriver) execute(method string, params map[string]interface{}) (*JSONRPCResponse, error) {
	request := JSONRPCRequest{
		Method: method,
		Params: params,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), driverTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pd.path)
	cmd.Stdin = bytes.NewReader(requestJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("driver execution timed out after %v", driverTimeout)
		}
		return nil, fmt.Errorf("driver execution failed: %w, stderr: %s", err, stderr.String())
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, output: %s", err, stdout.String())
	}

	if !response.Success {
		return nil, fmt.Errorf("driver returned error: %s", response.Error)
	}

	return &response, nil
}

func (pd *PluginDriver) Name() string {
	return pd.name
}

func (pd *PluginDriver) Version() string {
	return pd.version
}

// ExtractSchema extracts the database schema using the driver
func (pd *PluginDriver) ExtractSchema(params ExtractParams) (*models.SchemaSnapshot, error) {
	paramsMap := map[string]interface{}{
		"host":              params.Host,
		"port":              params.Port,
		"user":              params.User,
		"password":          params.Password,
		"database":          params.Database,
		"connection_string": params.ConnectionString,
		"verify_data":       params.VerifyData,
		"verify_row_counts": params.VerifyRowCounts,
		"workers":           params.Workers,
	}

	response, err := pd.execute(MethodExtractSchema, paramsMap)
	if err != nil {
		return nil, err
	}

	var snapshot models.SchemaSnapshot
	if err := json.Unmarshal(response.Data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse schema response: %w", err)
	}

	return &snapshot, nil
}

func (pd *PluginDriver) SupportedFeatures() DriverFeatures {
	return pd.features
}

// findDriverExecutable searches for a driver executable
// Looks in:
// 1. ./bin/dbc-driver-<name> (local development)
// 2. Executable directory (same folder as dbc binary)
// 3. ~/.dbc/drivers/<name>/dbc-driver-<name> (user installed)
// 4. Current directory
// 5. PATH
func findDriverExecutable(driverName string) (string, error) {
	exeName := "dbc-driver-" + driverName
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}

	// 1. Check ./bin directory (local development)
	binPath := filepath.Join("bin", exeName)
	if fileExists(binPath) {
		return binPath, nil
	}

	// 2. Check same directory as executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		driverPath := filepath.Join(execDir, exeName)
		if fileExists(driverPath) {
			return driverPath, nil
		}
	}

	// 3. Check user's driver directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		driverPath := filepath.Join(homeDir, ".dbc", "drivers", driverName, exeName)
		if fileExists(driverPath) {
			return driverPath, nil
		}
	}

	// 4. Check current directory
	if fileExists(exeName) {
		return exeName, nil
	}

	// 5. Check PATH
	path, err := exec.LookPath(exeName)
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("driver executable '%s' not found", exeName)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
