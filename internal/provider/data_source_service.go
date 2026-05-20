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

var _ datasource.DataSource = &ServiceDataSource{}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

type ServiceDataSource struct {
	client *graphql.Client
}

type ServiceDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ProjectId types.String `tfsdk:"project_id"`
}

func (d *ServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Railway service by name within a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the service belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (d *ServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := findServiceByName(ctx, *d.client, data.ProjectId.ValueString(), data.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	data.Id = types.StringValue(service.Id)
	data.Name = types.StringValue(service.Name)
	data.ProjectId = types.StringValue(service.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findServiceByName(ctx context.Context, client graphql.Client, projectId string, name string) (*Service, error) {
	var after *string
	candidates := make([]namedLookupCandidate[Service], 0)

	for {
		response, err := listServicesForDataSource(ctx, client, projectId, after)

		if err != nil {
			return nil, err
		}

		for _, edge := range response.Project.Services.Edges {
			candidates = append(candidates, namedLookupCandidate[Service]{
				name:  edge.Node.Name,
				value: edge.Node.Service,
			})
		}

		if !response.Project.Services.PageInfo.HasNextPage {
			break
		}

		after = &response.Project.Services.PageInfo.EndCursor

		if *after == "" {
			return nil, fmt.Errorf("services pagination indicated another page without an end cursor")
		}
	}

	service, err := selectUniqueByName("Railway service", name, fmt.Sprintf("project %q", projectId), candidates)

	if err != nil {
		return nil, err
	}

	return &service, nil
}
