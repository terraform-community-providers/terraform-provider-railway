package provider

import (
	"context"
	"net/http"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Khan/genqlient/graphql"
)

var (
	tokenEnvVar             = "RAILWAY_TOKEN"
	projectTokenEnvVar      = "RAILWAY_PROJECT_TOKEN"
	errMissingAuthToken     = "Required token could not be found. Set token or project_token in the provider configuration, or set RAILWAY_TOKEN or RAILWAY_PROJECT_TOKEN."
	errConflictingAuthToken = "Exactly one authentication kind must be configured. Set either token/RAILWAY_TOKEN or project_token/RAILWAY_PROJECT_TOKEN, not both."
)

const railwayGraphQLEndpoint = "https://backboard.railway.com/graphql/v2?source=terraform_provider_railway"

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
	Token        types.String `tfsdk:"token"`
	ProjectToken types.String `tfsdk:"project_token"`
}

func (p *RailwayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "railway"
	resp.Version = p.version
}

func (p *RailwayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Railway account or workspace token. Defaults to RAILWAY_TOKEN.",
				Optional:            true,
				Sensitive:           true,
			},
			"project_token": schema.StringAttribute{
				MarkdownDescription: "Railway project token. Defaults to RAILWAY_PROJECT_TOKEN.",
				Optional:            true,
				Sensitive:           true,
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
	if data.Token.IsUnknown() || data.ProjectToken.IsUnknown() {
		return
	}

	token := data.Token.ValueString()
	if token == "" {
		token = os.Getenv(tokenEnvVar)
	}
	projectToken := data.ProjectToken.ValueString()
	if projectToken == "" {
		projectToken = os.Getenv(projectTokenEnvVar)
	}

	if token == "" && projectToken == "" {
		resp.Diagnostics.AddError("Missing API token", errMissingAuthToken)
		return
	}
	if token != "" && projectToken != "" {
		resp.Diagnostics.AddError("Conflicting API tokens", errConflictingAuthToken)
		return
	}

	headerName := "Authorization"
	headerValue := "Bearer " + token
	if projectToken != "" {
		headerName = "Project-Access-Token"
		headerValue = projectToken
	}

	httpClient := http.Client{
		Transport: &authedTransport{
			headerName:  headerName,
			headerValue: headerValue,
			wrapped:     http.DefaultTransport,
		},
	}

	client := graphql.NewClient(railwayGraphQLEndpoint, &httpClient)

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
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewEnvironmentDataSource,
		NewServiceDataSource,
		NewCustomDomainDataSource,
		NewServiceDomainDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RailwayProvider{
			version: version,
		}
	}
}
