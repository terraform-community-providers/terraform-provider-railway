package provider

import (
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64 // tokens per second
	lastRefill   time.Time
}

// NewRateLimiter creates a new rate limiter with the specified requests per second
// Returns nil if rps <= 0 (disabled)
func NewRateLimiter(rps float64) *RateLimiter {
	if rps <= 0 {
		return nil
	}
	return &RateLimiter{
		tokens:     rps, // Start with full bucket
		maxTokens:  rps,
		refillRate: rps,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()
		// Refill tokens based on elapsed time
		now := time.Now()
		elapsed := now.Sub(rl.lastRefill).Seconds()
		rl.tokens = math.Min(rl.maxTokens, rl.tokens+elapsed*rl.refillRate)
		rl.lastRefill = now

		if rl.tokens >= 1 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time for next token
		waitTime := time.Duration((1-rl.tokens)/rl.refillRate*1000) * time.Millisecond
		rl.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to try again
		}
	}
}

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)

	return t.wrapped.RoundTrip(req)
}

// retryTransport wraps an http.RoundTripper and adds retry logic for 429 responses
type retryTransport struct {
	wrapped     http.RoundTripper
	config      RetryConfig
	rateLimiter *RateLimiter
}

// NewRetryTransport creates a new retry transport with the given configuration
func NewRetryTransport(wrapped http.RoundTripper, config RetryConfig, rateLimiter *RateLimiter) http.RoundTripper {
	return &retryTransport{
		wrapped:     wrapped,
		config:      config,
		rateLimiter: rateLimiter,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Apply proactive rate limiting if configured
	if t.rateLimiter != nil {
		if err := t.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		// Clone request body for retry if needed
		if req.Body != nil && attempt > 0 && req.GetBody != nil {
			// Body was already read in previous attempt, need to reset
			newBody, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = newBody
		}
		if req.Body != nil && req.GetBody == nil {
			// Read and store body for potential retries
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			req.Body = io.NopCloser(&bodyReader{data: bodyBytes})
			bodyCopy := bodyBytes // capture for closure
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(&bodyReader{data: bodyCopy}), nil
			}
		}

		resp, err = t.wrapped.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// If not a 429, return immediately
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Don't retry if this was the last attempt
		if attempt == t.config.MaxRetries {
			break
		}

		// Calculate backoff duration
		backoff := t.calculateBackoff(attempt, resp)

		tflog.Warn(ctx, "Rate limited by Railway API, retrying",
			map[string]interface{}{
				"attempt":     attempt + 1,
				"max_retries": t.config.MaxRetries,
				"backoff_ms":  backoff.Milliseconds(),
			})

		// Close the response body before retrying
		resp.Body.Close()

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return resp, nil
}

// calculateBackoff determines the backoff duration for a retry attempt
func (t *retryTransport) calculateBackoff(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try parsing as seconds
		if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try parsing as HTTP date
		if parsedTime, err := http.ParseTime(retryAfter); err == nil {
			return time.Until(parsedTime)
		}
	}

	// Exponential backoff: initialBackoff * 2^attempt
	backoff := float64(t.config.InitialBackoff) * math.Pow(2, float64(attempt))

	// Add jitter (±25%)
	jitter := 0.75 + rand.Float64()*0.5 // 0.75 to 1.25
	backoff *= jitter

	// Cap at max backoff
	if backoff > float64(t.config.MaxBackoff) {
		backoff = float64(t.config.MaxBackoff)
	}

	return time.Duration(backoff)
}

// bodyReader is a simple io.Reader that reads from a byte slice
type bodyReader struct {
	data []byte
	pos  int
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
