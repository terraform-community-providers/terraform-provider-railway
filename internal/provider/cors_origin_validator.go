package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type corsOriginValidator struct{}

type atMostOneWildcardValidator struct{}

func (corsOriginValidator) Description(context.Context) string {
	return "must be `*` or an HTTP(S) origin without a path, query, or fragment"
}

func (v corsOriginValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v corsOriginValidator) ValidateString(
	ctx context.Context,
	request validator.StringRequest,
	response *validator.StringResponse,
) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if err := validateCorsOrigin(value); err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid CORS Origin",
			fmt.Sprintf("%q %s: %s.", value, v.Description(ctx), err),
		)
	}
}

func validateCorsOrigin(value string) error {
	if value == "*" {
		return nil
	}
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("must not contain surrounding whitespace")
	}
	if strings.Count(value, "*") > 1 {
		return fmt.Errorf("may contain at most one wildcard")
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return fmt.Errorf("must use a lowercase HTTP or HTTPS scheme")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must use the HTTP or HTTPS scheme")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("must include a host")
	}
	if parsed.User != nil {
		return fmt.Errorf("must not include user information")
	}
	if strings.HasSuffix(parsed.Host, ":") {
		return fmt.Errorf("must not include an empty port")
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return fmt.Errorf("must not include a path, query, or fragment")
	}
	if strings.Contains(value, "*") && !strings.Contains(parsed.Hostname(), "*") {
		return fmt.Errorf("wildcards are only allowed in the host")
	}

	if port := parsed.Port(); port != "" {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return fmt.Errorf("must include a valid port")
		}
		if len(port) > 1 && port[0] == '0' {
			return fmt.Errorf("must use a canonical port without leading zeros")
		}
	}
	return nil
}

func (atMostOneWildcardValidator) Description(context.Context) string {
	return "must contain at most one wildcard"
}

func (v atMostOneWildcardValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v atMostOneWildcardValidator) ValidateString(
	ctx context.Context,
	request validator.StringRequest,
	response *validator.StringResponse,
) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}
	if strings.Count(request.ConfigValue.ValueString(), "*") > 1 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Wildcard Count",
			fmt.Sprintf("%q %s.", request.ConfigValue.ValueString(), v.Description(ctx)),
		)
	}
}
