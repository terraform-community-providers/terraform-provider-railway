package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DeploymentTriggerResource{}
var _ resource.ResourceWithImportState = &DeploymentTriggerResource{}

func NewDeploymentTriggerResource() resource.Resource {
	return &DeploymentTriggerResource{}
}

type DeploymentTriggerResource struct {
	client *graphql.Client
}

type DeploymentTriggerResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Branch        types.String `tfsdk:"branch"`
	CheckSuites   types.Bool   `tfsdk:"check_suites"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *DeploymentTriggerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_trigger"
}

func (r *DeploymentTriggerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway deployment trigger.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the trigger.",
				Computed:            true,
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Repository for the trigger.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
				},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Branch for the trigger.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"check_suites": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for check suites to complete before deploying. Default `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment for the trigger.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service for the trigger.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project for the trigger.",
				Computed:            true,
			},
		},
	}
}

func (r *DeploymentTriggerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	input := DeploymentTriggerCreateInput{
		Provider:      "github",
		Repository:    data.Repository.ValueString(),
		Branch:        data.Branch.ValueString(),
		CheckSuites:   data.CheckSuites.ValueBool(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
		ProjectId:     service.Service.ProjectId,
	}

	response, err := createDeploymentTrigger(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a deployment trigger")

	trigger := response.DeploymentTriggerCreate.DeploymentTrigger

	data.Id = types.StringValue(trigger.Id)
	data.Repository = types.StringValue(trigger.Repository)
	data.Branch = types.StringValue(trigger.Branch)
	data.CheckSuites = types.BoolValue(trigger.CheckSuites)
	data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
	data.ServiceId = types.StringValue(trigger.ServiceId)
	data.ProjectId = types.StringValue(trigger.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := listDeploymentTriggers(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment triggers, got error: %s", err))
		return
	}

	triggers := response.DeploymentTriggers.Edges
	var trigger DeploymentTrigger

	if len(triggers) == 0 {
		resp.Diagnostics.AddError("Client Error", "No deployment triggers found")
		return
	}

	for _, t := range triggers {
		if t.Node.Id == data.Id.ValueString() {
			trigger = t.Node.DeploymentTrigger
			break
		}
	}

	if trigger.Id == "" {
		resp.Diagnostics.AddError("Client Error", "No deployment trigger found")
		return
	}

	data.Id = types.StringValue(trigger.Id)
	data.Repository = types.StringValue(trigger.Repository)
	data.Branch = types.StringValue(trigger.Branch)
	data.CheckSuites = types.BoolValue(trigger.CheckSuites)
	data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
	data.ServiceId = types.StringValue(trigger.ServiceId)
	data.ProjectId = types.StringValue(trigger.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := DeploymentTriggerUpdateInput{
		Repository:  data.Repository.ValueString(),
		Branch:      data.Branch.ValueString(),
		CheckSuites: data.CheckSuites.ValueBool(),
	}

	response, err := updateDeploymentTrigger(ctx, *r.client, data.Id.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a deployment trigger")

	trigger := response.DeploymentTriggerUpdate.DeploymentTrigger

	data.Id = types.StringValue(trigger.Id)
	data.Repository = types.StringValue(trigger.Repository)
	data.Branch = types.StringValue(trigger.Branch)
	data.CheckSuites = types.BoolValue(trigger.CheckSuites)
	data.EnvironmentId = types.StringValue(trigger.EnvironmentId)
	data.ServiceId = types.StringValue(trigger.ServiceId)
	data.ProjectId = types.StringValue(trigger.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteDeploymentTrigger(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a deployment trigger")
}

func (r *DeploymentTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_name. Got: %q", req.ID),
		)

		return
	}

	service, err := getService(ctx, *r.client, parts[0])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	projectId := service.Service.ProjectId
	environmentId, err := findEnvironment(ctx, *r.client, projectId, parts[1])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	response, err := listDeploymentTriggers(ctx, *r.client, projectId, *environmentId, parts[0])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment triggers, got error: %s", err))
		return
	}

	triggers := response.DeploymentTriggers.Edges

	if len(triggers) == 0 {
		resp.Diagnostics.AddError("Client Error", "No deployment triggers found")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), triggers[0].Node.DeploymentTrigger.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
}
