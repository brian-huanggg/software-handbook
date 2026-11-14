package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/brian-huanggg/software-handbook/templates/authkit/respond"
)

// Limiter is an in-memory token bucket per key: each key may spend Burst
// requests at once and regains one every Every. It is per process, so with
// several API replicas the effective limit is multiplied; move it to a shared
// store (e.g. Redis) when you scale out.
//
// A nil *Limiter allows everything, so tests and callers can opt out.
type Limiter struct {
	every time.Duration
	burst float64
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewLimiter allows burst requests per key, refilled at one per every.
func NewLimiter(every time.Duration, burst int) *Limiter {
	if every <= 0 || burst <= 0 {
		panic("middleware: NewLimiter needs every > 0 and burst > 0")
	}
	return &Limiter{every: every, burst: float64(burst), now: time.Now, buckets: map[string]*bucket{}}
}

// WithClock replaces the time source. Test hook.
func (l *Limiter) WithClock(now func() time.Time) *Limiter {
	l.now = now
	return l
}

// Allow spends one token for key. When none is left it reports false and how
// long until the next token.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	if l == nil {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	b, found := l.buckets[key]
	if !found {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = l.refilled(b, now)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) * float64(l.every))
}

func (l *Limiter) refilled(b *bucket, now time.Time) float64 {
	return math.Min(l.burst, b.tokens+float64(now.Sub(b.last))/float64(l.every))
}

// sweep drops buckets that have refilled completely: they behave exactly like
// a missing bucket, so memory stays proportional to recently active keys.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if l.refilled(b, now) >= l.burst {
			delete(l.buckets, k)
		}
	}
}

// RetryAfter sets the Retry-After header in whole seconds, rounded up.
func RetryAfter(w http.ResponseWriter, d time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(d.Seconds()))))
}

// RateLimit answers 429 once the key returned by key(r) runs out of tokens.
func RateLimit(l *Limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if l == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok, wait := l.Allow(key(r)); !ok {
				RetryAfter(w, wait)
				respond.Error(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP keys by the TCP peer address. IPv6 is keyed by its /64, since one
// subscriber usually owns a whole /64 and could otherwise rotate addresses.
//
// Behind a reverse proxy every client shares the proxy's address: derive the
// IP from the proxy's header there, and only trust that header when the
// proxy is the sole way in.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	switch {
	case ip == nil:
		return host
	case ip.To4() != nil:
		return ip.To4().String()
	default:
		return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
	}
}
