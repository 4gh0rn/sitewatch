package stats

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/config"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// Constants for better maintainability
const (
	HoursPerDay     = 24
	DaysPerWeek     = 7
	MonthsPerYear   = 12
	
	LatencyPrecision = 2
	UptimePrecision  = 2
	
	DefaultChartDataPoints = 24
	MaxChartDataPoints     = 100
	
	// Latency distribution buckets in milliseconds
	LatencyBucket1  = 10
	LatencyBucket2  = 50
	LatencyBucket3  = 100
	LatencyBucket4  = 200
	LatencyBucket5  = 500
)

// roundToDecimalPlaces rounds a value to specified decimal places
func roundToDecimalPlaces(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}

// validateLogData validates ping log data for consistency
func validateLogData(pingLog models.PingLog) error {
	if pingLog.SiteID == "" {
		return fmt.Errorf("empty site ID")
	}
	if pingLog.Success && pingLog.Latency == nil {
		// This might be acceptable for some configurations, just log a warning
		log := logger.Default().WithComponent("stats").WithSite(pingLog.SiteID, "")
		log.Warn("Successful ping without latency data")
	}
	if pingLog.Latency != nil && *pingLog.Latency < 0 {
		return fmt.Errorf("negative latency: %f for site %s", *pingLog.Latency, pingLog.SiteID)
	}
	return nil
}

// TimeframeStats holds statistics for a specific timeframe
type TimeframeStats struct {
	TotalChecks     int
	SuccessChecks   int
	PrimaryTotal    int
	PrimarySuccess  int
	SecondaryTotal  int
	SecondarySuccess int
	
	// Latency statistics
	Latencies       []float64
	MinLatency      float64
	MaxLatency      float64
	SumLatency      float64
	JitterValues    []float64    // Jitter (standard deviation) values
	
	// Provider-specific extended statistics
	PrimaryMinLatencies []float64
	PrimaryMaxLatencies []float64 
	PrimaryJitterValues []float64
	SecondaryMinLatencies []float64
	SecondaryMaxLatencies []float64
	SecondaryJitterValues []float64
	
	// Packet statistics  
	TotalPacketsSent      int
	TotalPacketsReceived  int
	TotalPacketsDuplicates int
	PacketLossValues      []float64
	
	// Provider-specific packet stats
	PrimaryPacketsSent       int
	PrimaryPacketsReceived   int
	PrimaryPacketsDuplicates int
	PrimaryPacketLossValues  []float64
	SecondaryPacketsSent     int
	SecondaryPacketsReceived int
	SecondaryPacketsDuplicates int
	SecondaryPacketLossValues []float64
}

// NewTimeframeStats creates a new TimeframeStats instance
func NewTimeframeStats() *TimeframeStats {
	return &TimeframeStats{
		MinLatency: math.MaxFloat64,
	}
}

// AddLog processes a log entry for this timeframe
func (ts *TimeframeStats) AddLog(log models.PingLog) {
	ts.TotalChecks++
	
	// Packet statistics (always collected)
	ts.TotalPacketsSent += log.PacketsSent
	ts.TotalPacketsReceived += log.PacketsRecv
	ts.TotalPacketsDuplicates += log.PacketsDuplicates
	if log.PacketLoss != nil {
		ts.PacketLossValues = append(ts.PacketLossValues, *log.PacketLoss)
	}
	
	if log.Success {
		ts.SuccessChecks++
		
		// Add latency data if available
		if log.Latency != nil {
			latency := *log.Latency
			ts.Latencies = append(ts.Latencies, latency)
			ts.SumLatency += latency
			
			if latency < ts.MinLatency {
				ts.MinLatency = latency
			}
			if latency > ts.MaxLatency {
				ts.MaxLatency = latency
			}
		}
		
		// Extended latency statistics
		if log.Jitter != nil {
			ts.JitterValues = append(ts.JitterValues, *log.Jitter)
		}
	}
	
	// Provider-specific stats
	if log.Target == "primary" {
		ts.PrimaryTotal++
		ts.PrimaryPacketsSent += log.PacketsSent
		ts.PrimaryPacketsReceived += log.PacketsRecv
		ts.PrimaryPacketsDuplicates += log.PacketsDuplicates
		if log.PacketLoss != nil {
			ts.PrimaryPacketLossValues = append(ts.PrimaryPacketLossValues, *log.PacketLoss)
		}
		
		if log.Success {
			ts.PrimarySuccess++
			// Provider-specific extended latency stats
			if log.MinLatency != nil {
				ts.PrimaryMinLatencies = append(ts.PrimaryMinLatencies, *log.MinLatency)
			}
			if log.MaxLatency != nil {
				ts.PrimaryMaxLatencies = append(ts.PrimaryMaxLatencies, *log.MaxLatency)
			}
			if log.Jitter != nil {
				ts.PrimaryJitterValues = append(ts.PrimaryJitterValues, *log.Jitter)
			}
		}
	} else if log.Target == "secondary" {
		ts.SecondaryTotal++
		ts.SecondaryPacketsSent += log.PacketsSent
		ts.SecondaryPacketsReceived += log.PacketsRecv
		ts.SecondaryPacketsDuplicates += log.PacketsDuplicates
		if log.PacketLoss != nil {
			ts.SecondaryPacketLossValues = append(ts.SecondaryPacketLossValues, *log.PacketLoss)
		}
		
		if log.Success {
			ts.SecondarySuccess++
			// Provider-specific extended latency stats
			if log.MinLatency != nil {
				ts.SecondaryMinLatencies = append(ts.SecondaryMinLatencies, *log.MinLatency)
			}
			if log.MaxLatency != nil {
				ts.SecondaryMaxLatencies = append(ts.SecondaryMaxLatencies, *log.MaxLatency)
			}
			if log.Jitter != nil {
				ts.SecondaryJitterValues = append(ts.SecondaryJitterValues, *log.Jitter)
			}
		}
	}
}

// GetUptimePercentage calculates uptime percentage for this timeframe
func (ts *TimeframeStats) GetUptimePercentage() float64 {
	if ts.TotalChecks == 0 {
		return 0
	}
	return roundToDecimalPlaces(float64(ts.SuccessChecks)/float64(ts.TotalChecks)*100, UptimePrecision)
}

// GetMeanLatency calculates mean latency for this timeframe
func (ts *TimeframeStats) GetMeanLatency() float64 {
	if len(ts.Latencies) == 0 {
		return 0
	}
	return roundToDecimalPlaces(ts.SumLatency/float64(len(ts.Latencies)), LatencyPrecision)
}

// GetProviderUptime calculates uptime percentage for a specific provider
func (ts *TimeframeStats) GetProviderUptime(provider string) float64 {
	switch provider {
	case "primary":
		if ts.PrimaryTotal == 0 {
			return 0
		}
		return roundToDecimalPlaces(float64(ts.PrimarySuccess)/float64(ts.PrimaryTotal)*100, UptimePrecision)
	case "secondary":
		if ts.SecondaryTotal == 0 {
			return 0
		}
		return roundToDecimalPlaces(float64(ts.SecondarySuccess)/float64(ts.SecondaryTotal)*100, UptimePrecision)
	default:
		return 0
	}
}

// GetProviderMeanLatency calculates mean latency for a specific provider
func (ts *TimeframeStats) GetProviderMeanLatency(provider string, allLogs []models.PingLog, siteID string) float64 {
	var sum float64
	var count int
	
	for _, log := range allLogs {
		if log.SiteID != siteID || !log.Success || log.Latency == nil || log.Target != provider {
			continue
		}
		sum += *log.Latency
		count++
	}
	
	if count == 0 {
		return 0
	}
	return roundToDecimalPlaces(sum/float64(count), LatencyPrecision)
}

// GetMeanJitter calculates mean jitter across all measurements
func (ts *TimeframeStats) GetMeanJitter() float64 {
	if len(ts.JitterValues) == 0 {
		return 0
	}
	sum := 0.0
	for _, jitter := range ts.JitterValues {
		sum += jitter
	}
	return roundToDecimalPlaces(sum/float64(len(ts.JitterValues)), LatencyPrecision)
}

