package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

var _ resource.Resource = &ServiceInstanceResource{}
var _ resource.ResourceWithConfigValidators = &ServiceInstanceResource{}
var _ resource.ResourceWithImportState = &ServiceInstanceResource{}

func NewServiceInstanceResource() resource.Resource {
	return &ServiceInstanceResource{}
}

type ServiceInstanceResource struct {
	client *graphql.Client
}

type ServiceInstanceResourceModel struct {
	Id                                 types.String  `tfsdk:"id"`
	ProjectId                          types.String  `tfsdk:"project_id"`
	EnvironmentId                      types.String  `tfsdk:"environment_id"`
	ServiceId                          types.String  `tfsdk:"service_id"`
	CronSchedule                       types.String  `tfsdk:"cron_schedule"`
	SourceImage                        types.String  `tfsdk:"source_image"`
	SourceImagePrivateRegistryUsername types.String  `tfsdk:"source_image_registry_username"`
	SourceImagePrivateRegistryPassword types.String  `tfsdk:"source_image_registry_password"`
	SourceRepo                         types.String  `tfsdk:"source_repo"`
	SourceRepoBranch                   types.String  `tfsdk:"source_repo_branch"`
	RootDirectory                      types.String  `tfsdk:"root_directory"`
	ConfigPath                         types.String  `tfsdk:"config_path"`
	Regions                            types.Map     `tfsdk:"regions"`
	EffectiveRegions                   types.Map     `tfsdk:"effective_regions"`
	StartCommand                       types.String  `tfsdk:"start_command"`
	HealthcheckPath                    types.String  `tfsdk:"healthcheck_path"`
	HealthcheckTimeout                 types.Int64   `tfsdk:"healthcheck_timeout"`
	VCpus                              types.Float64 `tfsdk:"vcpus"`
	MemoryGb                           types.Float64 `tfsdk:"memory_gb"`
}

type ServiceInstanceRegionModel struct {
	NumReplicas types.Int64 `tfsdk:"num_replicas"`
}

var serviceInstanceRegionAttrTypes = map[string]attr.Type{
	"num_replicas": types.Int64Type,
}

func (r *ServiceInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_instance"
}

func (r *ServiceInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Railway service instance in one environment, including its source, deployment settings, and resource limits. Omitted attributes reset to Railway defaults. Removing this resource deletes the environment instance without deleting the project-level service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project containing the service and environment.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"cron_schedule": schema.StringAttribute{
				MarkdownDescription: "Cron schedule for this service instance. Only allowed when the total number of replicas across all regions is `1`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.OneOf(""),
						stringvalidator.UTF8LengthAtLeast(9),
					),
				},
			},
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Container image used by this service instance. Omit to disconnect the image source.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"source_image_registry_username": schema.StringAttribute{
				MarkdownDescription: "Username used to pull the service instance image from a private registry.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"source_image_registry_password": schema.StringAttribute{
				MarkdownDescription: "Password used to pull the service instance image from a private registry.",
				Optional:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "Source repository used by this service instance.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"source_repo_branch": schema.StringAttribute{
				MarkdownDescription: "Repository branch deployed by this service instance.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Root directory used to build this service instance.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Railway configuration file used by this service instance.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"regions": schema.MapNestedAttribute{
				MarkdownDescription: "Region and replica overrides for this service instance. Omit to let Railway choose the effective placement. After import, configure this attribute explicitly to retain the existing placement as an override.",
				Optional:            true,
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"num_replicas": schema.Int64Attribute{
							MarkdownDescription: "Number of replicas to deploy.",
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
			"effective_regions": schema.MapNestedAttribute{
				MarkdownDescription: "Regions and replica counts currently present in Railway's environment configuration.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"num_replicas": schema.Int64Attribute{
							MarkdownDescription: "Effective number of replicas.",
							Computed:            true,
						},
					},
				},
			},
			"start_command": schema.StringAttribute{
				MarkdownDescription: "Command used to start this service instance. Omit to reset the command override.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"healthcheck_path": schema.StringAttribute{
				MarkdownDescription: "HTTP path Railway uses to check this service instance. Omit to disable the healthcheck.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"healthcheck_timeout": schema.Int64Attribute{
				MarkdownDescription: "Maximum healthcheck duration in seconds. Omit or set to zero to reset the timeout.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"vcpus": schema.Float64Attribute{
				MarkdownDescription: "Number of vCPUs allocated to this service instance. Omit or set to zero to reset the CPU override.",
				Optional:            true,
				Computed:            true,
				Default:             float64default.StaticFloat64(0),
				Validators: []validator.Float64{
					float64validator.Any(
						float64validator.OneOf(0),
						float64validator.AtLeast(0.01),
					),
				},
			},
			"memory_gb": schema.Float64Attribute{
				MarkdownDescription: "Memory in GB allocated to this service instance. Omit or set to zero to reset the memory override.",
				Optional:            true,
				Computed:            true,
				Default:             float64default.StaticFloat64(0),
				Validators: []validator.Float64{
					float64validator.Any(
						float64validator.OneOf(0),
						float64validator.AtLeast(0.01),
					),
				},
			},
		},
	}
}

