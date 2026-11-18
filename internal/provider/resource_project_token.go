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

var _ resource.Resource = &ProjectTokenResource{}
var _ resource.ResourceWithImportState = &ProjectTokenResource{}

func NewProjectTokenResource() resource.Resource {
	return &ProjectTokenResource{}
}

type ProjectTokenResource struct {
	client *graphql.Client
}

type ProjectTokenResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ProjectId     types.String `tfsdk:"project_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	Token         types.String `tfsdk:"token"`
}

func (r *ProjectTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_token"
}

func (r *ProjectTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway project token. Project tokens cannot be updated, changing any attribute replaces the token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project token.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project token.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the project token belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the project token has access to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Value of the project token. Only available when created by Terraform, it is not populated on import.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *ProjectTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ProjectTokenCreateInput{
		Name:          data.Name.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	response, err := createProjectToken(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project token, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a project token")

	// The create mutation only returns the token value, so look up the id.
	tokens, err := getProjectTokens(ctx, *r.client, data.ProjectId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project token after create, got error: %s", err))
		return
	}

	var created *ProjectToken

	// Names are not unique, so take the most recently created match.
	for _, edge := range tokens.ProjectTokens.Edges {
		token := edge.Node.ProjectToken

		if token.Name == data.Name.ValueString() && token.EnvironmentId == data.EnvironmentId.ValueString() &&
			(created == nil || token.CreatedAt.After(created.CreatedAt)) {
			created = &token
		}
	}

	if created == nil {
		resp.Diagnostics.AddError("Client Error", "Unable to find project token after create")
		return
	}

	data.Id = types.StringValue(created.Id)
	data.Token = types.StringValue(response.ProjectTokenCreate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getProjectTokens(ctx, *r.client, data.ProjectId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project token, got error: %s", err))
		return
	}

	found := false

	for _, edge := range response.ProjectTokens.Edges {
		token := edge.Node.ProjectToken

		if token.Id == data.Id.ValueString() {
			found = true
			data.Name = types.StringValue(token.Name)
			data.ProjectId = types.StringValue(token.ProjectId)
			data.EnvironmentId = types.StringValue(token.EnvironmentId)
		}
	}

	// Token was deleted outside of Terraform.
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// The token value is only known at creation time, keep what is in state.
	if data.Token.IsNull() || data.Token.IsUnknown() {
		data.Token = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectTokenResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteProjectToken(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project token, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a project token")
}

func (r *ProjectTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:project_token_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
