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

var _ resource.Resource = &SharedVariableResource{}
var _ resource.ResourceWithImportState = &SharedVariableResource{}

func NewSharedVariableResource() resource.Resource {
	return &SharedVariableResource{}
}

type SharedVariableResource struct {
	client *graphql.Client
}

type SharedVariableResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Value         types.String `tfsdk:"value"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *SharedVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_variable"
}

func (r *SharedVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway shared variable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the variable.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the variable.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the variable.",
				Required:            true,
				Sensitive:           true,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the variable belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the variable belongs to.",
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

func (r *SharedVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SharedVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *SharedVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := VariableUpsertInput{
		Name:          data.Name.ValueString(),
		Value:         data.Value.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}

	_, err := upsertVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create shared variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a shared variable")

	err = getSharedVariable(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read shared variable after creating it, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SharedVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *SharedVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := getSharedVariable(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read shared variable, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SharedVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *SharedVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := VariableUpsertInput{
		Name:          data.Name.ValueString(),
		Value:         data.Value.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}

	_, err := upsertVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update shared variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a shared variable")

	err = getSharedVariable(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read shared variable after updating it, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SharedVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *SharedVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := VariableDeleteInput{
		Name:          data.Name.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}

	_, err := deleteVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete shared variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a shared variable")
}

func (r *SharedVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: project_id:environment_id:name. Got: %q", req.ID),
		)

		return
	}

	environmentId, err := findEnvironment(ctx, *r.client, parts[0], parts[1])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
}

func getSharedVariable(ctx context.Context, client graphql.Client, projectId string, environmentId string, name string, data *SharedVariableResourceModel) error {
	response, err := getSharedVariables(ctx, client, projectId, environmentId)

	if err != nil {
		return err
	}

	if value, ok := response.Variables[name]; ok {
		data.Id = types.StringValue(fmt.Sprintf("%s:%s:%s", projectId, environmentId, name))
		data.Name = types.StringValue(name)
		data.Value = types.StringValue(fmt.Sprintf("%v", value))
		data.ProjectId = types.StringValue(projectId)
		data.EnvironmentId = types.StringValue(environmentId)
	}

	return nil
}
