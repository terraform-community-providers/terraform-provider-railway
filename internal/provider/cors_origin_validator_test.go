package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCorsOriginValidator(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value string
		valid bool
	}{
		"all origins":            {value: "*", valid: true},
		"HTTP origin":            {value: "http://localhost:3000", valid: true},
		"HTTPS origin":           {value: "https://example.com", valid: true},
		"wildcard host":          {value: "https://*.example.com", valid: true},
		"IPv6 host":              {value: "https://[::1]:8443", valid: true},
		"explicit port":          {value: "https://example.com:8443", valid: true},
		"empty":                  {value: "", valid: false},
		"missing scheme":         {value: "example.com", valid: false},
		"unsupported scheme":     {value: "ftp://example.com", valid: false},
		"missing host":           {value: "https://", valid: false},
		"user information":       {value: "https://user@example.com", valid: false},
		"path":                   {value: "https://example.com/path", valid: false},
		"trailing slash":         {value: "https://example.com/", valid: false},
		"query":                  {value: "https://example.com?key=value", valid: false},
		"fragment":               {value: "https://example.com#fragment", valid: false},
		"multiple wildcards":     {value: "https://*.*.example.com", valid: false},
		"wildcard outside host":  {value: "https://example.com?value=*", valid: false},
		"surrounding whitespace": {value: " https://example.com", valid: false},
		"uppercase scheme":       {value: "HTTPS://example.com", valid: false},
		"empty port":             {value: "https://example.com:", valid: false},
		"leading-zero port":      {value: "https://example.com:080", valid: false},
		"invalid port":           {value: "https://example.com:65536", valid: false},
		"zero port":              {value: "https://example.com:0", valid: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := validator.StringRequest{
				ConfigValue: types.StringValue(test.value),
				Path:        path.Root("allowed_origins"),
			}
			var response validator.StringResponse
			corsOriginValidator{}.ValidateString(context.Background(), request, &response)
			if actual := !response.Diagnostics.HasError(); actual != test.valid {
				t.Fatalf("expected valid=%t, got diagnostics: %v", test.valid, response.Diagnostics)
			}
		})
	}
}

func TestCorsOriginValidatorSkipsUnknownValues(t *testing.T) {
	t.Parallel()

	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		request := validator.StringRequest{
			ConfigValue: value,
			Path:        path.Root("allowed_origins"),
		}
		var response validator.StringResponse
		corsOriginValidator{}.ValidateString(context.Background(), request, &response)
		if response.Diagnostics.HasError() {
			t.Fatalf("expected no diagnostics, got: %v", response.Diagnostics)
		}
	}
}

func TestAtMostOneWildcardValidator(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"Authorization": true,
		"*":             true,
		"x-amz-*":       true,
		"x-*-*":         false,
	}
	for value, valid := range tests {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			request := validator.StringRequest{
				ConfigValue: types.StringValue(value),
				Path:        path.Root("allowed_headers"),
			}
			var response validator.StringResponse
			atMostOneWildcardValidator{}.ValidateString(context.Background(), request, &response)
			if actual := !response.Diagnostics.HasError(); actual != valid {
				t.Fatalf("expected valid=%t, got diagnostics: %v", valid, response.Diagnostics)
			}
		})
	}
}
