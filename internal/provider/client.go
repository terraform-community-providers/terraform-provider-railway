package provider

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
)

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)

	debug := os.Getenv("HTTP_DEBUG") == "1"

	if debug {
		dump, err := httputil.DumpRequestOut(req, true)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to dump Railway request: %v\n", err)
		} else {
			dump = bytes.ReplaceAll(
				dump,
				[]byte("Bearer "+t.token),
				[]byte("Bearer [REDACTED]"),
			)

			fmt.Fprintf(os.Stderr, "\nRailway request:\n%s\n", dump)
		}
	}

	resp, err := t.wrapped.RoundTrip(req)

	if debug && resp != nil {
		dump, dumpErr := httputil.DumpResponse(resp, true)

		if dumpErr != nil {
			fmt.Fprintf(os.Stderr, "Unable to dump Railway response: %v\n", dumpErr)
		} else {
			fmt.Fprintf(os.Stderr, "\nRailway response:\n%s\n", dump)
		}
	}

	return resp, err
}