// GetMeanPacketLoss calculates mean packet loss percentage
func (ts *TimeframeStats) GetMeanPacketLoss() float64 {
	if len(ts.PacketLossValues) == 0 {
		return 0
	}
	sum := 0.0
	for _, loss := range ts.PacketLossValues {
		sum += loss
	}
	return roundToDecimalPlaces(sum/float64(len(ts.PacketLossValues)), UptimePrecision)
}

// GetProviderMeanJitter calculates mean jitter for a specific provider
func (ts *TimeframeStats) GetProviderMeanJitter(provider string) float64 {
	var values []float64
	switch provider {
	case "primary":
		values = ts.PrimaryJitterValues
	case "secondary":
		values = ts.SecondaryJitterValues
	default:
		return 0
	}
	
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, jitter := range values {
		sum += jitter
	}
	return roundToDecimalPlaces(sum/float64(len(values)), LatencyPrecision)
}

// GetProviderMeanPacketLoss calculates mean packet loss for a specific provider
func (ts *TimeframeStats) GetProviderMeanPacketLoss(provider string) float64 {
	var values []float64
	switch provider {
	case "primary":
		values = ts.PrimaryPacketLossValues
	case "secondary":
		values = ts.SecondaryPacketLossValues
	default:
		return 0
	}
	
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, loss := range values {
		sum += loss
	}
	return roundToDecimalPlaces(sum/float64(len(values)), UptimePrecision)
}

// GetProviderMinLatency calculates minimum latency for a specific provider
func (ts *TimeframeStats) GetProviderMinLatency(provider string) float64 {
	var values []float64
	switch provider {
	case "primary":
		values = ts.PrimaryMinLatencies
	case "secondary":
		values = ts.SecondaryMinLatencies
	default:
		return 0
	}
	
	if len(values) == 0 {
		return 0
	}
	
	min := values[0]
	for _, latency := range values[1:] {
		if latency < min {
			min = latency
		}
	}
	return roundToDecimalPlaces(min, LatencyPrecision)
}

// GetProviderMaxLatency calculates maximum latency for a specific provider
func (ts *TimeframeStats) GetProviderMaxLatency(provider string) float64 {
	var values []float64
	switch provider {
	case "primary":
		values = ts.PrimaryMaxLatencies
	case "secondary":
		values = ts.SecondaryMaxLatencies
	default:
		return 0
	}
	
	if len(values) == 0 {
		return 0
	}
	
	max := values[0]
	for _, latency := range values[1:] {
		if latency > max {
			max = latency
		}
	}
	return roundToDecimalPlaces(max, LatencyPrecision)
}

// GetLatencyDistribution calculates latency distribution in predefined buckets
func (ts *TimeframeStats) GetLatencyDistribution() []float64 {
	distribution := make([]float64, 6) // 6 buckets: 0-10, 10-50, 50-100, 100-200, 200-500, 500+
	
	for _, latency := range ts.Latencies {
		var bucketIndex int
		if latency <= LatencyBucket1 {
			bucketIndex = 0
		} else if latency <= LatencyBucket2 {
			bucketIndex = 1
		} else if latency <= LatencyBucket3 {
			bucketIndex = 2
		} else if latency <= LatencyBucket4 {
			bucketIndex = 3
		} else if latency <= LatencyBucket5 {
			bucketIndex = 4
		} else {
			bucketIndex = 5
		}
		distribution[bucketIndex]++
	}
	
	return distribution
}

// GetAllLogs returns all ping logs from storage
func GetAllLogs(app *config.AppState) []models.PingLog {
	if storageImpl, ok := app.Storage.(interface{ GetAllLogs() ([]models.PingLog, error) }); ok {
		logs, err := storageImpl.GetAllLogs()
		if err != nil {
			log := logger.Default().WithComponent("stats-storage")
			log.Error("Failed to get all logs from storage", "error", err)
			return []models.PingLog{}
		}
		return logs
	}
	return []models.PingLog{}
}

// CalculateSiteStatistics calculates comprehensive statistics for a site
func CalculateSiteStatistics(app *config.AppState, siteID string) models.SiteStatistics {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	// Use UTC time to avoid timezone issues
	now := time.Now().UTC()
	day24h := now.Add(-HoursPerDay * time.Hour)
	day7d := now.Add(-DaysPerWeek * HoursPerDay * time.Hour)
	month12 := now.AddDate(-1, 0, 0) // 12 months ago
	
	// Initialize timeframe statistics
	stats := map[string]*TimeframeStats{
		"all": NewTimeframeStats(),
		"24h": NewTimeframeStats(),
		"7d":  NewTimeframeStats(),
		"12m": NewTimeframeStats(),
	}
	
	var lastIncidentTime time.Time
	var lastIncidentDuration string
	
	// Get all logs from storage
	allLogs := GetAllLogs(app)
	
	// Analyze ping logs in a single pass
	for _, pingLog := range allLogs {
		if pingLog.SiteID != siteID {
			continue
		}
		
		// Validate log data
		if err := validateLogData(pingLog); err != nil {
			log := logger.Default().WithComponent("stats").WithSite(siteID, "")
			log.Warn("Invalid log data, skipping", "error", err)
			continue
		}
		
		// Process for all timeframes
		stats["all"].AddLog(pingLog)
		
		// Track failures for incident detection
		if !pingLog.Success && pingLog.Timestamp.After(lastIncidentTime) {
			lastIncidentTime = pingLog.Timestamp
		}
		
		// Check timeframes and add to appropriate stats
		logTime := pingLog.Timestamp
		if logTime.After(day24h) {
			stats["24h"].AddLog(pingLog)
		}
		if logTime.After(day7d) {
			stats["7d"].AddLog(pingLog)
		}
		if logTime.After(month12) {
			stats["12m"].AddLog(pingLog)
		}
	}
	
	// Get all stats for convenience
	allStats := stats["all"]
	stats24h := stats["24h"]
	stats7d := stats["7d"]
	stats12m := stats["12m"]
	
	// Calculate latency statistics
	var avgLatency, minLatencyResult, maxLatencyResult float64
	
	if len(allStats.Latencies) > 0 {
		avgLatency = allStats.GetMeanLatency()
		minLatencyResult = roundToDecimalPlaces(allStats.MinLatency, LatencyPrecision)
		maxLatencyResult = roundToDecimalPlaces(allStats.MaxLatency, LatencyPrecision)
	} else {
		minLatencyResult = 0
		maxLatencyResult = 0
	}
	
	// Calculate success rate (FIXED: now uses actual successful checks)
	var successRate float64
	if allStats.TotalChecks > 0 {
		successRate = roundToDecimalPlaces(float64(allStats.SuccessChecks)/float64(allStats.TotalChecks)*100, UptimePrecision)
	}
	
	// Format last incident
	var lastIncident string
	if !lastIncidentTime.IsZero() {
		diff := now.Sub(lastIncidentTime)
		if diff < time.Hour {
			lastIncident = fmt.Sprintf("%dm ago", int(diff.Minutes()))
		} else if diff < HoursPerDay*time.Hour {
			lastIncident = fmt.Sprintf("%dh ago", int(diff.Hours()))
		} else {
			lastIncident = fmt.Sprintf("%dd ago", int(diff.Hours()/HoursPerDay))
		}
		// TODO: Implement proper incident duration tracking
		lastIncidentDuration = "~5min" 
	} else {
		lastIncident = "None"
		lastIncidentDuration = "N/A"
	}
	
	// Determine current latencies (from recent status)
	var currentLatencyPrimary, currentLatencySecondary *float64
	if status, exists := app.SiteStatus[siteID]; exists {
		currentLatencyPrimary = status.PrimaryLatency
		currentLatencySecondary = status.SecondaryLatency
	}
	
	// Calculate provider-specific mean latencies
	meanLatencyPrimary := stats["all"].GetProviderMeanLatency("primary", allLogs, siteID)
	meanLatencySecondary := stats["all"].GetProviderMeanLatency("secondary", allLogs, siteID)
	
	return models.SiteStatistics{
		// Current latencies
		CurrentLatencyPrimary:    currentLatencyPrimary,
		CurrentLatencySecondary:  currentLatencySecondary,
		MeanLatencyPrimary:       meanLatencyPrimary,
		MeanLatencySecondary:     meanLatencySecondary,
		
		// Extended latency statistics
		MinLatencyPrimary:        allStats.GetProviderMinLatency("primary"),
		MinLatencySecondary:      allStats.GetProviderMinLatency("secondary"),
		MaxLatencyPrimary:        allStats.GetProviderMaxLatency("primary"),
		MaxLatencySecondary:      allStats.GetProviderMaxLatency("secondary"),
		JitterPrimary:            allStats.GetProviderMeanJitter("primary"),
		JitterSecondary:          allStats.GetProviderMeanJitter("secondary"),
		
		// Packet statistics (using extended packet data)
		PacketsReceivedPrimary:   allStats.PrimaryPacketsReceived,
		PacketsReceivedSecondary: allStats.SecondaryPacketsReceived,
		TotalPacketsPrimary:      allStats.PrimaryPacketsSent,
		TotalPacketsSecondary:    allStats.SecondaryPacketsSent,
		PacketLossPrimary:        allStats.GetProviderMeanPacketLoss("primary"),
		PacketLossSecondary:      allStats.GetProviderMeanPacketLoss("secondary"),
		DuplicatePacketsPrimary:  allStats.PrimaryPacketsDuplicates,
		DuplicatePacketsSecondary: allStats.SecondaryPacketsDuplicates,
		
		// Uptime statistics by timeframe
		Uptime24h:                stats24h.GetUptimePercentage(),
		Uptime7d:                 stats7d.GetUptimePercentage(),
		Uptime12m:                stats12m.GetUptimePercentage(),
		
		// Provider-specific uptime (24h)
		UptimePrimary:            stats24h.GetProviderUptime("primary"),
		UptimeSecondary:          stats24h.GetProviderUptime("secondary"),
		PrimaryUptime24h:         stats24h.GetProviderUptime("primary"),
		SecondaryUptime24h:       stats24h.GetProviderUptime("secondary"),
		
		// Provider-specific uptime (7d)
		PrimaryUptime7d:          stats7d.GetProviderUptime("primary"),
		SecondaryUptime7d:        stats7d.GetProviderUptime("secondary"),
		
		// Provider-specific uptime (12m)
		PrimaryUptime12m:         stats12m.GetProviderUptime("primary"),
		SecondaryUptime12m:       stats12m.GetProviderUptime("secondary"),
		
		// Performance statistics
		AvgLatency:               avgLatency,
		MinLatency:               minLatencyResult,
		MaxLatency:               maxLatencyResult,
		SuccessRate:              successRate,
		TotalChecks:              allStats.TotalChecks,
		
		// Incident tracking
		LastIncident:             lastIncident,
		LastIncidentDuration:     lastIncidentDuration,
	}
}

