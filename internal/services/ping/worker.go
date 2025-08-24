package ping

import (
	"context"
	"time"

	"sitewatch/internal/config"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// StartPingWorkers starts ping workers for all enabled sites
func StartPingWorkers(ctx context.Context, appState *config.AppState) {
	log := logger.Default().WithComponent("ping-workers")
	log.Info("Starting ping workers")
	
	// Start result processor
	go ProcessResults(ctx, appState)
	
	// Start ping workers for each site
	enabledCount := 0
	for _, site := range appState.Sites {
		if !site.Enabled {
			log.Debug("Site disabled, skipping", "site_id", site.ID, "site_name", site.Name)
			continue
		}
		
		log.Info("Starting ping worker for site", "site_id", site.ID, "site_name", site.Name)
		go PingWorker(ctx, appState, site)
		enabledCount++
	}
	
	log.Info("All ping workers started", "enabled_sites", enabledCount, "total_sites", len(appState.Sites))
}

// PingWorker handles pinging for a specific site
func PingWorker(ctx context.Context, appState *config.AppState, site models.Site) {
	log := logger.Default().WithSite(site.ID, site.Name)
	
	interval := time.Duration(site.Interval) * time.Second
	if interval == 0 {
		interval = appState.Config.Ping.DefaultInterval
	}
	
	log.Debug("Ping worker initialized", "interval", interval.String())
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Immediate first ping
	PingSite(appState, site)
	
	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping ping worker")
			return
		case <-ticker.C:
			PingSite(appState, site)
		}
	}
}

// ProcessResults processes ping results and updates metrics
func ProcessResults(ctx context.Context, appState *config.AppState) {
	log := logger.Default().WithComponent("result-processor")
	log.Info("Starting result processor")
	
	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping result processor")
			return
		case result := <-appState.ResultChan:
			log.Debug("Processing ping result", "site_id", result.SiteID, "line_type", result.LineType, "success", result.Success)
			HandlePingResult(appState, result)
		}
	}
}