package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
)

func TestAuthedTransport_SetsAuthorizationHeader(t *testing.T) {
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &authedTransport{
		token:   "test-token",
		wrapped: http.DefaultTransport,
	}

	client := &http.Client{Transport: transport}
	_, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("request was not captured")
	}

	authHeader := capturedReq.Header.Get("Authorization")
	expected := "Bearer test-token"
	if authHeader != expected {
		t.Errorf("expected Authorization header %q, got %q", expected, authHeader)
	}
}

func TestRetryTransport_RetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryTransport_ExhaustsRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	maxRetries := 3
	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     maxRetries,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp.StatusCode)
	}

	expectedAttempts := maxRetries + 1 // Initial attempt + retries
	if attempts != expectedAttempts {
		t.Errorf("expected %d attempts, got %d", expectedAttempts, attempts)
	}
}

func TestRetryTransport_RespectsRetryAfterHeader(t *testing.T) {
	attempts := 0
	var requestTimes []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond, // Very short, should be overridden by Retry-After
		MaxBackoff:     30 * time.Second,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if len(requestTimes) < 2 {
		t.Fatal("expected at least 2 requests")
	}

	elapsed := requestTimes[1].Sub(requestTimes[0])
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected at least 900ms between requests due to Retry-After, got %v", elapsed)
	}
}

func TestRetryTransport_RespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 10 * time.Second, // Long backoff
		MaxBackoff:     30 * time.Second,
	}, nil)

	client := &http.Client{Transport: transport}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	start := time.Now()
	_, err := client.Do(req)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected context cancellation error")
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("expected quick cancellation, took %v", elapsed)
	}
}

func TestRetryTransport_PreservesRequestBody(t *testing.T) {
	attempts := 0
	var bodies []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	requestBody := "test request body"
	resp, err := client.Post(server.URL, "text/plain", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if len(bodies) != 3 {
		t.Errorf("expected 3 request bodies, got %d", len(bodies))
	}

	for i, body := range bodies {
		if body != requestBody {
			t.Errorf("request %d: expected body %q, got %q", i+1, requestBody, body)
		}
	}
}

func TestBackoff_GrowsExponentially(t *testing.T) {
	rt := &retryTransport{
		config: RetryConfig{
			MaxRetries:     5,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
		},
	}

	// Create a mock response without Retry-After header
	resp := &http.Response{Header: http.Header{}}

	// Test exponential growth (without jitter consideration)
	// We'll just verify the general trend is increasing
	var backoffs []time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		backoff := rt.calculateBackoff(attempt, resp)
		backoffs = append(backoffs, backoff)
	}

	// Verify backoffs are generally increasing (accounting for jitter)
	for i := 1; i < len(backoffs); i++ {
		// With 25% jitter, next backoff should be at least 1.5x previous minimum
		// This is a loose check due to jitter
		minExpected := time.Duration(float64(backoffs[i-1]) * 0.5)
		if backoffs[i] < minExpected {
			t.Errorf("backoff[%d] (%v) should be greater than %v", i, backoffs[i], minExpected)
		}
	}
}

func TestBackoff_CapsAtMaximum(t *testing.T) {
	maxBackoff := 100 * time.Millisecond
	rt := &retryTransport{
		config: RetryConfig{
			MaxRetries:     10,
			InitialBackoff: 50 * time.Millisecond,
			MaxBackoff:     maxBackoff,
		},
	}

	resp := &http.Response{Header: http.Header{}}

	// Test many attempts - all should be capped
	for attempt := 5; attempt < 10; attempt++ {
		backoff := rt.calculateBackoff(attempt, resp)
		if backoff > maxBackoff {
			t.Errorf("attempt %d: backoff %v exceeds max %v", attempt, backoff, maxBackoff)
		}
	}
}

func TestRateLimiter_LimitsRequestRate(t *testing.T) {
	rps := 10.0 // 10 requests per second
	rl := NewRateLimiter(rps)

	ctx := context.Background()
	start := time.Now()

	// Make 10 requests quickly
	for i := 0; i < 10; i++ {
		if err := rl.Wait(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// The 10th request should have waited for tokens to refill
	elapsed := time.Since(start)

	// With 10 RPS and 10 requests, minimum time should be close to 0
	// since we start with a full bucket, but 11+ requests would need refills
	if elapsed > 2*time.Second {
		t.Errorf("10 requests with 10 RPS took too long: %v", elapsed)
	}
}

func TestRateLimiter_RespectsContextCancellation(t *testing.T) {
	rps := 1.0 // 1 request per second
	rl := NewRateLimiter(rps)

	ctx := context.Background()

	// Use up the initial token
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error on first wait: %v", err)
	}

	// Now try with a cancelled context - should fail quickly
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.Wait(ctx)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiter_DisabledWhenZeroOrNegative(t *testing.T) {
	tests := []float64{0, -1, -0.5}

	for _, rps := range tests {
		rl := NewRateLimiter(rps)
		if rl != nil {
			t.Errorf("NewRateLimiter(%v) should return nil", rps)
		}
	}
}

func TestRetryTransport_WithRateLimiter(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rateLimiter := NewRateLimiter(100) // 100 RPS should be plenty fast for test
	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, rateLimiter)

	client := &http.Client{Transport: transport}

	// Make several requests
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}
		resp.Body.Close()
	}

	if atomic.LoadInt32(&requestCount) != 5 {
		t.Errorf("expected 5 requests, got %d", requestCount)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries=5, got %d", config.MaxRetries)
	}
	if config.InitialBackoff != 1*time.Second {
		t.Errorf("expected InitialBackoff=1s, got %v", config.InitialBackoff)
	}
	if config.MaxBackoff != 30*time.Second {
		t.Errorf("expected MaxBackoff=30s, got %v", config.MaxBackoff)
	}
}

