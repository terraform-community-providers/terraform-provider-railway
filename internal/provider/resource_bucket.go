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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BucketResource{}
var _ resource.ResourceWithImportState = &BucketResource{}

func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

type BucketResource struct {
	client *graphql.Client
}

type BucketResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Region        types.String `tfsdk:"region"`
}

func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway S3-compatible storage bucket.\n\n~> Railway's public API cannot delete a bucket outright, only remove it from its environment. Destroying this resource removes the bucket from the environment and renames the project-level leftover to `<name>-deleted-<id prefix>` so the name can be reused. The leftover still counts towards the project's bucket quota until it is deleted in the Railway dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bucket.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the bucket.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the bucket belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the bucket is provisioned in.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region the bucket is provisioned in. One of `sjc`, `iad`, `ams`, `sin`. **Default** `sjc`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("sjc"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("sjc", "iad", "ams", "sin"),
				},
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	response, err := createBucket(ctx, *r.client, BucketCreateInput{
		ProjectId: data.ProjectId.ValueString(),
		Name:      &name,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a bucket")

	bucket := response.BucketCreate.Bucket

	data.Id = types.StringValue(bucket.Id)
	data.Name = types.StringValue(bucket.Name)
	data.ProjectId = types.StringValue(bucket.ProjectId)

	// The bucket only exists at the project level until it is provisioned into
	// an environment through a config patch. Save state first so a failed step
	// below leaves a tainted resource instead of an untracked bucket.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	// bucketCreate silently assigns a random name when the requested one is
	// taken (typically by a bucket that was removed from every environment but
	// still exists at the project level). Renaming surfaces a clear error.
	if bucket.Name != name {
		renamed, err := updateBucket(ctx, *r.client, bucket.Id, BucketUpdateInput{Name: name})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Bucket was created as %q instead of %q and could not be renamed, got error: %s", bucket.Name, name, err))
			return
		}

		data.Name = types.StringValue(renamed.BucketUpdate.Bucket.Name)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}

	patch := map[string]interface{}{
		"buckets": map[string]interface{}{
			bucket.Id: map[string]interface{}{
				"region":    data.Region.ValueString(),
				"isCreated": true,
			},
		},
	}

	message := fmt.Sprintf("Create bucket %s", bucket.Name)

	_, err = commitEnvironmentPatch(ctx, *r.client, data.EnvironmentId.ValueString(), patch, message)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to provision bucket in environment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "provisioned a bucket in environment")
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getBucket(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}

	id := data.Id.ValueString()

	instance, ok := bucketInstanceFromConfig(response.Environment.Config, id)

	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	for _, edge := range response.Project.Buckets.Edges {
		if edge.Node.Id == id {
			data.Name = types.StringValue(edge.Node.Name)
			data.ProjectId = types.StringValue(edge.Node.ProjectId)
		}
	}

	data.EnvironmentId = types.StringValue(response.Environment.Id)

	if region, ok := instance["region"].(string); ok && region != "" {
		data.Region = types.StringValue(region)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *BucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := updateBucket(ctx, *r.client, data.Id.ValueString(), BucketUpdateInput{
		Name: data.Name.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a bucket")

	bucket := response.BucketUpdate.Bucket

	data.Id = types.StringValue(bucket.Id)
	data.Name = types.StringValue(bucket.Name)
	data.ProjectId = types.StringValue(bucket.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *BucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	patch := map[string]interface{}{
		"buckets": map[string]interface{}{
			data.Id.ValueString(): map[string]interface{}{
				"isDeleted": true,
			},
		},
	}

	message := fmt.Sprintf("Delete bucket %s", data.Name.ValueString())

	_, err := commitEnvironmentPatch(ctx, *r.client, data.EnvironmentId.ValueString(), patch, message)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a bucket")

	// The public API has no way to delete the project-level bucket record; the
	// patch above only removes it from the environment. Rename the leftover so
	// its name can be reused (e.g. on replace). It still counts towards the
	// project's bucket quota until removed in the Railway dashboard.
	_, err = updateBucket(ctx, *r.client, data.Id.ValueString(), BucketUpdateInput{
		Name: deletedBucketName(data.Name.ValueString(), data.Id.ValueString()),
	})

	if err != nil {
		resp.Diagnostics.AddWarning("Bucket name not released", fmt.Sprintf("Bucket was removed from the environment but could not be renamed, so its name stays reserved in the project, got error: %s", err))
	}
}

// deletedBucketName is the name given to a bucket's project-level leftover
// after it has been removed from its environment.
func deletedBucketName(name string, id string) string {
	return fmt.Sprintf("%s-deleted-%s", name, id[:8])
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:bucket_id. Got: %q", req.ID),
		)

		return
	}

	response, err := getEnvironment(ctx, *r.client, parts[0])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), response.Environment.ProjectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// bucketInstanceFromConfig returns the bucket's entry in an environment config,
// or false when the bucket is absent or marked as deleted.
func bucketInstanceFromConfig(config map[string]interface{}, id string) (map[string]interface{}, bool) {
	buckets, _ := config["buckets"].(map[string]interface{})
	instance, ok := buckets[id].(map[string]interface{})

	if !ok {
		return nil, false
	}

	if deleted, _ := instance["isDeleted"].(bool); deleted {
		return nil, false
	}

	return instance, true
}