// GenerateChartData generates chart data for a site with improved structure and error handling
func GenerateChartData(app *config.AppState, siteID string) models.ChartData {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	now := time.Now().UTC()
	day24h := now.Add(-HoursPerDay * time.Hour)
	
	// Get all logs from storage
	allLogs := GetAllLogs(app)
	if len(allLogs) == 0 {
		log := logger.Default().WithComponent("stats-chart")
		log.Warn("No logs available for chart generation")
		return models.ChartData{}
	}
	
	// Generate latency timeline (last 24h, hourly buckets)
	latencyData := generateLatencyChart(allLogs, siteID, now, DefaultChartDataPoints)
	
	// Generate uptime overview (last 7 days, daily buckets)
	uptimeData := generateUptimeChart(allLogs, siteID, now, DaysPerWeek)
	
	// Generate SLA comparison (last 12 months, monthly buckets)
	slaData := generateSLAChart(allLogs, siteID, now, MonthsPerYear)
	
	// Generate response time distribution (last 24h)
	distributionData := generateDistributionChart(allLogs, siteID, day24h)
	
	// Generate yearly uptime chart (last 12 months for SLA tracking)
	yearlyData := generateYearlyChart(allLogs, siteID, now, MonthsPerYear)
	
	// Generate extended ping data charts
	packetLossData := generatePacketLossChart(allLogs, siteID, now, DefaultChartDataPoints)
	jitterData := generateJitterChart(allLogs, siteID, now, DefaultChartDataPoints)
	minLatencyData, maxLatencyData := generateLatencyMinMaxChart(allLogs, siteID, now, DefaultChartDataPoints)
	
	return models.ChartData{
		// Latency timeline (24h)
		LatencyChartLabels:        latencyData.Labels,
		LatencyChartDataPrimary:   latencyData.PrimaryData,
		LatencyChartDataSecondary: latencyData.SecondaryData,

		// Uptime overview (7d)
		UptimeChartLabels:        uptimeData.Labels,
		UptimeChartData:          uptimeData.CombinedData,
		UptimeChartDataPrimary:   uptimeData.PrimaryData,
		UptimeChartDataSecondary: uptimeData.SecondaryData,

		// SLA comparison (12m)
		SLAChartLabels:        slaData.Labels,
		SLAChartDataPrimary:   slaData.PrimaryData,
		SLAChartDataSecondary: slaData.SecondaryData,

		// Response time distribution (24h)
		DistributionChartLabels:   distributionData.Labels,
		DistributionChartData:     distributionData.CombinedData,
		DistributionPrimaryData:   distributionData.PrimaryData,
		DistributionSecondaryData: distributionData.SecondaryData,

		// Yearly SLA tracking (365d)
		YearlyUptimeLabels:        yearlyData.Labels,
		YearlyUptimeData:          yearlyData.CombinedData,
		YearlyUptimeDataPrimary:   yearlyData.PrimaryData,
		YearlyUptimeDataSecondary: yearlyData.SecondaryData,
		
		// Extended ping data charts (24h)
		PacketLossChartLabels:        packetLossData.Labels,
		PacketLossChartDataPrimary:   packetLossData.PrimaryData,
		PacketLossChartDataSecondary: packetLossData.SecondaryData,
		
		JitterChartLabels:        jitterData.Labels,
		JitterChartDataPrimary:   jitterData.PrimaryData,
		JitterChartDataSecondary: jitterData.SecondaryData,
		
		LatencyMinMaxChartLabels:        minLatencyData.Labels,
		LatencyMinChartDataPrimary:      minLatencyData.PrimaryData,
		LatencyMinChartDataSecondary:    minLatencyData.SecondaryData,
		LatencyMaxChartDataPrimary:      maxLatencyData.PrimaryData,
		LatencyMaxChartDataSecondary:    maxLatencyData.SecondaryData,
	}
}

// ChartDataResult represents structured chart data
type ChartDataResult struct {
	Labels        []string
	CombinedData  []float64
	PrimaryData   []float64
	SecondaryData []float64
}

