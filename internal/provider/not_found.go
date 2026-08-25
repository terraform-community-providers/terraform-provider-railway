package provider

import (
	"errors"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

// errNotFound is wrapped by helpers that look an object up in a list (variables,
// domains, proxies) when no entry matches.
var errNotFound = errors.New("not found")

// isNotFoundError reports whether err means the requested object does not
// exist, either because the Railway GraphQL API said so (e.g. "Project not
// found") or because a lookup helper wrapped errNotFound.
//
// Only GraphQL response errors are inspected for the "not found" text, so
// internal sentinel errors such as "deploy is not found" (partial data on an
// object that does exist) are not mistaken for a deleted resource.
//
// Read implementations use this to remove the resource from state, which is
// how Terraform expects out-of-band deletion to be signalled so the next plan
// recreates the resource instead of failing.
func isNotFoundError(err error) bool {
	if errors.Is(err, errNotFound) {
		return true
	}
	var list gqlerror.List
	if errors.As(err, &list) {
		for _, e := range list {
			if e != nil && strings.Contains(strings.ToLower(e.Message), "not found") {
				return true
			}
		}
		return false
	}
	var single *gqlerror.Error
	if errors.As(err, &single) {
		return strings.Contains(strings.ToLower(single.Message), "not found")
	}
	return false
}
