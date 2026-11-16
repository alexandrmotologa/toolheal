package executor

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/alexandrmotologa/toolheal/internal/config"
)

// CircuitState represents the current condition of the circuit breaker.
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateHalfOpen CircuitState = "HALF_OPEN"
	StateOpen     CircuitState = "OPEN"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open: upstream tool is currently failing")
)

// CircuitBreaker guards tool executions against cascading upstream failures.
type CircuitBreaker struct {
	mu                  sync.Mutex
	state               CircuitState
	consecutiveFailures int
	failureThreshold    int
	resetTimeout        time.Duration
	lastFailureTime     time.Time
	backoffBase         time.Duration
	backoffMax          time.Duration
}

// NewCircuitBreaker initializes a circuit breaker with config parameters.
func NewCircuitBreaker(cfg *config.CircuitBreakerConfig) *CircuitBreaker {
	if cfg == nil {
		cfg = &config.DefaultConfig().CircuitBreaker
	}

	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: cfg.MaxConsecutiveFailures,
		resetTimeout:     time.Duration(cfg.ResetTimeoutMs) * time.Millisecond,
		backoffBase:      time.Duration(cfg.BackoffBaseMs) * time.Millisecond,
		backoffMax:       time.Duration(cfg.BackoffMaxMs) * time.Millisecond,
	}
}

// Execute wraps a tool execution action with circuit breaker protection.
func (cb *CircuitBreaker) Execute(action func() (any, error)) (any, error) {
	cb.mu.Lock()

	// Check if state should transition from OPEN to HALF_OPEN
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return nil, ErrCircuitOpen
		}
	}

	cb.mu.Unlock()

	result, err := action()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.consecutiveFailures++
		cb.lastFailureTime = time.Now()

		if cb.consecutiveFailures >= cb.failureThreshold {
			cb.state = StateOpen
		}
		return nil, err
	}

	// Success: reset failures and close circuit
	if cb.state == StateHalfOpen || cb.consecutiveFailures > 0 {
		cb.state = StateClosed
		cb.consecutiveFailures = 0
	}

	return result, nil
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ComputeBackoff computes exponential backoff with jitter for a given retry attempt.
func (cb *CircuitBreaker) ComputeBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	factor := math.Pow(2, float64(attempt-1))
	backoffMs := float64(cb.backoffBase.Milliseconds()) * factor
	maxMs := float64(cb.backoffMax.Milliseconds())

	if backoffMs > maxMs {
		backoffMs = maxMs
	}

	// Add random jitter of +/- 20%
	jitter := (rand.Float64()*0.4 - 0.2) * backoffMs
	totalMs := backoffMs + jitter
	if totalMs < 0 {
		totalMs = 0
	}

	return time.Duration(totalMs) * time.Millisecond
}
