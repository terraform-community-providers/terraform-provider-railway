package provider

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Khan/genqlient/graphql"
)

var (
	envVarName          = "RAILWAY_TOKEN"
	errMissingAuthToken = "Required token could not be found. Please set the token using an input variable in the provider configuration block or by using the `" + envVarName + "` environment variable."
)

func uuidRegex() *regexp.Regexp {
	return regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
}

var _ provider.Provider = &RailwayProvider{}

type RailwayProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type RailwayProviderModel struct {
	Token            types.String  `tfsdk:"token"`
	MaxRetries       types.Int64   `tfsdk:"max_retries"`
	InitialBackoffMs types.Int64   `tfsdk:"initial_backoff_ms"`
	MaxBackoffMs     types.Int64   `tfsdk:"max_backoff_ms"`
	RateLimitRps     types.Float64 `tfsdk:"rate_limit_rps"`
}

func (p *RailwayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "railway"
	resp.Version = p.version
}

func (p *RailwayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "The token used to authenticate with Railway.",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of retry attempts for rate-limited requests (HTTP 429). Defaults to 5.",
				Optional:            true,
			},
			"initial_backoff_ms": schema.Int64Attribute{
				MarkdownDescription: "Initial backoff duration in milliseconds for retry attempts. Defaults to 1000 (1 second).",
				Optional:            true,
			},
			"max_backoff_ms": schema.Int64Attribute{
				MarkdownDescription: "Maximum backoff duration in milliseconds for retry attempts. Defaults to 30000 (30 seconds).",
				Optional:            true,
			},
			"rate_limit_rps": schema.Float64Attribute{
				MarkdownDescription: "Proactive rate limit in requests per second. Set to 0 (default) to disable proactive rate limiting.",
				Optional:            true,
			},
		},
	}
}

func (p *RailwayProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RailwayProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	token := ""

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	// If a token wasn't set in the provider configuration block, try and fetch it
	// from the environment variable.
	if token == "" {
		token = os.Getenv(envVarName)
	}

	// If we still don't have a token at this point, we return an error.
	if token == "" {
		resp.Diagnostics.AddError("Missing API token", errMissingAuthToken)
		return
	}

	// Build retry configuration with defaults
	retryConfig := DefaultRetryConfig()

	if !data.MaxRetries.IsNull() {
		retryConfig.MaxRetries = int(data.MaxRetries.ValueInt64())
	}
	if !data.InitialBackoffMs.IsNull() {
		retryConfig.InitialBackoff = time.Duration(data.InitialBackoffMs.ValueInt64()) * time.Millisecond
	}
	if !data.MaxBackoffMs.IsNull() {
		retryConfig.MaxBackoff = time.Duration(data.MaxBackoffMs.ValueInt64()) * time.Millisecond
	}

	// Create rate limiter if configured (disabled by default)
	var rateLimiter *RateLimiter
	if !data.RateLimitRps.IsNull() && data.RateLimitRps.ValueFloat64() > 0 {
		rateLimiter = NewRateLimiter(data.RateLimitRps.ValueFloat64())
	}

	// Build transport chain: auth -> retry -> default
	transport := NewRetryTransport(
		&authedTransport{
			token:   token,
			wrapped: http.DefaultTransport,
		},
		retryConfig,
		rateLimiter,
	)

	httpClient := http.Client{
		Transport: transport,
	}

	baseClient := graphql.NewClient("https://backboard.railway.app/graphql/v2?source=terraform_provider_railway", &httpClient)

	// Wrap with retry logic for GraphQL-level rate limits
	var client graphql.Client = NewRetryableClient(baseClient, retryConfig)

	resp.DataSourceData = &client
	resp.ResourceData = &client
}

func (p *RailwayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewEnvironmentResource,
		NewServiceResource,
		NewVariableResource,
		NewVariableCollectionResource,
		NewSharedVariableResource,
		NewCustomDomainResource,
		NewServiceDomainResource,
		NewTcpProxyResource,
	}
}

func (p *RailwayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RailwayProvider{
			version: version,
		}
	}
}
