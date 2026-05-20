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

var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

type EnvironmentDataSource struct {
	client *graphql.Client
}

type EnvironmentDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ProjectId types.String `tfsdk:"project_id"`
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Railway environment by name within a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the environment.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the environment belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := findEnvironmentByName(ctx, *d.client, data.ProjectId.ValueString(), data.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	data.Id = types.StringValue(environment.Id)
	data.Name = types.StringValue(environment.Name)
	data.ProjectId = types.StringValue(environment.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findEnvironmentByName(ctx context.Context, client graphql.Client, projectId string, name string) (*Environment, error) {
	var after *string
	candidates := make([]namedLookupCandidate[Environment], 0)

	for {
		response, err := listEnvironmentsForDataSource(ctx, client, projectId, after)

		if err != nil {
			return nil, err
		}

		for _, edge := range response.Environments.Edges {
			candidates = append(candidates, namedLookupCandidate[Environment]{
				name:  edge.Node.Name,
				value: edge.Node.Environment,
			})
		}

		if !response.Environments.PageInfo.HasNextPage {
			break
		}

		after = &response.Environments.PageInfo.EndCursor

		if *after == "" {
			return nil, fmt.Errorf("environments pagination indicated another page without an end cursor")
		}
	}

	environment, err := selectUniqueByName("Railway environment", name, fmt.Sprintf("project %q", projectId), candidates)

	if err != nil {
		return nil, err
	}

	return &environment, nil
}
