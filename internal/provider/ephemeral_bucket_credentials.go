package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var errBucketCredentialsUnavailable = errors.New("Railway returned no credentials for the bucket")

var _ ephemeral.EphemeralResource = &BucketCredentialsEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &BucketCredentialsEphemeralResource{}

func NewBucketCredentialsEphemeralResource() ephemeral.EphemeralResource {
	return &BucketCredentialsEphemeralResource{}
}

type BucketCredentialsEphemeralResource struct {
	client *graphql.Client
}

type BucketCredentialsEphemeralResourceModel struct {
	ProjectId       types.String `tfsdk:"project_id"`
	EnvironmentId   types.String `tfsdk:"environment_id"`
	BucketId        types.String `tfsdk:"bucket_id"`
	AccessKeyId     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	Endpoint        types.String `tfsdk:"endpoint"`
	BucketName      types.String `tfsdk:"bucket_name"`
	Region          types.String `tfsdk:"region"`
	UrlStyle        types.String `tfsdk:"url_style"`
}

type bucketS3Credentials struct {
	AccessKeyId     string
	SecretAccessKey string
	Endpoint        string
	BucketName      string
	Region          string
	UrlStyle        string
}

func (r *BucketCredentialsEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket_credentials"
}

func (r *BucketCredentialsEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	idValidators := []validator.String{
		stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches S3-compatible Railway bucket credentials without persisting the returned values in Terraform plan or state.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project.",
				Required:            true,
				Validators:          idValidators,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment.",
				Required:            true,
				Validators:          idValidators,
			},
			"bucket_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bucket.",
				Required:            true,
				Validators:          idValidators,
			},
			"access_key_id": schema.StringAttribute{
				MarkdownDescription: "S3 access key identifier.",
				Computed:            true,
				Sensitive:           true,
			},
			"secret_access_key": schema.StringAttribute{
				MarkdownDescription: "S3 secret access key.",
				Computed:            true,
				Sensitive:           true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "S3 endpoint.",
				Computed:            true,
			},
			"bucket_name": schema.StringAttribute{
				MarkdownDescription: "Globally unique S3 bucket name.",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "S3 region.",
				Computed:            true,
			},
			"url_style": schema.StringAttribute{
				MarkdownDescription: "S3 URL style.",
				Computed:            true,
			},
		},
	}
}

func (r *BucketCredentialsEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *BucketCredentialsEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data BucketCredentialsEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials, err := waitForBucketS3Credentials(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket credentials, got error: %s", err))
		return
	}
	data.AccessKeyId = types.StringValue(credentials.AccessKeyId)
	data.SecretAccessKey = types.StringValue(credentials.SecretAccessKey)
	data.Endpoint = types.StringValue(credentials.Endpoint)
	data.BucketName = types.StringValue(credentials.BucketName)
	data.Region = types.StringValue(credentials.Region)
	data.UrlStyle = types.StringValue(credentials.UrlStyle)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
func readBucketS3Credentials(
	ctx context.Context,
	client graphql.Client,
	projectId string,
	environmentId string,
	bucketId string,
) (bucketS3Credentials, error) {
	response, err := getBucketS3Credentials(ctx, client, projectId, environmentId, bucketId)
	if err != nil {
		return bucketS3Credentials{}, err
	}

	// Railway returns a list, but its CLI treats exactly one credential set per bucket as an invariant.
	if len(response.BucketS3Credentials) == 0 {
		return bucketS3Credentials{}, errBucketCredentialsUnavailable
	}
	if len(response.BucketS3Credentials) > 1 {
		return bucketS3Credentials{}, fmt.Errorf("Railway returned multiple credential sets for the bucket")
	}

	credentials := response.BucketS3Credentials[0]
	return bucketS3Credentials{
		AccessKeyId:     credentials.AccessKeyId,
		SecretAccessKey: credentials.SecretAccessKey,
		Endpoint:        credentials.Endpoint,
		BucketName:      credentials.BucketName,
		Region:          credentials.Region,
		UrlStyle:        credentials.UrlStyle,
	}, nil
}
func waitForBucketS3Credentials(
	ctx context.Context,
	client graphql.Client,
	projectId string,
	environmentId string,
	bucketId string,
) (bucketS3Credentials, error) {
	const timeout = 30 * time.Second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		credentials, err := readBucketS3Credentials(ctx, client, projectId, environmentId, bucketId)
		if err == nil {
			return credentials, nil
		}
		if !errors.Is(err, errBucketCredentialsUnavailable) {
			return bucketS3Credentials{}, err
		}

		select {
		case <-ctx.Done():
			return bucketS3Credentials{}, ctx.Err()
		case <-timer.C:
			return bucketS3Credentials{}, fmt.Errorf("timed out waiting for Railway bucket credentials")
		case <-ticker.C:
		}
	}
}
