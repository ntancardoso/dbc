package db

import (
	"encoding/json"
	"github.com/ntancardoso/dbc/internal/models"
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

const (
	MethodExtractSchema = "extract_schema"
	MethodGetVersion    = "get_version"
	MethodGetFeatures   = "get_features"
)

type ExtractSchemaRequest struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	User            string `json:"user"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	VerifyData      bool   `json:"verify_data"`
	VerifyRowCounts bool   `json:"verify_row_counts"`
	Workers         int    `json:"workers"`
}

type ExtractSchemaResponse struct {
	Snapshot *models.SchemaSnapshot `json:"snapshot"`
}

type GetVersionResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type GetFeaturesResponse struct {
	Features DriverFeatures `json:"features"`
}
