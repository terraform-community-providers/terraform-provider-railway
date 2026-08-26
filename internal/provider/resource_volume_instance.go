package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

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
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

const (
	volumeInstancePollInterval = time.Second
	volumeInstancePollTimeout  = 2 * time.Minute
)

var _ resource.Resource = &VolumeInstanceResource{}
var _ resource.ResourceWithImportState = &VolumeInstanceResource{}
var _ resource.ResourceWithModifyPlan = &VolumeInstanceResource{}

func NewVolumeInstanceResource() resource.Resource {
	return &VolumeInstanceResource{}
}

type VolumeInstanceResource struct {
	client *graphql.Client
}

type VolumeInstanceResourceModel struct {
	Id            types.String `tfsdk:"id"`
	VolumeId      types.String `tfsdk:"volume_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	MountPath     types.String `tfsdk:"mount_path"`
	SizeMB        types.Int64  `tfsdk:"size_mb"`
}

func (r *VolumeInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_instance"
}

func (r *VolumeInstanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one environment deployment of a project-level Railway volume and its optional service mount. Deleting or replacing this resource destroys the environment deployment and its data. Data is not transferred when `volume_id` or `environment_id` changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment-specific volume instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"volume_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project-level volume. Changing this value replaces the resource, destroying this environment deployment and its data without transferring it.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment containing the volume instance. Changing this value replaces the resource, destroying the old environment deployment and its data without transferring it.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service to mount the volume on. Omit to keep the environment volume instance detached.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
					stringvalidator.AlsoRequires(path.MatchRoot("mount_path")),
				},
			},
			"mount_path": schema.StringAttribute{
				MarkdownDescription: "Absolute path where the volume is mounted in the service container. Required when `service_id` is set and must be omitted otherwise.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^/`), "must be an absolute path"),
					stringvalidator.AlsoRequires(path.MatchRoot("service_id")),
				},
			},
			"size_mb": schema.Int64Attribute{
				MarkdownDescription: "Provisioned size of the volume instance in megabytes. Updates may increase this value but cannot decrease it.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
		},
	}
}

func (r *VolumeInstanceResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	var plan VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() ||
		state.VolumeId.IsNull() || state.VolumeId.IsUnknown() ||
		plan.VolumeId.IsNull() || plan.VolumeId.IsUnknown() ||
		state.EnvironmentId.IsNull() || state.EnvironmentId.IsUnknown() ||
		plan.EnvironmentId.IsNull() || plan.EnvironmentId.IsUnknown() {
		return
	}
	if plan.VolumeId.ValueString() != state.VolumeId.ValueString() ||
		plan.EnvironmentId.ValueString() != state.EnvironmentId.ValueString() {
		return
	}
	if state.SizeMB.IsNull() || state.SizeMB.IsUnknown() ||
		plan.SizeMB.IsNull() || plan.SizeMB.IsUnknown() {
		return
	}
	if plan.SizeMB.ValueInt64() < state.SizeMB.ValueInt64() {
		resp.Diagnostics.AddAttributeError(
			path.Root("size_mb"),
			"Volume size cannot be decreased",
			fmt.Sprintf(
				"`size_mb` cannot be reduced from %d to %d because Railway volume data cannot be safely shrunk. Choose a value of at least %d.",
				state.SizeMB.ValueInt64(),
				plan.SizeMB.ValueInt64(),
				state.SizeMB.ValueInt64(),
			),
		)
	}
}

