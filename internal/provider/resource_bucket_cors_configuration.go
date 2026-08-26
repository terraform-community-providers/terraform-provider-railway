package provider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BucketCorsConfigurationResource{}
var _ resource.ResourceWithImportState = &BucketCorsConfigurationResource{}

func NewBucketCorsConfigurationResource() resource.Resource {
	return &BucketCorsConfigurationResource{}
}

type BucketCorsConfigurationResource struct {
	client *graphql.Client
}

type managedBucketState struct {
	exists   bool
	deployed bool
}

type BucketCorsConfigurationResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	BucketId      types.String `tfsdk:"bucket_id"`
	CorsRules     types.List   `tfsdk:"cors_rules"`
}

type BucketCorsRuleModel struct {
	Id             types.String `tfsdk:"id"`
	AllowedHeaders types.Set    `tfsdk:"allowed_headers"`
	AllowedMethods types.Set    `tfsdk:"allowed_methods"`
	AllowedOrigins types.Set    `tfsdk:"allowed_origins"`
	ExposeHeaders  types.Set    `tfsdk:"expose_headers"`
	MaxAgeSeconds  types.Int64  `tfsdk:"max_age_seconds"`
}

var bucketCorsRuleAttrTypes = map[string]attr.Type{
	"id":              types.StringType,
	"allowed_headers": types.SetType{ElemType: types.StringType},
	"allowed_methods": types.SetType{ElemType: types.StringType},
	"allowed_origins": types.SetType{ElemType: types.StringType},
	"expose_headers":  types.SetType{ElemType: types.StringType},
	"max_age_seconds": types.Int64Type,
}

var emptyStringSet = types.SetValueMust(types.StringType, []attr.Value{})

func (r *BucketCorsConfigurationResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_bucket_cors_configuration"
}

func (r *BucketCorsConfigurationResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	idValidators := []validator.String{
		stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the ordered CORS configuration of a Railway bucket through its S3-compatible API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bucket CORS configuration.",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project that owns the bucket.",
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
			"bucket_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the bucket.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: idValidators,
			},
			"cors_rules": schema.ListNestedAttribute{
				MarkdownDescription: "Ordered CORS rules. S3 applies the first rule matching an incoming request.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 100),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Optional identifier for the CORS rule.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(""),
							Validators: []validator.String{
								stringvalidator.UTF8LengthAtMost(255),
							},
						},
						"allowed_headers": schema.SetAttribute{
							MarkdownDescription: "Request headers allowed in CORS preflight requests.",
							Optional:            true,
							Computed:            true,
							Default:             setdefault.StaticValue(emptyStringSet),
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.All(
										stringvalidator.UTF8LengthAtLeast(1),
										atMostOneWildcardValidator{},
									),
								),
							},
						},
						"allowed_methods": schema.SetAttribute{
							MarkdownDescription: "HTTP methods allowed for matching origins.",
							Required:            true,
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.ValueStringsAre(
									stringvalidator.OneOf("DELETE", "GET", "HEAD", "POST", "PUT"),
								),
							},
						},
						"allowed_origins": schema.SetAttribute{
							MarkdownDescription: "Origins allowed to access the bucket.",
							Required:            true,
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
								setvalidator.ValueStringsAre(
									corsOriginValidator{},
								),
							},
						},
						"expose_headers": schema.SetAttribute{
							MarkdownDescription: "Response headers exposed to browser clients.",
							Optional:            true,
							Computed:            true,
							Default:             setdefault.StaticValue(emptyStringSet),
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.UTF8LengthAtLeast(1),
								),
							},
						},
						"max_age_seconds": schema.Int64Attribute{
							MarkdownDescription: "Duration in seconds that browsers may cache a preflight response.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							Validators: []validator.Int64{
								int64validator.Between(0, math.MaxInt32),
							},
						},
					},
				},
			},
		},
	}
}

