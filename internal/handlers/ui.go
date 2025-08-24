package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/config"
	"sitewatch/internal/models"
	"sitewatch/internal/services/ping"
	"sitewatch/internal/services/stats"
)

// UI Handlers

// HandleDashboard - GET / or /dashboard - Main dashboard page
func HandleDashboard(c *fiber.Ctx) error {
	overview := stats.CalculateOverviewData(config.GlobalAppState)
	
	return c.Render("pages/dashboard", fiber.Map{
		"Sites":    config.GlobalAppState.Sites,
		"Overview": overview,
	})
}

// HandleUIOverview - GET /ui/overview - Overview stats fragment
func HandleUIOverview(c *fiber.Ctx) error {
	overview := stats.CalculateOverviewData(config.GlobalAppState)
	return c.Render("fragments/overview", overview)
}

// HandleUISites - GET /ui/sites - Sites grid fragment
func HandleUISites(c *fiber.Ctx) error {
	// Use thread-safe snapshots instead of direct locking
	sites := config.GlobalAppState.GetSitesSnapshot()
	statusMap := config.GlobalAppState.GetSiteStatusSnapshot()
	
	type SiteWithStatus struct {
		models.Site
		Status                 models.SiteStatus     `json:"status"`
		Stats                  models.SiteStatistics `json:"stats"`
		PrimaryLatencyString   string                `json:"primary_latency_string,omitempty"`
		SecondaryLatencyString string                `json:"secondary_latency_string,omitempty"`
	}
	
	var sitesWithStatus []SiteWithStatus
	for _, site := range sites {
		status, exists := statusMap[site.ID]
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
		
		// Calculate extended statistics for this site
		siteStats := stats.CalculateSiteStatistics(config.GlobalAppState, site.ID)
		
		siteWithStatus := SiteWithStatus{
			Site:   site,
			Status: *status,
			Stats:  siteStats,
		}
		
		// Format latency strings
		if status.PrimaryLatency != nil {
			siteWithStatus.PrimaryLatencyString = fmt.Sprintf("%.1f", *status.PrimaryLatency)
		}
		if status.SecondaryLatency != nil {
			siteWithStatus.SecondaryLatencyString = fmt.Sprintf("%.1f", *status.SecondaryLatency)
		}
		
		sitesWithStatus = append(sitesWithStatus, siteWithStatus)
	}
	
	return c.Render("fragments/sites", fiber.Map{
		"Sites": sitesWithStatus,
	})
}

// HandleUIDetails - GET /ui/details/:siteId - Site details modal fragment
func HandleUIDetails(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Find site info using thread-safe method
	siteInfo, exists := config.GlobalAppState.FindSite(siteID)
	if !exists {
		return c.SendString("<p class='text-red-600'>Site not found</p>")
	}
	
	// Get site status using thread-safe method
	status, exists := config.GlobalAppState.GetSiteStatus(siteID)
	if !exists {
		return c.SendString("<p class='text-red-600'>Site status not found</p>")
	}
	
	return c.Render("fragments/details", fiber.Map{
		"Site":   *siteInfo,
		"Status": *status,
	})
}

// HandleUILogs - GET /ui/logs - Logs page with filters
func HandleUILogs(c *fiber.Ctx) error {
	sites := config.GlobalAppState.GetSitesSnapshot()
	return c.Render("pages/logs", fiber.Map{
		"Sites": sites,
	})
}

// HandleUILogsTable - GET /ui/logs-table - Logs table with data
func HandleUILogsTable(c *fiber.Ctx) error {
	// Parse query parameters (same as API)
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
		logs = []models.PingLog{}
	}
	
	return c.Render("fragments/logs-table", fiber.Map{
		"Logs":  logs,
		"Total": len(logs),
		"Filters": fiber.Map{
			"site":    siteID,
			"success": successParam,
			"limit":   limit,
		},
	})
}

// HandleUIChartData - GET /ui/chart-data/:siteId/:chartType/:range - Dynamic chart data for time ranges
func HandleUIChartData(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	chartType := c.Params("chartType")
	timeRange := c.Params("range")
	
	// Validate parameters
	if siteID == "" || chartType == "" || timeRange == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Missing parameters"})
	}
	
	// Generate chart data based on type and range
	chartData := stats.GenerateChartDataForRange(config.GlobalAppState, siteID, chartType, timeRange)
	
	return c.JSON(chartData)
}

