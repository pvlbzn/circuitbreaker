package circuitbreaker

import (
	"errors"
	"math/rand"
	"testing"
	"time"
)

// makeService creates an unreliable service which fails with `failureRate` chance.
// Latency for execution ranges from `latencyFrom` to `latencyTo`
func makeService(latencyFrom, latencyTo, failureRate int) func() (any, error) {
	return func() (any, error) {
		duration := time.Duration(latencyFrom+rand.Intn(latencyTo-latencyFrom)) * time.Millisecond
		time.Sleep(duration)

		if rand.Intn(100) < failureRate {
			return nil, errors.New("service failed")
		}

		return "OK", nil
	}
}

func TestCallNeverFailing(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 2*time.Second, 2*time.Second)
	// Never failing service
	s := makeService(20, 200, 0)

	res, err := cb.Call(s)
	if err != nil {
		t.Errorf("service shouldn't fail, got `%s` error", err)
	}

	if res != "OK" {
		t.Errorf("service always returns `OK`, got `%s`", res)
	}
}

func TestOpenState(t *testing.T) {
	cb := NewCircuitBreaker(1, 2, 2*time.Second, 2*time.Second)
	// Always failing service
	s := makeService(20, 200, 100)

	_, err := cb.Call(s)
	if err == nil {
		t.Errorf("service should return error, got `nil`")
	}

	if cb.state != open {
		t.Errorf("circuit breaker should open on failure, got `%s`", cb.state)
	}
}

func TestHalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(1, 1, 1*time.Second, 1*time.Second)
	// Always failing service
	alwaysFailing := makeService(20, 200, 100)
	neverFailing := makeService(20, 200, 0)

	// Transition to closed state
	cb.Call(alwaysFailing)
	// Wait for 2 seconds
	time.Sleep(2 * time.Second)
	// Transition to half open state
	cb.Call(neverFailing)

	if cb.state != halfOpen {
		t.Errorf("state should move to half open, got `%s`", cb.state)
	}
}

func TestRecovery(t *testing.T) {
	cb := NewCircuitBreaker(1, 1, 1*time.Second, 1*time.Second)
	// Always failing service
	alwaysFailing := makeService(20, 200, 100)
	neverFailing := makeService(20, 200, 0)

	// Transition to closed state
	cb.Call(alwaysFailing)

	// Wait for 2 seconds
	time.Sleep(2 * time.Second)
	// Transition to half open state
	cb.Call(neverFailing)

	// Wait again
	time.Sleep(2 * time.Second)
	// Transition to closed state
	cb.Call(neverFailing)

	if cb.state != closed {
		t.Errorf("state should move to closed, got `%s`", cb.state)
	}
}