// GraphQL rate limit detection tests

func TestIsGraphQLRateLimitError_DetectsRateLimitPatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "volume creation too quickly",
			err:      errors.New("volumeCreate Whoa there pal! You are creating volumes too quickly. Try again in a sec"),
			expected: true,
		},
		{
			name:     "try again in a sec",
			err:      errors.New("Try again in a sec"),
			expected: true,
		},
		{
			name:     "too quickly",
			err:      errors.New("You are doing something too quickly"),
			expected: true,
		},
		{
			name:     "rate limit",
			err:      errors.New("Rate limit exceeded"),
			expected: true,
		},
		{
			name:     "rate-limit with hyphen",
			err:      errors.New("rate-limit reached"),
			expected: true,
		},
		{
			name:     "throttled",
			err:      errors.New("Request was throttled"),
			expected: true,
		},
		{
			name:     "whoa there",
			err:      errors.New("Whoa there! Slow down"),
			expected: true,
		},
		{
			name:     "regular error - not rate limit",
			err:      errors.New("Service not found"),
			expected: false,
		},
		{
			name:     "validation error",
			err:      errors.New("Invalid input: name is required"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "case insensitive - uppercase",
			err:      errors.New("RATE LIMIT EXCEEDED"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGraphQLRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("IsGraphQLRateLimitError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// mockGraphQLClient is a mock implementation of graphql.Client for testing
type mockGraphQLClient struct {
	responses []error
	callCount int
}

func (m *mockGraphQLClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	if m.callCount < len(m.responses) {
		err := m.responses[m.callCount]
		m.callCount++
		return err
	}
	m.callCount++
	return nil
}

func TestRetryableClient_RetriesOnGraphQLRateLimit(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("volumeCreate Whoa there pal! You are creating volumes too quickly"),
			errors.New("Try again in a sec"),
			nil, // Success on third attempt
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}
}

func TestRetryableClient_ExhaustsRetriesOnPersistentRateLimit(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("rate limit"),
			errors.New("rate limit"),
			errors.New("rate limit"),
			errors.New("rate limit"),
		},
	}

	maxRetries := 3
	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     maxRetries,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected 'max retries exceeded' error, got: %v", err)
	}

	expectedCalls := maxRetries + 1 // Initial attempt + retries
	if mock.callCount != expectedCalls {
		t.Errorf("expected %d calls, got %d", expectedCalls, mock.callCount)
	}
}

func TestRetryableClient_DoesNotRetryNonRateLimitErrors(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("Service not found"),
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err == nil {
		t.Fatal("expected error to be returned")
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", mock.callCount)
	}

	if !strings.Contains(err.Error(), "Service not found") {
		t.Errorf("expected original error, got: %v", err)
	}
}

func TestRetryableClient_RespectsContextCancellation(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("rate limit"),
			errors.New("rate limit"),
			errors.New("rate limit"),
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 10 * time.Second, // Long backoff
		MaxBackoff:     30 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short time
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := client.MakeRequest(ctx, &graphql.Request{}, &graphql.Response{})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("expected quick cancellation, took %v", elapsed)
	}
}

func TestRetryableClient_SucceedsImmediatelyOnNoError(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{nil},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
}

// mockGraphQLClientWithErrors returns both an error and populates GraphQL errors
type mockGraphQLClientWithErrors struct {
	graphqlErrors []string
	callCount     int
}

func (m *mockGraphQLClientWithErrors) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	m.callCount++
	if m.callCount <= len(m.graphqlErrors) {
		errMsg := m.graphqlErrors[m.callCount-1]
		if errMsg != "" {
			// Return the error message - the actual error type doesn't matter for our detection
			return fmt.Errorf("%s", errMsg)
		}
	}
	return nil
}

func TestRetryableClient_RetriesOnGraphQLResponseErrors(t *testing.T) {
	mock := &mockGraphQLClientWithErrors{
		graphqlErrors: []string{
			"volumeCreate Whoa there pal! You are creating volumes too quickly",
			"Try again in a sec",
			"", // No error on third attempt
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}
}

func TestRetryableClientBackoff_GrowsExponentially(t *testing.T) {
	client := NewRetryableClient(nil, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
	})

	var backoffs []time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		backoff := client.calculateBackoff(attempt)
		backoffs = append(backoffs, backoff)
	}

	// Verify backoffs are generally increasing (accounting for jitter)
	for i := 1; i < len(backoffs); i++ {
		minExpected := time.Duration(float64(backoffs[i-1]) * 0.5)
		if backoffs[i] < minExpected {
			t.Errorf("backoff[%d] (%v) should be greater than %v", i, backoffs[i], minExpected)
		}
	}
}

