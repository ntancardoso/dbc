package db

import (
	"github.com/ntancardoso/dbc/internal/models"
)

type Driver interface {
	Name() string
	Version() string
	ExtractSchema(params ExtractParams) (*models.SchemaSnapshot, error)
	SupportedFeatures() DriverFeatures
}

type ExtractParams struct {
	Host             string
	Port             int
	User             string
	Password         string
	Database         string
	ConnectionString string
	VerifyData       bool
	VerifyRowCounts  bool
	Workers          int
}

type DriverFeatures struct {
	SupportsChecksums   bool
	SupportsRowCounts   bool
	SupportsIndexes     bool
	SupportsForeignKeys bool
	SupportsConstraints bool
}

type DriverMetadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Path        string `json:"path"`
}
