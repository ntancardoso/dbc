package core

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/ntancardoso/dbc/internal/db"
)

const version = "0.1.0"

func Run(args []string) error {
	_ = godotenv.Load()

	if len(args) < 2 {
		printUsage()
		return nil
	}

	command := args[1]

	switch command {
	case "capture", "save", "snapshot":
		return runCapture(args[2:])
	case "compare", "diff":
		return runCompare(args[2:])
	case "list", "ls":
		return runList(args[2:])
	case "show":
		return runShow(args[2:])
	case "driver":
		return runDriver(args[2:])
	case "version", "--version", "-v":
		fmt.Printf("dbc version %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s (use 'dbc help' for usage)", command)
	}
}

func runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)

	dbType := fs.String("dbtype", "", "Database type (mysql, postgres, sqlserver, sqlite)")
	host := fs.String("host", "", "Database host")
	port := fs.Int("port", 0, "Database port")
	user := fs.String("user", "", "Database user")
	password := fs.String("password", "", "Database password")
	database := fs.String("database", "", "Database name or file path (for sqlite)")

	outputDir := fs.String("output", "", "Output directory for snapshots")
	verifyData := fs.Bool("verify-data", false, "Verify data with checksums")
	verifyRowCounts := fs.Bool("verify-counts", true, "Get exact row counts")
	workers := fs.Int("workers", 10, "Number of parallel workers")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	cfg := DefaultConfig()
	cfg.LoadFromEnv()

	if *dbType != "" {
		cfg.DBType = *dbType
	}
	if *host != "" {
		cfg.Host = *host
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if *user != "" {
		cfg.User = *user
	}
	if *password != "" {
		cfg.Password = *password
	}
	if *database != "" {
		cfg.Database = *database
	}
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}
	cfg.VerifyData = *verifyData
	cfg.VerifyRowCounts = *verifyRowCounts
	cfg.Workers = *workers

	var snapshotKey string
	if fs.NArg() > 0 {
		snapshotKey = fs.Arg(0)
	}

	if cfg.Database == "" {
		return fmt.Errorf("database name is required (use --database or DB_NAME)")
	}

	fmt.Printf("Capturing snapshot of %s database '%s'...\n", cfg.DBType, cfg.Database)

	driver, err := db.NewPluginDriver(cfg.DBType)
	if err != nil {
		return fmt.Errorf("failed to load driver: %w", err)
	}

	connStr := cfg.GetConnectionString()

	params := db.ExtractParams{
		Host:             cfg.Host,
		Port:             cfg.Port,
		User:             cfg.User,
		Password:         cfg.Password,
		Database:         cfg.Database,
		ConnectionString: connStr,
		VerifyData:       cfg.VerifyData,
		VerifyRowCounts:  cfg.VerifyRowCounts,
		Workers:          cfg.Workers,
	}

	snapshot, err := driver.ExtractSchema(params)
	if err != nil {
		return fmt.Errorf("failed to extract schema: %w", err)
	}

	if snapshotKey == "" {
		snapshotKey = fmt.Sprintf("snapshot_%s", snapshot.Timestamp.Format("20060102_150405"))
	}

	snapshot.Key = snapshotKey
	snapshot.Host = cfg.Host

	storage := NewSnapshotStorage(cfg.OutputDir)
	if err := storage.Save(snapshot); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	fmt.Printf("✓ Snapshot captured: %s\n", snapshotKey)
	fmt.Printf("  Database: %s\n", cfg.Database)
	fmt.Printf("  Tables: %d\n", len(snapshot.Tables))
	fmt.Printf("  Saved to: %s\n", cfg.OutputDir)

	return nil
}