// generateLatencyChart generates latency chart data (hourly)
func generateLatencyChart(allLogs []models.PingLog, siteID string, now time.Time, hours int) ChartDataResult {
	var labels []string
	var primaryLatencies, secondaryLatencies []float64
	
	for i := hours - 1; i >= 0; i-- {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		labels = append(labels, hourStart.Format("15:04"))
		
		// Filter logs for this specific hour
		var hourLogs []models.PingLog
		for _, log := range allLogs {
			if log.SiteID == siteID && !log.Timestamp.Before(hourStart) && log.Timestamp.Before(hourEnd) {
				hourLogs = append(hourLogs, log)
			}
		}
		

		
		// Calculate mean latencies for this hour only
		var primarySum, secondarySum float64
		var primaryCount, secondaryCount int
		
		for _, log := range hourLogs {
			if log.Success && log.Latency != nil {
				if log.Target == "primary" {
					primarySum += *log.Latency
					primaryCount++
				} else if log.Target == "secondary" {
					secondarySum += *log.Latency
					secondaryCount++
				}
			}
		}
		
		var primaryMean, secondaryMean float64
		if primaryCount > 0 {
			primaryMean = primarySum / float64(primaryCount)
		}
		if secondaryCount > 0 {
			secondaryMean = secondarySum / float64(secondaryCount)
		}
		
		primaryLatencies = append(primaryLatencies, primaryMean)
		secondaryLatencies = append(secondaryLatencies, secondaryMean)
	}
	
	// Add detailed debugging output
	log := logger.Default().WithComponent("chart-latency")
	
	// Count non-zero values
	nonZeroPrimary := 0
	nonZeroSecondary := 0
	for _, val := range primaryLatencies {
		if val > 0 {
			nonZeroPrimary++
		}
	}
	for _, val := range secondaryLatencies {
		if val > 0 {
			nonZeroSecondary++
		}
	}
	
	// Get sample of actual logs for debugging
	sampleLogCount := 0
	var sampleLogTimes []string
	for _, log := range allLogs {
		if log.SiteID == siteID && sampleLogCount < 5 {
			sampleLogTimes = append(sampleLogTimes, log.Timestamp.Format("2006-01-02 15:04:05 UTC"))
			sampleLogCount++
		}
	}
	
	log.Info("Generated hourly latency chart data", 
		"site_id", siteID, 
		"hours", hours,
		"total_logs", len(allLogs),
		"labels_count", len(labels),
		"primary_count", len(primaryLatencies),
		"secondary_count", len(secondaryLatencies),
		"non_zero_primary", nonZeroPrimary,
		"non_zero_secondary", nonZeroSecondary,
		"sample_primary_first", func() []float64 { 
			if len(primaryLatencies) >= 3 { 
				return primaryLatencies[:3] 
			} 
			return primaryLatencies 
		}(),
		"sample_primary_last", func() []float64 { 
			if len(primaryLatencies) >= 3 { 
				return primaryLatencies[len(primaryLatencies)-3:] 
			} 
			return primaryLatencies 
		}(),
		"sample_secondary_first", func() []float64 { 
			if len(secondaryLatencies) >= 3 { 
				return secondaryLatencies[:3] 
			} 
			return secondaryLatencies 
		}(),
		"sample_secondary_last", func() []float64 { 
			if len(secondaryLatencies) >= 3 { 
				return secondaryLatencies[len(secondaryLatencies)-3:] 
			} 
			return secondaryLatencies 
		}(),
		"sample_labels", func() []string { 
			if len(labels) >= 3 { 
				return labels[:3] 
			} 
			return labels 
		}(),
		"sample_log_times", sampleLogTimes,
		"now_utc", now.Format("2006-01-02 15:04:05 UTC"))
	
	// Filter out empty buckets to show only periods with real data
	filteredResult := filterEmptyBuckets(labels, primaryLatencies, secondaryLatencies)
	
	return filteredResult
}

// generateLatencyChartMinutely generates latency chart data with minute-level granularity
func generateLatencyChartMinutely(allLogs []models.PingLog, siteID string, now time.Time, minutes int) ChartDataResult {
	var labels []string
	var primaryLatencies, secondaryLatencies []float64
	
	for i := minutes - 1; i >= 0; i-- {
		minuteStart := now.Add(time.Duration(-i) * time.Minute).Truncate(time.Minute)
		minuteEnd := minuteStart.Add(time.Minute)
		
		labels = append(labels, minuteStart.Format("15:04"))
		
		// Filter logs for this specific minute
		var minuteLogs []models.PingLog
		for _, log := range allLogs {
			if log.SiteID == siteID && !log.Timestamp.Before(minuteStart) && log.Timestamp.Before(minuteEnd) {
				minuteLogs = append(minuteLogs, log)
			}
		}
		
		// Calculate mean latencies for this minute
		var primarySum, secondarySum float64
		var primaryCount, secondaryCount int
		
		for _, log := range minuteLogs {
			if log.Success && log.Latency != nil {
				if log.Target == "primary" {
					primarySum += *log.Latency
					primaryCount++
				} else if log.Target == "secondary" {
					secondarySum += *log.Latency
					secondaryCount++
				}
			}
		}
		
		var primaryMean, secondaryMean float64
		if primaryCount > 0 {
			primaryMean = primarySum / float64(primaryCount)
		}
		if secondaryCount > 0 {
			secondaryMean = secondarySum / float64(secondaryCount)
		}
		
		primaryLatencies = append(primaryLatencies, primaryMean)
		secondaryLatencies = append(secondaryLatencies, secondaryMean)
	}
	
	// Filter out empty buckets to show only periods with real data
	return filterEmptyBuckets(labels, primaryLatencies, secondaryLatencies)
}

// generateLatencyChart5Minutes generates latency chart data with 5-minute buckets
func generateLatencyChart5Minutes(allLogs []models.PingLog, siteID string, now time.Time, periods int) ChartDataResult {
	var labels []string
	var primaryLatencies, secondaryLatencies []float64
	
	for i := periods - 1; i >= 0; i-- {
		periodStart := now.Add(time.Duration(-i*5) * time.Minute).Truncate(5 * time.Minute)
		periodEnd := periodStart.Add(5 * time.Minute)
		
		labels = append(labels, periodStart.Format("15:04"))
		
		// Filter logs for this 5-minute period
		var periodLogs []models.PingLog
		for _, log := range allLogs {
			if log.SiteID == siteID && !log.Timestamp.Before(periodStart) && log.Timestamp.Before(periodEnd) {
				periodLogs = append(periodLogs, log)
			}
		}
		
		// Calculate mean latencies for this 5-minute period
		var primarySum, secondarySum float64
		var primaryCount, secondaryCount int
		
		for _, log := range periodLogs {
			if log.Success && log.Latency != nil {
				if log.Target == "primary" {
					primarySum += *log.Latency
					primaryCount++
				} else if log.Target == "secondary" {
					secondarySum += *log.Latency
					secondaryCount++
				}
			}
		}
		
		var primaryMean, secondaryMean float64
		if primaryCount > 0 {
			primaryMean = primarySum / float64(primaryCount)
		}
		if secondaryCount > 0 {
			secondaryMean = secondarySum / float64(secondaryCount)
		}
		
		primaryLatencies = append(primaryLatencies, primaryMean)
		secondaryLatencies = append(secondaryLatencies, secondaryMean)
	}
	
	// Filter out empty buckets to show only periods with real data
	return filterEmptyBuckets(labels, primaryLatencies, secondaryLatencies)
}

