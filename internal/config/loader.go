package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// LoadConfig loads configuration from config.yaml
func (app *AppState) LoadConfig() error {
	// Get config path from environment or use default
	configPath := GetConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	if err := yaml.Unmarshal(data, &app.Config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Set defaults
	if app.Config.Server.Host == "" {
		app.Config.Server.Host = "0.0.0.0"
	}
	if app.Config.Server.Port == 0 {
		app.Config.Server.Port = 8080
	}
	if app.Config.Ping.DefaultInterval == 0 {
		app.Config.Ping.DefaultInterval = 30 * time.Second
	}
	if app.Config.Ping.Timeout == 0 {
		app.Config.Ping.Timeout = 5 * time.Second
	}
	if app.Config.Ping.PacketCount <= 0 {
		app.Config.Ping.PacketCount = 3 // Default to 3 packets for better statistics
	}
	if app.Config.Metrics.Path == "" {
		app.Config.Metrics.Path = "/metrics"
	}
	
	// Storage defaults
	if app.Config.Storage.Type == "" {
		app.Config.Storage.Type = "sqlite"
	}
	if app.Config.Storage.SQLitePath == "" {
		app.Config.Storage.SQLitePath = "data/ping_monitor.db"
	}
	// MaxMemoryLogs removed - only SQLite storage is used now
	
	// Auth defaults
	if app.Config.Auth.UI.SessionName == "" {
		app.Config.Auth.UI.SessionName = "sitewatch_session"
	}
	if app.Config.Auth.UI.ExpiresHours == 0 {
		app.Config.Auth.UI.ExpiresHours = 24
	}
	
	// Apply environment variable overrides
	LoadEnvOverrides(&app.Config)

	return nil
}

// LoadSites loads site configuration from sites.yaml
func (app *AppState) LoadSites() error {
	// Get sites path from environment or use default
	sitesPath := GetSitesPath()
	
	data, err := os.ReadFile(sitesPath)
	if err != nil {
		return fmt.Errorf("reading sites file %s: %w", sitesPath, err)
	}

	var sitesConfig models.SitesConfig
	if err := yaml.Unmarshal(data, &sitesConfig); err != nil {
		return fmt.Errorf("parsing sites config: %w", err)
	}

	// Thread-safe assignment
	app.Mu.Lock()
	app.Sites = sitesConfig.Sites
	app.Mu.Unlock()
	
	log := logger.Default().WithComponent("config")
	log.Info("Sites loaded", "count", len(sitesConfig.Sites), "path", sitesPath)

	return nil
}

// GetSitesSnapshot returns a thread-safe snapshot of sites
func (app *AppState) GetSitesSnapshot() []models.Site {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	// Return a copy to prevent race conditions
	sites := make([]models.Site, len(app.Sites))
	copy(sites, app.Sites)
	return sites
}

// GetSiteStatusSnapshot returns a thread-safe snapshot of site status
func (app *AppState) GetSiteStatusSnapshot() map[string]*models.SiteStatus {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	// Return a deep copy to prevent race conditions
	statusMap := make(map[string]*models.SiteStatus, len(app.SiteStatus))
	for id, status := range app.SiteStatus {
		if status != nil {
			statusCopy := *status // Copy the struct
			statusMap[id] = &statusCopy
		}
	}
	return statusMap
}

// FindSite returns a copy of a site by ID (thread-safe)
func (app *AppState) FindSite(siteID string) (*models.Site, bool) {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	for _, site := range app.Sites {
		if site.ID == siteID {
			siteCopy := site // Copy the struct
			return &siteCopy, true
		}
	}
	return nil, false
}

// GetSiteStatus returns a copy of site status by ID (thread-safe)  
func (app *AppState) GetSiteStatus(siteID string) (*models.SiteStatus, bool) {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	status, exists := app.SiteStatus[siteID]
	if !exists || status == nil {
		return nil, false
	}
	
	statusCopy := *status // Copy the struct
	return &statusCopy, true
}