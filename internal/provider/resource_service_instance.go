package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ServiceInstanceResource{}
var _ resource.ResourceWithImportState = &ServiceInstanceResource{}

func NewServiceInstanceResource() resource.Resource {
	return &ServiceInstanceResource{}
}

type ServiceInstanceResource struct {
	client *graphql.Client
}

type ServiceInstanceResourceModel struct {
	Id                 types.String `tfsdk:"id"`
	ServiceId          types.String `tfsdk:"service_id"`
	EnvironmentId      types.String `tfsdk:"environment_id"`
	BuildCommand       types.String `tfsdk:"build_command"`
	Builder            types.String `tfsdk:"builder"`
	CronSchedule       types.String `tfsdk:"cron_schedule"`
	HealthcheckPath    types.String `tfsdk:"healthcheck_path"`
	HealthcheckTimeout types.Int64  `tfsdk:"healthcheck_timeout"`
	NumReplicas        types.Int64  `tfsdk:"num_replicas"`
	Region             types.String `tfsdk:"region"`
	RootDirectory      types.String `tfsdk:"root_directory"`
	StartCommand       types.String `tfsdk:"start_command"`
	SourceImage        types.String `tfsdk:"source_image"`
	SourceRepo         types.String `tfsdk:"source_repo"`
}

func (r *ServiceInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_instance"
}

