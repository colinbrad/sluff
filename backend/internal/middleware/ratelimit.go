package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter is an IP-based token bucket rate limiter.
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

// NewRateLimiter creates a rate limiter with the given requests per second (r)
// and burst size (b).
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

// Limit returns middleware that rate limits by IP address.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := rl.limiterFor(ip)
		if !limiter.Allow() {
			http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, exists := rl.limiters[ip]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rl.r, rl.b)
	rl.limiters[ip] = limiter

	// Cleanup old limiters every 5 minutes
	if len(rl.limiters) == 1 {
		go rl.cleanup()
	}

	return limiter
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// Remove entries that haven't been used recently
		for ip, limiter := range rl.limiters {
			if limiter.AllowN(time.Now(), 0) {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}
