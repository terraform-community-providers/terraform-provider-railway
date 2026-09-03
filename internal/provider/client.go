package provider

import "net/http"

type authedTransport struct {
	headerName  string
	headerValue string
	wrapped     http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	authedRequest := req.Clone(req.Context())
	authedRequest.Header.Del("Authorization")
	authedRequest.Header.Del("Project-Access-Token")
	authedRequest.Header.Set(t.headerName, t.headerValue)

	return t.wrapped.RoundTrip(authedRequest)
}
