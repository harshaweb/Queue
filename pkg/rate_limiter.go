package pkg

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	tokens     int64
	capacity   int64
	refillRate int64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// TokenBucket represents a token bucket rate limiter
type TokenBucket struct {
	*RateLimiter
}

// SlidingWindow represents a sliding window rate limiter
type SlidingWindow struct {
	requests []time.Time
	limit    int
	window   time.Duration
	mu       sync.Mutex
}

// RateLimiterConfig configures rate limiting behavior
type RateLimiterConfig struct {
	Type       string        `json:"type"`        // "token_bucket" or "sliding_window"
	Capacity   int64         `json:"capacity"`    // Max tokens or requests
	RefillRate int64         `json:"refill_rate"` // Tokens per second or requests per window
	Window     time.Duration `json:"window"`      // For sliding window
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		RateLimiter: &RateLimiter{
			tokens:     capacity,
			capacity:   capacity,
			refillRate: refillRate,
			lastRefill: time.Now(),
		},
	}
}

// Allow checks if a request can proceed
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if N requests can proceed
func (tb *TokenBucket) AllowN(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

// WaitN blocks until N tokens are available
func (tb *TokenBucket) WaitN(ctx context.Context, n int64) error {
	for {
		if tb.AllowN(n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// Continue waiting
		}
	}
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int64(elapsed.Seconds()) * rl.refillRate

	if tokensToAdd > 0 {
		rl.tokens = min(rl.capacity, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		requests: make([]time.Time, 0, limit*2),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request can proceed in sliding window
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Remove old requests
	validRequests := 0
	for i, reqTime := range sw.requests {
		if reqTime.After(cutoff) {
			sw.requests = sw.requests[i:]
			validRequests = len(sw.requests)
			break
		}
	}

	if validRequests == 0 {
		sw.requests = sw.requests[:0]
	}

	// Check if we can add this request
	if len(sw.requests) < sw.limit {
		sw.requests = append(sw.requests, now)
		return true
	}

	return false
}

// Wait blocks until a request can proceed in sliding window
func (sw *SlidingWindow) Wait(ctx context.Context) error {
	for {
		if sw.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// Continue waiting
		}
	}
}

// Stats returns rate limiter statistics
func (tb *TokenBucket) Stats() RateLimiterStats {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	return RateLimiterStats{
		Type:       "token_bucket",
		Capacity:   tb.capacity,
		Available:  tb.tokens,
		RefillRate: tb.refillRate,
	}
}

// Stats returns sliding window statistics
func (sw *SlidingWindow) Stats() RateLimiterStats {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)
	activeRequests := 0

	for _, reqTime := range sw.requests {
		if reqTime.After(cutoff) {
			activeRequests++
		}
	}

	return RateLimiterStats{
		Type:           "sliding_window",
		Capacity:       int64(sw.limit),
		Available:      int64(sw.limit - activeRequests),
		ActiveRequests: int64(activeRequests),
		Window:         sw.window,
	}
}

// RateLimiterStats provides rate limiter statistics
type RateLimiterStats struct {
	Type           string        `json:"type"`
	Capacity       int64         `json:"capacity"`
	Available      int64         `json:"available"`
	RefillRate     int64         `json:"refill_rate,omitempty"`
	ActiveRequests int64         `json:"active_requests,omitempty"`
	Window         time.Duration `json:"window,omitempty"`
}

// Helper function for min
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