// generatePacketLossChartMinutely generates packet loss chart data with minute-level granularity
func generatePacketLossChartMinutely(allLogs []models.PingLog, siteID string, now time.Time, minutes int) ChartDataResult {
	var labels []string
	var primaryPacketLoss, secondaryPacketLoss []float64
	
	for i := minutes - 1; i >= 0; i-- {
		minuteStart := now.Add(time.Duration(-i) * time.Minute).Truncate(time.Minute)
		minuteEnd := minuteStart.Add(time.Minute)
		
		labels = append(labels, minuteStart.Format("15:04"))
		
		var primaryLossSum, secondaryLossSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(minuteStart) || !log.Timestamp.Before(minuteEnd) {
				continue
			}
			
			if log.Target == "primary" && log.PacketLoss != nil {
				primaryLossSum += *log.PacketLoss
				primaryCount++
			} else if log.Target == "secondary" && log.PacketLoss != nil {
				secondaryLossSum += *log.PacketLoss
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryPacketLoss = append(primaryPacketLoss, primaryLossSum/float64(primaryCount))
		} else {
			primaryPacketLoss = append(primaryPacketLoss, 0)
		}
		
		if secondaryCount > 0 {
			secondaryPacketLoss = append(secondaryPacketLoss, secondaryLossSum/float64(secondaryCount))
		} else {
			secondaryPacketLoss = append(secondaryPacketLoss, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryPacketLoss, secondaryPacketLoss)
}

// generatePacketLossChart5Minutes generates packet loss chart data with 5-minute buckets
func generatePacketLossChart5Minutes(allLogs []models.PingLog, siteID string, now time.Time, periods int) ChartDataResult {
	var labels []string
	var primaryPacketLoss, secondaryPacketLoss []float64
	
	for i := periods - 1; i >= 0; i-- {
		periodStart := now.Add(time.Duration(-i*5) * time.Minute).Truncate(5 * time.Minute)
		periodEnd := periodStart.Add(5 * time.Minute)
		
		labels = append(labels, periodStart.Format("15:04"))
		
		var primaryLossSum, secondaryLossSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(periodStart) || !log.Timestamp.Before(periodEnd) {
				continue
			}
			
			if log.Target == "primary" && log.PacketLoss != nil {
				primaryLossSum += *log.PacketLoss
				primaryCount++
			} else if log.Target == "secondary" && log.PacketLoss != nil {
				secondaryLossSum += *log.PacketLoss
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryPacketLoss = append(primaryPacketLoss, primaryLossSum/float64(primaryCount))
		} else {
			primaryPacketLoss = append(primaryPacketLoss, 0)
		}
		
		if secondaryCount > 0 {
			secondaryPacketLoss = append(secondaryPacketLoss, secondaryLossSum/float64(secondaryCount))
		} else {
			secondaryPacketLoss = append(secondaryPacketLoss, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryPacketLoss, secondaryPacketLoss)
}

// generateJitterChartMinutely generates jitter chart data with minute-level granularity
func generateJitterChartMinutely(allLogs []models.PingLog, siteID string, now time.Time, minutes int) ChartDataResult {
	var labels []string
	var primaryJitter, secondaryJitter []float64
	
	for i := minutes - 1; i >= 0; i-- {
		minuteStart := now.Add(time.Duration(-i) * time.Minute).Truncate(time.Minute)
		minuteEnd := minuteStart.Add(time.Minute)
		
		labels = append(labels, minuteStart.Format("15:04"))
		
		var primaryJitterSum, secondaryJitterSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(minuteStart) || !log.Timestamp.Before(minuteEnd) {
				continue
			}
			
			if log.Target == "primary" && log.Jitter != nil {
				primaryJitterSum += *log.Jitter
				primaryCount++
			} else if log.Target == "secondary" && log.Jitter != nil {
				secondaryJitterSum += *log.Jitter
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryJitter = append(primaryJitter, primaryJitterSum/float64(primaryCount))
		} else {
			primaryJitter = append(primaryJitter, 0)
		}
		
		if secondaryCount > 0 {
			secondaryJitter = append(secondaryJitter, secondaryJitterSum/float64(secondaryCount))
		} else {
			secondaryJitter = append(secondaryJitter, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryJitter, secondaryJitter)
}

// generateJitterChart5Minutes generates jitter chart data with 5-minute buckets
func generateJitterChart5Minutes(allLogs []models.PingLog, siteID string, now time.Time, periods int) ChartDataResult {
	var labels []string
	var primaryJitter, secondaryJitter []float64
	
	for i := periods - 1; i >= 0; i-- {
		periodStart := now.Add(time.Duration(-i*5) * time.Minute).Truncate(5 * time.Minute)
		periodEnd := periodStart.Add(5 * time.Minute)
		
		labels = append(labels, periodStart.Format("15:04"))
		
		var primaryJitterSum, secondaryJitterSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(periodStart) || !log.Timestamp.Before(periodEnd) {
				continue
			}
			
			if log.Target == "primary" && log.Jitter != nil {
				primaryJitterSum += *log.Jitter
				primaryCount++
			} else if log.Target == "secondary" && log.Jitter != nil {
				secondaryJitterSum += *log.Jitter
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryJitter = append(primaryJitter, primaryJitterSum/float64(primaryCount))
		} else {
			primaryJitter = append(primaryJitter, 0)
		}
		
		if secondaryCount > 0 {
			secondaryJitter = append(secondaryJitter, secondaryJitterSum/float64(secondaryCount))
		} else {
			secondaryJitter = append(secondaryJitter, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryJitter, secondaryJitter)
}

// filterEmptyBuckets removes time buckets that have no data for any line
// NOTE: 0 values are considered valid data (e.g. 0% packet loss), only filter truly empty buckets
func filterEmptyBuckets(labels []string, primaryData, secondaryData []float64) ChartDataResult {
	var filteredLabels []string
	var filteredPrimary, filteredSecondary []float64
	
	// Keep buckets that have data in at least one line (including 0 values)
	for i := 0; i < len(labels); i++ {
		hasPrimaryData := i < len(primaryData)
		hasSecondaryData := i < len(secondaryData)
		
		// Include bucket if we have data for either line (even if value is 0)
		if hasPrimaryData || hasSecondaryData {
			filteredLabels = append(filteredLabels, labels[i])
			
			if i < len(primaryData) {
				filteredPrimary = append(filteredPrimary, primaryData[i])
			} else {
				filteredPrimary = append(filteredPrimary, 0)
			}
			
			if i < len(secondaryData) {
				filteredSecondary = append(filteredSecondary, secondaryData[i])
			} else {
				filteredSecondary = append(filteredSecondary, 0)
			}
		}
	}
	
	// Fallback: if no data found, keep at least the last bucket to avoid empty charts
	if len(filteredLabels) == 0 && len(labels) > 0 {
		lastIdx := len(labels) - 1
		filteredLabels = append(filteredLabels, labels[lastIdx])
		
		if lastIdx < len(primaryData) {
			filteredPrimary = append(filteredPrimary, primaryData[lastIdx])
		} else {
			filteredPrimary = append(filteredPrimary, 0)
		}
		
		if lastIdx < len(secondaryData) {
			filteredSecondary = append(filteredSecondary, secondaryData[lastIdx])
		} else {
			filteredSecondary = append(filteredSecondary, 0)
		}
	}
	
	return ChartDataResult{
		Labels:        filteredLabels,
		PrimaryData:   filteredPrimary,
		SecondaryData: filteredSecondary,
	}
}

// generatePacketLossChart generates packet loss chart data
func generatePacketLossChart(allLogs []models.PingLog, siteID string, now time.Time, hours int) ChartDataResult {
	var labels []string
	var primaryPacketLoss, secondaryPacketLoss []float64
	
	for i := hours - 1; i >= 0; i-- {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		labels = append(labels, hourStart.Format("15:04"))
		
		var primaryLossSum, secondaryLossSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(hourStart) || !log.Timestamp.Before(hourEnd) {
				continue
			}
			
			if log.Target == "primary" && log.PacketLoss != nil {
				primaryLossSum += *log.PacketLoss
				primaryCount++
			} else if log.Target == "secondary" && log.PacketLoss != nil {
				secondaryLossSum += *log.PacketLoss
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryPacketLoss = append(primaryPacketLoss, primaryLossSum/float64(primaryCount))
		} else {
			primaryPacketLoss = append(primaryPacketLoss, 0)
		}
		
		if secondaryCount > 0 {
			secondaryPacketLoss = append(secondaryPacketLoss, secondaryLossSum/float64(secondaryCount))
		} else {
			secondaryPacketLoss = append(secondaryPacketLoss, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryPacketLoss, secondaryPacketLoss)
}

// generateJitterChart generates jitter chart data
func generateJitterChart(allLogs []models.PingLog, siteID string, now time.Time, hours int) ChartDataResult {
	var labels []string
	var primaryJitter, secondaryJitter []float64
	
	for i := hours - 1; i >= 0; i-- {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		labels = append(labels, hourStart.Format("15:04"))
		
		var primaryJitterSum, secondaryJitterSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(hourStart) || !log.Timestamp.Before(hourEnd) {
				continue
			}
			
			if log.Target == "primary" && log.Jitter != nil {
				primaryJitterSum += *log.Jitter
				primaryCount++
			} else if log.Target == "secondary" && log.Jitter != nil {
				secondaryJitterSum += *log.Jitter
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryJitter = append(primaryJitter, primaryJitterSum/float64(primaryCount))
		} else {
			primaryJitter = append(primaryJitter, 0)
		}
		
		if secondaryCount > 0 {
			secondaryJitter = append(secondaryJitter, secondaryJitterSum/float64(secondaryCount))
		} else {
			secondaryJitter = append(secondaryJitter, 0)
		}
	}
	
	// Filter out empty buckets to show only periods with real data
	return filterEmptyBuckets(labels, primaryJitter, secondaryJitter)
}

// generateLatencyMinMaxChart generates min/max latency chart data
func generateLatencyMinMaxChart(allLogs []models.PingLog, siteID string, now time.Time, hours int) (ChartDataResult, ChartDataResult) {
	var labels []string
	var primaryMin, primaryMax, secondaryMin, secondaryMax []float64
	
	for i := hours - 1; i >= 0; i-- {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		labels = append(labels, hourStart.Format("15:04"))
		
		var primaryMinVal, primaryMaxVal, secondaryMinVal, secondaryMaxVal float64
		var primaryMinSet, primaryMaxSet, secondaryMinSet, secondaryMaxSet bool
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(hourStart) || !log.Timestamp.Before(hourEnd) {
				continue
			}
			
			if log.Target == "primary" {
				if log.MinLatency != nil {
					if !primaryMinSet || *log.MinLatency < primaryMinVal {
						primaryMinVal = *log.MinLatency
						primaryMinSet = true
					}
				}
				if log.MaxLatency != nil {
					if !primaryMaxSet || *log.MaxLatency > primaryMaxVal {
						primaryMaxVal = *log.MaxLatency
						primaryMaxSet = true
					}
				}
			} else if log.Target == "secondary" {
				if log.MinLatency != nil {
					if !secondaryMinSet || *log.MinLatency < secondaryMinVal {
						secondaryMinVal = *log.MinLatency
						secondaryMinSet = true
					}
				}
				if log.MaxLatency != nil {
					if !secondaryMaxSet || *log.MaxLatency > secondaryMaxVal {
						secondaryMaxVal = *log.MaxLatency
						secondaryMaxSet = true
					}
				}
			}
		}
		
		if primaryMinSet {
			primaryMin = append(primaryMin, primaryMinVal)
		} else {
			primaryMin = append(primaryMin, 0)
		}
		
		if primaryMaxSet {
			primaryMax = append(primaryMax, primaryMaxVal)
		} else {
			primaryMax = append(primaryMax, 0)
		}
		
		if secondaryMinSet {
			secondaryMin = append(secondaryMin, secondaryMinVal)
		} else {
			secondaryMin = append(secondaryMin, 0)
		}
		
		if secondaryMaxSet {
			secondaryMax = append(secondaryMax, secondaryMaxVal)
		} else {
			secondaryMax = append(secondaryMax, 0)
		}
	}
	
	minResult := ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryMin,
		SecondaryData: secondaryMin,
	}
	
	maxResult := ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryMax,
		SecondaryData: secondaryMax,
	}
	
	return minResult, maxResult
}

// generateLatencyChartDaily generates latency chart data (daily)
func generateLatencyChartDaily(allLogs []models.PingLog, siteID string, now time.Time, days int) ChartDataResult {
	var labels []string
	var primaryLatencies, secondaryLatencies []float64
	
	for i := days - 1; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)
		
		labels = append(labels, dayStart.Format("Jan 2"))
		
		// Filter logs for this specific day
		var dayLogs []models.PingLog
		for _, log := range allLogs {
			if log.SiteID == siteID && !log.Timestamp.Before(dayStart) && log.Timestamp.Before(dayEnd) {
				dayLogs = append(dayLogs, log)
			}
		}
		
		// Calculate mean latencies for this day only
		var primarySum, secondarySum float64
		var primaryCount, secondaryCount int
		
		for _, log := range dayLogs {
			if log.Success && log.Latency != nil {
				if log.Target == "primary" {
					primarySum += *log.Latency
					primaryCount++
				} else if log.Target == "secondary" {
					secondarySum += *log.Latency
					secondaryCount++
				}
			}
		}
		
		var primaryMean, secondaryMean float64
		if primaryCount > 0 {
			primaryMean = primarySum / float64(primaryCount)
		}
		if secondaryCount > 0 {
			secondaryMean = secondarySum / float64(secondaryCount)
		}
		
		primaryLatencies = append(primaryLatencies, primaryMean)
		secondaryLatencies = append(secondaryLatencies, secondaryMean)
	}
	
	return ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryLatencies,
		SecondaryData: secondaryLatencies,
	}
}

// generatePacketLossChartDaily generates packet loss chart data (daily aggregation)
func generatePacketLossChartDaily(allLogs []models.PingLog, siteID string, now time.Time, days int) ChartDataResult {
	var labels []string
	var primaryPacketLoss, secondaryPacketLoss []float64
	
	for i := days - 1; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)
		
		labels = append(labels, dayStart.Format("Jan 2"))
		
		var primaryLossSum, secondaryLossSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(dayStart) || !log.Timestamp.Before(dayEnd) {
				continue
			}
			
			if log.Target == "primary" && log.PacketLoss != nil {
				primaryLossSum += *log.PacketLoss
				primaryCount++
			} else if log.Target == "secondary" && log.PacketLoss != nil {
				secondaryLossSum += *log.PacketLoss
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryPacketLoss = append(primaryPacketLoss, primaryLossSum/float64(primaryCount))
		} else {
			primaryPacketLoss = append(primaryPacketLoss, 0)
		}
		
		if secondaryCount > 0 {
			secondaryPacketLoss = append(secondaryPacketLoss, secondaryLossSum/float64(secondaryCount))
		} else {
			secondaryPacketLoss = append(secondaryPacketLoss, 0)
		}
	}
	
	return filterEmptyBuckets(labels, primaryPacketLoss, secondaryPacketLoss)
}

// generateJitterChartDaily generates jitter chart data (daily aggregation)
func generateJitterChartDaily(allLogs []models.PingLog, siteID string, now time.Time, days int) ChartDataResult {
	var labels []string
	var primaryJitter, secondaryJitter []float64
	
	for i := days - 1; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)
		
		labels = append(labels, dayStart.Format("Jan 2"))
		
		var primaryJitterSum, secondaryJitterSum float64
		var primaryCount, secondaryCount int
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(dayStart) || !log.Timestamp.Before(dayEnd) {
				continue
			}
			
			if log.Target == "primary" && log.Jitter != nil {
				primaryJitterSum += *log.Jitter
				primaryCount++
			} else if log.Target == "secondary" && log.Jitter != nil {
				secondaryJitterSum += *log.Jitter
				secondaryCount++
			}
		}
		
		if primaryCount > 0 {
			primaryJitter = append(primaryJitter, primaryJitterSum/float64(primaryCount))
		} else {
			primaryJitter = append(primaryJitter, 0)
		}
		
		if secondaryCount > 0 {
			secondaryJitter = append(secondaryJitter, secondaryJitterSum/float64(secondaryCount))
		} else {
			secondaryJitter = append(secondaryJitter, 0)
		}
	}
	
	return ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryJitter,
		SecondaryData: secondaryJitter,
	}
}

