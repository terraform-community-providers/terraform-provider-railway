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

var _ resource.Resource = &VariableResource{}
var _ resource.ResourceWithImportState = &VariableResource{}

func NewVariableResource() resource.Resource {
	return &VariableResource{}
}

type VariableResource struct {
	client *graphql.Client
}

type VariableResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Value         types.String `tfsdk:"value"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *VariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (r *VariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway variable.",
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
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the variable belongs to.",
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
				Computed:            true,
			},
		},
	}
}

func (r *VariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	input := VariableUpsertInput{
		Name:          data.Name.ValueString(),
		Value:         data.Value.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     service.Service.ProjectId,
	}

	_, err = upsertVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a variable")

	err = getVariable(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable after creating it, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := getVariable(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	input := VariableUpsertInput{
		Name:          data.Name.ValueString(),
		Value:         data.Value.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     service.Service.ProjectId,
	}

	_, err = upsertVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a variable")

	err = getVariable(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable after updating it, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := VariableDeleteInput{
		Name:          data.Name.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}

	_, err := deleteVariable(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a variable")
}

func (r *VariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_name:name. Got: %q", req.ID),
		)

		return
	}

	service, err := getService(ctx, *r.client, parts[0])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	projectId := service.Service.ProjectId
	environmentId, err := findEnvironment(ctx, *r.client, projectId, parts[1])

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
}

func getVariable(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, name string, data *VariableResourceModel) error {
	response, err := getVariables(ctx, client, projectId, environmentId, serviceId)

	if err != nil {
		return err
	}

	if value, ok := response.Variables[name]; ok {
		data.Id = types.StringValue(fmt.Sprintf("%s:%s:%s", serviceId, environmentId, name))
		data.Name = types.StringValue(name)
		data.Value = types.StringValue(fmt.Sprintf("%v", value))
		data.ProjectId = types.StringValue(projectId)
		data.EnvironmentId = types.StringValue(environmentId)
		data.ServiceId = types.StringValue(serviceId)
	}

	return nil
}
