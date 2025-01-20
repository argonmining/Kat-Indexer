package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	mu       sync.Mutex
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		rl.mu.Lock()
		now := time.Now()

		// Remove old requests outside the window
		if reqs, exists := rl.requests[ip]; exists {
			var valid []time.Time
			for _, req := range reqs {
				if now.Sub(req) <= rl.window {
					valid = append(valid, req)
				}
			}
			rl.requests[ip] = valid
		}

		// Check if limit is exceeded
		if len(rl.requests[ip]) >= rl.limit {
			rl.mu.Unlock()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Add current request
		rl.requests[ip] = append(rl.requests[ip], now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