// generateLatencyMinMaxChartDaily generates min/max latency chart data (daily aggregation)
func generateLatencyMinMaxChartDaily(allLogs []models.PingLog, siteID string, now time.Time, days int) (ChartDataResult, ChartDataResult) {
	var labels []string
	var primaryMin, primaryMax, secondaryMin, secondaryMax []float64
	
	for i := days - 1; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)
		
		labels = append(labels, dayStart.Format("Jan 2"))
		
		var primaryMinVal, primaryMaxVal, secondaryMinVal, secondaryMaxVal float64
		var primaryMinSet, primaryMaxSet, secondaryMinSet, secondaryMaxSet bool
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(dayStart) || !log.Timestamp.Before(dayEnd) {
				continue
			}
			
			if log.Target == "primary" {
				if log.MinLatency != nil {
					if !primaryMinSet || *log.MinLatency < primaryMinVal {
						primaryMinVal = *log.MinLatency
						primaryMinSet = true
					}
				}
				if log.MaxLatency != nil {
					if !primaryMaxSet || *log.MaxLatency > primaryMaxVal {
						primaryMaxVal = *log.MaxLatency
						primaryMaxSet = true
					}
				}
			} else if log.Target == "secondary" {
				if log.MinLatency != nil {
					if !secondaryMinSet || *log.MinLatency < secondaryMinVal {
						secondaryMinVal = *log.MinLatency
						secondaryMinSet = true
					}
				}
				if log.MaxLatency != nil {
					if !secondaryMaxSet || *log.MaxLatency > secondaryMaxVal {
						secondaryMaxVal = *log.MaxLatency
						secondaryMaxSet = true
					}
				}
			}
		}
		
		if primaryMinSet {
			primaryMin = append(primaryMin, primaryMinVal)
		} else {
			primaryMin = append(primaryMin, 0)
		}
		
		if primaryMaxSet {
			primaryMax = append(primaryMax, primaryMaxVal)
		} else {
			primaryMax = append(primaryMax, 0)
		}
		
		if secondaryMinSet {
			secondaryMin = append(secondaryMin, secondaryMinVal)
		} else {
			secondaryMin = append(secondaryMin, 0)
		}
		
		if secondaryMaxSet {
			secondaryMax = append(secondaryMax, secondaryMaxVal)
		} else {
			secondaryMax = append(secondaryMax, 0)
		}
	}
	
	minResult := ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryMin,
		SecondaryData: secondaryMin,
	}
	
	maxResult := ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryMax,
		SecondaryData: secondaryMax,
	}
	
	return minResult, maxResult
}

