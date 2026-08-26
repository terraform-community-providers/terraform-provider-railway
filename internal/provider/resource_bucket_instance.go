package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

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

var _ resource.Resource = &BucketInstanceResource{}
var _ resource.ResourceWithImportState = &BucketInstanceResource{}
var _ resource.ResourceWithModifyPlan = &BucketInstanceResource{}

func NewBucketInstanceResource() resource.Resource {
	return &BucketInstanceResource{}
}

type BucketInstanceResource struct {
	client *graphql.Client
}

type BucketInstanceResourceModel struct {
	Id            types.String `tfsdk:"id"`
	BucketId      types.String `tfsdk:"bucket_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Region        types.String `tfsdk:"region"`
}

func (r *BucketInstanceResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_bucket_instance"
}

func (r *BucketInstanceResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	idValidators := []validator.String{
		stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deploys a project-level Railway bucket to one environment. Destroying this resource undeploys only this environment instance and preserves the bucket and its other instances. Changing the bucket or environment replaces the instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier of the bucket instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project-level bucket.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: idValidators,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment where the bucket is deployed.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: idValidators,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Railway bucket region: `ams`, `iad`, `sjc`, or `sin`. Railway does not support changing the region after creation; create a different bucket and migrate its data instead.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ams", "iad", "sjc", "sin"),
				},
			},
		},
	}
}

func (r *BucketInstanceResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state BucketInstanceResourceModel
	var plan BucketInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() ||
		state.Region.IsNull() || state.Region.IsUnknown() ||
		plan.Region.IsNull() || plan.Region.IsUnknown() ||
		state.Region.Equal(plan.Region) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("region"),
		"Bucket Instance Region Cannot Be Changed",
		"Railway does not support changing a bucket region after creation. Create a different bucket and migrate its data instead.",
	)
}

func (r *BucketInstanceResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
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

func (r *BucketInstanceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data BucketInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(ctx, data, railway.CreateBucketPatch(
		data.BucketId.ValueString(),
		data.Region.ValueString(),
	)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deploy bucket to environment, got error: %s", err))
		return
	}
	if err := waitForManagedBucketInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
		data.Region.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for bucket instance, got error: %s", err))
		return
	}

	data.Id = types.StringValue(bucketInstanceResourceId(data))
	tflog.Trace(ctx, "created a bucket instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketInstanceResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data BucketInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, deployed, err := getManagedBucketDeployment(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket instance, got error: %s", err))
		return
	}
	if !deployed {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(bucketInstanceResourceId(data))
	data.Region = types.StringValue(bucket.Region)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketInstanceResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data BucketInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(ctx, data, railway.CreateBucketPatch(
		data.BucketId.ValueString(),
		data.Region.ValueString(),
	)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket instance, got error: %s", err))
		return
	}
	if err := waitForManagedBucketInstance(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
		data.Region.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for bucket instance, got error: %s", err))
		return
	}

	data.Id = types.StringValue(bucketInstanceResourceId(data))
	tflog.Trace(ctx, "updated a bucket instance")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketInstanceResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data BucketInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.patchEnvironment(ctx, data, railway.DeleteBucketPatch(data.BucketId.ValueString())); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket instance, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a bucket instance")
}

func (r *BucketInstanceResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: environment_id:bucket_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_id"), parts[1])...)
}

func (r *BucketInstanceResource) patchEnvironment(
	ctx context.Context,
	data BucketInstanceResourceModel,
	patch railway.EnvironmentConfig,
) error {
	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		"Manage Terraform bucket instance",
	)
}

func bucketInstanceResourceId(data BucketInstanceResourceModel) string {
	return data.EnvironmentId.ValueString() + ":" + data.BucketId.ValueString()
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

func waitForManagedBucketInstance(
	ctx context.Context,
	client graphql.Client,
	environmentID string,
	bucketID string,
	region string,
) error {
	return waitForManagedBucketInstanceState(ctx, func() (bool, error) {
		bucket, deployed, err := getManagedBucketDeployment(ctx, client, environmentID, bucketID)
		return deployed && bucket.Region == region, err
	})
}

func waitForManagedBucketInstanceState(
	ctx context.Context,
	converged func() (bool, error),
) error {
	const timeout = 30 * time.Second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		done, err := converged()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timed out waiting for Railway bucket instance")
		case <-ticker.C:
		}
	}
}
