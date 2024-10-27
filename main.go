package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/pvlbzn/circuitbreaker/circuitbreaker"
)

// runUnreliableService runs a mock service which fails 33% of time
// with latency of `latencyFrom` to `latencyTo` ms.
func makeService(latencyFrom, latencyTo int) func() (any, error) {
	return func() (any, error) {
		duration := time.Duration(latencyFrom+rand.Intn(latencyTo-latencyFrom)) * time.Millisecond
		time.Sleep(duration)

		if rand.Intn(3) != 0 {
			return nil, errors.New("service failed")
		}

		return "OK", nil
	}
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	cb := circuitbreaker.NewCircuitBreaker(
		2,
		2,
		2*time.Second,
		2*time.Second,
	)
	service := makeService(20, 350)

	for i := 0; i < 50; i++ {
		res, err := cb.Call(service)
		if err != nil {
			slog.Debug("service request failed", "error", err)
		} else {
			slog.Debug("service request succeeded", "result", res)
		}

		time.Sleep(1 * time.Second)
		fmt.Println("-----------------------------")
	}
}
