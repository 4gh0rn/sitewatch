package config

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
	"sitewatch/internal/storage"
)

// Prometheus metrics - exported for use by other packages
var (
	PingChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_checks_total",
			Help: "Total number of ping checks performed",
		},
		[]string{"site_id", "line_type", "success"},
	)

	PingLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ping_latency_seconds",
			Help:    "Histogram of ping latencies in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"site_id", "line_type"},
	)

	SiteStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_status",
			Help: "Current status of site lines (1=online, 0=offline)",
		},
		[]string{"site_id", "line_type"},
	)

	SiteBothOnlineGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_both_lines_online",
			Help: "Both lines online status for site (1=both online, 0=at least one offline)",
		},
		[]string{"site_id"},
	)

	SiteInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "site_info",
			Help: "Site information with labels",
		},
		[]string{"site_id", "name", "location"},
	)
	
	// Extended ping metrics
	PacketLossGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ping_packet_loss_percentage",
			Help: "Packet loss percentage for site lines",
		},
		[]string{"site_id", "line_type"},
	)
	
	JitterHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ping_jitter_seconds",
			Help:    "Histogram of ping jitter (standard deviation) in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
		[]string{"site_id", "line_type"},
	)
	
	PacketsSentCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_packets_sent_total",
			Help: "Total number of ping packets sent",
		},
		[]string{"site_id", "line_type"},
	)
	
	PacketsReceivedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_packets_received_total", 
			Help: "Total number of ping packets received",
		},
		[]string{"site_id", "line_type"},
	)
	
	PacketsDuplicatesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ping_packets_duplicates_total",
			Help: "Total number of duplicate ping packets received",
		},
		[]string{"site_id", "line_type"},
	)
	
	// Application performance metrics
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)
	
	ActiveConnectionsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
		[]string{"type"},
	)
	
	MemoryUsageGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
		[]string{"type"},
	)
	
	GoroutinesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "goroutines_total",
			Help: "Number of goroutines currently running",
		},
		[]string{},
	)
	
	// Circuit breaker metrics
	CircuitBreakerStateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"site_id", "line_type"},
	)
	
	CircuitBreakerTripsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_trips_total",
			Help: "Total number of circuit breaker state transitions",
		},
		[]string{"site_id", "line_type", "to_state"},
	)
)

// AppState represents the global application state - exported for use by other packages
type AppState struct {
	Config      models.Config
	Sites       []models.Site
	SiteStatus  map[string]*models.SiteStatus
	Storage     storage.Storage
	Mu          sync.RWMutex // Protects Sites, SiteStatus maps
	StartTime   time.Time
	TotalChecks int64 // Use atomic operations for this field
	ResultChan  chan models.PingResult
}

// Global application state instance
var GlobalAppState *AppState

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(PingChecksTotal)
	prometheus.MustRegister(PingLatencyHistogram)
	prometheus.MustRegister(SiteStatusGauge)
	prometheus.MustRegister(SiteBothOnlineGauge)
	prometheus.MustRegister(SiteInfoGauge)
	
	// Register extended ping metrics
	prometheus.MustRegister(PacketLossGauge)
	prometheus.MustRegister(JitterHistogram)
	prometheus.MustRegister(PacketsSentCounter)
	prometheus.MustRegister(PacketsReceivedCounter)
	prometheus.MustRegister(PacketsDuplicatesCounter)
	
	// Register application performance metrics
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(ActiveConnectionsGauge)
	prometheus.MustRegister(MemoryUsageGauge)
	prometheus.MustRegister(GoroutinesGauge)
	
	// Register circuit breaker metrics
	prometheus.MustRegister(CircuitBreakerStateGauge)
	prometheus.MustRegister(CircuitBreakerTripsTotal)
}

// InitStorage initializes the storage backend
func (app *AppState) InitStorage() error {
	storage, err := storage.CreateStorage(app.Config)
	if err != nil {
		return err
	}
	app.Storage = storage
	
	log := logger.Default().WithComponent("config")
	log.Info("Storage initialized", "type", app.Config.Storage.Type)
	return nil
}

// InitializeSiteStatus initializes status tracking for all sites
func (app *AppState) InitializeSiteStatus() {
	app.Mu.Lock()
	defer app.Mu.Unlock()

	if app.SiteStatus == nil {
		app.SiteStatus = make(map[string]*models.SiteStatus)
	}

	for _, site := range app.Sites {
		app.SiteStatus[site.ID] = &models.SiteStatus{
			SiteID:           site.ID,
			PrimaryOnline:    false,
			SecondaryOnline:  false,
			BothOnline:       false,
			LastCheck:        time.Now(),
		}

		// Initialize Prometheus metrics
		SiteInfoGauge.WithLabelValues(site.ID, site.Name, site.Location).Set(1)
		SiteStatusGauge.WithLabelValues(site.ID, "primary").Set(0)
		SiteStatusGauge.WithLabelValues(site.ID, "secondary").Set(0)
		SiteBothOnlineGauge.WithLabelValues(site.ID).Set(0)
	}
}

