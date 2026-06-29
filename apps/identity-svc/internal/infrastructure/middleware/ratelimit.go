package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements Redis-backed rate limiting per IP address.
type RateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	prefix string
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(client *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
		prefix: "ratelimit:",
	}
}

// Middleware returns the rate limiting HTTP middleware handler.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.prefix + r.RemoteAddr

		count, err := rl.client.Incr(r.Context(), key).Result()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			rl.client.Expire(r.Context(), key, rl.window)
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, rl.limit-int(count))))

		if count > int64(rl.limit) {
			ttl, err := rl.client.TTL(r.Context(), key).Result()
			if err == nil {
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			}
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