func runCompare(args []string) error {
	var positionalArgs []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positionalArgs = append(positionalArgs, args[i])
		}
	}

	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	outputDir := fs.String("output", "", "Snapshot directory")
	format := fs.String("format", "text", "Output format (text, json, html)")
	if err := fs.Parse(flagArgs); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if len(positionalArgs) < 2 {
		return fmt.Errorf("compare requires two snapshot keys")
	}

	key1 := positionalArgs[0]
	key2 := positionalArgs[1]

	cfg := DefaultConfig()
	cfg.LoadFromEnv()
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}

	storage := NewSnapshotStorage(cfg.OutputDir)

	fmt.Fprintf(os.Stderr, "Loading snapshots...\n")
	snapshot1, err := storage.Load(key1)
	if err != nil {
		return fmt.Errorf("failed to load snapshot '%s': %w", key1, err)
	}

	snapshot2, err := storage.Load(key2)
	if err != nil {
		return fmt.Errorf("failed to load snapshot '%s': %w", key2, err)
	}

	fmt.Fprintf(os.Stderr, "Comparing: %s → %s\n\n", key1, key2)
	changeSet := CompareSnapshots(snapshot1, snapshot2)

	var output string
	switch *format {
	case "json":
		jsonOutput, err := FormatChangeSetJSON(changeSet, key1, key2)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		output = jsonOutput
	case "html":
		htmlOutput, err := FormatChangeSetHTML(changeSet, key1, key2)
		if err != nil {
			return fmt.Errorf("failed to format HTML: %w", err)
		}
		output = htmlOutput
	default:
		output = FormatChangeSet(changeSet, key1, key2)
	}

	fmt.Println(output)

	return nil
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	outputDir := fs.String("output", "", "Snapshot directory")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	cfg := DefaultConfig()
	cfg.LoadFromEnv()
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}

	storage := NewSnapshotStorage(cfg.OutputDir)

	snapshots, err := storage.List()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	fmt.Printf("Snapshots in %s:\n\n", cfg.OutputDir)
	fmt.Printf("%-20s %-15s %-25s %s\n", "KEY", "DATABASE", "TIMESTAMP", "TABLES")
	fmt.Println(strings.Repeat("-", 80))

	for _, snapshot := range snapshots {
		fmt.Printf("%-20s %-15s %-25s %d\n",
			snapshot.Key,
			snapshot.Database,
			snapshot.Timestamp.Format("2006-01-02 15:04:05"),
			snapshot.Tables,
		)
	}

	return nil
}

func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	outputDir := fs.String("output", "", "Snapshot directory")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("show requires a snapshot key")
	}

	key := fs.Arg(0)

	cfg := DefaultConfig()
	cfg.LoadFromEnv()
	if *outputDir != "" {
		cfg.OutputDir = *outputDir
	}

	storage := NewSnapshotStorage(cfg.OutputDir)

	snapshot, err := storage.Load(key)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	fmt.Printf("=== Snapshot: %s ===\n\n", key)
	fmt.Printf("Database: %s\n", snapshot.Database)
	fmt.Printf("Timestamp: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Tables: %d\n\n", len(snapshot.Tables))

	fmt.Println("Tables:")
	for _, table := range snapshot.Tables {
		fmt.Printf("  %s\n", table.Name)
		fmt.Printf("    Columns: %d\n", len(table.Columns))
		fmt.Printf("    Indexes: %d\n", len(table.Indexes))
		fmt.Printf("    Foreign Keys: %d\n", len(table.ForeignKeys))
		fmt.Printf("    Rows: %d\n", table.RowCount)
	}

	return nil
}

func runDriver(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("driver command requires a subcommand (list, install, uninstall, info)")
	}

	subcommand := args[0]

	cfg := DefaultConfig()
	cfg.LoadFromEnv()

	regMgr, err := db.NewRegistryManager(cfg.RegistryURL)
	if err != nil {
		return fmt.Errorf("failed to create registry manager: %w", err)
	}

	switch subcommand {
	case "list":
		return runDriverList(regMgr, args[1:])
	case "install":
		return runDriverInstall(regMgr, args[1:])
	case "uninstall":
		return runDriverUninstall(regMgr, args[1:])
	case "info":
		return runDriverInfo(regMgr, args[1:])
	case "update":
		return runDriverUpdate(regMgr, args[1:])
	default:
		return fmt.Errorf("unknown driver subcommand: %s", subcommand)
	}
}

