package ping

import (
	"fmt"
	"sync"
	"time"
	
	"sitewatch/internal/config"
	"sitewatch/internal/logger"
)

// CircuitBreakerManager manages circuit breakers for ping operations
type CircuitBreakerManager struct {
	breakers   map[string]*CircuitBreaker
	mu         sync.RWMutex
	maxFailures int
	resetTimeout time.Duration
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(maxFailures int, resetTimeout time.Duration) *CircuitBreakerManager {
	manager := &CircuitBreakerManager{
		breakers:     make(map[string]*CircuitBreaker),
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
	
	return manager
}

// GetBreaker returns a circuit breaker for the given site and line type
func (cbm *CircuitBreakerManager) GetBreaker(siteID, lineType string) *CircuitBreaker {
	key := fmt.Sprintf("%s-%s", siteID, lineType)
	
	cbm.mu.RLock()
	breaker, exists := cbm.breakers[key]
	cbm.mu.RUnlock()
	
	if exists {
		return breaker
	}
	
	// Create new breaker
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	// Double-check pattern
	if breaker, exists := cbm.breakers[key]; exists {
		return breaker
	}
	
	name := fmt.Sprintf("%s/%s", siteID, lineType)
	breaker = NewCircuitBreaker(name, cbm.maxFailures, cbm.resetTimeout)
	
	// Set state change callback for metrics
	breaker.SetOnStateChange(func(name string, from, to CircuitBreakerState) {
		log := logger.Default().WithComponent("circuit-breaker")
		log.Info("Circuit breaker state changed",
			"name", name,
			"from", stateToString(from),
			"to", stateToString(to))
		
		// Update Prometheus metrics
		switch to {
		case StateClosed:
			config.CircuitBreakerStateGauge.WithLabelValues(siteID, lineType).Set(0)
		case StateHalfOpen:
			config.CircuitBreakerStateGauge.WithLabelValues(siteID, lineType).Set(1)
		case StateOpen:
			config.CircuitBreakerStateGauge.WithLabelValues(siteID, lineType).Set(2)
		}
		
		config.CircuitBreakerTripsTotal.WithLabelValues(siteID, lineType, stateToString(to)).Inc()
	})
	
	cbm.breakers[key] = breaker
	
	log := logger.Default().WithComponent("circuit-breaker")
	log.Info("Created circuit breaker", 
		"name", name,
		"max_failures", cbm.maxFailures,
		"reset_timeout", cbm.resetTimeout)
	
	return breaker
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() map[string]CircuitBreakerStats {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	stats := make(map[string]CircuitBreakerStats, len(cbm.breakers))
	for key, breaker := range cbm.breakers {
		stats[key] = CircuitBreakerStats{
			Name:     breaker.name,
			State:    breaker.GetState(),
			Failures: breaker.GetFailures(),
		}
	}
	
	return stats
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	Name     string               `json:"name"`
	State    CircuitBreakerState  `json:"state"`
	Failures int                  `json:"failures"`
}

// stateToString converts circuit breaker state to string
func stateToString(state CircuitBreakerState) string {
	switch state {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Global circuit breaker manager instance
var globalCircuitBreakerManager *CircuitBreakerManager
var cbManagerOnce sync.Once

// GetGlobalCircuitBreakerManager returns the global circuit breaker manager instance
func GetGlobalCircuitBreakerManager() *CircuitBreakerManager {
	cbManagerOnce.Do(func() {
		// Default configuration: 3 failures within 30 seconds opens the circuit for 60 seconds
		globalCircuitBreakerManager = NewCircuitBreakerManager(3, 60*time.Second)
	})
	return globalCircuitBreakerManager
}