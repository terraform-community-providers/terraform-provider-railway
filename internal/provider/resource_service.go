package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
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

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithImportState = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

type ServiceResource struct {
	client *graphql.Client
}

type ServiceResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	CronSchedule  types.String `tfsdk:"cron_schedule"`
	SourceImage   types.String `tfsdk:"source_image"`
	SourceRepo    types.String `tfsdk:"source_repo"`
	RootDirectory types.String `tfsdk:"root_directory"`
	ConfigPath    types.String `tfsdk:"config_path"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service.\n\n> ⚠️ **NOTE:** All the other settings not specified here are recommended to be specified in the Railway config file.",
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
				MarkdownDescription: "Cron schedule of the service.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(9),
				},
			},
			"source_image": schema.StringAttribute{
				MarkdownDescription: "Source image of the service. Conflicts with `source_repo`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"source_repo": schema.StringAttribute{
				MarkdownDescription: "Source repository of the service. Conflicts with `source_image`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(3),
				},
			},
			"root_directory": schema.StringAttribute{
				MarkdownDescription: "Directory to user for the service.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Railway config file.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

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

	instanceInput := buildServiceInstanceInput(data)

	_, err = updateServiceInstance(ctx, *r.client, data.Id.ValueString(), instanceInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service settings, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created service settings")

	err = getAndBuildServiceInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

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

	instanceInput := buildServiceInstanceInput(data)

	_, err = updateServiceInstance(ctx, *r.client, data.Id.ValueString(), instanceInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service settings, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated service settings")

	err = getAndBuildServiceInstance(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service settings, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceResourceModel

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
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildServiceInstanceInput(data *ServiceResourceModel) ServiceInstanceUpdateInput {
	var instanceInput ServiceInstanceUpdateInput

	// Update the source attributes
	if !data.SourceImage.IsNull() {
		instanceInput.Source = &ServiceSourceInput{
			Image: data.SourceImage.ValueStringPointer(),
		}
	} else if !data.SourceRepo.IsNull() {
		instanceInput.Source = &ServiceSourceInput{
			Repo: data.SourceRepo.ValueStringPointer(),
		}
	} else {
		instanceInput.Source = &ServiceSourceInput{}
	}

	if !data.CronSchedule.IsNull() {
		instanceInput.CronSchedule = data.CronSchedule.ValueStringPointer()
	}

	if !data.RootDirectory.IsNull() {
		instanceInput.RootDirectory = data.RootDirectory.ValueString()
	}

	if !data.ConfigPath.IsNull() {
		instanceInput.RailwayConfigFile = data.ConfigPath.ValueString()
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

	if response.ServiceInstance.CronSchedule != nil {
		data.CronSchedule = types.StringValue(*response.ServiceInstance.CronSchedule)
	}

	if response.ServiceInstance.RootDirectory != nil && len(*response.ServiceInstance.RootDirectory) != 0 {
		data.RootDirectory = types.StringValue(*response.ServiceInstance.RootDirectory)
	}

	if response.ServiceInstance.RailwayConfigFile != nil && len(*response.ServiceInstance.RailwayConfigFile) != 0 {
		data.ConfigPath = types.StringValue(*response.ServiceInstance.RailwayConfigFile)
	}

	if response.ServiceInstance.Source != nil {
		if response.ServiceInstance.Source.Image != nil {
			data.SourceImage = types.StringValue(*response.ServiceInstance.Source.Image)
		}

		if response.ServiceInstance.Source.Repo != nil {
			data.SourceRepo = types.StringValue(*response.ServiceInstance.Source.Repo)
		}
	}

	return nil
}
