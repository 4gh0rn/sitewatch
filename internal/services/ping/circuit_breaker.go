package ping

import (
	"sync"
	"time"
	
	"sitewatch/internal/logger"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreaker implements the circuit breaker pattern for ping operations
type CircuitBreaker struct {
	name           string
	maxFailures    int
	resetTimeout   time.Duration
	state          CircuitBreakerState
	failures       int
	lastFailTime   time.Time
	mu             sync.RWMutex
	onStateChange  func(name string, from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// SetOnStateChange sets a callback function for state changes
func (cb *CircuitBreaker) SetOnStateChange(fn func(name string, from, to CircuitBreakerState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// Call executes the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Call(fn func() error) error {
	log := logger.Default().WithComponent("circuit-breaker").WithSite(cb.name, "")
	
	if !cb.canExecute() {
		log.Warn("Circuit breaker is open, call blocked", 
			"state", cb.getStateString(),
			"failures", cb.getFailures())
		return &CircuitBreakerError{
			Name:  cb.name,
			State: cb.getStateString(),
		}
	}
	
	err := fn()
	
	if err != nil {
		cb.recordFailure()
		log.Debug("Circuit breaker recorded failure", 
			"error", err,
			"failures", cb.getFailures(),
			"state", cb.getStateString())
	} else {
		cb.recordSuccess()
		log.Debug("Circuit breaker recorded success", 
			"failures", cb.getFailures(),
			"state", cb.getStateString())
	}
	
	return err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			// Transition to half-open state
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.resetTimeout {
				cb.setState(StateHalfOpen)
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == StateHalfOpen
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure records a failure and potentially changes state
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures++
	cb.lastFailTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
	}
}

// recordSuccess records a success and potentially changes state
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case StateHalfOpen:
		cb.setState(StateClosed)
		cb.failures = 0
	case StateClosed:
		// Reset failure count on success in closed state
		if cb.failures > 0 {
			cb.failures = 0
		}
	}
}

// setState changes the state and notifies listeners
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	
	if cb.onStateChange != nil && oldState != newState {
		// Call callback without holding lock to prevent deadlocks
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

// GetState returns the current state (thread-safe)
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count (thread-safe)
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// getStateString returns state as string (must hold read lock)
func (cb *CircuitBreaker) getStateString() string {
	switch cb.state {
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

// getFailures returns failures (must hold read lock)
func (cb *CircuitBreaker) getFailures() int {
	return cb.failures
}

// CircuitBreakerError represents an error when circuit breaker is open
type CircuitBreakerError struct {
	Name  string
	State string
}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker '" + e.Name + "' is " + e.State
}