package provider

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestBucketCorsRulesRoundTripPreservesOrder(t *testing.T) {
	ctx := context.Background()
	expected := []types.CORSRule{
		{
			ID:             aws.String("browser-uploads"),
			AllowedHeaders: []string{"*", "Content-Type"},
			AllowedMethods: []string{"PUT"},
			AllowedOrigins: []string{"https://uploads.example.com"},
			ExposeHeaders:  []string{"ETag"},
			MaxAgeSeconds:  aws.Int32(3600),
		},
		{
			ID:             aws.String("browser-reads"),
			AllowedMethods: []string{"GET", "HEAD"},
			AllowedOrigins: []string{"https://example.com"},
		},
	}

	value, diagnostics := flattenBucketCorsRules(ctx, expected)
	if diagnostics.HasError() {
		t.Fatalf("flattening CORS rules returned diagnostics: %v", diagnostics)
	}

	actual, diagnostics := expandBucketCorsRules(ctx, value)
	if diagnostics.HasError() {
		t.Fatalf("expanding CORS rules returned diagnostics: %v", diagnostics)
	}
	want := append([]types.CORSRule(nil), expected...)
	want[1].AllowedHeaders = []string{}
	want[1].ExposeHeaders = []string{}
	if diff := cmp.Diff(want, actual, cmpopts.IgnoreUnexported(types.CORSRule{})); diff != "" {
		t.Fatalf("CORS rules changed during round trip (-want +got):\n%s", diff)
	}
}

func TestBucketUsesPathStyle(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		urlStyle     string
		usePathStyle bool
		valid        bool
	}{
		"path":                {urlStyle: "path", usePathStyle: true, valid: true},
		"virtual":             {urlStyle: "virtual", usePathStyle: false, valid: true},
		"virtual host":        {urlStyle: "virtual-host", usePathStyle: false, valid: true},
		"case insensitive":    {urlStyle: "VIRTUAL-HOST", usePathStyle: false, valid: true},
		"unsupported default": {urlStyle: "auto", valid: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual, err := bucketUsesPathStyle(test.urlStyle)
			if (err == nil) != test.valid {
				t.Fatalf("expected valid=%t, got error: %v", test.valid, err)
			}
			if err == nil && actual != test.usePathStyle {
				t.Fatalf("expected usePathStyle=%t, got %t", test.usePathStyle, actual)
			}
		})
	}
}