func (r *VolumeInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(
		ctx,
		data,
		railway.CreateVolumePatch(data.VolumeId.ValueString(), data.SizeMB.ValueInt64()),
		"Create volume instance with Terraform",
	); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume instance, got error: %s", err))
		return
	}

	instance, found, err := waitForManagedVolumeInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created volume instance, got error: %s", err))
		return
	}
	if !found {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Timed out waiting for volume %s to be created in environment %s",
				data.VolumeId.ValueString(),
				data.EnvironmentId.ValueString(),
			),
		)
		return
	}

	data.Id = types.StringValue(instance.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ServiceId.IsNull() {
		if err := updateManagedVolumeInstanceAttachment(ctx, *r.client, data); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to attach created volume instance, got error: %s", err))
			return
		}
	}
	if err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created volume instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a volume instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.readFound(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume instance, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var state VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.SizeMB.ValueInt64() > state.SizeMB.ValueInt64() {
		if err := r.patchEnvironment(
			ctx,
			data,
			railway.ResizeVolumePatch(data.VolumeId.ValueString(), data.SizeMB.ValueInt64()),
			"Resize volume instance with Terraform",
		); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resize volume instance, got error: %s", err))
			return
		}
	}

	if attachmentChanged(state, data) {
		if !state.ServiceId.IsNull() && (data.ServiceId.IsNull() || state.ServiceId.ValueString() != data.ServiceId.ValueString()) {
			if err := detachManagedVolumeInstance(ctx, *r.client, state); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to detach volume instance before updating its mount, got error: %s", err))
				return
			}
		}
		if !data.ServiceId.IsNull() {
			if err := updateManagedVolumeInstanceAttachment(ctx, *r.client, data); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume instance mount, got error: %s", err))
				return
			}
		}
	}
	if err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read updated volume instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a volume instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VolumeInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, found, err := getManagedVolumeInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to verify volume instance before deleting it, got error: %s", err))
		return
	}
	if !found {
		return
	}
	if instance.State == nil || *instance.State != VolumeStateDeleting {
		if instance.ServiceId != nil {
			detachData := data
			applyManagedVolumeInstance(&detachData, instance)
			if err := detachManagedVolumeInstance(ctx, *r.client, detachData); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to detach volume instance before deleting it, got error: %s", err))
				return
			}
		}

		response, err := deleteManagedVolumeInstance(
			ctx,
			*r.client,
			data.VolumeId.ValueString(),
			data.EnvironmentId.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume instance, got error: %s", err))
			return
		}
		if !response.VolumeInstanceUpdate {
			resp.Diagnostics.AddError("Client Error", "Railway did not delete the volume instance")
			return
		}
	}

	_, deleted, err := waitForManagedVolumeInstanceState(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
		func(_ ManagedVolumeInstance, found bool) bool {
			return !found
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to verify deleted volume instance, got error: %s", err))
		return
	}
	if !deleted {
		resp.Diagnostics.AddError("Client Error", "Timed out waiting for Railway to delete the volume instance")
		return
	}

	tflog.Trace(ctx, "deleted a volume instance")
}

func (r *VolumeInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	environmentId, volumeId, ok := parseVolumeInstanceImportId(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:volume_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), volumeId)...)
}

func parseVolumeInstanceImportId(id string) (string, string, bool) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (r *VolumeInstanceResource) patchEnvironment(
	ctx context.Context,
	data VolumeInstanceResourceModel,
	patch railway.EnvironmentConfig,
	commitMessage string,
) error {
	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		commitMessage,
	)
}

func (r *VolumeInstanceResource) read(ctx context.Context, data *VolumeInstanceResourceModel) error {
	found, err := r.readFound(ctx, data)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf(
			"volume %s is not available in environment %s",
			data.VolumeId.ValueString(),
			data.EnvironmentId.ValueString(),
		)
	}
	return nil
}

func (r *VolumeInstanceResource) readFound(ctx context.Context, data *VolumeInstanceResourceModel) (bool, error) {
	instance, found, err := getManagedVolumeInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
	)
	if err != nil || !found {
		return found, err
	}

	applyManagedVolumeInstance(data, instance)
	return true, nil
}

func applyManagedVolumeInstance(data *VolumeInstanceResourceModel, instance ManagedVolumeInstance) {
	data.Id = types.StringValue(instance.Id)
	data.VolumeId = types.StringValue(instance.VolumeId)
	data.EnvironmentId = types.StringValue(instance.EnvironmentId)
	data.SizeMB = types.Int64Value(int64(instance.SizeMB))
	if instance.ServiceId == nil {
		data.ServiceId = types.StringNull()
		data.MountPath = types.StringNull()
	} else {
		data.ServiceId = types.StringValue(*instance.ServiceId)
		data.MountPath = types.StringValue(instance.MountPath)
	}
}

func attachmentChanged(state VolumeInstanceResourceModel, plan VolumeInstanceResourceModel) bool {
	if state.ServiceId.IsNull() != plan.ServiceId.IsNull() || state.MountPath.IsNull() != plan.MountPath.IsNull() {
		return true
	}
	if !state.ServiceId.IsNull() && state.ServiceId.ValueString() != plan.ServiceId.ValueString() {
		return true
	}
	return !state.MountPath.IsNull() && state.MountPath.ValueString() != plan.MountPath.ValueString()
}

