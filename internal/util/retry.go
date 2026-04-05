package util

import (
	"fmt"
	"time"
)

// RetryConfig holds configuration for retry logic.
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  2,
		InitialWait: 500 * time.Millisecond,
		MaxWait:     4 * time.Second,
		Multiplier:  2.0,
	}
}

// Retry executes fn with exponential backoff retries.
// Returns the result of the first successful call, or the last error.
func Retry[T any](cfg RetryConfig, fn func() (T, error)) (T, error) {
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < cfg.MaxRetries {
			Debug("Retrying after error", "attempt", attempt+1, "maxRetries", cfg.MaxRetries, "wait", wait, "error", err)
			time.Sleep(wait)
			wait = time.Duration(float64(wait) * cfg.Multiplier)
			if wait > cfg.MaxWait {
				wait = cfg.MaxWait
			}
		}
	}

	var zero T
	return zero, fmt.Errorf("all %d attempts failed: %w", cfg.MaxRetries+1, lastErr)
}
