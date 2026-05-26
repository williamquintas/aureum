package circuitbreaker

import (
	"time"

	"github.com/sony/gobreaker"
)

type Config struct {
	Name        string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
	ReadyToTrip func(counts gobreaker.Counts) bool
}

func NewCircuitBreaker(config Config) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: config.ReadyToTrip,
	})
}

func DefaultConfig(name string) Config {
	var maxReq uint32 = 3
	return Config{
		Name:        name,
		MaxRequests: maxReq,
		Interval:    30 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}
}

func ExecuteWithFallback[T any](cb *gobreaker.CircuitBreaker,
	fn func() (T, error), fallback func() (T, error),
) (T, error) {
	result, err := cb.Execute(func() (interface{}, error) {
		return fn()
	})
	if err != nil {
		return fallback()
	}
	return result.(T), nil
}
