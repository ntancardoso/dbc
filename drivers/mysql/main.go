package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	driverName    = "mysql"
	driverVersion = "1.0.0"
)

func main() {
	request, err := readRequest()
	if err != nil {
		writeErrorResponse(fmt.Sprintf("Failed to read request: %v", err))
		os.Exit(1)
	}

	switch request.Method {
	case "get_version":
		handleGetVersion()
	case "get_features":
		handleGetFeatures()
	case "extract_schema":
		handleExtractSchema(request.Params)
	default:
		writeErrorResponse(fmt.Sprintf("Unknown method: %s", request.Method))
		os.Exit(1)
	}
}

type Request struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func readRequest() (*Request, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func writeResponse(data interface{}) {
	resp := Response{
		Success: true,
		Data:    data,
	}

	output, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
}

func writeErrorResponse(errMsg string) {
	resp := Response{
		Success: false,
		Error:   errMsg,
	}

	output, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal error response: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
}

func handleGetVersion() {
	data := map[string]string{
		"name":    driverName,
		"version": driverVersion,
	}
	writeResponse(data)
}

func handleGetFeatures() {
	features := map[string]interface{}{
		"features": map[string]bool{
			"SupportsChecksums":   true,
			"SupportsRowCounts":   true,
			"SupportsIndexes":     true,
			"SupportsForeignKeys": true,
			"SupportsConstraints": true,
		},
	}
	writeResponse(features)
}

func handleExtractSchema(params map[string]interface{}) {
	host := getString(params, "host", "localhost")
	port := getInt(params, "port", 3306)
	user := getString(params, "user", "root")
	password := getString(params, "password", "")
	database := getString(params, "database", "")
	verifyData := getBool(params, "verify_data", false)
	verifyRowCounts := getBool(params, "verify_row_counts", true)

	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		writeErrorResponse(fmt.Sprintf("Failed to connect: %v", err))
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		writeErrorResponse(fmt.Sprintf("Failed to ping database: %v", err))
		return
	}

	snapshot, err := extractMySQLSchema(db, database, verifyData, verifyRowCounts)
	if err != nil {
		writeErrorResponse(fmt.Sprintf("Failed to extract schema: %v", err))
		return
	}

	writeResponse(snapshot)
}

func extractMySQLSchema(db *sql.DB, database string, verifyData, verifyRowCounts bool) (map[string]interface{}, error) {
	startTime := time.Now()

	tables, err := getTables(db, database, verifyData, verifyRowCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	snapshot := map[string]interface{}{
		"database":  database,
		"timestamp": time.Now().Format(time.RFC3339),
		"tables":    tables,
		"metadata": map[string]interface{}{
			"version":           driverVersion,
			"verify_data":       verifyData,
			"verify_row_counts": verifyRowCounts,
			"workers":           1,
			"duration":          time.Since(startTime).String(),
		},
	}

	return snapshot, nil
}

func getString(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getInt(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultValue
}

func getBool(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}