// HandleUIEnhancedFragment - GET /ui/enhanced-fragment/:siteId - Enhanced details fragment for dashboard tab
func HandleUIEnhancedFragment(c *fiber.Ctx) error {
	siteID := c.Params("siteId")
	
	// Get site info using thread-safe method
	siteInfo, exists := config.GlobalAppState.FindSite(siteID)
	if !exists {
		return c.Status(404).SendString("Site not found")
	}
	
	// Get site status using thread-safe method
	status, exists := config.GlobalAppState.GetSiteStatus(siteID)
	if !exists {
		status = &models.SiteStatus{
			PrimaryOnline:   false,
			SecondaryOnline: false,
			BothOnline:      false,
		}
	}
	
	// Calculate statistics and chart data
	statistics := stats.CalculateSiteStatistics(config.GlobalAppState, siteID)
	chartData := stats.GenerateChartData(config.GlobalAppState, siteID)
	recentEvents := stats.GetRecentEvents(config.GlobalAppState, siteID, 10)
	
	// Convert chart data to JSON strings for templates
	latencyLabelsJSON, _ := json.Marshal(chartData.LatencyChartLabels)
	latencyPrimaryJSON, _ := json.Marshal(chartData.LatencyChartDataPrimary)
	latencySecondaryJSON, _ := json.Marshal(chartData.LatencyChartDataSecondary)
	uptimeLabelsJSON, _ := json.Marshal(chartData.UptimeChartLabels)
	uptimeDataJSON, _ := json.Marshal(chartData.UptimeChartData)
	uptimePrimaryJSON, _ := json.Marshal(chartData.UptimeChartDataPrimary)
	uptimeSecondaryJSON, _ := json.Marshal(chartData.UptimeChartDataSecondary)
	slaLabelsJSON, _ := json.Marshal(chartData.SLAChartLabels)
	slaPrimaryJSON, _ := json.Marshal(chartData.SLAChartDataPrimary)
	slaSecondaryJSON, _ := json.Marshal(chartData.SLAChartDataSecondary)
	yearlyLabelsJSON, _ := json.Marshal(chartData.YearlyUptimeLabels)
	yearlyDataJSON, _ := json.Marshal(chartData.YearlyUptimeData)
	yearlyPrimaryJSON, _ := json.Marshal(chartData.YearlyUptimeDataPrimary)
	yearlySecondaryJSON, _ := json.Marshal(chartData.YearlyUptimeDataSecondary)
	distributionLabelsJSON, _ := json.Marshal(chartData.DistributionChartLabels)
	distributionDataJSON, _ := json.Marshal(chartData.DistributionChartData)
	distributionPrimaryJSON, _ := json.Marshal(chartData.DistributionPrimaryData)
	distributionSecondaryJSON, _ := json.Marshal(chartData.DistributionSecondaryData)
	
	// Extended ping data JSON
	packetLossLabelsJSON, _ := json.Marshal(chartData.PacketLossChartLabels)
	packetLossPrimaryJSON, _ := json.Marshal(chartData.PacketLossChartDataPrimary)
	packetLossSecondaryJSON, _ := json.Marshal(chartData.PacketLossChartDataSecondary)
	jitterLabelsJSON, _ := json.Marshal(chartData.JitterChartLabels)
	jitterPrimaryJSON, _ := json.Marshal(chartData.JitterChartDataPrimary)
	jitterSecondaryJSON, _ := json.Marshal(chartData.JitterChartDataSecondary)
	latencyMinMaxLabelsJSON, _ := json.Marshal(chartData.LatencyMinMaxChartLabels)
	latencyMinPrimaryJSON, _ := json.Marshal(chartData.LatencyMinChartDataPrimary)
	latencyMaxPrimaryJSON, _ := json.Marshal(chartData.LatencyMaxChartDataPrimary)
	latencyMinSecondaryJSON, _ := json.Marshal(chartData.LatencyMinChartDataSecondary)
	latencyMaxSecondaryJSON, _ := json.Marshal(chartData.LatencyMaxChartDataSecondary)

	return c.Render("components/enhanced-fragment", fiber.Map{
		"Site":         *siteInfo,
		"Status":       *status,
		"Statistics":   statistics,
		"ChartData":    chartData,
		"RecentEvents": recentEvents,
		// SLA Configuration 
		"PrimarySLA":   siteInfo.GetPrimarySLAUptime(),
		"SecondarySLA": siteInfo.GetSecondarySLAUptime(),
		"CombinedSLA":  siteInfo.GetCombinedSLAUptime(),
		// JSON-encoded chart data for JavaScript
		"LatencyChartLabels":        string(latencyLabelsJSON),
		"LatencyChartDataPrimary":   string(latencyPrimaryJSON),
		"LatencyChartDataSecondary": string(latencySecondaryJSON),
		"UptimeChartLabels":         string(uptimeLabelsJSON),
		"UptimeChartData":           string(uptimeDataJSON),
		"UptimeChartPrimaryData":    string(uptimePrimaryJSON),
		"UptimeChartSecondaryData":  string(uptimeSecondaryJSON),
		"SLAChartLabels":            string(slaLabelsJSON),
		"SLAChartPrimaryData":       string(slaPrimaryJSON),
		"SLAChartSecondaryData":     string(slaSecondaryJSON),
		"YearlyUptimeLabels":        string(yearlyLabelsJSON),
		"YearlyUptimeData":          string(yearlyDataJSON),
		"YearlyUptimePrimaryData":   string(yearlyPrimaryJSON),
		"YearlyUptimeSecondaryData": string(yearlySecondaryJSON),
		"DistributionChartLabels":   string(distributionLabelsJSON),
		"DistributionChartData":     string(distributionDataJSON),
		"DistributionPrimaryData":   string(distributionPrimaryJSON),
		"DistributionSecondaryData": string(distributionSecondaryJSON),
		// Extended ping data for templates
		"PacketLossChartLabels":        string(packetLossLabelsJSON),
		"PacketLossChartDataPrimary":   string(packetLossPrimaryJSON),
		"PacketLossChartDataSecondary": string(packetLossSecondaryJSON),
		"JitterChartLabels":            string(jitterLabelsJSON),
		"JitterChartDataPrimary":       string(jitterPrimaryJSON),
		"JitterChartDataSecondary":     string(jitterSecondaryJSON),
		"LatencyMinMaxChartLabels":     string(latencyMinMaxLabelsJSON),
		"LatencyMinChartDataPrimary":   string(latencyMinPrimaryJSON),
		"LatencyMaxChartDataPrimary":   string(latencyMaxPrimaryJSON),
		"LatencyMinChartDataSecondary": string(latencyMinSecondaryJSON),
		"LatencyMaxChartDataSecondary": string(latencyMaxSecondaryJSON),
	})
}