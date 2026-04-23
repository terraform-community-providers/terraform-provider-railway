package provider

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// Default retry configuration
	defaultMaxRetries     = 5
	defaultInitialBackoff = 1 * time.Second
	defaultMaxBackoff     = 30 * time.Second
)

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)

	return t.wrapped.RoundTrip(req)
}

// retryTransport wraps an http.RoundTripper and adds retry logic for 429 and 5xx responses
type retryTransport struct {
	wrapped http.RoundTripper
}

// NewRetryTransport creates a new retry transport
func NewRetryTransport(wrapped http.RoundTripper) http.RoundTripper {
	return &retryTransport{
		wrapped: wrapped,
	}
}

// isRetryableStatusCode checks if the HTTP status code should trigger a retry
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return false
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= defaultMaxRetries; attempt++ {
		// Clone request body for retry if needed
		if req.Body != nil && attempt > 0 && req.GetBody != nil {
			newBody, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = newBody
		}
		if req.Body != nil && req.GetBody == nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			req.Body = io.NopCloser(&bodyReader{data: bodyBytes})
			bodyCopy := bodyBytes
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(&bodyReader{data: bodyCopy}), nil
			}
		}

		resp, err = t.wrapped.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if !isRetryableStatusCode(resp.StatusCode) {
			return resp, nil
		}

		if attempt == defaultMaxRetries {
			break
		}

		backoff := calculateBackoff(attempt, resp)

		tflog.Warn(ctx, "Retryable HTTP error from Railway API, retrying",
			map[string]interface{}{
				"attempt":     attempt + 1,
				"max_retries": defaultMaxRetries,
				"backoff_ms":  backoff.Milliseconds(),
				"status_code": resp.StatusCode,
			})

		resp.Body.Close()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return resp, nil
}

// calculateBackoff determines the backoff duration for a retry attempt
func calculateBackoff(attempt int, resp *http.Response) time.Duration {
	// Check for Retry-After header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
		if parsedTime, err := http.ParseTime(retryAfter); err == nil {
			return time.Until(parsedTime)
		}
	}

	// Exponential backoff: initialBackoff * 2^attempt
	backoff := float64(defaultInitialBackoff) * math.Pow(2, float64(attempt))

	if backoff > float64(defaultMaxBackoff) {
		backoff = float64(defaultMaxBackoff)
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

// GraphQL-level retry patterns

var graphQLRetryablePatterns = []string{
	// Rate limiting
	"too quickly",
	"try again in a sec",
	"rate limit",
	"rate-limit",
	"throttl",
	"whoa there",
	// Gateway errors
	"504 gateway timeout",
	"502 bad gateway",
	"503 service unavailable",
	"gateway timeout",
	"bad gateway",
	"service unavailable",
	"connection reset",
	"connection refused",
}

// IsRetryableGraphQLError checks if an error should trigger a retry
func IsRetryableGraphQLError(err error) bool {
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

// RetryableClient wraps a graphql.Client with retry logic
type RetryableClient struct {
	client graphql.Client
}

// NewRetryableClient creates a new RetryableClient
func NewRetryableClient(client graphql.Client) *RetryableClient {
	return &RetryableClient{
		client: client,
	}
}

// MakeRequest implements graphql.Client interface with retry logic
func (c *RetryableClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	var lastErr error

	for attempt := 0; attempt <= defaultMaxRetries; attempt++ {
		err := c.client.MakeRequest(ctx, req, resp)

		if err == nil && resp != nil && len(resp.Errors) > 0 {
			err = fmt.Errorf("%v", resp.Errors)
		}

		if err == nil {
			return nil
		}

		if !IsRetryableGraphQLError(err) {
			return err
		}

		lastErr = err

		if attempt == defaultMaxRetries {
			break
		}

		backoff := calculateGraphQLBackoff(attempt)

		tflog.Warn(ctx, "Retryable GraphQL error detected, retrying",
			map[string]interface{}{
				"attempt":     attempt + 1,
				"max_retries": defaultMaxRetries,
				"backoff_ms":  backoff.Milliseconds(),
				"error":       err.Error(),
			})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateGraphQLBackoff determines the backoff duration for a retry attempt
func calculateGraphQLBackoff(attempt int) time.Duration {
	backoff := float64(defaultInitialBackoff) * math.Pow(2, float64(attempt))

	if backoff > float64(defaultMaxBackoff) {
		backoff = float64(defaultMaxBackoff)
	}

	return time.Duration(backoff)
}
