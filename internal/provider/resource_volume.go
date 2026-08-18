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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const detachedVolumeMountPath = "/data"

var _ resource.Resource = &VolumeResource{}
var _ resource.ResourceWithImportState = &VolumeResource{}

func NewVolumeResource() resource.Resource {
	return &VolumeResource{}
}

type VolumeResource struct {
	client *graphql.Client
}

type VolumeResourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ProjectId types.String `tfsdk:"project_id"`
}

func (r *VolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *VolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages only the project-level identity of a Railway persistent volume. Creation does not deploy the volume to any environment. Use `railway_volume_instance` to create each environment deployment and optionally mount it to a service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the volume.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the volume.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project that owns the volume. Changing this value replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (r *VolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desiredName := data.Name.ValueString()

	response, err := createManagedVolume(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		detachedVolumeMountPath,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create volume, got error: %s", err))
		return
	}

	data.Id = types.StringValue(response.VolumeCreate.Id)
	data.Name = types.StringValue(response.VolumeCreate.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResponse, err := updateManagedVolume(ctx, *r.client, data.Id.ValueString(), VolumeUpdateInput{
		Name: desiredName,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to name created volume, got error: %s", err))
		return
	}

	data.Name = types.StringValue(updateResponse.VolumeUpdate.Name)
	tflog.Trace(ctx, "created a volume")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, found, err := getManagedVolume(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.Id.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read volume, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(volume.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := updateManagedVolume(ctx, *r.client, data.Id.ValueString(), VolumeUpdateInput{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update volume, got error: %s", err))
		return
	}

	data.Name = types.StringValue(response.VolumeUpdate.Name)
	tflog.Trace(ctx, "updated a volume")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, found, err := getManagedVolume(ctx, *r.client, data.ProjectId.ValueString(), data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to verify volume before deleting it, got error: %s", err))
		return
	}
	if !found {
		return
	}

	response, err := deleteManagedVolume(ctx, *r.client, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete volume, got error: %s", err))
		return
	}
	if !response.VolumeDelete {
		resp.Diagnostics.AddError("Client Error", "Railway did not delete the volume")
		return
	}

	tflog.Trace(ctx, "deleted a volume")
}

func (r *VolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectId, volumeId, ok := parseVolumeImportId(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:volume_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), volumeId)...)
}

func parseVolumeImportId(id string) (string, string, bool) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func getManagedVolume(ctx context.Context, client graphql.Client, projectId string, volumeId string) (ManagedVolume, bool, error) {
	var after *string
	for {
		response, err := getManagedVolumes(ctx, client, projectId, after)
		if err != nil {
			return ManagedVolume{}, false, err
		}

		for _, edge := range response.Project.Volumes.Edges {
			if edge.Node.Id == volumeId {
				return edge.Node.ManagedVolume, true, nil
			}
		}

		if !response.Project.Volumes.PageInfo.HasNextPage {
			return ManagedVolume{}, false, nil
		}
		endCursor := response.Project.Volumes.PageInfo.EndCursor
		if endCursor == nil || *endCursor == "" {
			return ManagedVolume{}, false, fmt.Errorf(
				"Railway volume connection has a next page but no end cursor",
			)
		}
		after = endCursor
	}
}
