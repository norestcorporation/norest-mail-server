package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// Limiter provides rate limiting functionality with a simple in-memory implementation.
// This is designed to be extensible for distributed implementations later.
type Limiter struct {
	mu     sync.RWMutex
	buckets map[string]*bucket
	config *Config
}

type Config struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window for the limit
}

type bucket struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// NewLimiter creates a new rate limiter with the given configuration.
func NewLimiter(config *Config) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		config:  config,
	}
}

// Allow checks if a request should be allowed for the given key.
// It returns true if allowed, false if rate limited.
func (l *Limiter) Allow(key string) bool {
	l.mu.RLock()
	b, exists := l.buckets[key]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		b = &bucket{
			timestamps: make([]time.Time, 0),
		}
		l.buckets[key] = b
		l.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Window)

	// Remove timestamps outside the window
	validTimestamps := make([]time.Time, 0)
	for _, ts := range b.timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	b.timestamps = validTimestamps

	// Check if under limit
	if len(b.timestamps) < l.config.Requests {
		b.timestamps = append(b.timestamps, now)
		return true
	}

	return false
}

// Cleanup removes old buckets to prevent memory leaks.
// This should be called periodically in a background goroutine.
func (l *Limiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.config.Window * 2)
	for key, b := range l.buckets {
		b.mu.Lock()
		if len(b.timestamps) == 0 || b.timestamps[len(b.timestamps)-1].Before(cutoff) {
			delete(l.buckets, key)
		}
		b.mu.Unlock()
	}
}

// Middleware returns HTTP middleware for rate limiting.
func (l *Limiter) Middleware(keyFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if !l.Allow(key) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				w.Header().Set("Retry-After", "60")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKey extracts the IP address from a request for rate limiting.
func IPKey(r *http.Request) string {
	return r.RemoteAddr
}

// UserKey extracts the user ID from a request for rate limiting.
// This requires the user ID to be set in the request context.
func UserKey(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return IPKey(r)
}
