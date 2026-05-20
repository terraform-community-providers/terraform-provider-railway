package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

type ProjectDataSource struct {
	client *graphql.Client
}

type ProjectDataSourceModel struct {
	Id                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Private            types.Bool   `tfsdk:"private"`
	HasPrDeploys       types.Bool   `tfsdk:"has_pr_deploys"`
	WorkspaceId        types.String `tfsdk:"workspace_id"`
	DefaultEnvironment types.Object `tfsdk:"default_environment"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a Railway project by name within a workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the project.",
				Computed:            true,
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Privacy of the project.",
				Computed:            true,
			},
			"has_pr_deploys": schema.BoolAttribute{
				MarkdownDescription: "Whether the project has PR deploys enabled.",
				Computed:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the project belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"default_environment": schema.SingleNestedAttribute{
				MarkdownDescription: "Default environment of the project. When multiple exist, the oldest is considered.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Identifier of the default environment.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the default environment.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := findProjectByName(ctx, *d.client, data.WorkspaceId.ValueString(), data.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.Private = types.BoolValue(!project.IsPublic)
	data.HasPrDeploys = types.BoolValue(project.PrDeploys)

	if project.Workspace != nil {
		data.WorkspaceId = types.StringValue(project.Workspace.Id)
	}

	defaultEnvironment, err := defaultEnvironmentForProjectDataSource(project)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read default environment, got error: %s", err))
		return
	}

	data.DefaultEnvironment = types.ObjectValueMust(
		defaultEnvironmentAttrTypes,
		map[string]attr.Value{
			"id":   types.StringValue(defaultEnvironment.Id),
			"name": types.StringValue(defaultEnvironment.Name),
		},
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findProjectByName(ctx context.Context, client graphql.Client, workspaceId string, name string) (*Project, error) {
	var after *string
	candidates := make([]namedLookupCandidate[Project], 0)

	for {
		response, err := listProjectsForDataSource(ctx, client, workspaceId, after)

		if err != nil {
			return nil, err
		}

		for _, edge := range response.Projects.Edges {
			candidates = append(candidates, namedLookupCandidate[Project]{
				name:  edge.Node.Name,
				value: edge.Node.Project,
			})
		}

		if !response.Projects.PageInfo.HasNextPage {
			break
		}

		after = &response.Projects.PageInfo.EndCursor

		if *after == "" {
			return nil, fmt.Errorf("projects pagination indicated another page without an end cursor")
		}
	}

	project, err := selectUniqueByName("Railway project", name, fmt.Sprintf("workspace %q", workspaceId), candidates)

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func defaultEnvironmentForProjectDataSource(project *Project) (*ProjectEnvironmentsProjectEnvironmentsConnectionEdgesProjectEnvironmentsConnectionEdgeNodeEnvironment, error) {
	noOfEnvironments := len(project.Environments.Edges)

	if noOfEnvironments < 1 {
		return nil, fmt.Errorf("expected at least one environment, got %d", noOfEnvironments)
	}

	sort.SliceStable(project.Environments.Edges, func(i, j int) bool {
		return project.Environments.Edges[i].Node.CreatedAt.Before(project.Environments.Edges[j].Node.CreatedAt)
	})

	return &project.Environments.Edges[0].Node, nil
}
