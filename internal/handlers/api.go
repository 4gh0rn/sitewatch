package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/config"
	"sitewatch/internal/models"
	"sitewatch/internal/services/ping"
	"sitewatch/internal/services/stats"
)

// API Handlers

// HandleGetSites - GET /api/sites - List all sites with status overview
func HandleGetSites(c *fiber.Ctx) error {
	config.GlobalAppState.Mu.RLock()
	defer config.GlobalAppState.Mu.RUnlock()
	
	type SiteOverview struct {
		models.Site
		Status models.SiteStatus `json:"status"`
	}
	
	var overview []SiteOverview
	for _, site := range config.GlobalAppState.Sites {
		status, exists := config.GlobalAppState.SiteStatus[site.ID]
		if !exists {
			// Default status if not found
			status = &models.SiteStatus{
				SiteID:          site.ID,
				PrimaryOnline:   false,
				SecondaryOnline: false,
				BothOnline:      false,
				LastCheck:       time.Now(),
			}
		}
		
		overview = append(overview, SiteOverview{
			Site:   site,
			Status: *status,
		})
	}
	
	return c.JSON(fiber.Map{
		"sites": overview,
		"total": len(overview),
		"timestamp": time.Now(),
	})
}

// HandleGetSiteStatus - GET /api/sites/{siteId}/status - Serverguard compatible endpoint
// Returns "OK" (HTTP 200) if at least one line is online, "FAILURE" (HTTP 200) if all lines are offline
func HandleGetSiteStatus(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	config.GlobalAppState.Mu.RLock()
	status, exists := config.GlobalAppState.SiteStatus[siteID]
	config.GlobalAppState.Mu.RUnlock()
	
	if !exists {
		return c.Status(200).SendString("FAILURE")
	}
	
	// Site is considered successful if at least one line is online
	if status.PrimaryOnline || status.SecondaryOnline {
		return c.Status(200).SendString("OK")
	}
	
	return c.Status(200).SendString("FAILURE")
}

// HandleGetSiteDetails - GET /api/sites/{siteId}/details - Detailed site information
func HandleGetSiteDetails(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Find site info
	var siteInfo *models.Site
	for _, site := range config.GlobalAppState.Sites {
		if site.ID == siteID {
			siteInfo = &site
			break
		}
	}
	
	if siteInfo == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Site not found",
		})
	}
	
	config.GlobalAppState.Mu.RLock()
	status, exists := config.GlobalAppState.SiteStatus[siteID]
	config.GlobalAppState.Mu.RUnlock()
	
	if !exists {
		return c.Status(404).JSON(fiber.Map{
			"error": "Site status not found",
		})
	}
	
	return c.JSON(fiber.Map{
		"site": siteInfo,
		"status": status,
		"timestamp": time.Now(),
	})
}

// HandleGetLogs - GET /api/logs - Get ping logs with optional filtering
func HandleGetLogs(c *fiber.Ctx) error {
	// Parse query parameters
	siteID := c.Query("site", "")
	successParam := c.Query("success", "")
	limitParam := c.Query("limit", "100")
	
	// Parse success filter
	var success *bool
	if successParam != "" {
		if successParam == "true" {
			val := true
			success = &val
		} else if successParam == "false" {
			val := false
			success = &val
		}
	}
	
	// Parse limit
	limit := 100
	if limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}
	
	// Get filtered logs
	logs, err := ping.GetFilteredLogs(config.GlobalAppState, siteID, success, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get logs",
		})
	}
	
	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": len(logs),
		"filters": fiber.Map{
			"site":    siteID,
			"success": successParam,
			"limit":   limit,
		},
	})
}

// HandleGetSiteStatistics - GET /api/sites/:siteId/statistics - Get extended site statistics
func HandleGetSiteStatistics(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Calculate extended statistics
	statistics := stats.CalculateSiteStatistics(config.GlobalAppState, siteID)
	
	return c.JSON(fiber.Map{
		"site_id":    siteID,
		"statistics": statistics,
		"timestamp":  time.Now(),
	})
}

// HandleGetSiteChartData - GET /api/sites/:siteId/charts - Get comprehensive chart data
func HandleGetSiteChartData(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Generate comprehensive chart data
	chartData := stats.GenerateChartData(config.GlobalAppState, siteID)
	
	return c.JSON(fiber.Map{
		"site_id":    siteID,
		"chart_data": chartData,
		"timestamp":  time.Now(),
	})
}

// HandleSiteTest - POST /api/sites/:siteId/test - Run manual ping test
func HandleSiteTest(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Find the site
	var site *models.Site
	for _, s := range config.GlobalAppState.Sites {
		if s.ID == siteID {
			site = &s
			break
		}
	}
	
	if site == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Site not found",
		})
	}
	
	type TestResult struct {
		IP        string    `json:"ip"`
		Success   bool      `json:"success"`
		Latency   *float64  `json:"latency,omitempty"`
		Error     string    `json:"error,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}
	
	type TestResponse struct {
		Primary   *TestResult `json:"primary,omitempty"`
		Secondary *TestResult `json:"secondary,omitempty"`
	}
	
	response := TestResponse{}
	now := time.Now()
	
	// Test primary IP
	if site.PrimaryIP != "" {
		success, latency, errorMsg := ping.PingIPSync(config.GlobalAppState, site.PrimaryIP)
		result := &TestResult{
			IP:        site.PrimaryIP,
			Success:   success,
			Timestamp: now,
		}
		
		if !success {
			result.Error = errorMsg
		} else if latency != nil {
			result.Latency = latency
		}
		
		response.Primary = result
	}
	
	// Test secondary IP (if exists)
	if site.SecondaryIP != "" {
		success, latency, errorMsg := ping.PingIPSync(config.GlobalAppState, site.SecondaryIP)
		result := &TestResult{
			IP:        site.SecondaryIP,
			Success:   success,
			Timestamp: now,
		}
		
		if !success {
			result.Error = errorMsg
		} else if latency != nil {
			result.Latency = latency
		}
		
		response.Secondary = result
	}
	
	return c.JSON(response)
}

// HandleHealth - GET /api/health - Health check endpoint
func HandleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"timestamp": time.Now(),
		"uptime":    time.Since(config.GlobalAppState.StartTime).Seconds(),
	})
}