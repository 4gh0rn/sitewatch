package middleware

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"sitewatch/internal/config"
	"sitewatch/internal/logger"
)

// MetricsMiddleware collects HTTP request metrics
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Increment active connections
		config.ActiveConnectionsGauge.WithLabelValues("http").Inc()
		defer config.ActiveConnectionsGauge.WithLabelValues("http").Dec()
		
		// Process request
		err := c.Next()
		
		// Calculate request duration
		duration := time.Since(start).Seconds()
		
		// Get status code
		status := c.Response().StatusCode()
		statusStr := strconv.Itoa(status)
		
		// Record metrics
		config.HTTPRequestsTotal.WithLabelValues(
			c.Method(),
			c.Route().Path,
			statusStr,
		).Inc()
		
		config.HTTPRequestDuration.WithLabelValues(
			c.Method(),
			c.Route().Path,
		).Observe(duration)
		
		return err
	}
}

// UpdateSystemMetrics updates system-wide metrics periodically
func UpdateSystemMetrics() {
	log := logger.Default().WithComponent("metrics")
	
	// Update memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	config.MemoryUsageGauge.WithLabelValues("alloc").Set(float64(memStats.Alloc))
	config.MemoryUsageGauge.WithLabelValues("total_alloc").Set(float64(memStats.TotalAlloc))
	config.MemoryUsageGauge.WithLabelValues("sys").Set(float64(memStats.Sys))
	config.MemoryUsageGauge.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
	config.MemoryUsageGauge.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
	config.MemoryUsageGauge.WithLabelValues("stack_inuse").Set(float64(memStats.StackInuse))
	config.MemoryUsageGauge.WithLabelValues("stack_sys").Set(float64(memStats.StackSys))
	
	// Update goroutine count
	numGoroutines := runtime.NumGoroutine()
	config.GoroutinesGauge.WithLabelValues().Set(float64(numGoroutines))
	
	log.Debug("System metrics updated",
		"mem_alloc_mb", float64(memStats.Alloc)/1024/1024,
		"mem_sys_mb", float64(memStats.Sys)/1024/1024,
		"heap_alloc_mb", float64(memStats.HeapAlloc)/1024/1024,
		"goroutines", numGoroutines,
	)
}

// StartMetricsUpdater starts a goroutine that periodically updates system metrics
func StartMetricsUpdater(interval time.Duration) {
	log := logger.Default().WithComponent("metrics")
	log.Info("Starting metrics updater", "interval", interval)
	
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			UpdateSystemMetrics()
		}
	}()
}