func (r *ServiceInstanceResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		serviceInstanceSourceValidator{},
		serviceInstanceCronScheduleValidator{},
	}
}

type serviceInstanceSourceValidator struct{}

func (serviceInstanceSourceValidator) Description(context.Context) string {
	return "service instance source configuration must select one complete image or repository source"
}

func (v serviceInstanceSourceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (serviceInstanceSourceValidator) ValidateResource(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	values := []types.String{
		data.SourceImage,
		data.SourceRepo,
		data.SourceRepoBranch,
		data.RootDirectory,
		data.ConfigPath,
		data.SourceImagePrivateRegistryUsername,
		data.SourceImagePrivateRegistryPassword,
	}
	for _, value := range values {
		if value.IsUnknown() {
			return
		}
	}

	image := data.SourceImage.ValueString()
	repo := data.SourceRepo.ValueString()
	branch := data.SourceRepoBranch.ValueString()
	username := data.SourceImagePrivateRegistryUsername.ValueString()
	password := data.SourceImagePrivateRegistryPassword.ValueString()

	if image != "" && repo != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_image"),
			"Conflicting service instance sources",
			"`source_image` and `source_repo` cannot both be configured.",
		)
	}
	if (repo == "") != (branch == "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_repo"),
			"Incomplete repository source",
			"`source_repo` and `source_repo_branch` must be configured together.",
		)
	}
	if image != "" && (data.RootDirectory.ValueString() != "" || data.ConfigPath.ValueString() != "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_image"),
			"Invalid image source configuration",
			"`root_directory` and `config_path` can only be configured for repository sources.",
		)
	}
	if (username == "") != (password == "") {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_image_registry_username"),
			"Incomplete registry credentials",
			"`source_image_registry_username` and `source_image_registry_password` must be configured together.",
		)
	}
	if username != "" && image == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("source_image_registry_username"),
			"Registry credentials require an image source",
			"Private registry credentials can only be configured with `source_image`.",
		)
	}
}

type serviceInstanceCronScheduleValidator struct{}

func (serviceInstanceCronScheduleValidator) Description(context.Context) string {
	return "`cron_schedule` can only be set when the total number of replicas across all regions is 1"
}

func (v serviceInstanceCronScheduleValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (serviceInstanceCronScheduleValidator) ValidateResource(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.CronSchedule.IsUnknown() || data.CronSchedule.ValueString() == "" ||
		data.Regions.IsNull() || data.Regions.IsUnknown() {
		return
	}

	var regions map[string]ServiceInstanceRegionModel
	resp.Diagnostics.Append(data.Regions.ElementsAs(ctx, &regions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var replicas int64 = 1
	if len(regions) > 0 {
		replicas = 0
		for _, region := range regions {
			if region.NumReplicas.IsNull() || region.NumReplicas.IsUnknown() {
				replicas++
			} else {
				replicas += region.NumReplicas.ValueInt64()
			}
		}
	}
	if replicas != 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("cron_schedule"),
			"Invalid cron schedule with multiple replicas",
			fmt.Sprintf("`cron_schedule` requires exactly one replica; configured regions contain %d.", replicas),
		)
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
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.resolveProjectID(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine service instance project, got error: %s", err))
		return
	}
	if err := r.create(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service instance, got error: %s", err))
		return
	}

	data.Id = types.StringValue(serviceInstanceResourceId(data))
	data.EffectiveRegions = data.Regions
	if data.EffectiveRegions.IsNull() || data.EffectiveRegions.IsUnknown() {
		data.EffectiveRegions = types.MapValueMust(
			types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes},
			map[string]attr.Value{},
		)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to configure created service instance, got error: %s", err))
		return
	}
	attached, err := r.read(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created service instance, got error: %s", err))
		return
	}
	if !attached {
		resp.Diagnostics.AddError("Client Error", "Railway did not create the service instance in the requested environment.")
		return
	}
	tflog.Trace(ctx, "created service instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attached, err := r.read(ctx, &data)
	if err != nil {
		exists, verificationErr := r.serviceInstanceExists(ctx, data)
		if verificationErr != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf(
					"Unable to read service instance settings, got error: %s. Unable to verify service instance existence, got error: %s",
					err,
					verificationErr,
				),
			)
			return
		}
		if !exists {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service instance settings, got error: %s", err))
		return
	}
	if !attached {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service instance settings, got error: %s", err))
		return
	}

	data.Id = types.StringValue(serviceInstanceResourceId(data))
	attached, err := r.read(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read updated service instance settings, got error: %s", err))
		return
	}
	if !attached {
		resp.Diagnostics.AddError("Client Error", "Railway removed the service instance from the environment while updating it.")
		return
	}
	tflog.Trace(ctx, "updated service instance settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		railway.DeleteServiceInstancePatch(data.ServiceId.ValueString()),
		"Delete Terraform service instance",
	); err != nil {
		exists, verificationErr := r.serviceInstanceExists(ctx, data)
		if verificationErr != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf(
					"Unable to delete service instance, got error: %s. Unable to verify service instance existence, got error: %s",
					err,
					verificationErr,
				),
			)
			return
		}
		if !exists {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service instance, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted service instance")
}

func (r *ServiceInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:service_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[1])...)
	service, err := getService(ctx, *r.client, parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine imported service instance project, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), service.Service.ProjectId)...)
	// Adopt the current placement during import so omitting regions in configuration plans a reset.
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx,
		path.Root("regions"),
		types.MapValueMust(types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes}, map[string]attr.Value{}),
	)...)
}

func (r *ServiceInstanceResource) apply(ctx context.Context, data ServiceInstanceResourceModel) error {
	regions, err := serviceInstanceRegions(ctx, data.Regions)
	if err != nil {
		return err
	}

	current, err := getManagedServiceInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.ServiceId.ValueString(),
	)
	if err != nil {
		return err
	}

	if err := commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		expandServiceInstanceEnvironmentPatch(data, regions, current.Environment.Config),
		"Manage Terraform service instance",
	); err != nil {
		return err
	}

	return nil
}

func (r *ServiceInstanceResource) create(ctx context.Context, data ServiceInstanceResourceModel) error {
	regions, err := serviceInstanceRegions(ctx, data.Regions)
	if err != nil {
		return err
	}

	environment, err := getEnvironmentServiceInstances(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.EnvironmentId.ValueString(),
		nil,
	)
	if err != nil {
		return err
	}

	patch := expandServiceInstanceEnvironmentPatch(data, regions, environment.Environment.Config)
	service := patch.Services[data.ServiceId.ValueString()]
	service.IsCreated = true
	patch.Services[data.ServiceId.ValueString()] = service

	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		"Create Terraform service instance",
	)
}

func (r *ServiceInstanceResource) read(ctx context.Context, data *ServiceInstanceResourceModel) (bool, error) {
	response, err := getManagedServiceInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.ServiceId.ValueString(),
	)
	if err != nil {
		return false, err
	}

	if _, attached := response.Environment.Config.Services[data.ServiceId.ValueString()]; !attached {
		return false, nil
	}

	instance := response.ServiceInstance
	if instance.Source != nil && instance.Source.Image != nil {
		data.SourceImage = types.StringValue(*instance.Source.Image)
	} else {
		data.SourceImage = types.StringValue("")
	}
	if instance.Source != nil && instance.Source.Repo != nil {
		data.SourceRepo = types.StringValue(*instance.Source.Repo)
	} else {
		data.SourceRepo = types.StringValue("")
	}
	data.SourceRepoBranch = types.StringValue(serviceInstanceSourceBranch(
		response.Environment.Config,
		data.ServiceId.ValueString(),
	))
	data.CronSchedule = optionalStringValue(instance.CronSchedule)
	data.RootDirectory = optionalStringValue(instance.RootDirectory)
	data.ConfigPath = optionalStringValue(instance.RailwayConfigFile)
	if data.Regions.IsNull() || data.Regions.IsUnknown() {
		data.Regions = types.MapNull(types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes})
	} else {
		data.Regions = serviceInstanceRegionValue(response.Environment.Config, data.ServiceId.ValueString())
	}
	data.EffectiveRegions = serviceInstanceRegionValue(response.Environment.Config, data.ServiceId.ValueString())
	if instance.StartCommand != nil {
		data.StartCommand = types.StringValue(*instance.StartCommand)
	} else {
		data.StartCommand = types.StringValue("")
	}
	if instance.HealthcheckPath != nil {
		data.HealthcheckPath = types.StringValue(*instance.HealthcheckPath)
	} else {
		data.HealthcheckPath = types.StringValue("")
	}
	if instance.HealthcheckTimeout != nil {
		data.HealthcheckTimeout = types.Int64Value(int64(*instance.HealthcheckTimeout))
	} else {
		data.HealthcheckTimeout = types.Int64Value(0)
	}

	vcpus, memoryGb := flattenServiceInstanceLimits(response.ServiceInstanceLimitOverride)
	data.VCpus = types.Float64Value(vcpus)
	data.MemoryGb = types.Float64Value(memoryGb)
	data.Id = types.StringValue(serviceInstanceResourceId(*data))
	return true, nil
}