func runDriverList(regMgr *db.RegistryManager, args []string) error {
	fs := flag.NewFlagSet("driver list", flag.ExitOnError)
	installed := fs.Bool("installed", false, "List only installed drivers")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if *installed {
		drivers, err := regMgr.ListInstalledDrivers()
		if err != nil {
			return err
		}

		if len(drivers) == 0 {
			fmt.Println("No drivers installed")
			return nil
		}

		fmt.Println("Installed drivers:")
		for _, d := range drivers {
			fmt.Printf("  %s  v%s  %s\n", d.Name, d.Version, d.Path)
		}
	} else {
		registry, err := regMgr.FetchRegistry()
		if err != nil {
			return err
		}

		fmt.Println("Available drivers:")
		for name, info := range registry.Drivers {
			installed := ""
			if regMgr.IsDriverInstalled(name) {
				installed = " (installed)"
			}
			fmt.Printf("  %-12s v%-8s %s%s\n", name, info.Version, info.Description, installed)
		}
	}

	return nil
}

func runDriverInstall(regMgr *db.RegistryManager, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("install requires a driver name")
	}

	driverName := args[0]

	if regMgr.IsDriverInstalled(driverName) {
		fmt.Printf("Driver '%s' is already installed\n", driverName)
		return nil
	}

	return regMgr.InstallDriver(driverName)
}

func runDriverUninstall(regMgr *db.RegistryManager, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("uninstall requires a driver name")
	}

	driverName := args[0]
	return regMgr.UninstallDriver(driverName)
}

func runDriverInfo(_ *db.RegistryManager, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("info requires a driver name")
	}

	driverName := args[0]

	fmt.Printf("Driver information for: %s\n", driverName)
	fmt.Println("(Driver info logic not yet implemented)")

	return nil
}

func runDriverUpdate(_ *db.RegistryManager, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("update requires a driver name")
	}

	driverName := args[0]

	fmt.Printf("Updating driver: %s\n", driverName)
	fmt.Println("(Driver update logic not yet implemented)")

	return nil
}


func printUsage() {
	usage := `dbc - Database Comparison Tool

Usage:
  dbc <command> [options]

Commands:
  capture [key]            Capture database snapshot (aliases: save, snapshot)
  compare <key1> <key2>    Compare two snapshots (alias: diff)
  list                     List all snapshots (alias: ls)
  show <key>               Show snapshot details
  driver <subcommand>      Manage database drivers

Driver Subcommands:
  driver list              List available drivers
  driver list --installed  List installed drivers
  driver install <name>    Install a driver
  driver uninstall <name>  Uninstall a driver
  driver info <name>       Show driver information
  driver update <name>     Update a driver

Capture Options:
  --type <type>            Database type (mysql, postgres, sqlserver, sqlite)
  --host <host>            Database host (default: localhost)
  --port <port>            Database port (default: 3306 for mysql)
  --user <user>            Database user (default: root)
  --password <password>    Database password
  --database <name>        Database name (required)
  --output-dir <dir>       Output directory (default: ./db_snapshots)
  --workers <n>            Number of parallel workers (default: 10)
  --verify-data            Verify data with checksums (default: false)
  --verify-counts          Get exact row counts (default: true)

Environment Variables:
  DB_TYPE                  Database type
  DB_HOST                  Database host
  DB_PORT                  Database port
  DB_USER                  Database user
  DB_PASSWORD              Database password
  DB_NAME                  Database name
  DBC_OUTPUT_DIR           Output directory
  DBC_WORKERS              Number of workers
  DBC_AUTO_INSTALL         Auto-install drivers (default: true)

Examples:
  # First time setup - install MySQL driver
  dbc driver install mysql

  # Capture snapshot with auto-generated key
  dbc capture --database mydb

  # Capture snapshot with custom key
  dbc capture baseline --database mydb

  # Compare two snapshots
  dbc compare baseline v1.2.3

  # List available drivers
  dbc driver list

Version: %s
`
	fmt.Printf(usage, version)
}