func (r *ServiceInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: strings.TrimSpace(`
Per-environment configuration for an existing Railway service. Railway's data
model treats a service as project-scoped (identity: ` + "`name`, `project_id`" + `)
and a *service instance* as the per-environment configuration: source,
replica count, region, build/start commands, etc.

Railway does not expose explicit create/delete on a service instance — it
exists implicitly when the parent service exists in an environment. This
resource therefore maps Create + Update to ` + "`serviceInstanceUpdate`" + ` and
treats Delete as a no-op (the instance disappears with the parent service).

Deploy branch is owned by ` + "`railway_deployment_trigger`" + `, not here.
		`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the parent service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "Override the Nixpacks/Railpack build command for this instance.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"builder": schema.StringAttribute{
				MarkdownDescription: "Builder to use: `NIXPACKS` (default), `RAILPACK`, `PAKETO`, or `HEROKU`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("NIXPACKS", "RAILPACK", "PAKETO", "HEROKU"),
				},
			},
			"cron_schedule": schema.StringAttribute{
				MarkdownDescription: "Cron schedule for this instance (cron-like syntax, e.g. `*/5 * * * *`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"healthcheck_path": schema.StringAttribute{
				MarkdownDescription: "HTTP path Railway probes to consider the instance healthy.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"healthcheck_timeout": schema.Int64Attribute{
				MarkdownDescription: "Healthcheck timeout in seconds.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"num_replicas": schema.Int64Attribute{
				MarkdownDescription: "Number of replicas to run.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Railway region (e.g. `us-east4-eqdc4a`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Repository sub-directory for builds.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "Container start command override.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Docker image source (e.g. `redis:7-alpine`). Mutually exclusive with `source_repo`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
				},
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "GitHub `owner/name` repo source. Mutually exclusive with `source_image`. Branch lives on `railway_deployment_trigger`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServiceInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyInstance(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "configured a service instance")

	if err := r.refreshFromRailway(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read back service instance, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refreshFromRailway(ctx, data); err != nil {
		if strings.Contains(err.Error(), "no service instance") {
			// Service was deleted in Railway; surface as state removal.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service instance, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyInstance(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a service instance")

	if err := r.refreshFromRailway(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read back service instance, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Railway service instances exist implicitly as long as the parent
	// service exists. There's no `serviceInstanceDelete` mutation. Treat
	// Delete as a no-op: Terraform removes the resource from state, the
	// underlying instance configuration remains until the service or
	// environment is destroyed.
	tflog.Trace(ctx, "service_instance delete is a no-op (Railway has no serviceInstanceDelete)")
}

func (r *ServiceInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected format service_id:environment_id. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
}

func (r *ServiceInstanceResource) applyInstance(ctx context.Context, data *ServiceInstanceResourceModel) error {
	input := ServiceInstanceUpdateInput{}

	if !data.BuildCommand.IsNull() && !data.BuildCommand.IsUnknown() {
		input.BuildCommand = data.BuildCommand.ValueStringPointer()
	}
	if !data.Builder.IsNull() && !data.Builder.IsUnknown() {
		v := Builder(data.Builder.ValueString())
		input.Builder = &v
	}
	if !data.CronSchedule.IsNull() && !data.CronSchedule.IsUnknown() {
		input.CronSchedule = data.CronSchedule.ValueStringPointer()
	}
	if !data.HealthcheckPath.IsNull() && !data.HealthcheckPath.IsUnknown() {
		input.HealthcheckPath = data.HealthcheckPath.ValueStringPointer()
	}
	if !data.HealthcheckTimeout.IsNull() && !data.HealthcheckTimeout.IsUnknown() {
		v := int(data.HealthcheckTimeout.ValueInt64())
		input.HealthcheckTimeout = &v
	}
	if !data.NumReplicas.IsNull() && !data.NumReplicas.IsUnknown() {
		v := int(data.NumReplicas.ValueInt64())
		input.NumReplicas = &v
	}
	if !data.Region.IsNull() && !data.Region.IsUnknown() {
		input.Region = data.Region.ValueStringPointer()
	}
	if !data.RootDirectory.IsNull() && !data.RootDirectory.IsUnknown() {
		input.RootDirectory = data.RootDirectory.ValueStringPointer()
	}
	if !data.StartCommand.IsNull() && !data.StartCommand.IsUnknown() {
		input.StartCommand = data.StartCommand.ValueStringPointer()
	}

	hasSource := (!data.SourceImage.IsNull() && !data.SourceImage.IsUnknown()) ||
		(!data.SourceRepo.IsNull() && !data.SourceRepo.IsUnknown())
	if hasSource {
		src := &ServiceSourceInput{}
		if !data.SourceImage.IsNull() && !data.SourceImage.IsUnknown() {
			src.Image = data.SourceImage.ValueStringPointer()
		}
		if !data.SourceRepo.IsNull() && !data.SourceRepo.IsUnknown() {
			src.Repo = data.SourceRepo.ValueStringPointer()
		}
		input.Source = src
	}

	_, err := updateServiceInstanceForEnv(ctx, *r.client, data.ServiceId.ValueString(), data.EnvironmentId.ValueString(), input)
	return err
}

func (r *ServiceInstanceResource) refreshFromRailway(ctx context.Context, data *ServiceInstanceResourceModel) error {
	resp, err := listServiceInstances(ctx, *r.client, data.ServiceId.ValueString())
	if err != nil {
		return err
	}

	for _, edge := range resp.Service.ServiceInstances.Edges {
		if edge.Node.EnvironmentId == data.EnvironmentId.ValueString() {
			si := edge.Node.ServiceInstance
			data.Id = types.StringValue(si.Id)
			data.BuildCommand = optionalString(si.BuildCommand)
			data.Builder = types.StringValue(string(si.Builder))
			data.CronSchedule = optionalString(si.CronSchedule)
			data.HealthcheckPath = optionalString(si.HealthcheckPath)
			if si.HealthcheckTimeout == 0 {
				data.HealthcheckTimeout = types.Int64Null()
			} else {
				data.HealthcheckTimeout = types.Int64Value(int64(si.HealthcheckTimeout))
			}
			if si.NumReplicas == 0 {
				data.NumReplicas = types.Int64Null()
			} else {
				data.NumReplicas = types.Int64Value(int64(si.NumReplicas))
			}
			data.Region = optionalString(si.Region)
			data.RootDirectory = optionalString(si.RootDirectory)
			data.StartCommand = optionalString(si.StartCommand)
			data.SourceImage = optionalString(si.Source.Image)
			data.SourceRepo = optionalString(si.Source.Repo)
			return nil
		}
	}

	return fmt.Errorf("no service instance for (service=%s, env=%s)", data.ServiceId.ValueString(), data.EnvironmentId.ValueString())
}

func optionalString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