func serviceInstanceResourceId(data ServiceInstanceResourceModel) string {
	return fmt.Sprintf("%s:%s", data.EnvironmentId.ValueString(), data.ServiceId.ValueString())
}

func (r *ServiceInstanceResource) resolveProjectID(ctx context.Context, data *ServiceInstanceResourceModel) error {
	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())
	if err != nil {
		return err
	}
	data.ProjectId = types.StringValue(service.Service.ProjectId)
	return nil
}

func (r *ServiceInstanceResource) serviceInstanceExists(
	ctx context.Context,
	data ServiceInstanceResourceModel,
) (bool, error) {
	var after *string
	environmentFound := false
	for {
		response, err := getServiceInstanceProjectEnvironments(
			ctx,
			*r.client,
			data.ProjectId.ValueString(),
			after,
		)
		if err != nil {
			return false, fmt.Errorf("unable to list project environments: %w", err)
		}

		environments := response.Project.Environments
		for _, edge := range environments.Edges {
			if edge.Node.Id == data.EnvironmentId.ValueString() {
				environmentFound = true
				break
			}
		}
		if environmentFound {
			break
		}
		if !environments.PageInfo.HasNextPage {
			return false, nil
		}
		if environments.PageInfo.EndCursor == "" {
			return false, fmt.Errorf(
				"Railway environment connection has a next page but no end cursor",
			)
		}
		after = &environments.PageInfo.EndCursor
	}

	after = nil
	for {
		response, err := getEnvironmentServiceInstances(
			ctx,
			*r.client,
			data.ProjectId.ValueString(),
			data.EnvironmentId.ValueString(),
			after,
		)
		if err != nil {
			return false, fmt.Errorf("unable to list environment service instances: %w", err)
		}

		instances := response.Environment.ServiceInstances
		for _, edge := range instances.Edges {
			if edge.Node.ServiceId == data.ServiceId.ValueString() {
				return true, nil
			}
		}
		if !instances.PageInfo.HasNextPage {
			return false, nil
		}
		if instances.PageInfo.EndCursor == "" {
			return false, fmt.Errorf(
				"Railway service instance connection has a next page but no end cursor",
			)
		}
		after = &instances.PageInfo.EndCursor
	}
}

