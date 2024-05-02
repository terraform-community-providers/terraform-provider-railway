package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"sort"
	"strings"

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

type VariableCollectionResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Variables     types.Map    `tfsdk:"variables"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
}

func (r *VariableCollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable_collection"
}

func (r *VariableCollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway variable collection. Group of variables managed as a whole. Any changes in collection are triggering service redeployment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the variable collection.",
				Computed:            true,
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Collection of variables.",
				ElementType:         types.StringType,
				Required:            true,
				//Sensitive:           true,
				// @todo: probably need a validator to prevent empty map and the same keys
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

	tfVariablesMap := data.Variables.Elements()
	variablesMap := make(map[string]interface{})
	for k, v := range tfVariablesMap {
		variablesMap[k] = v.(types.String).ValueString()
	}

	input := VariableCollectionUpsertInput{
		ServiceId:     data.ServiceId.ValueStringPointer(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     service.Service.ProjectId,
		Variables:     variablesMap,
	}

	_, err = upsertVariableCollection(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create variable collection, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a variable collection")

	variableNames := make([]string, 0, len(variablesMap))
	for name := range variablesMap {
		variableNames = append(variableNames, name)
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

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	variableNames := make([]string, 0)
	for name := range data.Variables.Elements() {
		variableNames = append(variableNames, name)
	}

	err = getVariableCollection(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames, data)

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

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	tfVariablesMapToUpsert := getVariablesToUpsert(data, state)
	variablesMapToUpsert := make(map[string]interface{})
	for k, v := range tfVariablesMapToUpsert {
		variablesMapToUpsert[k] = v.(types.String).ValueString()
	}

	if len(variablesMapToUpsert) > 0 {

		input := VariableCollectionUpsertInput{
			ServiceId:     data.ServiceId.ValueStringPointer(),
			EnvironmentId: data.EnvironmentId.ValueString(),
			ProjectId:     service.Service.ProjectId,
			Variables:     variablesMapToUpsert,
		}

		_, err = upsertVariableCollection(ctx, *r.client, input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upsert variables of variable collection, got error: %s", err))
			return
		}
	}

	variableNamesToDelete := getVariableNamesToDelete(data, state)

	if len(variableNamesToDelete) > 0 {

		err = deleteManyVariables(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNamesToDelete)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete variables of variable collection, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "updated a variable collection")

	allVariableNames := make([]string, 0)
	for k := range data.Variables.Elements() {
		allVariableNames = append(allVariableNames, k)
	}

	err = getVariableCollection(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), allVariableNames, data)

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

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	variableNames := make([]string, 0)
	for name := range data.Variables.Elements() {
		variableNames = append(variableNames, name)
	}

	err = deleteManyVariables(ctx, *r.client, service.Service.ProjectId, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete variable collection, got error: %s", err))
		return
	}

	_, err = redeployServiceInstance(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to redeploy service after variable collection updated, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a variable collection")
}

func (r *VariableCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// terraform import railway_variable_collection.sentry 89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging:SENTRY_KEY:SENTRY_SECRET
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
	environmentName := parts[1]
	variableNames := parts[2:]

	service, err := getService(ctx, *r.client, serviceId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	projectId := service.Service.ProjectId
	environmentId, err := findEnvironment(ctx, *r.client, projectId, environmentName)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment, got error: %s", err))
		return
	}

	variablesMap := make(map[string]types.String)
	for _, variableName := range variableNames {
		variablesMap[variableName] = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), serviceId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("variables"), variablesMap)...)

}

func getVariableCollection(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, names []string, data *VariableCollectionResourceModel) error {
	if len(names) == 0 {
		return errors.New("cannot get variable collection with no variable names")
	}

	response, err := getVariables(ctx, client, projectId, environmentId, serviceId)

	if err != nil {
		return err
	}

	variablesMap := make(map[string]attr.Value)
	for _, name := range names {
		if value, ok := response.Variables[name]; ok {
			if str, ok := value.(string); ok {
				variablesMap[name] = types.StringValue(str)
			} else {
				return errors.New(fmt.Sprintf("cannot convert variable %s to string", name))
			}
		}
	}

	data.Id = types.StringValue(getVariableCollectionId(ctx, serviceId, environmentId, names))
	data.EnvironmentId = types.StringValue(environmentId)
	data.ServiceId = types.StringValue(serviceId)
	data.Variables, _ = types.MapValue(types.StringType, variablesMap)

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
func getVariablesToUpsert(data, state *VariableCollectionResourceModel) map[string]attr.Value {
	dataVariablesMap := data.Variables.Elements()
	stateVariablesMap := state.Variables.Elements()

	variablesToUpsert := make(map[string]attr.Value)

	for dataName, dataValue := range dataVariablesMap {
		if stateValue, ok := stateVariablesMap[dataName]; !ok || (ok && !dataValue.Equal(stateValue)) {
			variablesToUpsert[dataName] = dataValue
		}
	}

	return variablesToUpsert
}

// getVariableNamesToDelete returns an array where entries are names of variables to delete. The criteria is the following:
// if variables is in the state, but not in the data, then it has to be deleted
func getVariableNamesToDelete(data, state *VariableCollectionResourceModel) []string {
	dataVariablesMap := data.Variables.Elements()
	stateVariablesMap := state.Variables.Elements()

	variableNamesToDelete := make([]string, 0)

	for stateName := range stateVariablesMap {
		if _, ok := dataVariablesMap[stateName]; !ok {
			variableNamesToDelete = append(variableNamesToDelete, stateName)
		}
	}

	return variableNamesToDelete
}
