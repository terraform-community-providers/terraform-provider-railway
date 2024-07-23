package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VariableCollectionResource{}
var _ resource.ResourceWithImportState = &VariableCollectionResource{}

func NewVariableCollectionResource() resource.Resource {
	return &VariableCollectionResource{}
}

type VariableCollectionResource struct {
	client *graphql.Client
}

type VariableCollectionResourceVariableModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

var variableAttrTypes = map[string]attr.Type{
	"name":  types.StringType,
	"value": types.StringType,
}

type VariableCollectionResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Variables     types.List   `tfsdk:"variables"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *VariableCollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable_collection"
}

func (r *VariableCollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway variable collection. Group of variables managed as a whole. Any changes in collection triggers service redeployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the variable collection.",
				Computed:            true,
			},
			"variables": schema.ListNestedAttribute{
				MarkdownDescription: "Collection of variables.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the variable.",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the variable.",
							Required:            true,
							Sensitive:           true,
						},
					},
				},
				Required: true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the variable collection belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the variable collection belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the variable collection belongs to.",
				Computed:            true,
			},
		},
	}
}

func (r *VariableCollectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VariableCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *VariableCollectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	variablesData := make([]VariableCollectionResourceVariableModel, 0, len(data.Variables.Elements()))

	resp.Diagnostics.Append(data.Variables.ElementsAs(ctx, &variablesData, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variablesMap := make(map[string]interface{})

	for _, v := range variablesData {
		variablesMap[v.Name.ValueString()] = v.Value.ValueString()
	}

	input := VariableCollectionUpsertInput{
		Variables:     variablesMap,
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     service.Service.ProjectId,
	}

	_, err = upsertVariableCollection(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create variable collection, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a variable collection")

	variableNames, diagErr := getVariableNames(ctx, data)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	err = getVariableCollection(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable collection after creating it, got error: %s", err))
		return
	}

	_, err = redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy service after variable collection created, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *VariableCollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variableNames, diagErr := getVariableNames(ctx, data)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	err := getVariableCollection(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable collection, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *VariableCollectionResourceModel
	var state *VariableCollectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variablesMapToUpsert, diagErr := getVariablesToUpsert(ctx, data, state)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	if len(variablesMapToUpsert) > 0 {
		input := VariableCollectionUpsertInput{
			ServiceId:     data.ServiceId.ValueStringPointer(),
			EnvironmentId: data.EnvironmentId.ValueString(),
			ProjectId:     state.ProjectId.ValueString(),
			Variables:     variablesMapToUpsert,
		}

		_, err := upsertVariableCollection(ctx, *r.client, input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upsert variables of variable collection, got error: %s", err))
			return
		}
	}

	variableNamesToDelete, diagErr := getVariableNamesToDelete(ctx, data, state)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	if len(variableNamesToDelete) > 0 {
		err := deleteManyVariables(ctx, *r.client, state.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNamesToDelete)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete variables of variable collection, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "updated a variable collection")

	allVariableNames, diagErr := getVariableNames(ctx, data)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	err := getVariableCollection(ctx, *r.client, state.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), allVariableNames, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable collection after updating it, got error: %s", err))
		return
	}

	_, err = redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy service after variable collection updated, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *VariableCollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variableNames, diagErr := getVariableNames(ctx, data)

	if diagErr != nil {
		resp.Diagnostics.Append(diagErr...)
		return
	}

	err := deleteManyVariables(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete variable collection, got error: %s", err))
		return
	}

	_, err = redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy service after variable collection deleted, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a variable collection")
}

func (r *VariableCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) < 3 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_name:name1:name2:name3:... Got: %q", req.ID),
		)

		return
	}

	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected import identifier with format: service_id:environment_name:name1:name2:name3:... Got: %q", req.ID),
			)

			return
		}
	}

	serviceId := parts[0]
	service, err := getService(ctx, *r.client, serviceId)

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

	variables := make([]attr.Value, 0, len(parts[2:]))

	for _, variableName := range parts[2:] {
		variables = append(variables, types.ObjectValueMust(variableAttrTypes, map[string]attr.Value{
			"name":  types.StringValue(variableName),
			"value": types.StringUnknown(),
		}))
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("variables"), variables)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), serviceId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
}

func getVariableNames(ctx context.Context, data *VariableCollectionResourceModel) ([]string, diag.Diagnostics) {
	length := len(data.Variables.Elements())
	variablesData := make([]VariableCollectionResourceVariableModel, 0, length)

	err := data.Variables.ElementsAs(ctx, &variablesData, false)

	if err != nil {
		return nil, err
	}

	variableNames := make([]string, 0, length)

	for _, v := range variablesData {
		variableNames = append(variableNames, v.Name.ValueString())
	}

	return variableNames, nil
}

func getVariableCollection(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, names []string, data *VariableCollectionResourceModel) error {
	if len(names) == 0 {
		return errors.New("cannot get variable collection with no variable names")
	}

	response, err := getVariables(ctx, client, projectId, environmentId, serviceId)

	if err != nil {
		return err
	}

	variables := make([]attr.Value, 0, len(names))

	for _, name := range names {
		if value, ok := response.Variables[name]; ok {
			if str, ok := value.(string); ok {
				variables = append(variables, types.ObjectValueMust(variableAttrTypes, map[string]attr.Value{
					"name":  types.StringValue(name),
					"value": types.StringValue(str),
				}))
			} else {
				return fmt.Errorf("cannot convert variable %s to string", name)
			}
		}
	}

	data.Id = types.StringValue(getVariableCollectionId(ctx, serviceId, environmentId, names))
	data.ProjectId = types.StringValue(projectId)
	data.EnvironmentId = types.StringValue(environmentId)
	data.ServiceId = types.StringValue(serviceId)
	data.Variables = types.ListValueMust(types.ObjectType{AttrTypes: variableAttrTypes}, variables)

	return nil
}

func getVariableCollectionId(ctx context.Context, serviceId, environmentId string, names []string) string {
	// we need stable identifiers for this synthetic resource. Since order of `names` is not really defined
	// (since they are keys of a map), I'm going to sort them first and then concat to one long id string
	namesSortedAsc := append([]string(nil), names...)
	sort.Strings(namesSortedAsc)

	return fmt.Sprintf("%s:%s:%s", serviceId, environmentId, strings.Join(namesSortedAsc, ":"))
}

func deleteManyVariables(ctx context.Context, client graphql.Client, projectId, environmentId, serviceId string, names []string) error {
	for _, name := range names {
		input := VariableDeleteInput{
			Name:          name,
			ServiceId:     &serviceId,
			EnvironmentId: environmentId,
			ProjectId:     projectId,
		}

		_, err := deleteVariable(ctx, client, input)

		if err != nil {
			return err
		}
	}

	return nil
}

// getVariablesToUpsert returns a map where entries have to be upserted. The criteria is the following:
// if entry is in the data, but not in the state, then it has to be created
// if entry is in the data and is in the state, and values are different, then it has to be updated
func getVariablesToUpsert(ctx context.Context, data, state *VariableCollectionResourceModel) (map[string]interface{}, diag.Diagnostics) {
	variablesData := make([]VariableCollectionResourceVariableModel, 0, len(data.Variables.Elements()))
	variablesState := make([]VariableCollectionResourceVariableModel, 0, len(state.Variables.Elements()))

	err := data.Variables.ElementsAs(ctx, &variablesData, false)

	if err != nil {
		return nil, err
	}

	err = state.Variables.ElementsAs(ctx, &variablesState, false)

	if err != nil {
		return nil, err
	}

	variablesToUpsert := make(map[string]interface{})
	variablesStateMap := make(map[string]string, len(variablesState))

	for _, v := range variablesState {
		variablesStateMap[v.Name.ValueString()] = v.Value.ValueString()
	}

	for _, v := range variablesData {
		if stateValue, ok := variablesStateMap[v.Name.ValueString()]; !ok || (ok && v.Value.ValueString() != stateValue) {
			variablesToUpsert[v.Name.ValueString()] = v.Value.ValueString()
		}
	}

	return variablesToUpsert, nil
}

// getVariableNamesToDelete returns an array where entries are names of variables to delete. The criteria is the following:
// if variables is in the state, but not in the data, then it has to be deleted
func getVariableNamesToDelete(ctx context.Context, data, state *VariableCollectionResourceModel) ([]string, diag.Diagnostics) {
	variableNamesData, err := getVariableNames(ctx, data)

	if err != nil {
		return nil, err
	}

	variableNamesState, err := getVariableNames(ctx, state)

	if err != nil {
		return nil, err
	}

	variableNamesDataMap := make(map[string]interface{}, len(variableNamesData))

	for _, v := range variableNamesData {
		variableNamesDataMap[v] = true
	}

	variableNamesToDelete := make([]string, 0)

	for _, v := range variableNamesState {
		if _, ok := variableNamesDataMap[v]; !ok {
			variableNamesToDelete = append(variableNamesToDelete, v)
		}
	}

	return variableNamesToDelete, nil
}
