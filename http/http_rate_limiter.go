package http

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	requests map[string]*requestCount
	mu       sync.RWMutex
	rate     int
	per      time.Duration
}

type requestCount struct {
	count    int
	lastTime time.Time
}

func NewRateLimiter(rate int, per time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*requestCount),
		rate:     rate,
		per:      per,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if req, exists := rl.requests[ip]; exists {
		if now.Sub(req.lastTime) > rl.per {
			req.count = 1
			req.lastTime = now
			return true
		}
		if req.count < rl.rate {
			req.count++
			return true
		}
		return false
	}

	rl.requests[ip] = &requestCount{1, now}
	return true
}

func RateLimiterMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(r.RemoteAddr) {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
