package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// RateLimiter provides IP-based rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with the given limit per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}

	// Background cleanup every window duration
	go rl.cleanup()

	return rl
}

// Middleware returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		entry, exists := rl.entries[ip]
		now := time.Now()

		if !exists || now.After(entry.expiresAt) {
			rl.entries[ip] = &rateLimitEntry{
				count:     1,
				expiresAt: now.Add(rl.window),
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > rl.limit {
			rl.mu.Unlock()
			abortErrorJSON(c, http.StatusTooManyRequests, "rate limit exceeded, try again later")
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.entries {
			if now.After(entry.expiresAt) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}
