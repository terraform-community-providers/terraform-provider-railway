package provider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestRetryTransport_429Retry(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("POST", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransport_5xxRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := atomic.Int32{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := attempts.Add(1)
				if attempt < 2 {
					w.WriteHeader(tt.statusCode)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"data":{}}`))
			}))
			defer server.Close()

			transport := NewRetryTransport(http.DefaultTransport)
			client := &http.Client{Transport: transport}

			req, _ := http.NewRequest("POST", server.URL, nil)
			resp, err := client.Do(req)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", resp.StatusCode)
			}
			if attempts.Load() != 2 {
				t.Fatalf("expected 2 attempts, got %d", attempts.Load())
			}
		})
	}
}

func TestRetryTransport_NoRetryOn400(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("POST", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if attempts.Load() != 1 {
		t.Fatalf("expected 1 attempt (no retry), got %d", attempts.Load())
	}
}

func TestRetryTransport_MaxRetriesExceeded(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	transport := NewRetryTransport(http.DefaultTransport)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("POST", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", resp.StatusCode)
	}
	// Should be maxRetries + 1 (initial + retries)
	if attempts.Load() != defaultMaxRetries+1 {
		t.Fatalf("expected %d attempts, got %d", defaultMaxRetries+1, attempts.Load())
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, false},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}

	for _, tt := range tests {
		result := isRetryableStatusCode(tt.code)
		if result != tt.expected {
			t.Errorf("isRetryableStatusCode(%d) = %v, expected %v", tt.code, result, tt.expected)
		}
	}
}

func TestIsRetryableGraphQLError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      gqlerror.List{{Message: "Rate limit exceeded"}},
			expected: true,
		},
		{
			name:     "too quickly error",
			err:      gqlerror.List{{Message: "creating volumes too quickly, please try again later"}},
			expected: true,
		},
		{
			name:     "5xx error message",
			err:      gqlerror.List{{Message: "502 Bad Gateway"}},
			expected: true,
		},
		{
			name:     "503 error",
			err:      gqlerror.List{{Message: "503 Service Unavailable"}},
			expected: true,
		},
		{
			name:     "504 error",
			err:      gqlerror.List{{Message: "504 Gateway Timeout"}},
			expected: true,
		},
		{
			name:     "validation error",
			err:      gqlerror.List{{Message: "Variable $id is required"}},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableGraphQLError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableGraphQLError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
