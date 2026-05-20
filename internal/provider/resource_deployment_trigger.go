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
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	Provider      types.String `tfsdk:"source_provider"`
	Repository    types.String `tfsdk:"repository"`
	Branch        types.String `tfsdk:"branch"`
	CheckSuites   types.Bool   `tfsdk:"check_suites"`
	RootDirectory types.String `tfsdk:"root_directory"`
}

func (r *DeploymentTriggerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_trigger"
}

func (r *DeploymentTriggerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway deployment trigger. One trigger per (service, environment) tuple binds a service-instance to a branch of a source repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the deployment trigger.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the trigger belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment scope for the trigger.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the trigger targets.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"source_provider": schema.StringAttribute{
				MarkdownDescription: "Source provider name (e.g. `github`). Renamed from `provider` to avoid collision with Terraform's reserved meta-argument. Create-time only; change requires replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Source repository in `owner/name` form.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
				},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Branch to deploy from. Updating this field updates the trigger in place.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"check_suites": schema.BoolAttribute{
				MarkdownDescription: "Whether the trigger waits for GitHub check suites to succeed before deploying.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Sub-directory within the repository to use for deploys. Not exposed on the Railway read API; preserved in state and may be updated.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// Railway's DeploymentTrigger type does not expose
					// rootDirectory on readback, so keep whatever state has
					// rather than tripping plan-consistency on every refresh.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DeploymentTriggerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	input := DeploymentTriggerCreateInput{
		ProjectId:     data.ProjectId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
		Provider:      data.Provider.ValueString(),
		Repository:    data.Repository.ValueString(),
		Branch:        data.Branch.ValueString(),
	}

	if !data.CheckSuites.IsNull() && !data.CheckSuites.IsUnknown() {
		v := data.CheckSuites.ValueBool()
		input.CheckSuites = &v
	}

	if !data.RootDirectory.IsNull() && !data.RootDirectory.IsUnknown() {
		input.RootDirectory = data.RootDirectory.ValueStringPointer()
	}

	response, err := createDeploymentTrigger(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a deployment trigger")

	trigger := response.DeploymentTriggerCreate.DeploymentTrigger
	setDeploymentTriggerState(&trigger, data)

	// root_directory is Optional+Computed and not returned by Railway. After
	// Create, Terraform requires a concrete value (Null or known) — leaving
	// it Unknown trips the "Provider returned invalid result object" check.
	if data.RootDirectory.IsUnknown() {
		data.RootDirectory = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	trigger, err := findDeploymentTrigger(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Id.ValueString())

	if err != nil {
		if strings.Contains(err.Error(), "no deployment trigger") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment trigger, got error: %s", err))
		return
	}

	setDeploymentTriggerState(trigger, data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *DeploymentTriggerResourceModel
	var state *DeploymentTriggerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	branch := data.Branch.ValueString()
	repository := data.Repository.ValueString()
	checkSuites := data.CheckSuites.ValueBool()

	input := DeploymentTriggerUpdateInput{
		Branch:      &branch,
		Repository:  &repository,
		CheckSuites: &checkSuites,
	}

	if !data.RootDirectory.IsNull() && !data.RootDirectory.IsUnknown() {
		input.RootDirectory = data.RootDirectory.ValueStringPointer()
	}

	response, err := updateDeploymentTrigger(ctx, *r.client, state.Id.ValueString(), input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update deployment trigger, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a deployment trigger")

	trigger := response.DeploymentTriggerUpdate.DeploymentTrigger
	setDeploymentTriggerState(&trigger, data)

	if data.RootDirectory.IsUnknown() {
		data.RootDirectory = types.StringNull()
	}

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

	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected format project_id:environment_id:service_id:trigger_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

// findDeploymentTrigger fetches the trigger by listing all triggers scoped to
// (project, env, service) and matching on id. Railway enforces at most one
// trigger per (service, env), but we filter by id anyway so a future API
// expansion that allows multiple doesn't silently match the wrong one.
func findDeploymentTrigger(ctx context.Context, client graphql.Client, projectId, environmentId, serviceId, triggerId string) (*DeploymentTrigger, error) {
	var after *string

	for {
		response, err := listDeploymentTriggers(ctx, client, projectId, environmentId, serviceId, after)
		if err != nil {
			return nil, err
		}

		for _, edge := range response.DeploymentTriggers.Edges {
			if edge.Node.Id == triggerId {
				t := edge.Node.DeploymentTrigger
				return &t, nil
			}
		}

		if !response.DeploymentTriggers.PageInfo.HasNextPage {
			break
		}
		after = &response.DeploymentTriggers.PageInfo.EndCursor
	}

	return nil, fmt.Errorf("no deployment trigger with id %q in (project=%s, env=%s, service=%s)", triggerId, projectId, environmentId, serviceId)
}

func setDeploymentTriggerState(t *DeploymentTrigger, data *DeploymentTriggerResourceModel) {
	data.Id = types.StringValue(t.Id)
	data.ProjectId = types.StringValue(t.ProjectId)
	data.EnvironmentId = types.StringValue(t.EnvironmentId)
	if t.ServiceId != "" {
		data.ServiceId = types.StringValue(t.ServiceId)
	}
	data.Provider = types.StringValue(t.Provider)
	data.Repository = types.StringValue(t.Repository)
	data.Branch = types.StringValue(t.Branch)
	data.CheckSuites = types.BoolValue(t.CheckSuites)
	// root_directory is intentionally not set from t: Railway does not return
	// it on read, so we keep whatever value is already on data (state).
}
