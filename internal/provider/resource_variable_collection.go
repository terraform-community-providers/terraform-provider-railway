package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	ProjectId     types.String `tfsdk:"project_id"`
}

func (r *VariableCollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	tflog.Info(ctx, "METADATA")
	resp.TypeName = req.ProviderTypeName + "_variable_collection"
}

func (r *VariableCollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	tflog.Info(ctx, "SCHEMA")
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway variable collection. Group of variables managed as a whole",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the variable collection.",
				Computed:            true,
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Collection of variables.",
				ElementType:         types.StringType,
				Required:            true,
				Sensitive:           true,
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
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the variable belongs to.",
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
	tflog.Info(ctx, "DOING CREATE")
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "READ")
	var data *VariableCollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variableNames := make([]string, 0)
	for name := range data.Variables.Elements() {
		variableNames = append(variableNames, name)
	}

	err := getVariableCollection(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), variableNames, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variable collection, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO
}

func (r *VariableCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO
}

func (r *VariableCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// TODO
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
		// TODO: should `types.StringNull` be used when ok == false instead of omitting a value from map?
		if value, ok := response.Variables[name]; ok {
			if str, ok := value.(string); ok {
				variablesMap[name] = types.StringValue(str)
			} else {
				return errors.New(fmt.Sprintf("cannot convert variable %s to string", name))
			}
		}
	}

	data.Id = types.StringValue(getVariableCollectionId(serviceId, environmentId, names))
	data.ProjectId = types.StringValue(projectId)
	data.EnvironmentId = types.StringValue(environmentId)
	data.ServiceId = types.StringValue(serviceId)
	data.Variables, _ = types.MapValue(types.StringType, variablesMap)

	return nil
}

func getVariableCollectionId(serviceId, environmentId string, names []string) string {

	// we need stable identifiers for this synthetic resource. Since order of `names` is not really defined
	// (since they are keys of a map), I'm going to sort them first and then concat to one long id string
	namesSortedAsc := append([]string(nil), names...)
	sort.Strings(namesSortedAsc)

	return fmt.Sprintf("%s:%s:%s", serviceId, environmentId, strings.Join(namesSortedAsc, ":"))
}
