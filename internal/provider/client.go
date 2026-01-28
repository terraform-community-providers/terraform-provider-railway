package provider

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Khan/genqlient/graphql"
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

// isRetryableStatusCode checks if the HTTP status code should trigger a retry
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,      // 429
		http.StatusBadGateway,             // 502
		http.StatusServiceUnavailable,     // 503
		http.StatusGatewayTimeout:         // 504
		return true
	default:
		return false
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

		// If not a retryable status code, return immediately
		if !isRetryableStatusCode(resp.StatusCode) {
			return resp, nil
		}

		// Don't retry if this was the last attempt
		if attempt == t.config.MaxRetries {
			break
		}

		// Calculate backoff duration
		backoff := t.calculateBackoff(attempt, resp)

		tflog.Warn(ctx, "Retryable HTTP error from Railway API, retrying",
			map[string]interface{}{
				"attempt":      attempt + 1,
				"max_retries":  t.config.MaxRetries,
				"backoff_ms":   backoff.Milliseconds(),
				"status_code":  resp.StatusCode,
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

// GraphQL-level rate limit detection and retry

// graphQLRateLimitPatterns contains error message patterns that indicate rate limiting
var graphQLRateLimitPatterns = []string{
	"too quickly",
	"try again in a sec",
	"rate limit",
	"rate-limit",
	"throttl",
	"whoa there",
}

// graphQLRetryablePatterns contains error message patterns that indicate retryable errors (gateway timeouts, etc.)
var graphQLRetryablePatterns = []string{
	"504 gateway timeout",
	"502 bad gateway",
	"503 service unavailable",
	"gateway timeout",
	"bad gateway",
	"service unavailable",
	"connection reset",
	"connection refused",
	"timeout",
}

// IsGraphQLRateLimitError checks if an error is a GraphQL-level rate limit error
func IsGraphQLRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	for _, pattern := range graphQLRateLimitPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// IsGraphQLRetryableError checks if an error is a retryable GraphQL error (gateway timeouts, etc.)
func IsGraphQLRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	for _, pattern := range graphQLRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// IsRetryableGraphQLError checks if an error should trigger a retry (rate limit OR gateway timeout, etc.)
func IsRetryableGraphQLError(err error) bool {
	return IsGraphQLRateLimitError(err) || IsGraphQLRetryableError(err)
}

// RetryableClient wraps a graphql.Client with retry logic for GraphQL-level rate limits
type RetryableClient struct {
	client graphql.Client
	config RetryConfig
}

// NewRetryableClient creates a new RetryableClient
func NewRetryableClient(client graphql.Client, config RetryConfig) *RetryableClient {
	return &RetryableClient{
		client: client,
		config: config,
	}
}

// MakeRequest implements graphql.Client interface with retry logic
func (c *RetryableClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		err := c.client.MakeRequest(ctx, req, resp)

		// Check for GraphQL errors in response even if err is nil
		if err == nil && resp != nil && len(resp.Errors) > 0 {
			// Convert GraphQL errors to an error for checking
			err = fmt.Errorf("%v", resp.Errors)
		}

		if err == nil {
			return nil
		}

		// Check if this is a retryable error (rate limit or gateway timeout, etc.)
		if !IsRetryableGraphQLError(err) {
			return err
		}

		lastErr = err

		// Don't retry if this was the last attempt
		if attempt == c.config.MaxRetries {
			break
		}

		// Calculate backoff
		backoff := c.calculateBackoff(attempt)

		tflog.Warn(ctx, "Retryable GraphQL error detected, retrying",
			map[string]interface{}{
				"attempt":     attempt + 1,
				"max_retries": c.config.MaxRetries,
				"backoff_ms":  backoff.Milliseconds(),
				"error":       err.Error(),
			})

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries exceeded for GraphQL error: %w", lastErr)
}

// calculateBackoff determines the backoff duration for a retry attempt
func (c *RetryableClient) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initialBackoff * 2^attempt
	backoff := float64(c.config.InitialBackoff) * math.Pow(2, float64(attempt))

	// Add jitter (±25%)
	jitter := 0.75 + rand.Float64()*0.5 // 0.75 to 1.25
	backoff *= jitter

	// Cap at max backoff
	if backoff > float64(c.config.MaxBackoff) {
		backoff = float64(c.config.MaxBackoff)
	}

	return time.Duration(backoff)
}

// GetUnderlyingClient returns the underlying graphql.Client for cases where
// direct access is needed (e.g., for generated code compatibility)
func (c *RetryableClient) GetUnderlyingClient() graphql.Client {
	return c.client
}
