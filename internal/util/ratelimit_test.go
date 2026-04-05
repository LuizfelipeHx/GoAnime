package util

import (
	"fmt"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// RateLimiter
// ---------------------------------------------------------------------------

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10.0) // 10 req/s -> 100ms interval
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	if rl.rate != 100*time.Millisecond {
		t.Errorf("rate = %v, want %v", rl.rate, 100*time.Millisecond)
	}
}

func TestRateLimiterEnforcesMinInterval(t *testing.T) {
	// 5 req/s = 200ms minimum interval
	rl := NewRateLimiter(5.0)

	// First call should not block.
	start := time.Now()
	rl.Wait()
	firstDur := time.Since(start)
	if firstDur > 50*time.Millisecond {
		t.Errorf("first Wait() took %v, expected near-instant", firstDur)
	}

	// Second call should block ~200ms.
	start = time.Now()
	rl.Wait()
	secondDur := time.Since(start)
	if secondDur < 150*time.Millisecond {
		t.Errorf("second Wait() took only %v, expected >= ~200ms", secondDur)
	}
	if secondDur > 400*time.Millisecond {
		t.Errorf("second Wait() took %v, expected <= ~300ms", secondDur)
	}
}

func TestRateLimiterAllowsAfterInterval(t *testing.T) {
	rl := NewRateLimiter(10.0) // 100ms interval
	rl.Wait()
	// Wait longer than the rate interval.
	time.Sleep(150 * time.Millisecond)

	start := time.Now()
	rl.Wait()
	dur := time.Since(start)
	if dur > 50*time.Millisecond {
		t.Errorf("Wait() after sleeping took %v, expected near-instant", dur)
	}
}

func TestGetJikanLimiter(t *testing.T) {
	rl := GetJikanLimiter()
	if rl == nil {
		t.Fatal("GetJikanLimiter() returned nil")
	}
	// Should return the same instance on subsequent calls
	rl2 := GetJikanLimiter()
	if rl != rl2 {
		t.Error("GetJikanLimiter() returned different instances")
	}
}

func TestGetAniListLimiter(t *testing.T) {
	rl := GetAniListLimiter()
	if rl == nil {
		t.Fatal("GetAniListLimiter() returned nil")
	}
	rl2 := GetAniListLimiter()
	if rl != rl2 {
		t.Error("GetAniListLimiter() returned different instances")
	}
}

// ---------------------------------------------------------------------------
// Retry
// ---------------------------------------------------------------------------

func TestRetrySuccessOnFirstAttempt(t *testing.T) {
	callCount := 0
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	result, err := Retry(cfg, func() (string, error) {
		callCount++
		return "ok", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetrySuccessOnSecondAttempt(t *testing.T) {
	callCount := 0
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	result, err := Retry(cfg, func() (int, error) {
		callCount++
		if callCount < 2 {
			return 0, fmt.Errorf("temporary error")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestRetryAllAttemptsFail(t *testing.T) {
	callCount := 0
	cfg := RetryConfig{
		MaxRetries:  2,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	result, err := Retry(cfg, func() (string, error) {
		callCount++
		return "", fmt.Errorf("error #%d", callCount)
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if result != "" {
		t.Errorf("result = %q, want empty", result)
	}
	// MaxRetries=2 means 3 total attempts (0, 1, 2)
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryZeroRetries(t *testing.T) {
	callCount := 0
	cfg := RetryConfig{
		MaxRetries:  0,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	_, err := Retry(cfg, func() (string, error) {
		callCount++
		return "", fmt.Errorf("fail")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (only initial attempt)", callCount)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 2 {
		t.Errorf("MaxRetries = %d, want 2", cfg.MaxRetries)
	}
	if cfg.InitialWait != 500*time.Millisecond {
		t.Errorf("InitialWait = %v, want 500ms", cfg.InitialWait)
	}
	if cfg.MaxWait != 4*time.Second {
		t.Errorf("MaxWait = %v, want 4s", cfg.MaxWait)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", cfg.Multiplier)
	}
}

func TestRetryBackoffCapped(t *testing.T) {
	callCount := 0
	cfg := RetryConfig{
		MaxRetries:  5,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     3 * time.Millisecond,
		Multiplier:  10.0, // aggressive multiplier
	}

	start := time.Now()
	_, _ = Retry(cfg, func() (string, error) {
		callCount++
		return "", fmt.Errorf("fail")
	})
	elapsed := time.Since(start)

	if callCount != 6 {
		t.Errorf("callCount = %d, want 6", callCount)
	}
	// With MaxWait=3ms and 5 retries, total sleep should be well under 100ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("retries took %v, expected much less due to MaxWait cap", elapsed)
	}
}