// generateUptimeChartHourly generates uptime chart data (hourly aggregation)
func generateUptimeChartHourly(allLogs []models.PingLog, siteID string, now time.Time, hours int) ChartDataResult {
	var labels []string
	var combinedData, primaryData, secondaryData []float64
	
	for i := hours - 1; i >= 0; i-- {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		labels = append(labels, hourStart.Format("15:04"))
		
		stats := NewTimeframeStats()
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(hourStart) || !log.Timestamp.Before(hourEnd) {
				continue
			}
			stats.AddLog(log)
		}
		
		// Calculate uptime percentages for this hour
		combinedUptime := stats.GetUptimePercentage()
		primaryUptime := stats.GetProviderUptime("primary")
		secondaryUptime := stats.GetProviderUptime("secondary")
		
		combinedData = append(combinedData, combinedUptime)
		primaryData = append(primaryData, primaryUptime)
		secondaryData = append(secondaryData, secondaryUptime)
	}
	
	return ChartDataResult{
		Labels:        labels,
		CombinedData:  combinedData,
		PrimaryData:   primaryData,
		SecondaryData: secondaryData,
	}
}

// generateUptimeChart generates uptime chart data
func generateUptimeChart(allLogs []models.PingLog, siteID string, now time.Time, days int) ChartDataResult {
	var labels []string
	var combinedData, primaryData, secondaryData []float64
	
	for i := days - 1; i >= 0; i-- {
		dayStart := now.AddDate(0, 0, -i).Truncate(HoursPerDay * time.Hour)
		dayEnd := dayStart.Add(HoursPerDay * time.Hour)
		
		labels = append(labels, dayStart.Format("Jan 2"))
		
		stats := NewTimeframeStats()
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(dayStart) || !log.Timestamp.Before(dayEnd) {
				continue
			}
			stats.AddLog(log)
		}
		
		combinedData = append(combinedData, stats.GetUptimePercentage())
		primaryData = append(primaryData, stats.GetProviderUptime("primary"))
		secondaryData = append(secondaryData, stats.GetProviderUptime("secondary"))
	}
	
	return ChartDataResult{
		Labels:        labels,
		CombinedData:  combinedData,
		PrimaryData:   primaryData,
		SecondaryData: secondaryData,
	}
}

// generateSLAChart generates SLA comparison chart data
func generateSLAChart(allLogs []models.PingLog, siteID string, now time.Time, months int) ChartDataResult {
	var labels []string
	var primaryData, secondaryData []float64
	
	for i := months - 1; i >= 0; i-- {
		monthStart := now.AddDate(0, -i, 0).Truncate(HoursPerDay * time.Hour)
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
		monthEnd := monthStart.AddDate(0, 1, 0)
		
		labels = append(labels, monthStart.Format("Jan 2006"))
		
		stats := NewTimeframeStats()
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(monthStart) || !log.Timestamp.Before(monthEnd) {
				continue
			}
			stats.AddLog(log)
		}
		
		primaryData = append(primaryData, stats.GetProviderUptime("primary"))
		secondaryData = append(secondaryData, stats.GetProviderUptime("secondary"))
	}
	
	return ChartDataResult{
		Labels:        labels,
		PrimaryData:   primaryData,
		SecondaryData: secondaryData,
	}
}

// generateDistributionChart generates response time distribution chart data
func generateDistributionChart(allLogs []models.PingLog, siteID string, since time.Time) ChartDataResult {
	distributionLabels := []string{"0-10ms", "10-50ms", "50-100ms", "100-200ms", "200-500ms", "500ms+"}
	
	stats := NewTimeframeStats()
	primaryStats := NewTimeframeStats()
	secondaryStats := NewTimeframeStats()
	
	for _, log := range allLogs {
		if log.SiteID != siteID || log.Timestamp.Before(since) || !log.Success || log.Latency == nil {
			continue
		}
		
		stats.AddLog(log)
		if log.Target == "primary" {
			primaryStats.AddLog(log)
		} else if log.Target == "secondary" {
			secondaryStats.AddLog(log)
		}
	}
	
	return ChartDataResult{
		Labels:        distributionLabels,
		CombinedData:  stats.GetLatencyDistribution(),
		PrimaryData:   primaryStats.GetLatencyDistribution(),
		SecondaryData: secondaryStats.GetLatencyDistribution(),
	}
}

// generateYearlyChart generates yearly uptime chart data
func generateYearlyChart(allLogs []models.PingLog, siteID string, now time.Time, months int) ChartDataResult {
	var labels []string
	var combinedData, primaryData, secondaryData []float64
	
	for i := months - 1; i >= 0; i-- {
		monthStart := now.AddDate(0, -i, 0).Truncate(HoursPerDay * time.Hour)
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
		monthEnd := monthStart.AddDate(0, 1, 0)
		
		labels = append(labels, monthStart.Format("Jan"))
		
		stats := NewTimeframeStats()
		
		for _, log := range allLogs {
			if log.SiteID != siteID || log.Timestamp.Before(monthStart) || !log.Timestamp.Before(monthEnd) {
				continue
			}
			stats.AddLog(log)
		}
		
		combinedData = append(combinedData, stats.GetUptimePercentage())
		primaryData = append(primaryData, stats.GetProviderUptime("primary"))
		secondaryData = append(secondaryData, stats.GetProviderUptime("secondary"))
	}
	
	return ChartDataResult{
		Labels:        labels,
		CombinedData:  combinedData,
		PrimaryData:   primaryData,
		SecondaryData: secondaryData,
	}
}

