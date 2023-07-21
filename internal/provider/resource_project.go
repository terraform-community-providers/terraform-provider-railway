package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type ProjectResource struct {
	client *graphql.Client
}

type ProejctResourceDefaultEnvironmentModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var defaultEnvironmentAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
}

type ProjectResourceModel struct {
	Id                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Private            types.Bool   `tfsdk:"private"`
	HasPrDeploys       types.Bool   `tfsdk:"has_pr_deploys"`
	TeamId             types.String `tfsdk:"team_id"`
	DefaultEnvironment types.Object `tfsdk:"default_environment"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Optional:            true,
				Default:             stringdefault.StaticString(""),
			},
			"private": schema.BoolAttribute{
				MarkdownDescription: "Privacy of the project. **Default** `true`.",
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
			},
			"has_pr_deploys": schema.BoolAttribute{
				MarkdownDescription: "Whether the project has PR deploys enabled. **Default** `false`.",
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
			},
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the team the project belongs to.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"default_environment": schema.SingleNestedAttribute{
				MarkdownDescription: "Default environment of the project. When multiple exist, the oldest is considered.",
				Optional:            true,
				Computed:            true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						defaultEnvironmentAttrTypes,
						map[string]attr.Value{
							"id":   types.StringUnknown(),
							"name": types.StringValue("production"),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Identifier of the default environment.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the default environment.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("production"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectResourceModel
	var defaultEnvironmentData *ProejctResourceDefaultEnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectCreateInput{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsPublic:    !data.Private.ValueBool(),
		PrDeploys:   data.HasPrDeploys.ValueBool(),
		TeamId:      data.TeamId.ValueStringPointer(),
	}

	resp.Diagnostics.Append(data.DefaultEnvironment.As(ctx, &defaultEnvironmentData, basetypes.ObjectAsOptions{})...)

	if resp.Diagnostics.HasError() {
		return
	}

	input.DefaultEnvironmentName = defaultEnvironmentData.Name.ValueString()

	response, err := createProject(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a project")

	project := response.ProjectCreate.Project

	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.Private = types.BoolValue(!project.IsPublic)
	data.HasPrDeploys = types.BoolValue(project.PrDeploys)

	if project.Team != nil {
		data.TeamId = types.StringValue(project.Team.Id)
	}

	noOfEnvironments := len(project.Environments.Edges)

	if noOfEnvironments != 1 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Expected exactly one environment, got %d", noOfEnvironments))
		return
	}

	data.DefaultEnvironment = types.ObjectValueMust(
		defaultEnvironmentAttrTypes,
		map[string]attr.Value{
			"id":   types.StringValue(project.Environments.Edges[0].Node.Id),
			"name": types.StringValue(project.Environments.Edges[0].Node.Name),
		},
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, enviroment, err := defaultEnvironmentForProject(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.Private = types.BoolValue(!project.IsPublic)
	data.HasPrDeploys = types.BoolValue(project.PrDeploys)

	if project.Team != nil {
		data.TeamId = types.StringValue(project.Team.Id)
	}

	data.DefaultEnvironment = types.ObjectValueMust(
		defaultEnvironmentAttrTypes,
		map[string]attr.Value{
			"id":   types.StringValue(enviroment.Id),
			"name": types.StringValue(enviroment.Name),
		},
	)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectResourceModel
	var defaultEnvironmentData *ProejctResourceDefaultEnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectUpdateInput{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		IsPublic:    !data.Private.ValueBool(),
		PrDeploys:   data.HasPrDeploys.ValueBool(),
	}

	resp.Diagnostics.Append(data.DefaultEnvironment.As(ctx, &defaultEnvironmentData, basetypes.ObjectAsOptions{})...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := updateProject(ctx, *r.client, data.Id.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a project")

	project := response.ProjectUpdate.Project

	noOfEnvironments := len(project.Environments.Edges)

	if noOfEnvironments < 1 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Expected at least one environment, got %d", noOfEnvironments))
		return
	}

	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.Private = types.BoolValue(!project.IsPublic)
	data.HasPrDeploys = types.BoolValue(project.PrDeploys)

	if project.Team != nil {
		data.TeamId = types.StringValue(project.Team.Id)
	}

	data.DefaultEnvironment = types.ObjectValueMust(
		defaultEnvironmentAttrTypes,
		map[string]attr.Value{
			"id":   types.StringValue(project.Environments.Edges[0].Node.Id),
			"name": types.StringValue(project.Environments.Edges[0].Node.Name),
		},
	)

	// Renaming of environments is not allowed

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteProject(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a project")
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func defaultEnvironmentForProject(ctx context.Context, client graphql.Client, projectId string) (*Project, *ProjectEnvironmentsProjectEnvironmentsConnectionEdgesProjectEnvironmentsConnectionEdgeNodeEnvironment, error) {
	response, err := getProject(ctx, client, projectId)

	if err != nil {
		return nil, nil, err
	}

	project := response.Project.Project
	noOfEnvironments := len(project.Environments.Edges)

	if noOfEnvironments < 1 {
		return nil, nil, fmt.Errorf("expected at least one environment, got %d", noOfEnvironments)
	}

	// Mark the oldest environment as the default
	sort.SliceStable(project.Environments.Edges, func(i, j int) bool {
		return project.Environments.Edges[i].Node.CreatedAt.Before(project.Environments.Edges[j].Node.CreatedAt)
	})

	return &project, &project.Environments.Edges[0].Node, nil
}
