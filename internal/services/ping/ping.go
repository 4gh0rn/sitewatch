package ping

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-ping/ping"
	"sitewatch/internal/config"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// PingSite pings both IPs of a site
func PingSite(appState *config.AppState, site models.Site) {
	// Ping primary IP
	go PingIP(appState, site.ID, site.PrimaryIP, "primary")
	
	// Ping secondary IP only if site has dual-line configuration
	if site.IsDualLine() {
		go PingIP(appState, site.ID, site.SecondaryIP, "secondary")
	}
}

// PingIP pings a specific IP address
func PingIP(appState *config.AppState, siteID, ip, lineType string) {
	log := logger.Default().WithPing(siteID, ip, lineType)
	
	result := models.PingResult{
		SiteID:    siteID,
		IP:        ip,
		LineType:  lineType,
		Timestamp: time.Now(),
	}
	
	log.Debug("Starting ping operation")
	
	// Get circuit breaker for this site/line combination
	cbManager := GetGlobalCircuitBreakerManager()
	cb := cbManager.GetBreaker(siteID, lineType)
	
	// Execute ping through circuit breaker
	err := cb.Call(func() error {
		return executePing(appState, &result)
	})
	
	if err != nil {
		// Check if it's a circuit breaker error
		if cbErr, ok := err.(*CircuitBreakerError); ok {
			result.Success = false
			result.Error = fmt.Sprintf("circuit breaker open: %s", cbErr.Error())
			log.Warn("Ping blocked by circuit breaker", "error", cbErr.Error())
		} else {
			// Regular ping error was already handled in executePing
			log.Debug("Ping completed with error", "error", err)
		}
	}
	
	// Send result to processor
	appState.ResultChan <- result
}

// executePing performs the actual ping operation
func executePing(appState *config.AppState, result *models.PingResult) error {
	log := logger.Default().WithPing(result.SiteID, result.IP, result.LineType)
	
	// Create pinger
	pinger, err := ping.NewPinger(result.IP)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create pinger: %v", err)
		log.Error("Failed to create pinger", "error", err)
		return err
	}
	
	// Configure pinger
	packetCount := appState.Config.Ping.PacketCount
	if packetCount <= 0 {
		packetCount = 3 // Default to 3 packets for better statistics
	}
	pinger.Count = packetCount
	pinger.Timeout = appState.Config.Ping.Timeout
	pinger.SetPrivileged(false) // Use unprivileged mode
	
	// Set packet size if configured
	if appState.Config.Ping.PacketSize > 0 {
		pinger.Size = appState.Config.Ping.PacketSize
	}
	
	// Run ping
	err = pinger.Run()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("ping failed: %v", err)
		log.Error("Ping execution failed", "error", err)
		return err
	}
	
	stats := pinger.Statistics()
	
	// Always capture packet statistics
	result.PacketsSent = stats.PacketsSent
	result.PacketsRecv = stats.PacketsRecv
	result.PacketsDuplicates = stats.PacketsRecvDuplicates
	
	// Calculate packet loss percentage
	if stats.PacketsSent > 0 {
		packetLoss := stats.PacketLoss
		result.PacketLoss = &packetLoss
	}
	
	if stats.PacketsRecv > 0 {
		result.Success = true
		
		// Average latency (existing)
		latencyMs := float64(stats.AvgRtt.Nanoseconds()) / 1000000.0
		result.Latency = &latencyMs
		
		// Extended latency statistics
		minLatencyMs := float64(stats.MinRtt.Nanoseconds()) / 1000000.0
		maxLatencyMs := float64(stats.MaxRtt.Nanoseconds()) / 1000000.0
		jitterMs := float64(stats.StdDevRtt.Nanoseconds()) / 1000000.0
		
		result.MinLatency = &minLatencyMs
		result.MaxLatency = &maxLatencyMs
		result.Jitter = &jitterMs
		
		log.Debug("Ping successful", 
			"latency_avg_ms", latencyMs,
			"latency_min_ms", minLatencyMs,
			"latency_max_ms", maxLatencyMs,
			"jitter_ms", jitterMs,
			"packets_sent", stats.PacketsSent,
			"packets_recv", stats.PacketsRecv,
			"packet_loss_pct", stats.PacketLoss,
			"duplicates", stats.PacketsRecvDuplicates)
	} else {
		result.Success = false
		result.Error = "no packets received"
		log.Warn("Ping failed - no packets received", 
			"packets_sent", stats.PacketsSent,
			"packet_loss_pct", stats.PacketLoss)
		return fmt.Errorf("no packets received")
	}
	
	return nil
}

// PingIPSync performs a synchronous ping for testing purposes
func PingIPSync(appState *config.AppState, ip string) (success bool, latency *float64, errorMsg string) {
	// Create pinger
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return false, nil, fmt.Sprintf("failed to create pinger: %v", err)
	}
	
	// Configure pinger
	packetCount := appState.Config.Ping.PacketCount
	if packetCount <= 0 {
		packetCount = 3 // Default to 3 packets for better statistics
	}
	pinger.Count = packetCount
	pinger.Timeout = appState.Config.Ping.Timeout
	pinger.SetPrivileged(false) // Use unprivileged mode
	
	// Set packet size if configured
	if appState.Config.Ping.PacketSize > 0 {
		pinger.Size = appState.Config.Ping.PacketSize
	}
	
	// Run ping
	err = pinger.Run()
	if err != nil {
		return false, nil, fmt.Sprintf("ping failed: %v", err)
	}
	
	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		latencyMs := float64(stats.AvgRtt.Nanoseconds()) / 1000000.0 // Convert to milliseconds
		return true, &latencyMs, ""
	} else {
		return false, nil, "no packets received"
	}
}

