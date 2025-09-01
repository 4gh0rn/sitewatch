package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// LoadEnvOverrides applies environment variable overrides to the configuration
func LoadEnvOverrides(cfg *models.Config) {
	log := logger.Default().WithComponent("config-env")
	
	// Server configuration
	if v := os.Getenv("SITEWATCH_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
		log.Info("Environment override applied", "setting", "Server.Host", "value", v)
	}
	if v := os.Getenv("SITEWATCH_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
			log.Info("Environment override applied", "setting", "Server.Port", "value", port)
		}
	}
	if v := os.Getenv("SITEWATCH_SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
			log.Info("Environment override applied", "setting", "Server.ReadTimeout", "value", d.String())
		}
	}
	if v := os.Getenv("SITEWATCH_SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
			log.Info("Environment override applied", "setting", "Server.WriteTimeout", "value", d.String())
		}
	}

	// Ping configuration
	if v := os.Getenv("SITEWATCH_PING_DEFAULT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Ping.DefaultInterval = d
			log.Info("Environment override applied", "setting", "Ping.DefaultInterval", "value", d.String())
		}
	}
	if v := os.Getenv("SITEWATCH_PING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Ping.Timeout = d
			log.Info("Environment override applied", "setting", "Ping.Timeout", "value", d.String())
		}
	}
	if v := os.Getenv("SITEWATCH_PING_PACKET_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			cfg.Ping.PacketSize = size
			log.Info("Environment override applied", "setting", "Ping.PacketSize", "value", size)
		}
	}
	if v := os.Getenv("SITEWATCH_PING_PACKET_COUNT"); v != "" {
		if count, err := strconv.Atoi(v); err == nil {
			cfg.Ping.PacketCount = count
			log.Info("Environment override applied", "setting", "Ping.PacketCount", "value", count)
		}
	}

	// Metrics configuration
	if v := os.Getenv("SITEWATCH_METRICS_ENABLED"); v != "" {
		cfg.Metrics.Enabled = parseBool(v)
		log.Info("Environment override applied", "setting", "Metrics.Enabled", "value", cfg.Metrics.Enabled)
	}
	if v := os.Getenv("SITEWATCH_METRICS_PATH"); v != "" {
		cfg.Metrics.Path = v
		log.Info("Environment override applied", "setting", "Metrics.Path", "value", v)
	}

	// Storage configuration
	if v := os.Getenv("SITEWATCH_STORAGE_TYPE"); v != "" {
		cfg.Storage.Type = v
		log.Info("Environment override applied", "setting", "Storage.Type", "value", v)
	}
	if v := os.Getenv("SITEWATCH_STORAGE_SQLITE_PATH"); v != "" {
		cfg.Storage.SQLitePath = v
		log.Info("Environment override applied", "setting", "Storage.SQLitePath", "value", v)
	}
	// MaxMemoryLogs removed - only SQLite storage is used now

	// Authentication configuration
	if v := os.Getenv("SITEWATCH_AUTH_ENABLED"); v != "" {
		cfg.Auth.Enabled = parseBool(v)
		log.Info("Environment override applied", "setting", "Auth.Enabled", "value", cfg.Auth.Enabled)
	}
	
	// UI Auth configuration
	if v := os.Getenv("SITEWATCH_AUTH_UI_SECRET"); v != "" {
		cfg.Auth.UI.Secret = v
		log.Info("Environment override applied", "setting", "Auth.UI.Secret", "value", "[REDACTED]")
	}
	if v := os.Getenv("SITEWATCH_AUTH_UI_SESSION_NAME"); v != "" {
		cfg.Auth.UI.SessionName = v
		log.Info("Environment override applied", "setting", "Auth.UI.SessionName", "value", v)
	}
	if v := os.Getenv("SITEWATCH_AUTH_UI_EXPIRES_HOURS"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil {
			cfg.Auth.UI.ExpiresHours = hours
			log.Info("Environment override applied", "setting", "Auth.UI.ExpiresHours", "value", hours)
		}
	}

	// API Tokens from environment
	// Format 1: JSON array
	if v := os.Getenv("SITEWATCH_AUTH_API_TOKENS"); v != "" {
		var tokens []models.APIToken
		if err := json.Unmarshal([]byte(v), &tokens); err == nil {
			cfg.Auth.API.Tokens = tokens
			log.Info("Environment override applied", "setting", "Auth.API.Tokens", "count", len(tokens), "source", "JSON")
		} else {
			// Format 2: Simple comma-separated tokens with default permissions
			tokenStrings := strings.Split(v, ",")
			tokens = []models.APIToken{}
			for i, token := range tokenStrings {
				token = strings.TrimSpace(token)
				if token != "" {
					tokens = append(tokens, models.APIToken{
						Token:       token,
						Name:        "ENV Token " + strconv.Itoa(i+1),
						Permissions: []string{"read"}, // Default permission
					})
				}
			}
			if len(tokens) > 0 {
				cfg.Auth.API.Tokens = tokens
				log.Info("Environment override applied", "setting", "Auth.API.Tokens", "count", len(tokens), "source", "comma-separated")
			}
		}
	}
	
	// Individual token support for simple deployments
	if v := os.Getenv("SITEWATCH_AUTH_API_TOKEN"); v != "" {
		// Single token with optional permissions
		permissions := []string{"read"} // default
		if p := os.Getenv("SITEWATCH_AUTH_API_TOKEN_PERMISSIONS"); p != "" {
			permissions = strings.Split(p, ",")
			for i := range permissions {
				permissions[i] = strings.TrimSpace(permissions[i])
			}
		}
		
		cfg.Auth.API.Tokens = []models.APIToken{
			{
				Token:       v,
				Name:        "ENV Token",
				Permissions: permissions,
			},
		}
		log.Info("Environment override applied", "setting", "Auth.API.Token", "permissions", permissions)
	}
}

// GetConfigPath returns the config file path from env or default
func GetConfigPath() string {
	if path := os.Getenv("SITEWATCH_CONFIG_PATH"); path != "" {
		return path
	}
	return "configs/config.yaml"
}

// GetSitesPath returns the sites file path from env or default
func GetSitesPath() string {
	if path := os.Getenv("SITEWATCH_SITES_PATH"); path != "" {
		return path
	}
	return "configs/sites.yaml"
}

// parseBool parses various boolean representations
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}