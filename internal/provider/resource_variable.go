package provider

import (
	"context"
	"fmt"
	"strings"

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

var _ resource.Resource = &VariableResource{}
var _ resource.ResourceWithImportState = &VariableResource{}

func NewVariableResource() resource.Resource {
	return &VariableResource{}
}

type VariableResource struct {
	client *graphql.Client
}

type VariableResourceModel struct {
	Id                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Value                 types.String `tfsdk:"value"`
	ValueWriteOnly        types.String `tfsdk:"value_wo"`
	ValueWriteOnlyVersion types.Int64  `tfsdk:"value_wo_version"`
	EnvironmentId         types.String `tfsdk:"environment_id"`
	ServiceId             types.String `tfsdk:"service_id"`
	ProjectId             types.String `tfsdk:"project_id"`
}

func (r *VariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (r *VariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service variable. Configure exactly one of `value` for a readable variable or `value_wo` with `value_wo_version` for a sealed variable whose value must not be stored in Terraform state. Changes to variables trigger service redeployment.",
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
				MarkdownDescription: "Readable value stored in Terraform state.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						replaceWhenUnsealing,
						"Changing a sealed variable to a readable variable requires replacement.",
						"Changing a sealed variable to a readable variable requires replacement.",
					),
				},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("value_wo")),
				},
			},
			"value_wo": schema.StringAttribute{
				MarkdownDescription: "Write-only value used to create a sealed variable. The value is never stored in Terraform state and cannot be retrieved from Railway.",
				Optional:            true,
				WriteOnly:           true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("value_wo_version")),
				},
			},
			"value_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version used to trigger updates to `value_wo`. Change this value whenever the write-only value changes. Imported sealed variables start at version 1, so use a greater version to rotate their value after import.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(path.MatchRoot("value_wo")),
				},
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
	var config *VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	data.ProjectId = types.StringValue(service.Service.ProjectId)
	if err := r.upsertVariable(ctx, data, config.ValueWriteOnly.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a variable")

	if data.isSealed() {
		data.Id = types.StringValue(variableID(
			data.ServiceId.ValueString(),
			data.EnvironmentId.ValueString(),
			data.Name.ValueString(),
		))
	} else {
		exists, err := getVariable(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable after creating it, got error: %s", err))
			return
		}
		if !exists {
			resp.Diagnostics.AddError("Client Error", "Railway did not return the variable after creating it")
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.isSealed() {
		exists, err := sealedVariableExists(
			ctx,
			*r.client,
			data.EnvironmentId.ValueString(),
			data.ServiceId.ValueString(),
			data.Name.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read sealed variable, got error: %s", err))
			return
		}
		if !exists {
			resp.State.RemoveResource(ctx)
			return
		}

		data.Id = types.StringValue(variableID(
			data.ServiceId.ValueString(),
			data.EnvironmentId.ValueString(),
			data.Name.ValueString(),
		))
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	exists, err := getVariable(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable, got error: %s", err))
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VariableResourceModel
	var config *VariableResourceModel
	var state *VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.ProjectId = state.ProjectId
	if err := r.upsertVariable(ctx, data, config.ValueWriteOnly.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update variable, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated a variable")

	if data.isSealed() {
		data.Id = types.StringValue(variableID(
			data.ServiceId.ValueString(),
			data.EnvironmentId.ValueString(),
			data.Name.ValueString(),
		))
	} else {
		exists, err := getVariable(ctx, *r.client, state.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Name.ValueString(), data)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable after updating it, got error: %s", err))
			return
		}
		if !exists {
			resp.Diagnostics.AddError("Client Error", "Railway did not return the variable after updating it")
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if data.isSealed() {
		err = r.deleteSealedVariable(ctx, data)
	} else {
		err = r.deleteReadableVariable(ctx, data)
	}
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

	sealed, err := sealedVariableExists(ctx, *r.client, *environmentId, parts[0], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to determine whether imported variable is sealed, got error: %s", err))
		return
	}
	if sealed {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value_wo_version"), 1)...)
	}
}

func getVariable(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, name string, data *VariableResourceModel) (bool, error) {
	response, err := getVariables(ctx, client, projectId, environmentId, serviceId)

	if err != nil {
		return false, err
	}

	value, exists := response.Variables[name]
	if !exists {
		return false, nil
	}

	data.Id = types.StringValue(variableID(serviceId, environmentId, name))
	data.Name = types.StringValue(name)
	data.Value = types.StringValue(fmt.Sprintf("%v", value))
	data.ProjectId = types.StringValue(projectId)
	data.EnvironmentId = types.StringValue(environmentId)
	data.ServiceId = types.StringValue(serviceId)

	return true, nil
}

func (r *VariableResource) upsertVariable(ctx context.Context, data *VariableResourceModel, writeOnlyValue string) error {
	if data.isSealed() {
		return r.upsertSealedVariable(ctx, data, writeOnlyValue)
	}
	return r.upsertReadableVariable(ctx, data)
}

func (r *VariableResource) upsertReadableVariable(ctx context.Context, data *VariableResourceModel) error {
	input := VariableUpsertInput{
		Name:          data.Name.ValueString(),
		Value:         data.Value.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}
	if _, err := upsertVariable(ctx, *r.client, input); err != nil {
		return err
	}
	_, err := redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())
	return err
}

func (r *VariableResource) upsertSealedVariable(ctx context.Context, data *VariableResourceModel, value string) error {
	patch := railway.SealedVariablePatch(data.ServiceId.ValueString(), data.Name.ValueString(), value)
	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		"Manage Terraform sealed variable",
	)
}

func (r *VariableResource) deleteReadableVariable(ctx context.Context, data *VariableResourceModel) error {
	input := VariableDeleteInput{
		Name:          data.Name.ValueString(),
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     data.ProjectId.ValueString(),
	}
	if _, err := deleteVariable(ctx, *r.client, input); err != nil {
		return err
	}
	_, err := redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())
	return err
}

func (r *VariableResource) deleteSealedVariable(ctx context.Context, data *VariableResourceModel) error {
	patch := railway.DeleteVariablePatch(data.ServiceId.ValueString(), data.Name.ValueString())
	return commitAndWaitForEnvironmentPatch(
		ctx,
		*r.client,
		data.EnvironmentId.ValueString(),
		patch,
		"Delete Terraform sealed variable",
	)
}

func sealedVariableExists(ctx context.Context, client graphql.Client, environmentId string, serviceId string, name string) (bool, error) {
	var after *string
	for {
		response, err := getEnvironmentVariables(ctx, client, environmentId, after)
		if err != nil {
			return false, err
		}

		variables := response.Environment.Variables
		for _, edge := range variables.Edges {
			variable := edge.Node
			if variable.ServiceId == serviceId &&
				variable.Name == name {
				return variable.IsSealed, nil
			}
		}

		if !variables.PageInfo.HasNextPage {
			return false, nil
		}
		after = &variables.PageInfo.EndCursor
	}
}

func (m *VariableResourceModel) isSealed() bool {
	return !m.ValueWriteOnlyVersion.IsNull()
}

func variableID(serviceID string, environmentID string, name string) string {
	return fmt.Sprintf("%s:%s:%s", serviceID, environmentID, name)
}

func replaceWhenUnsealing(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.StateValue.IsNull() && !req.PlanValue.IsNull()
}
