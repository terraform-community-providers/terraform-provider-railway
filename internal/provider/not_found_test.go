package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestIsNotFoundError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"nil":                    {err: nil, want: false},
		"plain error":            {err: errors.New("boom"), want: false},
		"internal sentinel text": {err: fmt.Errorf("deploy is not found"), want: false},
		"wrapped errNotFound":    {err: fmt.Errorf("variable %q: %w", "FOO", errNotFound), want: true},
		"gql list not found":     {err: gqlerror.List{{Message: "Project not found"}}, want: true},
		"gql list other":         {err: gqlerror.List{{Message: "Not Authorized"}}, want: false},
		"gql single not found":   {err: &gqlerror.Error{Message: "Service not found"}, want: true},
		"gql wrapped":            {err: fmt.Errorf("Unable to list custom domains, got error: %w", gqlerror.List{{Message: "Environment not found"}}), want: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isNotFoundError(tc.err); got != tc.want {
				t.Fatalf("isNotFoundError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
