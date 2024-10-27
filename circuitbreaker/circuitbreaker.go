package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type circuitBreakerState = string
type operation = func() (any, error)

const (
	closed   = "closed"
	open     = "open"
	halfOpen = "half-open"
)

type CircuitBreaker struct {
	mu sync.Mutex
	// Current state
	state circuitBreakerState

	// Count of consecutive failures, zeroed out on success
	failureCount int
	// Time record of the last failure
	lastFailureTime time.Time

	// Count of successful requests in `half-open` state
	successCount int
	// Number of consecutive failures before transitioning to `open` state
	failureThreshold int

	// Time interval before transitioning from `open` to `half-open` state
	recoveryTime time.Duration
	// Count of successful requests for transitioning to `close` state
	halfOpenThreshold int
	// Time interval request has to complete successfully
	timeout time.Duration
}

func NewCircuitBreaker(
	failureThreshold, halfOpenThreshold int,
	recoveryTime, timeout time.Duration,
) *CircuitBreaker {
	return &CircuitBreaker{
		state:             closed,
		failureThreshold:  failureThreshold,
		recoveryTime:      recoveryTime,
		halfOpenThreshold: halfOpenThreshold,
		timeout:           timeout,
	}
}

func (cb *CircuitBreaker) Call(fn operation) (any, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	slog.Debug("call", "state", cb.state)

	switch cb.state {
	case closed:
		// Healthy state, all requests are allowed
		return cb.processClosedState(fn)
	case open:
		// Faulty state, all requests are blocked
		return cb.processOpenState()
	case halfOpen:
		// Recovering state, allows limited requests
		return cb.processHalfOpenState(fn)
	default:
		return nil, errors.New(fmt.Sprintf("unknown state `%s`", cb.state))
	}
}

func (cb *CircuitBreaker) processClosedState(fn operation) (any, error) {
	// Attempt to run operation with `cb.timeout` timeout
	res, err := cb.runWithTimeout(fn)
	if err != nil {
		// Operation is timing out, start state transition checks
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		slog.Debug("request failed", "count", cb.failureCount, "state", "closed")

		// If we got more failures than threshold allows transition to open state.
		if cb.failureCount >= cb.failureThreshold {
			slog.Info("state transitioning to `open`", "state", "closed")
			cb.state = open
		}

		return nil, err
	}

	return res, nil
}

func (cb *CircuitBreaker) resetCircuit() {
	cb.failureCount = 0
	cb.state = closed
}

// processOpenState blocks all requests
func (cb *CircuitBreaker) processOpenState() (any, error) {
	// If time threshold since the last failure passed transition state to half open.
	if time.Since(cb.lastFailureTime) > cb.recoveryTime {
		slog.Info("state transitioning to `half-open`", "state", "open")
		cb.state = halfOpen
		cb.halfOpenThreshold = 0
		cb.failureCount = 0
		return nil, nil
	}

	// Not enough time passed since the last failure.
	return nil, errors.New("open state; request blocked")
}

// processHalfOpenState attempts to execute the operation and verifies eligibility
// for recovery.
func (cb *CircuitBreaker) processHalfOpenState(fn operation) (any, error) {
	res, err := cb.runWithTimeout(fn)
	if err != nil {
		// Operation is still failing, transition back to `open` state
		slog.Info("state transitioning to `open`", "state", "half-open")
		cb.state = open
		cb.lastFailureTime = time.Now()
		return nil, err
	}

	// Recovering is starting
	slog.Debug("successfull operation", "state", "half-open")
	cb.successCount++

	if cb.successCount >= cb.halfOpenThreshold {
		slog.Info("state transtioning to `closed`", "state", "half-open")
		cb.resetCircuit()
	}

	return res, nil
}

func (cb *CircuitBreaker) runWithTimeout(fn operation) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cb.timeout)
	defer cancel()

	type Message struct {
		result any
		err    error
	}

	resChan := make(chan Message, 1)

	go func() {
		res, err := fn()
		resChan <- Message{res, err}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("request timed out")
	case res := <-resChan:
		return res.result, res.err
	}
}