// GetRecentEvents returns recent status change events for a site with improved event detection
func GetRecentEvents(app *config.AppState, siteID string, limit int) []models.RecentEvent {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	// Get all logs from storage
	allLogs := GetAllLogs(app)
	if len(allLogs) == 0 {
		log := logger.Default().WithComponent("stats-events")
		log.Warn("No logs available for event detection")
		return []models.RecentEvent{}
	}
	
	var events []models.RecentEvent
	var lastStatus = make(map[string]bool) // target -> success
	
	// Analyze logs in chronological order to detect status changes
	for i := 0; i < len(allLogs); i++ {
		pingLog := allLogs[i]
		if pingLog.SiteID != siteID {
			continue
		}
		
		// Validate log data before processing
		if err := validateLogData(pingLog); err != nil {
			log := logger.Default().WithComponent("stats-events").WithSite(siteID, "")
			log.Warn("Skipping invalid log for event detection", "error", err)
			continue
		}
		
		// Check if this is a status change
		if prevStatus, exists := lastStatus[pingLog.Target]; exists && prevStatus != pingLog.Success {
			event := models.RecentEvent{
				Timestamp: pingLog.Timestamp,
				SiteID:    pingLog.SiteID,
				Target:    pingLog.Target,
			}
			
			// This log represents the NEW status after the change
			if pingLog.Success {
				event.Status = "restored"
				event.Message = fmt.Sprintf("%s connection restored", strings.Title(pingLog.Target))
				event.IsOutage = false
			} else {
				event.Status = "failed"
				event.Message = fmt.Sprintf("%s connection lost", strings.Title(pingLog.Target))
				event.IsOutage = true
			}
			
			events = append(events, event)
		}
		
		lastStatus[pingLog.Target] = pingLog.Success
	}
	
	// Reverse to get newest events first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	
	// Limit to requested number of events
	if len(events) > limit {
		events = events[:limit]
	}
	
	return events
}

// CalculateOverviewData calculates overall system statistics with improved accuracy
func CalculateOverviewData(app *config.AppState) models.OverviewData {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	// Get all logs from storage
	allLogs := GetAllLogs(app)
	
	totalSites := len(app.Sites)
	var onlineSites, offlineSites, degradedSites int
	var totalChecks int64
	var successfulChecks int64
	
	// Count site statuses with improved logic
	for _, site := range app.Sites {
		if !site.Enabled {
			continue
		}
		
		status, exists := app.SiteStatus[site.ID]
		if !exists {
			offlineSites++
			continue
		}
		
		if site.IsDualLine() {
			// Dual-line site
			if status.PrimaryOnline && status.SecondaryOnline {
				onlineSites++
			} else if status.PrimaryOnline || status.SecondaryOnline {
				// Count degraded sites as online (since at least one line works)
				onlineSites++
				degradedSites++
			} else {
				offlineSites++
			}
		} else {
			// Single-line site
			if status.PrimaryOnline {
				onlineSites++
			} else {
				offlineSites++
			}
		}
	}
	
	// Calculate overall uptime with improved accuracy
	totalChecks = atomic.LoadInt64(&app.TotalChecks)
	
	// Count successful checks from logs
	for _, log := range allLogs {
		if log.Success {
			successfulChecks++
		}
	}
	
	var uptimePercentage float64
	if len(allLogs) > 0 {
		uptimePercentage = roundToDecimalPlaces(float64(successfulChecks)/float64(len(allLogs))*100, UptimePrecision)
	}
	
	// Calculate uptime duration
	uptime := time.Since(app.StartTime)
	uptimeStr := FormatDuration(uptime)
	
	return models.OverviewData{
		TotalSites:       totalSites,
		OnlineSites:      onlineSites,
		OfflineSites:     offlineSites,
		DegradedSites:    degradedSites,
		UptimePercentage: uptimePercentage,
		TotalChecks:      totalChecks,
		Uptime:           uptimeStr,
	}
}

// GenerateChartDataForRange generates chart data for a specific chart type and time range
func GenerateChartDataForRange(app *config.AppState, siteID, chartType, timeRange string) interface{} {
	app.Mu.RLock()
	defer app.Mu.RUnlock()
	
	now := time.Now().UTC()
	allLogs := GetAllLogs(app)
	
	switch chartType {
	case "latency":
		switch timeRange {
		case "1h":
			return generateLatencyChartMinutely(allLogs, siteID, now, 60) // 60 minute points
		case "3h":
			return generateLatencyChart5Minutes(allLogs, siteID, now, 36) // 36 x 5-minute points
		case "12h":
			return generateLatencyChart5Minutes(allLogs, siteID, now, 144) // 144 x 5-minute points
		case "24h":
			return generateLatencyChart(allLogs, siteID, now, 24) // 24 hourly points
		case "7d":
			return generateLatencyChartDaily(allLogs, siteID, now, 7) // 7 daily points
		}
	case "uptime":
		switch timeRange {
		case "12h":
			// For sub-day ranges, use hourly aggregation
			return generateUptimeChartHourly(allLogs, siteID, now, 12) // 12 hourly points
		case "24h":
			return generateUptimeChartHourly(allLogs, siteID, now, 24) // 24 hourly points
		case "7d":
			return generateUptimeChart(allLogs, siteID, now, 7) // 7 daily points
		case "30d":
			return generateUptimeChart(allLogs, siteID, now, 30) // 30 daily points
		}
	case "yearly":
		// Always return 12 months for SLA tracking
		return generateSLAChart(allLogs, siteID, now, 12)
	case "distribution":
		// Always return last 24 hours distribution
		since := now.Add(-24 * time.Hour)
		return generateDistributionChart(allLogs, siteID, since)
	case "packet_loss":
		switch timeRange {
		case "1h":
			return generatePacketLossChartMinutely(allLogs, siteID, now, 60) // 60 minute points
		case "3h":
			return generatePacketLossChart5Minutes(allLogs, siteID, now, 36) // 36 x 5-minute points
		case "12h":
			return generatePacketLossChart5Minutes(allLogs, siteID, now, 144) // 144 x 5-minute points
		case "24h":
			return generatePacketLossChart(allLogs, siteID, now, 24) // 24 hourly points
		case "7d":
			return generatePacketLossChartDaily(allLogs, siteID, now, 7) // 7 daily points
		}
	case "jitter":
		switch timeRange {
		case "1h":
			return generateJitterChartMinutely(allLogs, siteID, now, 60) // 60 minute points
		case "3h":
			return generateJitterChart5Minutes(allLogs, siteID, now, 36) // 36 x 5-minute points
		case "12h":
			return generateJitterChart5Minutes(allLogs, siteID, now, 144) // 144 x 5-minute points
		case "24h":
			return generateJitterChart(allLogs, siteID, now, 24) // 24 hourly points
		case "7d":
			return generateJitterChartDaily(allLogs, siteID, now, 7) // 7 daily points
		}
	case "latency_minmax":
		switch timeRange {
		case "1h":
			minData, maxData := generateLatencyMinMaxChart(allLogs, siteID, now, 1)
			return fiber.Map{
				"min": minData,
				"max": maxData,
			}
		case "3h":
			minData, maxData := generateLatencyMinMaxChart(allLogs, siteID, now, 3)
			return fiber.Map{
				"min": minData,
				"max": maxData,
			}
		case "12h":
			minData, maxData := generateLatencyMinMaxChart(allLogs, siteID, now, 12)
			return fiber.Map{
				"min": minData,
				"max": maxData,
			}
		case "24h":
			minData, maxData := generateLatencyMinMaxChart(allLogs, siteID, now, 24)
			return fiber.Map{
				"min": minData,
				"max": maxData,
			}
		case "7d":
			minData, maxData := generateLatencyMinMaxChartDaily(allLogs, siteID, now, 7)
			return fiber.Map{
				"min": minData,
				"max": maxData,
			}
		}
	}
	
	return fiber.Map{"error": "Invalid chart type or range"}
}

// FormatDuration formats a duration in a human-readable way with improved precision
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < HoursPerDay*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours()) / HoursPerDay
		hours := int(d.Hours()) % HoursPerDay
		if hours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}