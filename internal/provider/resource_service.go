package provider

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

type ServiceResource struct {
	client *graphql.Client
}

type ServiceResourceVolumeModel struct {
	Id        types.String  `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	MountPath types.String  `tfsdk:"mount_path"`
	Size      types.Float64 `tfsdk:"size"`
}

var volumeAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"name":       types.StringType,
	"mount_path": types.StringType,
	"size":       types.Float64Type,
}

type ServiceResourceRegionModel struct {
	Region      types.String `tfsdk:"region"`
	NumReplicas types.Int64  `tfsdk:"num_replicas"`
}

var regionAttrTypes = map[string]attr.Type{
	"region":       types.StringType,
	"num_replicas": types.Int64Type,
}

// For JSON transformation
type numReplicas struct {
	NumReplicas int64 `json:"numReplicas"`
}

type ServiceResourceModel struct {
	Id                                 types.String `tfsdk:"id"`
	Name                               types.String `tfsdk:"name"`
	ProjectId                          types.String `tfsdk:"project_id"`
	CronSchedule                       types.String `tfsdk:"cron_schedule"`
	SourceImage                        types.String `tfsdk:"source_image"`
	SourceImagePrivateRegistryUsername types.String `tfsdk:"source_image_registry_username"`
	SourceImagePrivateRegistryPassword types.String `tfsdk:"source_image_registry_password"`
	SourceRepo                         types.String `tfsdk:"source_repo"`
	SourceRepoBranch                   types.String `tfsdk:"source_repo_branch"`
	RootDirectory                      types.String `tfsdk:"root_directory"`
	ConfigPath                         types.String `tfsdk:"config_path"`
	BuildCommand                       types.String `tfsdk:"build_command"`
	DockerfilePath                     types.String `tfsdk:"dockerfile_path"`
	WatchPatterns                      types.List   `tfsdk:"watch_patterns"`
	PredeployCommand                   types.String `tfsdk:"predeploy_command"`
	PredeployTimeout                   types.Int64  `tfsdk:"predeploy_timeout"`
	StartCommand                       types.String `tfsdk:"start_command"`
	HealthcheckPath                    types.String `tfsdk:"healthcheck_path"`
	HealthcheckTimeout                 types.Int64  `tfsdk:"healthcheck_timeout"`
	RestartPolicyType                  types.String `tfsdk:"restart_policy_type"`
	RestartPolicyMaxRetries            types.Int64  `tfsdk:"restart_policy_max_retries"`
	Serverless                         types.Bool   `tfsdk:"serverless"`
	Volume                             types.Object `tfsdk:"volume"`
	Regions                            types.List   `tfsdk:"regions"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"cron_schedule": schema.StringAttribute{
				MarkdownDescription: "Cron schedule of the service. Only allowed when total number of replicas across all regions is `1`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(9),
				},
			},
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Source image of the service. Conflicts with `source_repo`, `source_repo_branch`, `root_directory`, `config_path`, `build_command`, `dockerfile_path` and `watch_patterns`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo_branch")),
					stringvalidator.ConflictsWith(path.MatchRoot("root_directory")),
					stringvalidator.ConflictsWith(path.MatchRoot("config_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("build_command")),
					stringvalidator.ConflictsWith(path.MatchRoot("dockerfile_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("watch_patterns")),
				},
			},
			"source_image_registry_username": schema.StringAttribute{
				MarkdownDescription: "Private Docker registry credentials.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo_branch")),
					stringvalidator.ConflictsWith(path.MatchRoot("root_directory")),
					stringvalidator.ConflictsWith(path.MatchRoot("config_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("build_command")),
					stringvalidator.ConflictsWith(path.MatchRoot("dockerfile_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("watch_patterns")),
					stringvalidator.AlsoRequires(path.MatchRoot("source_image_registry_password")),
				},
			},
			"source_image_registry_password": schema.StringAttribute{
				MarkdownDescription: "Private Docker registry credentials.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo")),
					stringvalidator.ConflictsWith(path.MatchRoot("source_repo_branch")),
					stringvalidator.ConflictsWith(path.MatchRoot("root_directory")),
					stringvalidator.ConflictsWith(path.MatchRoot("config_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("build_command")),
					stringvalidator.ConflictsWith(path.MatchRoot("dockerfile_path")),
					stringvalidator.ConflictsWith(path.MatchRoot("watch_patterns")),
					stringvalidator.AlsoRequires(path.MatchRoot("source_image_registry_username")),
				},
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "Source repository of the service. Conflicts with `source_image`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
					stringvalidator.AlsoRequires(path.MatchRoot("source_repo_branch")),
				},
			},
			"source_repo_branch": schema.StringAttribute{
				MarkdownDescription: "Source repository branch to be used with `source_repo`. Must be specified if `source_repo` is specified.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.AlsoRequires(path.MatchRoot("source_repo")),
				},
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Directory to user for the service. Conflicts with `source_image`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Railway config file. Conflicts with `source_image`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"build_command": schema.StringAttribute{
				MarkdownDescription: "Command to run when building the service. Conflicts with `source_image` and `dockerfile_path`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_image")),
					stringvalidator.ConflictsWith(path.MatchRoot("dockerfile_path")),
				},
			},
			"dockerfile_path": schema.StringAttribute{
				MarkdownDescription: "Path to a custom Dockerfile. Conflicts with `source_image` and `build_command`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_image")),
				},
			},
			"watch_patterns": schema.ListAttribute{
				MarkdownDescription: "Gitignore-style patterns used to trigger deployments when matching files change. Patterns are relative to the repository root, regardless of `root_directory`. Conflicts with `source_image`. **Default** `[]`.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.UTF8LengthAtLeast(1)),
					listvalidator.ConflictsWith(path.MatchRoot("source_image")),
				},
			},
			"predeploy_command": schema.StringAttribute{
				MarkdownDescription: "Command to run before deploying the service. Combine multiple commands with `&&`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"predeploy_timeout": schema.Int64Attribute{
				MarkdownDescription: "Maximum time in seconds to wait for the predeploy command to succeed.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 3600),
				},
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "Command to start the service.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"healthcheck_path": schema.StringAttribute{
				MarkdownDescription: "HTTP path used to check deployment health.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"healthcheck_timeout": schema.Int64Attribute{
				MarkdownDescription: "Maximum time in seconds to wait for the health check to succeed.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 3600),
				},
			},
			"restart_policy_type": schema.StringAttribute{
				MarkdownDescription: "Restart policy: `ALWAYS`, `NEVER`, or `ON_FAILURE`. **Default** `ON_FAILURE`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALWAYS", "NEVER", "ON_FAILURE"),
				},
				Default: stringdefault.StaticString("ON_FAILURE"),
			},
			"restart_policy_max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum restart retries for the `ON_FAILURE` policy. **Default** `10`. Must be specified if `restart_policy_type` is `ON_FAILURE`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				Default: int64default.StaticInt64(10),
			},
			"serverless": schema.BoolAttribute{
				MarkdownDescription: "Allow the service to sleep when inactive. **Default** `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"volume": schema.SingleNestedAttribute{
				MarkdownDescription: "Volume connected to the service.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Identifier of the volume.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the volume.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
					"mount_path": schema.StringAttribute{
						MarkdownDescription: "Mount path of the volume.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.UTF8LengthAtLeast(1),
						},
					},
					"size": schema.Float64Attribute{
						MarkdownDescription: "Size of the volume in MB.",
						Computed:            true,
						PlanModifiers: []planmodifier.Float64{
							float64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "Regions with replicas to deploy service in.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"region": schema.StringAttribute{
							MarkdownDescription: "Region to deploy in.",
							Optional:            true,
							Computed:            true,
						},
						"num_replicas": schema.Int64Attribute{
							MarkdownDescription: "Number of replicas to deploy. **Default** `1`.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ServiceResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("source_image"),
			path.MatchRoot("source_repo"),
		),
		cronScheduleReplicasValidator{},
	}
}

