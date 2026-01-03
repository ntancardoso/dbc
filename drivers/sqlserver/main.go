package main

import (
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/microsoft/go-mssqldb"
)

const (
	driverName    = "sqlserver"
	driverVersion = "1.0.0"
)

type JSONRPCRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type JSONRPCResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func main() {
	request, err := readRequest()
	if err != nil {
		writeError(fmt.Sprintf("Failed to read request: %v", err))
		return
	}

	switch request.Method {
	case "get_version":
		handleGetVersion()
	case "get_features":
		handleGetFeatures()
	case "extract_schema":
		handleExtractSchema(request.Params)
	default:
		writeError(fmt.Sprintf("Unknown method: %s", request.Method))
	}
}

func readRequest() (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func writeResponse(data interface{}) {
	jsonData, _ := json.Marshal(data)
	response := JSONRPCResponse{
		Success: true,
		Data:    jsonData,
	}
	json.NewEncoder(os.Stdout).Encode(response)
}

func writeError(errMsg string) {
	response := JSONRPCResponse{
		Success: false,
		Error:   errMsg,
	}
	json.NewEncoder(os.Stdout).Encode(response)
}

func handleGetVersion() {
	writeResponse(map[string]string{
		"name":    driverName,
		"version": driverVersion,
	})
}

func handleGetFeatures() {
	writeResponse(map[string]interface{}{
		"SupportsChecksums":   true,
		"SupportsRowCounts":   true,
		"SupportsIndexes":     true,
		"SupportsForeignKeys": true,
		"SupportsConstraints": true,
	})
}

func handleExtractSchema(params map[string]interface{}) {
	connStr, ok := params["connection_string"].(string)
	if !ok || connStr == "" {
		writeError("connection_string is required")
		return
	}

	database, _ := params["database"].(string)
	verifyData, _ := params["verify_data"].(bool)
	verifyRowCounts, _ := params["verify_row_counts"].(bool)

	snapshot, err := extractSchema(connStr, database, verifyData, verifyRowCounts)
	if err != nil {
		writeError(fmt.Sprintf("Failed to extract schema: %v", err))
		return
	}

	writeResponse(snapshot)
}
