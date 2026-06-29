package circuitbreaker_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/pkg/circuitbreaker"
)

// CC-22: Healthy service test — circuit starts closed, allows requests, returns success.
func TestCC22_HealthyService(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:        "test-healthy",
		MaxRequests: 1,
		Interval:    time.Minute,
		Timeout:     10 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	// Circuit starts in closed state
	assert.Equal(t, gobreaker.StateClosed, cb.State(),
		"circuit breaker should start in closed state")

	// Allows requests through and returns success
	result, err := cb.Execute(func() (interface{}, error) {
		return "healthy result", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "healthy result", result)
}

// CC-23: Circuit opens after N consecutive failures.
func TestCC23_CircuitOpensAfterFailures(t *testing.T) {
	const tripThreshold uint32 = 3

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:        "test-open",
		MaxRequests: 1,
		Interval:    0,                // no interval reset
		Timeout:     30 * time.Second, // long enough to stay open for the test
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= tripThreshold
		},
	})

	// Initial state is closed
	require.Equal(t, gobreaker.StateClosed, cb.State())

	// Execute failing functions to trip the circuit
	for i := uint32(0); i < tripThreshold; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("service error")
		})
		assert.Error(t, err)
	}

	// Circuit should now be open
	assert.Equal(t, gobreaker.StateOpen, cb.State())

	// Any further call should fail immediately with ErrOpenState
	_, err := cb.Execute(func() (interface{}, error) {
		return "should not be called", nil
	})
	assert.ErrorIs(t, err, gobreaker.ErrOpenState,
		"open circuit should return ErrOpenState without calling the function")
}

// CC-24: Open circuit returns fallback/error immediately without calling the wrapped function.
func TestCC24_OpenCircuitReturnsFallbackImmediately(t *testing.T) {
	tripAfter := uint32(1)

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:        "test-fallback-immediate",
		MaxRequests: 1,
		Interval:    0,
		Timeout:     30 * time.Second, // stay open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= tripAfter
		},
	})

	// Trip the circuit by sending a failure
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, errors.New("service error")
	})
	require.Error(t, err)

	// Verify circuit is open
	require.Equal(t, gobreaker.StateOpen, cb.State())

	var callCount int32

	// Execute with fallback — the wrapped fn should NOT be called
	result, err := circuitbreaker.ExecuteWithFallback(cb,
		func() (string, error) {
			atomic.AddInt32(&callCount, 1)
			return "", errors.New("should not be called")
		},
		func() (string, error) {
			return "immediate fallback", nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "immediate fallback", result,
		"fallback result should be returned when circuit is open")
	assert.Equal(t, int32(0), atomic.LoadInt32(&callCount),
		"wrapped function must NOT be called when circuit is open")
}

// CC-25: Half-open to closed recovery — after timeout window, a successful request closes the
// circuit.
func TestCC25_HalfOpenToClosedRecovery(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:        "test-recovery",
		MaxRequests: 1,
		Interval:    0,
		Timeout:     50 * time.Millisecond, // short timeout for fast test
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})

	// Trip the circuit
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, errors.New("service error")
	})
	require.Error(t, err)
	require.Equal(t, gobreaker.StateOpen, cb.State())

	// Wait for the timeout to expire, transitioning the circuit to half-open
	time.Sleep(60 * time.Millisecond)

	// At this point the circuit should be half-open (ready to test a single request)
	// Execute a successful request — this should close the circuit
	result, err := cb.Execute(func() (interface{}, error) {
		return "recovered", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "recovered", result)

	// Circuit should now be closed again
	assert.Equal(t, gobreaker.StateClosed, cb.State(),
		"circuit should transition from half-open to closed after a successful request")

	// Verify normal operation resumes
	result2, err := cb.Execute(func() (interface{}, error) {
		return "normal operation", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "normal operation", result2)
}

// CC-26: Fallback function behavior — verify fallback data is returned when circuit is open, and
// that the fallback is bypassed when the circuit is closed and the function succeeds.
func TestCC26_FallbackFunctionBehavior(t *testing.T) {
	t.Run("fallback used when circuit is open", func(t *testing.T) {
		cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
			Name:        "test-fb-open",
			MaxRequests: 1,
			Interval:    0,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 1
			},
		})

		// Trip the circuit
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("service error")
		})
		require.Error(t, err)

		// ExecuteWithFallback should call fallback and return its data
		result, err := circuitbreaker.ExecuteWithFallback(cb,
			func() (string, error) {
				return "", errors.New("should not call")
			},
			func() (string, error) {
				return "fallback data", nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "fallback data", result)
	})

	t.Run("fallback NOT used when circuit is closed and function succeeds", func(t *testing.T) {
		cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
			Name:        "test-fb-closed-ok",
			MaxRequests: 1,
			Interval:    time.Minute,
			Timeout:     time.Minute,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures > 5
			},
		})

		require.Equal(t, gobreaker.StateClosed, cb.State())

		result, err := circuitbreaker.ExecuteWithFallback(cb,
			func() (string, error) {
				return "direct success", nil
			},
			func() (string, error) {
				return "fallback result", nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "direct success", result,
			"fallback should NOT be invoked when the direct function succeeds")
	})

	t.Run("fallback invoked when circuit is closed but function fails", func(t *testing.T) {
		cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
			Name:        "test-fb-closed-fail",
			MaxRequests: 1,
			Interval:    time.Minute,
			Timeout:     time.Minute,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				// Don't trip; we want to test closed + function error → fallback
				return false
			},
		})

		require.Equal(t, gobreaker.StateClosed, cb.State())

		result, err := circuitbreaker.ExecuteWithFallback(cb,
			func() (string, error) {
				return "", errors.New("function error")
			},
			func() (string, error) {
				return "fallback from fn error", nil
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "fallback from fn error", result,
			"fallback should be invoked when the function itself returns an error")
	})

	t.Run("fallback can return an error", func(t *testing.T) {
		cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
			Name:        "test-fb-error",
			MaxRequests: 1,
			Interval:    0,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 1
			},
		})

		// Trip the circuit
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("service error")
		})
		require.Error(t, err)

		// Both fn and fallback return errors
		result, err := circuitbreaker.ExecuteWithFallback(cb,
			func() (string, error) {
				return "", errors.New("should not call")
			},
			func() (string, error) {
				return "", errors.New("fallback also failed")
			},
		)
		require.Error(t, err)
		assert.Equal(t, "", result)
		assert.Contains(t, err.Error(), "fallback also failed")
	})
}

// TestDefaultConfig verifies that DefaultConfig returns sensible values.
func TestDefaultConfig(t *testing.T) {
	cfg := circuitbreaker.DefaultConfig("test-default")

	assert.Equal(t, "test-default", cfg.Name)
	assert.Equal(t, uint32(3), cfg.MaxRequests)
	assert.Equal(t, 30*time.Second, cfg.Interval)
	assert.Equal(t, 30*time.Second, cfg.Timeout)

	// Default trips after 5 consecutive failures
	require.NotNil(t, cfg.ReadyToTrip)
	assert.True(t, cfg.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 6}),
		"should trip with 6 consecutive failures")
	assert.False(t, cfg.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 5}),
		"should NOT trip with 5 consecutive failures (threshold is >5)")
	assert.False(t, cfg.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 0}))
}
