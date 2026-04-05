package util

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token-bucket rate limiter.
type RateLimiter struct {
	mu   sync.Mutex
	rate time.Duration // minimum interval between requests
	last time.Time
}

// NewRateLimiter creates a limiter allowing `perSecond` operations per second.
func NewRateLimiter(perSecond float64) *RateLimiter {
	interval := time.Duration(float64(time.Second) / perSecond)
	return &RateLimiter{rate: interval}
}

// Wait blocks until the next request is allowed.
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if elapsed := now.Sub(rl.last); elapsed < rl.rate {
		time.Sleep(rl.rate - elapsed)
	}
	rl.last = time.Now()
}

// Shared rate limiters for external APIs
var (
	jikanLimiter     *RateLimiter
	jikanLimiterOnce sync.Once

	anilistLimiter     *RateLimiter
	anilistLimiterOnce sync.Once
)

// GetJikanLimiter returns the shared Jikan API rate limiter (3 req/s).
func GetJikanLimiter() *RateLimiter {
	jikanLimiterOnce.Do(func() {
		jikanLimiter = NewRateLimiter(2.0) // well below 3/s to avoid 429s
	})
	return jikanLimiter
}

// GetAniListLimiter returns the shared AniList API rate limiter (10 req/s).
func GetAniListLimiter() *RateLimiter {
	anilistLimiterOnce.Do(func() {
		anilistLimiter = NewRateLimiter(9.0) // slightly below 10/s
	})
	return anilistLimiter
}
