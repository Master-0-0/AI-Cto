package auth

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token-bucket rate limiter per key.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // max requests
	interval time.Duration // per interval window
}

type bucket struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
	}
}

// Allow returns true if the key is allowed to proceed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]

	if !ok || now.After(b.resetAt) {
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(rl.interval)}
		return true
	}

	b.count++
	if b.count > rl.rate {
		return false
	}

	return true
}

// ResetAfter returns the duration until the bucket resets for a key.
func (rl *RateLimiter) ResetAfter(key string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		return 0
	}
	remaining := time.Until(b.resetAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Cleanup removes expired buckets periodically.
func (rl *RateLimiter) Cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for k, b := range rl.buckets {
				if now.After(b.resetAt) {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