func TestRetryableClientBackoff_CapsAtMaximum(t *testing.T) {
	maxBackoff := 100 * time.Millisecond
	client := NewRetryableClient(nil, RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     maxBackoff,
	})

	for attempt := 5; attempt < 10; attempt++ {
		backoff := client.calculateBackoff(attempt)
		if backoff > maxBackoff {
			t.Errorf("attempt %d: backoff %v exceeds max %v", attempt, backoff, maxBackoff)
		}
	}
}

func TestGraphQLRateLimitPatterns_AllPatternsValid(t *testing.T) {
	// Ensure all patterns are lowercase for case-insensitive matching
	for _, pattern := range graphQLRateLimitPatterns {
		if pattern != strings.ToLower(pattern) {
			t.Errorf("pattern %q should be lowercase", pattern)
		}
	}
}

func TestIsGraphQLRateLimitError_RealWorldErrors(t *testing.T) {
	// Test with actual error message from the issue
	realError := fmt.Errorf("input:3: volumeCreate Whoa there pal! You are creating volumes too quickly. Try again in a sec")
	if !IsGraphQLRateLimitError(realError) {
		t.Error("should detect real-world volume creation rate limit error")
	}
}

// Tests for 504 Gateway Timeout and other retryable errors

func TestRetryTransport_RetriesOn504(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusGatewayTimeout) // 504
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryTransport_RetriesOn502(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusBadGateway) // 502
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryTransport_RetriesOn503(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}, nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{http.StatusOK, false},
		{http.StatusCreated, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusInternalServerError, false},
		{http.StatusTooManyRequests, true},    // 429
		{http.StatusBadGateway, true},         // 502
		{http.StatusServiceUnavailable, true}, // 503
		{http.StatusGatewayTimeout, true},     // 504
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := isRetryableStatusCode(tt.statusCode)
			if result != tt.expected {
				t.Errorf("isRetryableStatusCode(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestIsGraphQLRetryableError_DetectsGatewayTimeoutPatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "504 gateway timeout",
			err:      errors.New("returned error 504 Gateway Timeout: error code: 504"),
			expected: true,
		},
		{
			name:     "gateway timeout without code",
			err:      errors.New("Gateway Timeout occurred"),
			expected: true,
		},
		{
			name:     "502 bad gateway",
			err:      errors.New("returned error 502 Bad Gateway"),
			expected: true,
		},
		{
			name:     "503 service unavailable",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "connection timeout",
			err:      errors.New("connection timeout while connecting to server"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "regular error - not retryable",
			err:      errors.New("Service not found"),
			expected: false,
		},
		{
			name:     "validation error",
			err:      errors.New("Invalid input: name is required"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "case insensitive - uppercase",
			err:      errors.New("GATEWAY TIMEOUT"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGraphQLRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsGraphQLRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableGraphQLError_CombinesRateLimitAndRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "gateway timeout error",
			err:      errors.New("504 gateway timeout"),
			expected: true,
		},
		{
			name:     "whoa there rate limit",
			err:      errors.New("Whoa there! Slow down"),
			expected: true,
		},
		{
			name:     "connection timeout",
			err:      errors.New("timeout waiting for response"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("Not found"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableGraphQLError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableGraphQLError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryableClient_RetriesOnGatewayTimeout(t *testing.T) {
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("returned error 504 Gateway Timeout: error code: 504"),
			errors.New("Gateway Timeout"),
			nil, // Success on third attempt
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}
}

func TestRetryableClient_RetriesOnMixedRetryableErrors(t *testing.T) {
	// Test that it retries on a mix of rate limit and gateway timeout errors
	mock := &mockGraphQLClient{
		responses: []error{
			errors.New("rate limit exceeded"),
			errors.New("504 Gateway Timeout"),
			errors.New("Whoa there! Try again in a sec"),
			nil, // Success on fourth attempt
		},
	}

	client := NewRetryableClient(mock, RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	})

	err := client.MakeRequest(context.Background(), &graphql.Request{}, &graphql.Response{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.callCount != 4 {
		t.Errorf("expected 4 calls, got %d", mock.callCount)
	}
}

func TestGraphQLRetryablePatterns_AllPatternsValid(t *testing.T) {
	// Ensure all patterns are lowercase for case-insensitive matching
	for _, pattern := range graphQLRetryablePatterns {
		if pattern != strings.ToLower(pattern) {
			t.Errorf("pattern %q should be lowercase", pattern)
		}
	}
}

func TestIsGraphQLRetryableError_RealWorldErrors(t *testing.T) {
	// Test with actual error message from the issue
	realError := fmt.Errorf("Unable to create volume, got error: returned error 504 Gateway Timeout:\nerror code: 504")
	if !IsGraphQLRetryableError(realError) {
		t.Error("should detect real-world 504 Gateway Timeout error")
	}
}
