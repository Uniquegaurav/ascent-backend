package httpx

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter is a fixed-window in-memory counter keyed by an arbitrary string
// (IP, phone, ...). Good enough for the current single-instance deployment;
// swap for Redis when the API scales horizontally.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*windowCount
}

type windowCount struct {
	start time.Time
	n     int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, hits: make(map[string]*windowCount)}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *RateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	// Lazy cleanup keeps the map bounded without a background goroutine.
	if len(l.hits) > 10_000 {
		for k, w := range l.hits {
			if now.Sub(w.start) > l.window {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.Sub(w.start) > l.window {
		l.hits[key] = &windowCount{start: now, n: 1}
		return true
	}
	w.n++
	return w.n <= l.limit
}

// ClientIP extracts the caller IP, trusting the first X-Forwarded-For entry
// (set by the hosting proxy) and falling back to the socket address.
func ClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if i := strings.IndexByte(fwd, ','); i >= 0 {
			fwd = fwd[:i]
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimit rejects requests over the per-IP limit with 429.
func RateLimit(l *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(ClientIP(r)) {
				Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
