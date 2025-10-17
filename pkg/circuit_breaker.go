package pkg

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker provides fault tolerance and prevents cascade failures
type CircuitBreaker struct {
	name              string
	maxFailures       int
	resetTimeout      time.Duration
	consecutiveErrors int
	lastFailureTime   time.Time
	state             CircuitState
	mu                sync.RWMutex
	onStateChange     func(from, to CircuitState)
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	Name          string        `json:"name"`
	MaxFailures   int           `json:"max_failures"`  // Failures to trigger open state
	ResetTimeout  time.Duration `json:"reset_timeout"` // Time to wait before half-open
	OnStateChange func(from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures == 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout == 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.Name == "" {
		config.Name = "default"
	}

	return &CircuitBreaker{
		name:          config.Name,
		maxFailures:   config.MaxFailures,
		resetTimeout:  config.ResetTimeout,
		state:         CircuitClosed,
		onStateChange: config.OnStateChange,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if circuit should be half-open
	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.setState(CircuitHalfOpen)
	}

	// Reject if circuit is open
	if cb.state == CircuitOpen {
		return fmt.Errorf("circuit breaker '%s' is open", cb.name)
	}

	// Execute the function
	err := fn()
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// onSuccess resets failure count and closes circuit if half-open
func (cb *CircuitBreaker) onSuccess() {
	cb.consecutiveErrors = 0
	if cb.state == CircuitHalfOpen {
		cb.setState(CircuitClosed)
	}
}

// onFailure increments failure count and may open circuit
func (cb *CircuitBreaker) onFailure() {
	cb.consecutiveErrors++
	cb.lastFailureTime = time.Now()

	if cb.consecutiveErrors >= cb.maxFailures {
		cb.setState(CircuitOpen)
	}
}

// setState changes circuit state and notifies listeners
func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil && oldState != newState {
		go cb.onStateChange(oldState, newState)
	}
}

// State returns current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:              cb.name,
		State:             cb.state,
		ConsecutiveErrors: cb.consecutiveErrors,
		LastFailureTime:   cb.lastFailureTime,
	}
}

// CircuitBreakerStats provides circuit breaker statistics
type CircuitBreakerStats struct {
	Name              string       `json:"name"`
	State             CircuitState `json:"state"`
	ConsecutiveErrors int          `json:"consecutive_errors"`
	LastFailureTime   time.Time    `json:"last_failure_time"`
}

// String returns human-readable circuit state
func (cs CircuitState) String() string {
	switch cs {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}
