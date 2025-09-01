package models

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Configuration structs
type Config struct {
	Server struct {
		Host         string        `yaml:"host"`
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"server"`
	Ping struct {
		DefaultInterval time.Duration `yaml:"default_interval"`
		Timeout         time.Duration `yaml:"timeout"`
		PacketSize      int           `yaml:"packet_size"`
		PacketCount     int           `yaml:"packet_count"`     // Number of packets per ping test
	} `yaml:"ping"`
	Metrics struct {
		Enabled bool   `yaml:"enabled"`
		Path    string `yaml:"path"`
	} `yaml:"metrics"`
	
	Storage struct {
		Type       string `yaml:"type"`        // Always "sqlite" for persistent storage
		SQLitePath string `yaml:"sqlite_path"` // Path to SQLite database file
	} `yaml:"storage"`
	
	Auth AuthConfig `yaml:"auth,omitempty"` // Authentication configuration
}

// SLA defines Service Level Agreement parameters
type SLA struct {
	Uptime      float64 `yaml:"uptime" json:"uptime"`           // Uptime percentage (e.g., 99.9)
	MaxLatency  *int    `yaml:"max_latency,omitempty" json:"max_latency,omitempty"` // Optional max latency in ms
	Restoration int     `yaml:"restoration,omitempty" json:"restoration,omitempty"` // Restoration time in minutes
}

// SLAConfig defines SLA configuration for a site
type SLAConfig struct {
	Primary   SLA `yaml:"primary,omitempty" json:"primary,omitempty"`     // Primary provider SLA
	Secondary SLA `yaml:"secondary,omitempty" json:"secondary,omitempty"` // Secondary provider SLA
	Combined  SLA `yaml:"combined,omitempty" json:"combined,omitempty"`   // Combined SLA for dual-line sites
}

type Site struct {
	ID          string    `yaml:"id" json:"id"`
	Name        string    `yaml:"name" json:"name"`
	Location    string    `yaml:"location" json:"location"`
	PrimaryIP   string    `yaml:"primary_ip" json:"primary_ip"`
	SecondaryIP string    `yaml:"secondary_ip,omitempty" json:"secondary_ip,omitempty"` // Optional für Single-Line Sites
	PrimaryProvider   string    `yaml:"primary_provider,omitempty" json:"primary_provider,omitempty"`     // Optional provider name
	SecondaryProvider string    `yaml:"secondary_provider,omitempty" json:"secondary_provider,omitempty"` // Optional provider name
	Interval    int       `yaml:"interval" json:"interval"` // Sekunden
	Enabled     bool      `yaml:"enabled" json:"enabled"`
	SLA         SLAConfig `yaml:"sla,omitempty" json:"sla,omitempty"` // SLA configuration
}

// IsDualLine returns true if site has both primary and secondary IP configured
func (s *Site) IsDualLine() bool {
	return s.SecondaryIP != ""
}

// GetPrimarySLAUptime returns the primary provider SLA uptime target or default 99.9%
func (s *Site) GetPrimarySLAUptime() float64 {
	if s.SLA.Primary.Uptime > 0 {
		return s.SLA.Primary.Uptime
	}
	return 99.9 // Default SLA
}

// GetSecondarySLAUptime returns the secondary provider SLA uptime target or default 99.9%
func (s *Site) GetSecondarySLAUptime() float64 {
	if s.SLA.Secondary.Uptime > 0 {
		return s.SLA.Secondary.Uptime
	}
	return 99.9 // Default SLA
}

// GetCombinedSLAUptime returns the combined SLA uptime target for dual-line sites or primary SLA for single-line
func (s *Site) GetCombinedSLAUptime() float64 {
	if s.IsDualLine() && s.SLA.Combined.Uptime > 0 {
		return s.SLA.Combined.Uptime
	}
	return s.GetPrimarySLAUptime()
}

// GetPrimaryMaxLatency returns the primary provider max latency SLA if configured
func (s *Site) GetPrimaryMaxLatency() *int {
	return s.SLA.Primary.MaxLatency
}

// GetSecondaryMaxLatency returns the secondary provider max latency SLA if configured
func (s *Site) GetSecondaryMaxLatency() *int {
	return s.SLA.Secondary.MaxLatency
}

type SitesConfig struct {
	Sites []Site `yaml:"sites"`
}

type SiteStatus struct {
	SiteID           string    `json:"site_id"`
	PrimaryOnline    bool      `json:"primary_online"`
	SecondaryOnline  bool      `json:"secondary_online"`
	BothOnline       bool      `json:"both_online"`
	PrimaryLatency   *float64  `json:"primary_latency,omitempty"`   // ms
	SecondaryLatency *float64  `json:"secondary_latency,omitempty"` // ms
	LastCheck        time.Time `json:"last_check"`
	PrimaryError     string    `json:"primary_error,omitempty"`
	SecondaryError   string    `json:"secondary_error,omitempty"`
}

// PingLog represents a single ping check log entry
type PingLog struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	SiteID    string    `json:"site_id"`
	SiteName  string    `json:"site_name"`
	Target    string    `json:"target"` // "primary" or "secondary"
	IP        string    `json:"ip"`
	Success   bool      `json:"success"`
	Latency   *float64  `json:"latency,omitempty"`
	Error     string    `json:"error,omitempty"`
	
	// Extended ping statistics
	PacketsSent      int      `json:"packets_sent"`
	PacketsRecv      int      `json:"packets_recv"`
	PacketsDuplicates int     `json:"packets_duplicates"`
	PacketLoss       *float64 `json:"packet_loss,omitempty"`
	MinLatency       *float64 `json:"min_latency,omitempty"`
	MaxLatency       *float64 `json:"max_latency,omitempty"`
	Jitter           *float64 `json:"jitter,omitempty"`
}

type PingResult struct {
	SiteID    string
	IP        string
	LineType  string // "primary" | "secondary"
	Success   bool
	Latency   *float64 // Milliseconds (AvgRtt)
	Error     string
	Timestamp time.Time
	
	// Extended ping statistics
	PacketsSent      int      // Number of packets sent
	PacketsRecv      int      // Number of packets received  
	PacketsDuplicates int     // Number of duplicate packets received
	PacketLoss       *float64 // Packet loss percentage (0-100)
	MinLatency       *float64 // Minimum RTT in milliseconds
	MaxLatency       *float64 // Maximum RTT in milliseconds  
	Jitter           *float64 // Standard deviation (jitter) in milliseconds
}

type OverviewData struct {
	TotalSites       int     `json:"total_sites"`
	OnlineSites      int     `json:"online_sites"`
	OfflineSites     int     `json:"offline_sites"`
	DegradedSites    int     `json:"degraded_sites"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalChecks      int64   `json:"total_checks"`
	Uptime           string  `json:"uptime"`
}

type DashboardData struct {
	Sites    []Site
	Overview OverviewData
}

type SiteStatistics struct {
	// Current latencies
	CurrentLatencyPrimary    *float64 `json:"current_latency_primary"`
	CurrentLatencySecondary  *float64 `json:"current_latency_secondary"`
	MeanLatencyPrimary       float64  `json:"mean_latency_primary"`
	MeanLatencySecondary     float64  `json:"mean_latency_secondary"`
	
	// Extended latency statistics
	MinLatencyPrimary        float64  `json:"min_latency_primary"`
	MinLatencySecondary      float64  `json:"min_latency_secondary"`
	MaxLatencyPrimary        float64  `json:"max_latency_primary"`
	MaxLatencySecondary      float64  `json:"max_latency_secondary"`
	JitterPrimary            float64  `json:"jitter_primary"`           // Standard deviation
	JitterSecondary          float64  `json:"jitter_secondary"`         // Standard deviation
	
	// Packet statistics
	PacketsReceivedPrimary   int      `json:"packets_received_primary"`
	PacketsReceivedSecondary int      `json:"packets_received_secondary"`
	TotalPacketsPrimary      int      `json:"total_packets_primary"`
	TotalPacketsSecondary    int      `json:"total_packets_secondary"`
	PacketLossPrimary        float64  `json:"packet_loss_primary"`      // Percentage
	PacketLossSecondary      float64  `json:"packet_loss_secondary"`    // Percentage
	DuplicatePacketsPrimary  int      `json:"duplicate_packets_primary"`
	DuplicatePacketsSecondary int     `json:"duplicate_packets_secondary"`
	
	// Uptime statistics by timeframe
	Uptime24h                float64  `json:"uptime_24h"`
	Uptime7d                 float64  `json:"uptime_7d"`
	Uptime12m                float64  `json:"uptime_12m"`
	
	// Provider-specific uptime (24h)
	UptimePrimary            float64  `json:"uptime_primary"`
	UptimeSecondary          float64  `json:"uptime_secondary"`
	PrimaryUptime24h         float64  `json:"primary_uptime_24h"`
	SecondaryUptime24h       float64  `json:"secondary_uptime_24h"`
	
	// Provider-specific uptime (7d)
	PrimaryUptime7d          float64  `json:"primary_uptime_7d"`
	SecondaryUptime7d        float64  `json:"secondary_uptime_7d"`
	
	// Provider-specific uptime (12m)
	PrimaryUptime12m         float64  `json:"primary_uptime_12m"`
	SecondaryUptime12m       float64  `json:"secondary_uptime_12m"`
	
	// Performance statistics
	AvgLatency               float64  `json:"avg_latency"`
	MinLatency               float64  `json:"min_latency"`
	MaxLatency               float64  `json:"max_latency"`
	SuccessRate              float64  `json:"success_rate"`
	TotalChecks              int      `json:"total_checks"`
	
	// Incident tracking
	LastIncident             string   `json:"last_incident"`
	LastIncidentDuration     string   `json:"last_incident_duration"`
}

type ChartData struct {
	// Latency timeline (24h)
	LatencyChartLabels        []string  `json:"latency_labels"`
	LatencyChartDataPrimary   []float64 `json:"latency_primary"`
	LatencyChartDataSecondary []float64 `json:"latency_secondary"`

	// Uptime overview (7d)
	UptimeChartLabels        []string  `json:"uptime_labels"`
	UptimeChartData          []float64 `json:"uptime_data"`
	UptimeChartDataPrimary   []float64 `json:"uptime_primary"`
	UptimeChartDataSecondary []float64 `json:"uptime_secondary"`

	// SLA comparison (12m)
	SLAChartLabels        []string  `json:"sla_labels"`
	SLAChartDataPrimary   []float64 `json:"sla_primary"`
	SLAChartDataSecondary []float64 `json:"sla_secondary"`

	// Response time distribution (24h)
	DistributionChartLabels   []string  `json:"distribution_labels"`
	DistributionChartData     []float64 `json:"distribution_data"`
	DistributionPrimaryData   []float64 `json:"distribution_primary"`
	DistributionSecondaryData []float64 `json:"distribution_secondary"`

	// Yearly SLA tracking (365d)
	YearlyUptimeLabels        []string  `json:"yearly_labels"`
	YearlyUptimeData          []float64 `json:"yearly_data"`
	YearlyUptimeDataPrimary   []float64 `json:"yearly_primary"`
	YearlyUptimeDataSecondary []float64 `json:"yearly_secondary"`
	
	// Extended Ping Data Charts
	PacketLossChartLabels        []string  `json:"packet_loss_chart_labels"`
	PacketLossChartDataPrimary   []float64 `json:"packet_loss_chart_data_primary"`
	PacketLossChartDataSecondary []float64 `json:"packet_loss_chart_data_secondary"`
	
	JitterChartLabels        []string  `json:"jitter_chart_labels"`
	JitterChartDataPrimary   []float64 `json:"jitter_chart_data_primary"`
	JitterChartDataSecondary []float64 `json:"jitter_chart_data_secondary"`
	
	LatencyMinMaxChartLabels        []string    `json:"latency_minmax_chart_labels"`
	LatencyMinChartDataPrimary      []float64   `json:"latency_min_chart_data_primary"`
	LatencyMaxChartDataPrimary      []float64   `json:"latency_max_chart_data_primary"`
	LatencyMinChartDataSecondary    []float64   `json:"latency_min_chart_data_secondary"`
	LatencyMaxChartDataSecondary    []float64   `json:"latency_max_chart_data_secondary"`
}

type RecentEvent struct {
	Timestamp time.Time
	Status    string
	Message   string
	SiteID    string
	Target    string
	IsOutage  bool
}

type TestResult struct {
	Success       bool     `json:"success"`
	LatencyPrimary   *float64 `json:"latency_primary,omitempty"`
	LatencySecondary *float64 `json:"latency_secondary,omitempty"`
	ErrorPrimary     string   `json:"error_primary,omitempty"`
	ErrorSecondary   string   `json:"error_secondary,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Global application state
type AppState struct {
	Config      Config
	Sites       []Site
	SiteStatus  map[string]*SiteStatus
	PingLogs    []PingLog // Deprecated: kept for compatibility, use storage instead
	LogCounter  int       // Deprecated: kept for compatibility, use storage instead
	Storage     interface{} // Storage backend (memory or SQLite) - temporarily using interface{}
	Mu          sync.RWMutex
	StartTime   time.Time
	TotalChecks int64
	ResultChan  chan PingResult
}

// Prometheus metrics
type Metrics struct {
	PingSuccessCounter prometheus.CounterVec
	PingLatencyGauge   prometheus.GaugeVec
	TotalSitesGauge    prometheus.Gauge
	OnlineSitesGauge   prometheus.Gauge
}

// Authentication configuration structs
type AuthConfig struct {
	Enabled bool          `yaml:"enabled"`                 // Enable/disable authentication
	UI      UIAuthConfig  `yaml:"ui,omitempty"`           // UI authentication settings
	API     APIAuthConfig `yaml:"api,omitempty"`          // API authentication settings
}

// UIAuthConfig defines UI session-based authentication
type UIAuthConfig struct {
	Secret       string `yaml:"secret"`                     // Session secret for UI access
	SessionName  string `yaml:"session_name,omitempty"`     // Cookie name for UI sessions
	ExpiresHours int    `yaml:"expires_hours,omitempty"`    // Session expiration in hours
}

// APIAuthConfig defines API token-based authentication
type APIAuthConfig struct {
	Tokens []APIToken `yaml:"tokens,omitempty"`           // List of API tokens
}

// APIToken represents an API access token with permissions
type APIToken struct {
	Token       string    `yaml:"token"`                   // The actual token value
	Name        string    `yaml:"name"`                    // Human-readable name/description
	Permissions []string  `yaml:"permissions,omitempty"`   // Permissions (read, test, admin)
	Expires     *string   `yaml:"expires,omitempty"`       // Expiration date (YYYY-MM-DD format)
	Created     time.Time `yaml:"created,omitempty"`       // Creation timestamp
}

// TokenPermission defines available permissions
type TokenPermission string

const (
	PermissionMetrics TokenPermission = "metrics" // Metrics access only (/metrics, /health)
	PermissionRead    TokenPermission = "read"    // Read access to API endpoints
	PermissionTest    TokenPermission = "test"    // Test/debug endpoints
	PermissionAdmin   TokenPermission = "admin"   // Administrative endpoints
)

// HasPermission checks if token has specific permission
func (t *APIToken) HasPermission(permission TokenPermission) bool {
	for _, p := range t.Permissions {
		perm := TokenPermission(p)
		// Direct permission match
		if perm == permission {
			return true
		}
		// Admin permission grants access to everything
		if perm == PermissionAdmin {
			return true
		}
	}
	return false
}

// IsExpired checks if token is expired
func (t *APIToken) IsExpired() bool {
	if t.Expires == nil {
		return false
	}
	
	expireDate, err := time.Parse("2006-01-02", *t.Expires)
	if err != nil {
		return true // If we can't parse, consider expired
	}
	
	return time.Now().After(expireDate)
}