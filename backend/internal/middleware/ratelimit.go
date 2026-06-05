// Package middleware contains HTTP middleware: JWT guide authentication and
// IP-based rate limiting.
package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is an IP-based token bucket rate limiter with a background
// cleanup goroutine tied to a context for graceful shutdown.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	r       rate.Limit
	b       int
}

// NewRateLimiter creates a rate limiter with r requests per second and burst
// size b. The cleanup goroutine stops when ctx is cancelled.
func NewRateLimiter(ctx context.Context, r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*entry),
		r:       r,
		b:       b,
	}
	go rl.cleanup(ctx)
	return rl
}

// Limit returns middleware that rate limits by client IP address.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !rl.limiterFor(ip).Allow() {
			http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if e, ok := rl.entries[ip]; ok {
		e.lastSeen = time.Now()
		return e.limiter
	}

	e := &entry{
		limiter:  rate.NewLimiter(rl.r, rl.b),
		lastSeen: time.Now(),
	}
	rl.entries[ip] = e
	return e.limiter
}

func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-10 * time.Minute)
			rl.mu.Lock()
			for ip, e := range rl.entries {
				if e.lastSeen.Before(cutoff) {
					delete(rl.entries, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}
