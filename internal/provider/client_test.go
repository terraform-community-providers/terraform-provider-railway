package provider

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestAuthedTransportUsesOnlyConfiguredHeaderAndClonesRequest(t *testing.T) {
	for _, test := range []struct {
		name        string
		headerName  string
		headerValue string
	}{
		{name: "bearer", headerName: "Authorization", headerValue: "Bearer bearer-secret"},
		{name: "project", headerName: "Project-Access-Token", headerValue: "project-secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, "https://example.com/graphql", strings.NewReader(`{"query":"query Viewer { me { id } }"}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "original-bearer")
			request.Header.Set("Project-Access-Token", "original-project")

			transport := &authedTransport{
				headerName:  test.headerName,
				headerValue: test.headerValue,
				wrapped: roundTripperFunc(func(got *http.Request) (*http.Response, error) {
					if got == request {
						t.Error("transport passed the caller request without cloning it")
					}
					if got.Header.Get(test.headerName) != test.headerValue {
						t.Errorf("%s = %q, want %q", test.headerName, got.Header.Get(test.headerName), test.headerValue)
					}
					other := "Authorization"
					if test.headerName == other {
						other = "Project-Access-Token"
					}
					if got.Header.Get(other) != "" {
						t.Errorf("unexpected %s header = %q", other, got.Header.Get(other))
					}
					body, err := io.ReadAll(got.Body)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(body), "bearer-secret") || strings.Contains(string(body), "project-secret") {
						t.Errorf("request body leaked an authentication secret: %s", body)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
				}),
			}
			if _, err := transport.RoundTrip(request); err != nil {
				t.Fatal(err)
			}
			if request.Header.Get("Authorization") != "original-bearer" || request.Header.Get("Project-Access-Token") != "original-project" {
				t.Errorf("caller request headers were mutated: %#v", request.Header)
			}
		})
	}
}