func updateManagedVolumeInstanceAttachment(ctx context.Context, client graphql.Client, data VolumeInstanceResourceModel) error {
	serviceId := data.ServiceId.ValueString()
	response, err := updateManagedVolumeInstance(
		ctx,
		client,
		data.VolumeId.ValueString(),
		data.EnvironmentId.ValueString(),
		&serviceId,
		data.MountPath.ValueString(),
	)
	if err != nil {
		return err
	}
	if !response.VolumeInstanceUpdate {
		return fmt.Errorf("Railway did not update the volume instance")
	}
	_, converged, err := waitForManagedVolumeInstanceState(
		ctx,
		client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
		func(instance ManagedVolumeInstance, found bool) bool {
			return found &&
				instance.ServiceId != nil &&
				*instance.ServiceId == serviceId &&
				instance.MountPath == data.MountPath.ValueString()
		},
	)
	if err != nil {
		return err
	}
	if !converged {
		return fmt.Errorf("timed out waiting for Railway to apply the volume instance mount")
	}
	return nil
}

func detachManagedVolumeInstance(ctx context.Context, client graphql.Client, data VolumeInstanceResourceModel) error {
	mountPath := detachedVolumeMountPath
	if !data.MountPath.IsNull() {
		mountPath = data.MountPath.ValueString()
	}
	response, err := updateManagedVolumeInstance(
		ctx,
		client,
		data.VolumeId.ValueString(),
		data.EnvironmentId.ValueString(),
		nil,
		mountPath,
	)
	if err != nil {
		return err
	}
	if !response.VolumeInstanceUpdate {
		return fmt.Errorf("Railway did not detach the volume instance")
	}
	_, converged, err := waitForManagedVolumeInstanceState(
		ctx,
		client,
		data.EnvironmentId.ValueString(),
		data.VolumeId.ValueString(),
		func(instance ManagedVolumeInstance, found bool) bool {
			return found && instance.ServiceId == nil
		},
	)
	if err != nil {
		return err
	}
	if !converged {
		return fmt.Errorf("timed out waiting for Railway to detach the volume instance")
	}
	return nil
}

func getManagedVolumeInstance(
	ctx context.Context,
	client graphql.Client,
	environmentId string,
	volumeId string,
) (ManagedVolumeInstance, bool, error) {
	var after *string
	for {
		response, err := getManagedVolumeInstances(ctx, client, environmentId, after)
		if err != nil {
			return ManagedVolumeInstance{}, false, err
		}

		for _, edge := range response.Environment.VolumeInstances.Edges {
			if edge.Node.VolumeId == volumeId {
				if edge.Node.State != nil && *edge.Node.State == VolumeStateDeleted {
					return ManagedVolumeInstance{}, false, nil
				}
				return edge.Node.ManagedVolumeInstance, true, nil
			}
		}

		if !response.Environment.VolumeInstances.PageInfo.HasNextPage {
			return ManagedVolumeInstance{}, false, nil
		}
		endCursor := response.Environment.VolumeInstances.PageInfo.EndCursor
		if endCursor == nil || *endCursor == "" {
			return ManagedVolumeInstance{}, false, fmt.Errorf(
				"Railway volume instance connection has a next page but no end cursor",
			)
		}
		after = endCursor
	}
}

func waitForManagedVolumeInstance(
	ctx context.Context,
	client graphql.Client,
	environmentId string,
	volumeId string,
) (ManagedVolumeInstance, bool, error) {
	return waitForManagedVolumeInstanceState(
		ctx,
		client,
		environmentId,
		volumeId,
		func(_ ManagedVolumeInstance, found bool) bool {
			return found
		},
	)
}

func waitForManagedVolumeInstanceState(
	ctx context.Context,
	client graphql.Client,
	environmentId string,
	volumeId string,
	matches func(ManagedVolumeInstance, bool) bool,
) (ManagedVolumeInstance, bool, error) {
	timeout := time.NewTimer(volumeInstancePollTimeout)
	defer timeout.Stop()
	ticker := time.NewTicker(volumeInstancePollInterval)
	defer ticker.Stop()

	for {
		instance, found, err := getManagedVolumeInstance(ctx, client, environmentId, volumeId)
		if err != nil {
			return ManagedVolumeInstance{}, false, err
		}
		if matches(instance, found) {
			return instance, true, nil
		}

		select {
		case <-ctx.Done():
			return ManagedVolumeInstance{}, false, ctx.Err()
		case <-timeout.C:
			return ManagedVolumeInstance{}, false, nil
		case <-ticker.C:
		}
	}
}