func (r *BucketCorsConfigurationResource) Configure(
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

func (r *BucketCorsConfigurationResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data BucketCorsConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.put(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create bucket CORS configuration, got error: %s", err))
		return
	}

	data.Id = types.StringValue(bucketCorsConfigurationResourceId(data))
	tflog.Trace(ctx, "created a bucket CORS configuration")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCorsConfigurationResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data BucketCorsConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketState, err := r.bucketState(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}
	if !bucketState.exists {
		resp.State.RemoveResource(ctx)
		return
	}
	if !bucketState.deployed {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}
	if err := r.read(ctx, &data); isMissingBucketCors(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket CORS configuration, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCorsConfigurationResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data BucketCorsConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.put(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update bucket CORS configuration, got error: %s", err))
		return
	}
	data.Id = types.StringValue(bucketCorsConfigurationResourceId(data))
	tflog.Trace(ctx, "updated a bucket CORS configuration")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BucketCorsConfigurationResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data BucketCorsConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketState, err := r.bucketState(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read bucket, got error: %s", err))
		return
	}
	if !bucketState.exists {
		return
	}
	if !bucketState.deployed {
		return
	}
	client, bucketName, err := r.s3Client(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to access bucket CORS configuration, got error: %s", err))
		return
	}
	_, err = client.DeleteBucketCors(ctx, &s3.DeleteBucketCorsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil && !isMissingBucketCors(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete bucket CORS configuration, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted a bucket CORS configuration")
}

func (r *BucketCorsConfigurationResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:environment_id:bucket_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_id"), parts[2])...)
}

func (r *BucketCorsConfigurationResource) put(
	ctx context.Context,
	data BucketCorsConfigurationResourceModel,
) error {
	_, deployed, err := getManagedBucketDeployment(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
	if err != nil {
		return err
	}
	if !deployed {
		return fmt.Errorf("bucket is not deployed to the configured environment")
	}

	client, bucketName, err := r.s3Client(ctx, data)
	if err != nil {
		return err
	}

	rules, diagnostics := expandBucketCorsRules(ctx, data.CorsRules)
	if diagnostics.HasError() {
		return bucketCorsDiagnosticError(diagnostics)
	}
	_, err = client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucketName),
		CORSConfiguration: &s3types.CORSConfiguration{
			CORSRules: rules,
		},
	})
	return err
}

func (r *BucketCorsConfigurationResource) bucketState(
	ctx context.Context,
	data BucketCorsConfigurationResourceModel,
) (managedBucketState, error) {
	_, found, err := getManagedBucket(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.BucketId.ValueString(),
	)
	if err != nil || !found {
		return managedBucketState{exists: found}, err
	}

	_, deployed, err := getManagedBucketDeployment(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
	return managedBucketState{
		exists:   true,
		deployed: deployed,
	}, err
}

func (r *BucketCorsConfigurationResource) read(
	ctx context.Context,
	data *BucketCorsConfigurationResourceModel,
) error {
	client, bucketName, err := r.s3Client(ctx, *data)
	if err != nil {
		return err
	}

	response, err := client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return err
	}
	corsRules, diagnostics := flattenBucketCorsRules(ctx, response.CORSRules)
	if diagnostics.HasError() {
		return bucketCorsDiagnosticError(diagnostics)
	}

	data.Id = types.StringValue(bucketCorsConfigurationResourceId(*data))
	data.CorsRules = corsRules
	return nil
}

func (r *BucketCorsConfigurationResource) s3Client(
	ctx context.Context,
	data BucketCorsConfigurationResourceModel,
) (*s3.Client, string, error) {
	bucketCredentials, err := waitForBucketS3Credentials(
		ctx,
		*r.client,
		data.ProjectId.ValueString(),
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
	if err != nil {
		return nil, "", err
	}

	usePathStyle, err := bucketUsesPathStyle(bucketCredentials.UrlStyle)
	if err != nil {
		return nil, "", err
	}
	config := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			bucketCredentials.AccessKeyId,
			bucketCredentials.SecretAccessKey,
			"",
		),
		Region: bucketCredentials.Region,
	}
	client := s3.NewFromConfig(config, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(bucketCredentials.Endpoint)
		options.UsePathStyle = usePathStyle
	})
	return client, bucketCredentials.BucketName, nil
}

func bucketUsesPathStyle(urlStyle string) (bool, error) {
	switch strings.ToLower(urlStyle) {
	case "path":
		return true, nil
	case "virtual", "virtual-host":
		return false, nil
	default:
		return false, fmt.Errorf("Railway returned unsupported bucket URL style %q", urlStyle)
	}
}

