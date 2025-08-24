package handlers

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/config"
)

// HandlePrometheusMetrics - GET /metrics - Prometheus format metrics
func HandlePrometheusMetrics(c *fiber.Ctx) error {
	// Set content type for Prometheus
	c.Set("Content-Type", "text/plain; charset=utf-8")
	
	var metrics strings.Builder
	
	// Write headers
	metrics.WriteString("# HELP ping_checks_total Total number of ping checks performed\n")
	metrics.WriteString("# TYPE ping_checks_total counter\n")
	metrics.WriteString("# HELP ping_latency_seconds Histogram of ping latencies in seconds\n")
	metrics.WriteString("# TYPE ping_latency_seconds histogram\n")
	metrics.WriteString("# HELP site_status Current status of site lines (1=online, 0=offline)\n")
	metrics.WriteString("# TYPE site_status gauge\n")
	metrics.WriteString("# HELP site_both_lines_online Both lines online status for site (1=both online, 0=at least one offline)\n")
	metrics.WriteString("# TYPE site_both_lines_online gauge\n")
	metrics.WriteString("# HELP site_info Site information with labels\n")
	metrics.WriteString("# TYPE site_info gauge\n")
	metrics.WriteString("# HELP site_sla_target SLA uptime targets for site providers\n")
	metrics.WriteString("# TYPE site_sla_target gauge\n")
	
	// Get current state
	config.GlobalAppState.Mu.RLock()
	defer config.GlobalAppState.Mu.RUnlock()
	
	// Export site status metrics
	for _, site := range config.GlobalAppState.Sites {
		status, exists := config.GlobalAppState.SiteStatus[site.ID]
		if !exists {
			continue
		}
		
		// Site info metric
		metrics.WriteString(fmt.Sprintf(
			"site_info{site_id=\"%s\",name=\"%s\",location=\"%s\"} 1\n",
			site.ID, site.Name, site.Location,
		))
		
		// SLA target metrics
		if site.SLA.Primary.Uptime > 0 {
			provider := site.PrimaryProvider
			if provider == "" {
				provider = "Primary"
			}
			metrics.WriteString(fmt.Sprintf(
				"site_sla_target{site_id=\"%s\",line_type=\"primary\",provider=\"%s\"} %.2f\n",
				site.ID, provider, site.GetPrimarySLAUptime(),
			))
		}
		
		if site.IsDualLine() && site.SLA.Secondary.Uptime > 0 {
			provider := site.SecondaryProvider
			if provider == "" {
				provider = "Secondary"
			}
			metrics.WriteString(fmt.Sprintf(
				"site_sla_target{site_id=\"%s\",line_type=\"secondary\",provider=\"%s\"} %.2f\n",
				site.ID, provider, site.GetSecondarySLAUptime(),
			))
		}
		
		if site.IsDualLine() && site.SLA.Combined.Uptime > 0 {
			metrics.WriteString(fmt.Sprintf(
				"site_sla_target{site_id=\"%s\",line_type=\"combined\",provider=\"Combined\"} %.2f\n",
				site.ID, site.GetCombinedSLAUptime(),
			))
		}
		
		// Site status metrics
		primaryOnline := 0
		secondaryOnline := 0
		bothOnline := 0
		
		if status.PrimaryOnline {
			primaryOnline = 1
		}
		if status.SecondaryOnline {
			secondaryOnline = 1
		}
		if status.BothOnline {
			bothOnline = 1
		}
		
		metrics.WriteString(fmt.Sprintf(
			"site_status{site_id=\"%s\",line_type=\"primary\"} %d\n",
			site.ID, primaryOnline,
		))
		metrics.WriteString(fmt.Sprintf(
			"site_status{site_id=\"%s\",line_type=\"secondary\"} %d\n",
			site.ID, secondaryOnline,
		))
		metrics.WriteString(fmt.Sprintf(
			"site_both_lines_online{site_id=\"%s\"} %d\n",
			site.ID, bothOnline,
		))
	}
	
	// Add app stats
	metrics.WriteString("# HELP app_uptime_seconds Application uptime in seconds\n")
	metrics.WriteString("# TYPE app_uptime_seconds gauge\n")
	metrics.WriteString(fmt.Sprintf("app_uptime_seconds %.2f\n", time.Since(config.GlobalAppState.StartTime).Seconds()))
	
	metrics.WriteString("# HELP app_total_checks Total number of ping checks performed\n")
	metrics.WriteString("# TYPE app_total_checks counter\n")
	metrics.WriteString(fmt.Sprintf("app_total_checks %d\n", atomic.LoadInt64(&config.GlobalAppState.TotalChecks)))
	
	metrics.WriteString("# HELP app_total_sites Total number of configured sites\n")
	metrics.WriteString("# TYPE app_total_sites gauge\n")
	metrics.WriteString(fmt.Sprintf("app_total_sites %d\n", len(config.GlobalAppState.Sites)))
	
	// Count active sites
	activeSites := 0
	for _, site := range config.GlobalAppState.Sites {
		if site.Enabled {
			activeSites++
		}
	}
	
	metrics.WriteString("# HELP app_active_sites Number of active sites\n")
	metrics.WriteString("# TYPE app_active_sites gauge\n")
	metrics.WriteString(fmt.Sprintf("app_active_sites %d\n", activeSites))
	
	return c.SendString(metrics.String())
}