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
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

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
		MarkdownDescription: "Manages a project-level, private S3-compatible Railway storage bucket and deploys it to one environment. Changing the project, environment, or region replaces the resource. Destroying the resource undeploys the bucket from the environment; Railway does not expose an API for deleting the project-level bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bucket.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the bucket.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project that owns the bucket. Changing this value replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment where the bucket is deployed. Changing this value replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Railway bucket region: `ams`, `iad`, `sjc`, or `sin`. Changing this value replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						replaceWhenBucketRegionChanges,
						"Changing the region of a deployed bucket requires replacement.",
						"Changing the region of a deployed bucket requires replacement.",
					),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ams", "iad", "sjc", "sin"),
				},
			},
		},
	}
}

func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	var data BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := createManagedBucket(ctx, *r.client, BucketCreateInput{
		Name:      data.Name.ValueString(),
		ProjectId: data.ProjectId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket, got error: %s", err))
		return
	}

	data.Id = types.StringValue(response.BucketCreate.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(ctx, data, railway.CreateBucketPatch(
		data.Id.ValueString(),
		data.Region.ValueString(),
	)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deploy bucket to environment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a bucket")
}

func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name, found, err := getManagedBucket(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.Id.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	data.Name = types.StringValue(name)

	bucketConfig, deployed, err := getManagedBucketDeployment(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.Id.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket environment configuration, got error: %s", err))
		return
	}
	if !deployed {
		data.Region = types.StringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if bucketConfig.Region != "" {
		data.Region = types.StringValue(bucketConfig.Region)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var state BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := updateManagedBucket(ctx, *r.client, data.Id.ValueString(), BucketUpdateInput{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket, got error: %s", err))
		return
	}

	if state.Region.IsNull() {
		if err := r.patchEnvironment(ctx, data, railway.CreateBucketPatch(
			data.Id.ValueString(),
			data.Region.ValueString(),
		)); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deploy bucket to environment, got error: %s", err))
			return
		}
	}

	data.Name = types.StringValue(response.BucketUpdate.Name)
	tflog.Trace(ctx, "updated a bucket")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(ctx, data, railway.DeleteBucketPatch(data.Id.ValueString())); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a bucket")
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:environment_id:bucket_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *BucketResource) patchEnvironment(ctx context.Context, data BucketResourceModel, patch railway.EnvironmentConfig) error {
	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		"Manage Terraform bucket",
	)
}

func replaceWhenBucketRegionChanges(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.StateValue.IsNull()
}

func getManagedBucketDeployment(
	ctx context.Context,
	client graphql.Client,
	environmentID string,
	bucketID string,
) (railway.BucketConfig, bool, error) {
	response, err := getEnvironmentConfig(ctx, client, environmentID)
	if err != nil {
		return railway.BucketConfig{}, false, err
	}

	bucket, found := response.Environment.Config.Bucket(bucketID)
	return bucket, found && !bucket.IsDeleted, nil
}

// Railway does not expose a bucket-by-ID query, so search the paginated project bucket connection.
func getManagedBucket(ctx context.Context, client graphql.Client, projectID string, bucketID string) (string, bool, error) {
	var after *string
	for {
		response, err := getManagedBuckets(ctx, client, projectID, after)
		if err != nil {
			return "", false, err
		}

		for _, edge := range response.Project.Buckets.Edges {
			if edge.Node.Id == bucketID {
				return edge.Node.Name, true, nil
			}
		}

		pageInfo := response.Project.Buckets.PageInfo
		if !pageInfo.HasNextPage {
			return "", false, nil
		}
		if pageInfo.EndCursor == nil || *pageInfo.EndCursor == "" {
			return "", false, fmt.Errorf("Railway bucket connection has a next page but no end cursor")
		}
		after = pageInfo.EndCursor
	}
}
