.PHONY: build build-drivers build-all test clean install run help

# Variables
BINARY_NAME=dbc
MAIN_PATH=./cmd/dbc
DRIVERS_DIR=./drivers
BIN_DIR=./bin
VERSION?=0.1.0

# Detect OS
ifeq ($(OS),Windows_NT)
    BINARY_EXT=.exe
    RM=del /Q
    RMDIR=rmdir /S /Q
    MKDIR=mkdir
else
    BINARY_EXT=
    RM=rm -f
    RMDIR=rm -rf
    MKDIR=mkdir -p
endif

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the core dbc binary
	@echo "Building dbc..."
	@go build -ldflags "-X github.com/ntancardoso/dbc/internal/core.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME)$(BINARY_EXT) $(MAIN_PATH)
	@echo "✓ Built $(BIN_DIR)/$(BINARY_NAME)$(BINARY_EXT)"

build-drivers: ## Build all database drivers
	@echo "Building drivers..."
	@$(MKDIR) $(BIN_DIR) 2>/dev/null || true
	@cd $(DRIVERS_DIR)/mysql && go build -o ../../$(BIN_DIR)/dbc-driver-mysql$(BINARY_EXT)
	@echo "✓ Built MySQL driver"
	@cd $(DRIVERS_DIR)/sqlite && go build -o ../../$(BIN_DIR)/dbc-driver-sqlite$(BINARY_EXT)
	@echo "✓ Built SQLite driver"
	@cd $(DRIVERS_DIR)/postgres && go build -o ../../$(BIN_DIR)/dbc-driver-postgres$(BINARY_EXT)
	@echo "✓ Built PostgreSQL driver"
	@cd $(DRIVERS_DIR)/sqlserver && go build -o ../../$(BIN_DIR)/dbc-driver-sqlserver$(BINARY_EXT)
	@echo "✓ Built SQL Server driver"
	@cd $(DRIVERS_DIR)/oracle && go build -o ../../$(BIN_DIR)/dbc-driver-oracle$(BINARY_EXT)
	@echo "✓ Built Oracle driver"

build-all: build build-drivers ## Build core binary and all drivers

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@$(RMDIR) $(BIN_DIR) 2>/dev/null || true
	@echo "✓ Cleaned"

install: build ## Install dbc to GOPATH/bin
	@echo "Installing..."
	@go install $(MAIN_PATH)
	@echo "✓ Installed to GOPATH/bin"

run: build ## Build and run dbc
	@$(BIN_DIR)/$(BINARY_NAME)$(BINARY_EXT)

# Development targets
dev-mysql: build-all ## Build and test MySQL driver locally
	@echo "Testing MySQL driver..."
	@$(BIN_DIR)/$(BINARY_NAME)$(BINARY_EXT) driver list
