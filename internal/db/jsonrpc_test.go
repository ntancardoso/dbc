package db

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCRequest(t *testing.T) {
	req := JSONRPCRequest{
		Method: "extract_schema",
		Params: map[string]interface{}{
			"host":     "localhost",
			"port":     3306,
			"database": "testdb",
		},
	}

	if req.Method != "extract_schema" {
		t.Errorf("Expected method 'extract_schema', got '%s'", req.Method)
	}

	if req.Params["host"] != "localhost" {
		t.Errorf("Expected host 'localhost', got '%v'", req.Params["host"])
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var unmarshaled JSONRPCRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.Method != req.Method {
		t.Errorf("Expected method '%s', got '%s'", req.Method, unmarshaled.Method)
	}
}

func TestJSONRPCResponse(t *testing.T) {
	resp := JSONRPCResponse{
		Success: true,
		Data:    json.RawMessage(`{"version":"1.0.0"}`),
	}

	if !resp.Success {
		t.Error("Expected success true")
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled JSONRPCResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !unmarshaled.Success {
		t.Error("Expected success true")
	}
}

func TestJSONRPCErrorResponse(t *testing.T) {
	resp := JSONRPCResponse{
		Success: false,
		Error:   "connection failed",
	}

	if resp.Success {
		t.Error("Expected success false")
	}

	if resp.Error != "connection failed" {
		t.Errorf("Expected error 'connection failed', got '%s'", resp.Error)
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled JSONRPCResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.Success {
		t.Error("Expected success false")
	}

	if unmarshaled.Error != resp.Error {
		t.Errorf("Expected error '%s', got '%s'", resp.Error, unmarshaled.Error)
	}
}

func TestExtractSchemaRequest(t *testing.T) {
	req := ExtractSchemaRequest{
		Host:            "localhost",
		Port:            3306,
		User:            "root",
		Password:        "secret",
		Database:        "testdb",
		VerifyData:      true,
		VerifyRowCounts: true,
		Workers:         10,
	}

	if req.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", req.Host)
	}

	if req.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", req.Port)
	}

	if req.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", req.Database)
	}

	if !req.VerifyData {
		t.Error("Expected VerifyData true")
	}

	if req.Workers != 10 {
		t.Errorf("Expected workers 10, got %d", req.Workers)
	}

	// Test JSON marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var unmarshaled ExtractSchemaRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.Host != req.Host {
		t.Errorf("Expected host '%s', got '%s'", req.Host, unmarshaled.Host)
	}
}

func TestGetVersionResponse(t *testing.T) {
	resp := GetVersionResponse{
		Name:    "mysql",
		Version: "1.0.0",
	}

	if resp.Name != "mysql" {
		t.Errorf("Expected name 'mysql', got '%s'", resp.Name)
	}

	if resp.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", resp.Version)
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled GetVersionResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.Name != resp.Name {
		t.Errorf("Expected name '%s', got '%s'", resp.Name, unmarshaled.Name)
	}

	if unmarshaled.Version != resp.Version {
		t.Errorf("Expected version '%s', got '%s'", resp.Version, unmarshaled.Version)
	}
}

func TestGetFeaturesResponse(t *testing.T) {
	resp := GetFeaturesResponse{
		Features: DriverFeatures{
			SupportsChecksums:   true,
			SupportsRowCounts:   true,
			SupportsIndexes:     true,
			SupportsForeignKeys: true,
			SupportsConstraints: true,
		},
	}

	if !resp.Features.SupportsChecksums {
		t.Error("Expected SupportsChecksums true")
	}

	// Test JSON marshaling
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled GetFeaturesResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !unmarshaled.Features.SupportsChecksums {
		t.Error("Expected SupportsChecksums true")
	}

	if !unmarshaled.Features.SupportsIndexes {
		t.Error("Expected SupportsIndexes true")
	}
}

func TestMethodConstants(t *testing.T) {
	if MethodExtractSchema != "extract_schema" {
		t.Errorf("Expected MethodExtractSchema 'extract_schema', got '%s'", MethodExtractSchema)
	}

	if MethodGetVersion != "get_version" {
		t.Errorf("Expected MethodGetVersion 'get_version', got '%s'", MethodGetVersion)
	}

	if MethodGetFeatures != "get_features" {
		t.Errorf("Expected MethodGetFeatures 'get_features', got '%s'", MethodGetFeatures)
	}
}