type cronScheduleReplicasValidator struct{}

func (v cronScheduleReplicasValidator) Description(ctx context.Context) string {
	return "`cron_schedule` can only be set when total number of replicas across all regions is 1"
}

func (v cronScheduleReplicasValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cronScheduleReplicasValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ServiceResourceModel
	var regionsData []ServiceResourceRegionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Regions.ElementsAs(ctx, &regionsData, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.CronSchedule.IsNull() || data.CronSchedule.IsUnknown() || data.Regions.IsNull() || data.Regions.IsUnknown() {
		return
	}

	// Sum up replicas, treating unknown or null as default of 1 each
	var sum int64

	for _, region := range regionsData {
		if region.NumReplicas.IsUnknown() || region.NumReplicas.IsNull() {
			sum += 1
		} else {
			sum += region.NumReplicas.ValueInt64()
		}
	}

	if sum != 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("cron_schedule"),
			"Invalid `cron_schedule` with multiple replicas",
			fmt.Sprintf(
				"`cron_schedule` can only be set when total number of replicas across all regions is 1. Found %d replicas.",
				sum,
			),
		)
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel
	var regionsData *[]ServiceResourceRegionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Regions.ElementsAs(ctx, &regionsData, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ServiceCreateInput{
		Name:      data.Name.ValueString(),
		ProjectId: data.ProjectId.ValueString(),
	}

	response, err := createService(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a service")

	service := response.ServiceCreate.Service

	data.Id = types.StringValue(service.Id)
	data.Name = types.StringValue(service.Name)
	data.ProjectId = types.StringValue(service.ProjectId)

	instanceInput := buildServiceInstanceInput(data, regionsData)

	_, err = updateServiceInstance(ctx, *r.client, data.Id.ValueString(), instanceInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service settings, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created service settings")

	if !data.Volume.IsNull() {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		volumeResponse, err := createVolume(ctx, *r.client, VolumeCreateInput{
			MountPath: volumeData.MountPath.ValueString(),
			ProjectId: data.ProjectId.ValueString(),
			ServiceId: data.Id.ValueStringPointer(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "created a volume")

		_, err = updateVolume(ctx, *r.client, volumeResponse.VolumeCreate.Volume.Id, VolumeUpdateInput{
			Name: volumeData.Name.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated a volume")
	}

	if !data.SourceRepo.IsNull() || !data.SourceImage.IsNull() {
		connectInput := buildServiceConnectInput(data)

		_, err := connectService(ctx, *r.client, data.Id.ValueString(), connectInput)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect repo or image to service, got error: %s", err))
			return
		}
	}

	err = getAndBuildServiceInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
		return
	}

	err = getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getService(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	service := response.Service.Service

	data.Id = types.StringValue(service.Id)
	data.Name = types.StringValue(service.Name)
	data.ProjectId = types.StringValue(service.ProjectId)

	err = getAndBuildServiceInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
		return
	}

	err = getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel
	var regionsData *[]ServiceResourceRegionModel

	var state *ServiceResourceModel
	var volumeState *ServiceResourceVolumeModel
	var regionsState *[]ServiceResourceRegionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.Regions.ElementsAs(ctx, &regionsData, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(state.Regions.ElementsAs(ctx, &regionsState, true)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.ValueString() != state.Name.ValueString() {
		input := ServiceUpdateInput{
			Name: data.Name.ValueString(),
		}

		response, err := updateService(ctx, *r.client, data.Id.ValueString(), input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated a service")

		service := response.ServiceUpdate.Service

		data.Id = types.StringValue(service.Id)
		data.Name = types.StringValue(service.Name)
		data.ProjectId = types.StringValue(service.ProjectId)
	}

	var instanceInput ServiceInstanceUpdateInput

	if reflect.DeepEqual(regionsData, regionsState) {
		instanceInput = buildServiceInstanceInput(data, nil)
	} else {
		instanceInput = buildServiceInstanceInput(data, regionsData)
	}

	_, err := updateServiceInstance(ctx, *r.client, data.Id.ValueString(), instanceInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service settings, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated service settings")

	// Delete volume if it was removed
	if data.Volume.IsNull() && !state.Volume.IsNull() {
		resp.Diagnostics.Append(state.Volume.As(ctx, &volumeState, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		_, err := deleteVolume(ctx, *r.client, volumeState.Id.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "deleted a volume")
	}

	// Create volume if it was added
	if !data.Volume.IsNull() && state.Volume.IsNull() {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		volumeResponse, err := createVolume(ctx, *r.client, VolumeCreateInput{
			MountPath: volumeData.MountPath.ValueString(),
			ProjectId: data.ProjectId.ValueString(),
			ServiceId: data.Id.ValueStringPointer(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "created a volume")

		_, err = updateVolume(ctx, *r.client, volumeResponse.VolumeCreate.Volume.Id, VolumeUpdateInput{
			Name: volumeData.Name.ValueString(),
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "updated a volume")
	}

	// Update volume if it was changed
	if !data.Volume.IsNull() && !state.Volume.IsNull() {
		resp.Diagnostics.Append(state.Volume.As(ctx, &volumeState, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		if volumeState.Name != volumeData.Name {
			_, err := updateVolume(ctx, *r.client, volumeState.Id.ValueString(), VolumeUpdateInput{
				Name: volumeData.Name.ValueString(),
			})

			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume, got error: %s", err))
				return
			}

			tflog.Trace(ctx, "updated a volume")
		}

		if volumeState.MountPath != volumeData.MountPath {
			_, err := updateVolumeInstance(ctx, *r.client, volumeState.Id.ValueString(), VolumeInstanceUpdateInput{
				MountPath: volumeData.MountPath.ValueString(),
				ServiceId: data.Id.ValueString(),
			})

			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume instance, got error: %s", err))
				return
			}

			tflog.Trace(ctx, "updated a volume instance")
		}
	}

	// Handling service connection with source repo or docker image
	err = updateServiceConnection(ctx, *r.client, data.Id.ValueString(), data, state)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service repo or image connection, got error: %s", err))
	}

	err = redeployAllInstances(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy services after update, got error: %s", err))
		return
	}

	err = getAndBuildServiceInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
		return
	}

	err = getAndBuildVolumeInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume settings, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceResourceModel
	var volumeData *ServiceResourceVolumeModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteService(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a service")

	if !data.Volume.IsNull() {
		resp.Diagnostics.Append(data.Volume.As(ctx, &volumeData, basetypes.ObjectAsOptions{})...)

		if resp.Diagnostics.HasError() {
			return
		}

		_, err := deleteVolume(ctx, *r.client, volumeData.Id.ValueString())

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "deleted a volume")
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildServiceInstanceInput(data *ServiceResourceModel, regionsData *[]ServiceResourceRegionModel) ServiceInstanceUpdateInput {
	var instanceInput ServiceInstanceUpdateInput

	if !data.CronSchedule.IsNull() {
		instanceInput.CronSchedule = data.CronSchedule.ValueStringPointer()
	}

	if !data.RootDirectory.IsNull() {
		instanceInput.RootDirectory = data.RootDirectory.ValueStringPointer()
	}

	if !data.ConfigPath.IsNull() {
		instanceInput.RailwayConfigFile = data.ConfigPath.ValueStringPointer()
	}

	if !data.BuildCommand.IsUnknown() {
		command := data.BuildCommand.ValueString()
		instanceInput.BuildCommand = &command
	}

	if !data.DockerfilePath.IsUnknown() {
		path := data.DockerfilePath.ValueString()
		instanceInput.DockerfilePath = &path
	}

	if !data.WatchPatterns.IsUnknown() {
		patterns := []string{}

		for _, pattern := range data.WatchPatterns.Elements() {
			patterns = append(patterns, pattern.(types.String).ValueString())
		}

		instanceInput.WatchPatterns = patterns
	}

	// Railway uses empty commands and paths to clear these overrides, not null.
	if !data.PredeployCommand.IsUnknown() {
		commands := []string{}

		if !data.PredeployCommand.IsNull() {
			commands = append(commands, data.PredeployCommand.ValueString())
		}

		instanceInput.PreDeployCommand = &commands
	}

	if !data.PredeployTimeout.IsNull() && !data.PredeployTimeout.IsUnknown() {
		timeout := int(data.PredeployTimeout.ValueInt64())
		instanceInput.PreDeployTimeoutSeconds = &timeout
	}

	if !data.StartCommand.IsUnknown() {
		command := data.StartCommand.ValueString()
		instanceInput.StartCommand = &command
	}

	if !data.HealthcheckPath.IsUnknown() {
		command := data.HealthcheckPath.ValueString()
		instanceInput.HealthcheckPath = &command
	}

	if !data.HealthcheckTimeout.IsNull() && !data.HealthcheckTimeout.IsUnknown() {
		timeout := int(data.HealthcheckTimeout.ValueInt64())
		instanceInput.HealthcheckTimeout = &timeout
	}

	instanceInput.RestartPolicyType = RestartPolicyType(data.RestartPolicyType.ValueString())
	instanceInput.RestartPolicyMaxRetries = int(data.RestartPolicyMaxRetries.ValueInt64())
	instanceInput.SleepApplication = data.Serverless.ValueBool()

	if regionsData != nil {
		multiRegionConfig := make(map[string]interface{})

		for _, region := range *regionsData {
			multiRegionConfig[region.Region.ValueString()] = numReplicas{
				NumReplicas: region.NumReplicas.ValueInt64(),
			}
		}

		instanceInput.MultiRegionConfig = &multiRegionConfig
	}

	if !data.SourceImagePrivateRegistryUsername.IsNull() {
		if instanceInput.RegistryCredentials == nil {
			instanceInput.RegistryCredentials = new(RegistryCredentialsInput)
		}
		instanceInput.RegistryCredentials.Username = data.SourceImagePrivateRegistryUsername.ValueString()
	}

	if !data.SourceImagePrivateRegistryPassword.IsNull() {
		if instanceInput.RegistryCredentials == nil {
			instanceInput.RegistryCredentials = new(RegistryCredentialsInput)
		}
		instanceInput.RegistryCredentials.Password = data.SourceImagePrivateRegistryPassword.ValueString()
	}

	return instanceInput
}

func getAndBuildServiceInstance(ctx context.Context, client graphql.Client, projectId string, serviceId string, data *ServiceResourceModel) error {
	// Read the service again to get the updated source attributes
	_, environment, err := defaultEnvironmentForProject(ctx, client, projectId)

	if err != nil {
		return err
	}

	response, err := getServiceInstance(ctx, client, environment.Id, serviceId)

	if err != nil {
		return err
	}

	instance := response.ServiceInstance

	if instance.CronSchedule != nil {
		data.CronSchedule = types.StringValue(*instance.CronSchedule)
	}

	if instance.RootDirectory != nil && len(*instance.RootDirectory) != 0 {
		data.RootDirectory = types.StringValue(*instance.RootDirectory)
	}

	if instance.RailwayConfigFile != nil && len(*instance.RailwayConfigFile) != 0 {
		data.ConfigPath = types.StringValue(*instance.RailwayConfigFile)
	}

	if instance.BuildCommand != nil && len(*instance.BuildCommand) != 0 {
		data.BuildCommand = types.StringValue(*instance.BuildCommand)
	}

	if instance.DockerfilePath != nil && len(*instance.DockerfilePath) != 0 {
		data.DockerfilePath = types.StringValue(*instance.DockerfilePath)
	}

	patterns, diags := types.ListValueFrom(ctx, types.StringType, instance.WatchPatterns)

	if diags.HasError() {
		return fmt.Errorf("unable to read watch patterns: %s", diags)
	}

	data.WatchPatterns = patterns

	if instance.PreDeployCommand != nil && len(*instance.PreDeployCommand) > 0 && len((*instance.PreDeployCommand)[0]) != 0 {
		data.PredeployCommand = types.StringValue((*instance.PreDeployCommand)[0])
	}

	if instance.PreDeployTimeoutSeconds != nil {
		data.PredeployTimeout = types.Int64Value(int64(*instance.PreDeployTimeoutSeconds))
	}

	if instance.StartCommand != nil && len(*instance.StartCommand) != 0 {
		data.StartCommand = types.StringValue(*instance.StartCommand)
	}

	if instance.HealthcheckPath != nil && len(*instance.HealthcheckPath) != 0 {
		data.HealthcheckPath = types.StringValue(*instance.HealthcheckPath)
	}

	if instance.HealthcheckTimeout != nil {
		data.HealthcheckTimeout = types.Int64Value(int64(*instance.HealthcheckTimeout))
	}

	data.RestartPolicyType = types.StringValue(string(instance.RestartPolicyType))
	data.RestartPolicyMaxRetries = types.Int64Value(int64(instance.RestartPolicyMaxRetries))

	data.Serverless = types.BoolValue(instance.SleepApplication)

	if response.ServiceInstance.Source != nil {
		if response.ServiceInstance.Source.Image != nil {
			data.SourceImage = types.StringValue(*response.ServiceInstance.Source.Image)
		}

		if response.ServiceInstance.Source.Repo != nil {
			data.SourceRepo = types.StringValue(*response.ServiceInstance.Source.Repo)

			triggersResponse, err := listDeploymentTriggers(ctx, client, projectId, environment.Id, serviceId)

			if err != nil {
				return err
			}

			// up to 1 deployment trigger is allowed for one (service, environment) pair. So, dealing with [0] only
			if edges := triggersResponse.DeploymentTriggers.Edges; len(edges) > 0 {
				data.SourceRepoBranch = types.StringValue(edges[0].Node.Branch)
			} else if data.SourceRepoBranch.IsNull() || data.SourceRepoBranch.IsUnknown() {
				// Only set to null if there's no existing value
				// This preserves the branch value during updates when triggers might not be immediately available
				data.SourceRepoBranch = types.StringNull()
			}
			// Otherwise keep the existing value from state/plan
		}
	}

	if len(response.ServiceInstance.LatestDeployment.Meta) != 0 {
		regions, err := getRegionsFromLatestDeployment(response.ServiceInstance.LatestDeployment)

		if err != nil {
			return err
		}

		data.Regions = types.ListValueMust(types.ObjectType{AttrTypes: regionAttrTypes}, regions)
	} else if data.Regions.IsUnknown() {
		data.Regions = types.ListNull(types.ObjectType{AttrTypes: regionAttrTypes})
	}

	return nil
}

func getRegionsFromLatestDeployment(latestDeployment getServiceInstanceServiceInstanceLatestDeployment) ([]attr.Value, error) {
	serviceManifest, ok := latestDeployment.Meta["serviceManifest"].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("serviceManifest is not found")
	}

	deploy, ok := serviceManifest["deploy"].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("deploy is not found")
	}

	multiRegionConfig, ok := deploy["multiRegionConfig"].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("multiRegionConfig is not found")
	}

	regions := make([]attr.Value, 0, len(multiRegionConfig))

	for _, region := range slices.Sorted(maps.Keys(multiRegionConfig)) {
		numReplicasMap, ok := multiRegionConfig[region].(map[string]interface{})

		if !ok {
			return nil, fmt.Errorf("numReplicas is not found")
		}

		numReplicas, exists := numReplicasMap["numReplicas"]

		if !exists {
			return nil, fmt.Errorf("numReplicas is not found")
		}

		regions = append(regions, types.ObjectValueMust(regionAttrTypes, map[string]attr.Value{
			"region":       types.StringValue(region),
			"num_replicas": types.Int64Value(int64(numReplicas.(float64))),
		}))
	}

	return regions, nil
}

func getAndBuildVolumeInstance(ctx context.Context, client graphql.Client, projectId string, serviceId string, data *ServiceResourceModel) error {
	data.Volume = types.ObjectNull(volumeAttrTypes)

	// Read the service again to get the updated source attributes
	_, environment, err := defaultEnvironmentForProject(ctx, client, projectId)

	if err != nil {
		return err
	}

	response, err := getVolumeInstances(ctx, client, projectId)

	if err != nil {
		return err
	}

	for _, volume := range response.Project.Volumes.Edges {
		for _, volumeInstance := range volume.Node.VolumeInstances.Edges {
			if volumeInstance.Node.ServiceId == serviceId && volumeInstance.Node.EnvironmentId == environment.Id {
				data.Volume = types.ObjectValueMust(
					volumeAttrTypes,
					map[string]attr.Value{
						"id":         types.StringValue(volume.Node.Id),
						"name":       types.StringValue(volume.Node.Name),
						"mount_path": types.StringValue(volumeInstance.Node.MountPath),
						"size":       types.Float64Value(float64(volumeInstance.Node.SizeMB)),
					},
				)
			}
		}
	}

	return nil
}

func updateServiceConnection(ctx context.Context, client graphql.Client, serviceId string, data *ServiceResourceModel, state *ServiceResourceModel) error {
	isSourceChanged := !state.SourceRepo.Equal(data.SourceRepo) || !state.SourceRepoBranch.Equal(data.SourceRepoBranch) || !state.SourceImage.Equal(data.SourceImage)
	isSourcesChangedToNull := isSourceChanged && data.SourceRepo.IsNull() && data.SourceRepoBranch.IsNull() && data.SourceImage.IsNull()

	// if all sources are eventually nulls, then just disconnecting all the sources
	if isSourcesChangedToNull {
		_, err := disconnectService(ctx, client, serviceId)
		return err
	}

	// if some sources are really changed we just propagating these values to Railway. Data is pre-validated and Railway knows what to do.
	if isSourceChanged {
		connectInput := buildServiceConnectInput(data)
		_, err := connectService(ctx, client, serviceId, connectInput)
		return err
	}

	return nil
}

// Build proper input which populates only required fields for each case: either Repo + Branch, or SourceImage
func buildServiceConnectInput(data *ServiceResourceModel) ServiceConnectInput {
	if !data.SourceRepo.IsNull() {
		// it is guaranteed by schema that both of them are specified or both empty
		return ServiceConnectInput{
			Repo:   data.SourceRepo.ValueStringPointer(),
			Branch: data.SourceRepoBranch.ValueStringPointer(),
		}
	} else if !data.SourceImage.IsNull() {
		return ServiceConnectInput{
			Image: data.SourceImage.ValueStringPointer(),
		}
	}

	return ServiceConnectInput{}
}

func redeployAllInstances(ctx context.Context, client graphql.Client, serviceId string) error {
	instances, err := getServiceInstances(ctx, client, serviceId)

	if err != nil {
		return err
	}

	for _, instance := range instances.Service.ServiceInstances.Edges {
		_, err := redeployServiceInstance(ctx, client, instance.Node.EnvironmentId, serviceId)

		if err != nil {
			return err
		}
	}

	tflog.Trace(ctx, "redeployed all service instances")

	return nil
}
