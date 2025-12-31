package core

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBType   string
	Host     string
	Port     int
	User     string
	Password string
	Database string

	OutputDir       string
	VerifyData      bool
	VerifyRowCounts bool
	Workers         int

	AutoInstall bool
	RegistryURL string

	Format string
}

func DefaultConfig() *Config {
	return &Config{
		DBType:          "mysql",
		Host:            "localhost",
		Port:            3306,
		User:            "root",
		Password:        "",
		Database:        "",
		OutputDir:       "./db_snapshots",
		VerifyData:      false,
		VerifyRowCounts: true,
		Workers:         10,
		AutoInstall:     true,
		RegistryURL:     "https://raw.githubusercontent.com/ntancardoso/dbc/main/registry/drivers.json",
		Format:          "both",
	}
}

func (c *Config) LoadFromEnv() {
	if val := os.Getenv("DB_TYPE"); val != "" {
		c.DBType = val
	}
	if val := os.Getenv("DB_HOST"); val != "" {
		c.Host = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			c.Port = port
		}
	}
	if val := os.Getenv("DB_USER"); val != "" {
		c.User = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		c.Password = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		c.Database = val
	}
	if val := os.Getenv("DBC_OUTPUT_DIR"); val != "" {
		c.OutputDir = val
	}
	if val := os.Getenv("DBC_VERIFY_DATA"); val != "" {
		c.VerifyData = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("DBC_VERIFY_COUNTS"); val != "" {
		c.VerifyRowCounts = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("DBC_WORKERS"); val != "" {
		if workers, err := strconv.Atoi(val); err == nil && workers > 0 {
			c.Workers = workers
		}
	}
	if val := os.Getenv("DBC_AUTO_INSTALL"); val != "" {
		c.AutoInstall = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("DBC_REGISTRY_URL"); val != "" {
		c.RegistryURL = val
	}
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) GetConnectionString() string {
	switch c.DBType {
	case "mysql":
		// MySQL connection strings don't use URL encoding in the same way
		// The go-sql-driver/mysql expects: user:password@tcp(host:port)/database
		// Passwords with special characters should still be properly escaped
		userInfo := c.User
		if c.Password != "" {
			userInfo += ":" + c.Password
		}
		return fmt.Sprintf("%s@tcp(%s:%d)/%s", userInfo, c.Host, c.Port, c.Database)
	case "postgres":
		// Use url.URL for proper encoding
		u := &url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
			Path:   "/" + c.Database,
		}
		if c.User != "" {
			if c.Password != "" {
				u.User = url.UserPassword(c.User, c.Password)
			} else {
				u.User = url.User(c.User)
			}
		}
		query := url.Values{}
		query.Set("sslmode", "disable")
		u.RawQuery = query.Encode()
		return u.String()
	case "sqlserver":
		// Use url.URL for proper encoding
		u := &url.URL{
			Scheme: "sqlserver",
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		}
		if c.User != "" {
			if c.Password != "" {
				u.User = url.UserPassword(c.User, c.Password)
			} else {
				u.User = url.User(c.User)
			}
		}
		query := url.Values{}
		query.Set("database", c.Database)
		u.RawQuery = query.Encode()
		return u.String()
	case "sqlite":
		return c.Database
	case "oracle":
		// Use url.URL for proper encoding
		u := &url.URL{
			Scheme: "oracle",
			Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
			Path:   "/" + c.Database,
		}
		if c.User != "" {
			if c.Password != "" {
				u.User = url.UserPassword(c.User, c.Password)
			} else {
				u.User = url.User(c.User)
			}
		}
		return u.String()
	default:
		return ""
	}
}
