package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ServiceDomainDataSource{}

func NewServiceDomainDataSource() datasource.DataSource {
	return &ServiceDomainDataSource{}
}

type ServiceDomainDataSource struct {
	client *graphql.Client
}

type ServiceDomainDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	Domain        types.String `tfsdk:"domain"`
	Suffix        types.String `tfsdk:"suffix"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
}

func (d *ServiceDomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_domain"
}

func (d *ServiceDomainDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Railway-generated domain for a service in an environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service domain.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Exact Railway-generated domain to look up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"suffix": schema.StringAttribute{
				MarkdownDescription: "Suffix of the Railway-generated domain.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project containing the service.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment containing the service domain.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service owning the service domain.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (d *ServiceDomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}

	d.client = client
}

func (d *ServiceDomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceDomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := listServiceDomains(ctx, *d.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list service domains, got error: %s", err))
		return
	}

	matches := make([]ServiceDomain, 0, 1)
	for _, result := range response.Domains.ServiceDomains {
		if result.ServiceDomain.Domain == data.Domain.ValueString() {
			matches = append(matches, result.ServiceDomain)
		}
	}
	if len(matches) != 1 {
		resp.Diagnostics.AddError("Service Domain Lookup Error", fmt.Sprintf("Expected exactly one service domain named %q for service %q in environment %q and project %q, found %d.", data.Domain.ValueString(), data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), data.ProjectId.ValueString(), len(matches)))
		return
	}

	domain := matches[0]
	data.Id = types.StringValue(domain.Id)
	data.Domain = types.StringValue(domain.Domain)
	data.Suffix = types.StringValue(domain.Suffix)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