// HandlePingResult handles a single ping result
func HandlePingResult(appState *config.AppState, result models.PingResult) {
	atomic.AddInt64(&appState.TotalChecks, 1)
	
	// Update Prometheus metrics
	successLabel := "false"
	if result.Success {
		successLabel = "true"
	}
	
	config.PingChecksTotal.WithLabelValues(result.SiteID, result.LineType, successLabel).Inc()
	
	// Update extended packet metrics
	config.PacketsSentCounter.WithLabelValues(result.SiteID, result.LineType).Add(float64(result.PacketsSent))
	config.PacketsReceivedCounter.WithLabelValues(result.SiteID, result.LineType).Add(float64(result.PacketsRecv))
	config.PacketsDuplicatesCounter.WithLabelValues(result.SiteID, result.LineType).Add(float64(result.PacketsDuplicates))
	
	// Update packet loss gauge
	if result.PacketLoss != nil {
		config.PacketLossGauge.WithLabelValues(result.SiteID, result.LineType).Set(*result.PacketLoss)
	}
	
	if result.Success {
		latencySeconds := *result.Latency / 1000.0 // Convert ms to seconds
		config.PingLatencyHistogram.WithLabelValues(result.SiteID, result.LineType).Observe(latencySeconds)
		config.SiteStatusGauge.WithLabelValues(result.SiteID, result.LineType).Set(1)
		
		// Update jitter histogram
		if result.Jitter != nil {
			jitterSeconds := *result.Jitter / 1000.0 // Convert ms to seconds
			config.JitterHistogram.WithLabelValues(result.SiteID, result.LineType).Observe(jitterSeconds)
		}
	} else {
		config.SiteStatusGauge.WithLabelValues(result.SiteID, result.LineType).Set(0)
	}
	
	// Add to ping logs
	var siteName string
	for _, site := range appState.Sites {
		if site.ID == result.SiteID {
			siteName = site.Name
			break
		}
	}
	
	AddPingLogToStorage(appState, result, siteName)
	
	// Update site status in memory
	UpdateSiteStatus(appState, result)
}

// AddPingLogToStorage adds a ping log entry to the configured storage backend
func AddPingLogToStorage(appState *config.AppState, result models.PingResult, siteName string) {
	log := logger.Default().WithComponent("storage").WithSite(result.SiteID, siteName)
	
	logEntry := models.PingLog{
		Timestamp: result.Timestamp,
		SiteID:    result.SiteID,
		SiteName:  siteName,
		Target:    result.LineType,
		IP:        result.IP,
		Success:   result.Success,
		Latency:   result.Latency,
		Error:     result.Error,
		
		// Extended statistics from PingResult
		PacketsSent:      result.PacketsSent,
		PacketsRecv:      result.PacketsRecv,
		PacketsDuplicates: result.PacketsDuplicates,
		PacketLoss:       result.PacketLoss,
		MinLatency:       result.MinLatency,
		MaxLatency:       result.MaxLatency,
		Jitter:           result.Jitter,
	}
	
	// Add to storage backend
	if err := appState.Storage.AddPingLog(logEntry); err != nil {
		log.Error("Failed to add ping log to storage", "error", err, "target", result.LineType, "ip", result.IP)
		// Fallback to in-memory logging - this functionality would need to be
		// implemented in the storage backends if needed
	} else {
		log.Debug("Ping log stored successfully", "target", result.LineType, "ip", result.IP, "success", result.Success)
	}
}

// GetFilteredLogs returns filtered ping logs from storage
func GetFilteredLogs(appState *config.AppState, siteID string, success *bool, limit int) ([]models.PingLog, error) {
	log := logger.Default().WithComponent("storage").WithSite(siteID, "")
	
	// Get logs from storage backend
	logs, err := appState.Storage.GetFilteredLogs(siteID, success, limit)
	if err != nil {
		log.Error("Failed to get logs from storage", "error", err, "limit", limit, "success_filter", success)
		return nil, err
	}
	
	log.Debug("Retrieved filtered logs", "count", len(logs), "limit", limit, "success_filter", success)
	return logs, nil
}

// UpdateSiteStatus updates site status in memory
func UpdateSiteStatus(appState *config.AppState, result models.PingResult) {
	appState.Mu.Lock()
	defer appState.Mu.Unlock()
	
	status, exists := appState.SiteStatus[result.SiteID]
	if !exists {
		return
	}
	
	// Update based on line type
	switch result.LineType {
	case "primary":
		status.PrimaryOnline = result.Success
		if result.Success {
			status.PrimaryLatency = result.Latency
			status.PrimaryError = ""
		} else {
			status.PrimaryLatency = nil
			status.PrimaryError = result.Error
		}
	case "secondary":
		status.SecondaryOnline = result.Success
		if result.Success {
			status.SecondaryLatency = result.Latency
			status.SecondaryError = ""
		} else {
			status.SecondaryLatency = nil
			status.SecondaryError = result.Error
		}
	}
	
	// Update combined status - depends on site configuration
	var site *models.Site
	for _, s := range appState.Sites {
		if s.ID == result.SiteID {
			site = &s
			break
		}
	}
	
	if site != nil {
		if site.IsDualLine() {
			// Dual-line: both must be online
			status.BothOnline = status.PrimaryOnline && status.SecondaryOnline
		} else {
			// Single-line: only primary needs to be online
			status.BothOnline = status.PrimaryOnline
		}
	}
	
	status.LastCheck = result.Timestamp
	
	// Update Prometheus gauge for combined status
	bothOnlineValue := float64(0)
	if status.BothOnline {
		bothOnlineValue = 1
	}
	config.SiteBothOnlineGauge.WithLabelValues(result.SiteID).Set(bothOnlineValue)
}