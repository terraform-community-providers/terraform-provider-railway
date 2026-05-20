package provider

import (
	"strings"
	"testing"
)

func TestSelectUniqueByName(t *testing.T) {
	tests := []struct {
		name        string
		candidates  []namedLookupCandidate[string]
		want        string
		errContains string
	}{
		{
			name: "single match",
			candidates: []namedLookupCandidate[string]{
				{name: "api", value: "svc-api"},
				{name: "web", value: "svc-web"},
			},
			want: "svc-api",
		},
		{
			name: "zero matches",
			candidates: []namedLookupCandidate[string]{
				{name: "web", value: "svc-web"},
			},
			errContains: "no Railway service named \"api\" found in project \"project-id\"",
		},
		{
			name: "multiple matches",
			candidates: []namedLookupCandidate[string]{
				{name: "api", value: "svc-api-1"},
				{name: "api", value: "svc-api-2"},
			},
			errContains: "found 2 Railway service entries named \"api\" in project \"project-id\"; expected exactly one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectUniqueByName("Railway service", "api", "project \"project-id\"", tt.candidates)

			if tt.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}

				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