func expandBucketCorsRules(ctx context.Context, value basetypes.ListValue) ([]s3types.CORSRule, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var models []BucketCorsRuleModel
	diagnostics.Append(value.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	rules := make([]s3types.CORSRule, 0, len(models))
	for _, model := range models {
		var allowedHeaders []string
		var allowedMethods []string
		var allowedOrigins []string
		var exposeHeaders []string
		diagnostics.Append(model.AllowedHeaders.ElementsAs(ctx, &allowedHeaders, false)...)
		diagnostics.Append(model.AllowedMethods.ElementsAs(ctx, &allowedMethods, false)...)
		diagnostics.Append(model.AllowedOrigins.ElementsAs(ctx, &allowedOrigins, false)...)
		diagnostics.Append(model.ExposeHeaders.ElementsAs(ctx, &exposeHeaders, false)...)
		if diagnostics.HasError() {
			return nil, diagnostics
		}

		rule := s3types.CORSRule{
			AllowedHeaders: allowedHeaders,
			AllowedMethods: allowedMethods,
			AllowedOrigins: allowedOrigins,
			ExposeHeaders:  exposeHeaders,
		}
		if model.Id.ValueString() != "" {
			rule.ID = aws.String(model.Id.ValueString())
		}
		if model.MaxAgeSeconds.ValueInt64() > 0 {
			rule.MaxAgeSeconds = aws.Int32(int32(model.MaxAgeSeconds.ValueInt64()))
		}
		rules = append(rules, rule)
	}
	return rules, diagnostics
}

func flattenBucketCorsRules(ctx context.Context, rules []s3types.CORSRule) (basetypes.ListValue, diag.Diagnostics) {
	models := make([]BucketCorsRuleModel, 0, len(rules))
	var diagnostics diag.Diagnostics
	for _, rule := range rules {
		allowedHeaderValues := rule.AllowedHeaders
		if allowedHeaderValues == nil {
			allowedHeaderValues = []string{}
		}
		allowedHeaders, stringSetDiagnostics := basetypes.NewSetValueFrom(ctx, basetypes.StringType{}, allowedHeaderValues)
		diagnostics.Append(stringSetDiagnostics...)
		allowedMethods, stringSetDiagnostics := basetypes.NewSetValueFrom(ctx, basetypes.StringType{}, rule.AllowedMethods)
		diagnostics.Append(stringSetDiagnostics...)
		allowedOrigins, stringSetDiagnostics := basetypes.NewSetValueFrom(ctx, basetypes.StringType{}, rule.AllowedOrigins)
		diagnostics.Append(stringSetDiagnostics...)
		exposeHeaderValues := rule.ExposeHeaders
		if exposeHeaderValues == nil {
			exposeHeaderValues = []string{}
		}
		exposeHeaders, stringSetDiagnostics := basetypes.NewSetValueFrom(ctx, basetypes.StringType{}, exposeHeaderValues)
		diagnostics.Append(stringSetDiagnostics...)
		if diagnostics.HasError() {
			return basetypes.ListValue{}, diagnostics
		}

		models = append(models, BucketCorsRuleModel{
			Id:             basetypes.NewStringValue(aws.ToString(rule.ID)),
			AllowedHeaders: allowedHeaders,
			AllowedMethods: allowedMethods,
			AllowedOrigins: allowedOrigins,
			ExposeHeaders:  exposeHeaders,
			MaxAgeSeconds:  basetypes.NewInt64Value(int64(aws.ToInt32(rule.MaxAgeSeconds))),
		})
	}

	value, listDiagnostics := basetypes.NewListValueFrom(
		ctx,
		basetypes.ObjectType{AttrTypes: bucketCorsRuleAttrTypes},
		models,
	)
	diagnostics.Append(listDiagnostics...)
	return value, diagnostics
}

func isMissingBucketCors(err error) bool {
	if err == nil {
		return false
	}

	var apiError smithy.APIError
	return errors.As(err, &apiError) && apiError.ErrorCode() == "NoSuchCORSConfiguration"
}

func bucketCorsDiagnosticError(diagnostics diag.Diagnostics) error {
	diagnostic := diagnostics.Errors()[0]
	return fmt.Errorf("%s: %s", diagnostic.Summary(), diagnostic.Detail())
}

func bucketCorsConfigurationResourceId(data BucketCorsConfigurationResourceModel) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		data.ProjectId.ValueString(),
		data.EnvironmentId.ValueString(),
		data.BucketId.ValueString(),
	)
}