func expandServiceInstanceEnvironmentPatch(
	data ServiceInstanceResourceModel,
	regions map[string]ServiceInstanceRegionModel,
	current railway.EnvironmentConfig,
) railway.EnvironmentConfig {
	var healthcheckTimeout *int64
	if data.HealthcheckTimeout.ValueInt64() > 0 {
		timeout := data.HealthcheckTimeout.ValueInt64()
		healthcheckTimeout = &timeout
	}

	var cronSchedule *string
	if data.CronSchedule.ValueString() != "" {
		cronSchedule = stringPointer(data.CronSchedule.ValueString())
	}

	var registryCredentials *railway.ServiceRegistryCredentialsConfig
	if data.SourceImagePrivateRegistryUsername.ValueString() != "" {
		registryCredentials = &railway.ServiceRegistryCredentialsConfig{
			Username: data.SourceImagePrivateRegistryUsername.ValueString(),
			Password: data.SourceImagePrivateRegistryPassword.ValueString(),
		}
	}

	multiRegionConfig := make(map[string]*railway.ServiceRegionConfig, len(regions))
	for regionName, region := range regions {
		multiRegionConfig[regionName] = &railway.ServiceRegionConfig{
			NumReplicas: region.NumReplicas.ValueInt64(),
		}
	}
	if service, ok := current.Services[data.ServiceId.ValueString()]; ok && service.Deploy != nil {
		for regionName := range service.Deploy.MultiRegionConfig {
			if _, managed := multiRegionConfig[regionName]; !managed {
				multiRegionConfig[regionName] = nil
			}
		}
	}
	var containers *railway.ServiceContainerLimitsConfig
	vcpus := data.VCpus.ValueFloat64()
	memoryGb := data.MemoryGb.ValueFloat64()
	if vcpus > 0 || memoryGb > 0 {
		containers = &railway.ServiceContainerLimitsConfig{}
		if vcpus > 0 {
			containers.CPU = &vcpus
		}
		if memoryGb > 0 {
			memoryBytes := memoryGb * 1_000_000_000
			containers.MemoryBytes = &memoryBytes
		}
	}

	return railway.EnvironmentConfig{
		Services: map[string]railway.ServiceConfig{
			data.ServiceId.ValueString(): {
				ConfigFile: stringPointer(data.ConfigPath.ValueString()),
				Source: &railway.ServiceSourceConfig{
					Image:         stringPointer(data.SourceImage.ValueString()),
					Repo:          stringPointer(data.SourceRepo.ValueString()),
					Branch:        stringPointer(data.SourceRepoBranch.ValueString()),
					RootDirectory: stringPointer(data.RootDirectory.ValueString()),
				},
				Deploy: &railway.ServiceDeployConfig{
					CronSchedule:       cronSchedule,
					HealthcheckPath:    stringPointer(data.HealthcheckPath.ValueString()),
					HealthcheckTimeout: healthcheckTimeout,
					LimitOverride: &railway.ServiceLimitOverrideConfig{
						Containers: containers,
					},
					MultiRegionConfig:   multiRegionConfig,
					RegistryCredentials: registryCredentials,
					StartCommand:        stringPointer(data.StartCommand.ValueString()),
				},
			},
		},
	}
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalStringValue(value *string) types.String {
	if value == nil {
		return types.StringValue("")
	}
	return types.StringValue(*value)
}

func serviceInstanceRegions(
	ctx context.Context,
	value types.Map,
) (map[string]ServiceInstanceRegionModel, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var regions map[string]ServiceInstanceRegionModel
	diagnostics := value.ElementsAs(ctx, &regions, false)
	if diagnostics.HasError() {
		return nil, fmt.Errorf("unable to decode service instance regions: %s", diagnostics[0].Summary())
	}
	return regions, nil
}

func serviceInstanceSourceBranch(config railway.EnvironmentConfig, serviceId string) string {
	service, ok := config.Services[serviceId]
	if !ok || service.Source == nil || service.Source.Branch == nil {
		return ""
	}
	return *service.Source.Branch
}

func serviceInstanceRegionValue(config railway.EnvironmentConfig, serviceId string) types.Map {
	service, ok := config.Services[serviceId]
	if !ok || service.Deploy == nil {
		return types.MapValueMust(types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes}, map[string]attr.Value{})
	}
	multiRegionConfig := service.Deploy.MultiRegionConfig
	if len(multiRegionConfig) == 0 {
		return types.MapValueMust(types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes}, map[string]attr.Value{})
	}

	regions := make(map[string]attr.Value, len(multiRegionConfig))
	for regionName, regionConfig := range multiRegionConfig {
		if regionConfig == nil {
			continue
		}
		regions[regionName] = types.ObjectValueMust(serviceInstanceRegionAttrTypes, map[string]attr.Value{
			"num_replicas": types.Int64Value(regionConfig.NumReplicas),
		})
	}
	return types.MapValueMust(types.ObjectType{AttrTypes: serviceInstanceRegionAttrTypes}, regions)
}

func flattenServiceInstanceLimits(limitOverride railway.ServiceLimitOverrideConfig) (float64, float64) {
	if limitOverride.Containers == nil {
		return 0, 0
	}

	var vcpus, memoryBytes float64
	if limitOverride.Containers.CPU != nil {
		vcpus = *limitOverride.Containers.CPU
	}
	if limitOverride.Containers.MemoryBytes != nil {
		memoryBytes = *limitOverride.Containers.MemoryBytes
	}
	return vcpus, memoryBytes / 1_000_000_000
}
