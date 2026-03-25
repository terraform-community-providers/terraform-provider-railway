package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &ServiceVariablesDataSource{}

func NewServiceVariablesDataSource() datasource.DataSource {
	return &ServiceVariablesDataSource{}
}

type ServiceVariablesDataSource struct {
	client *graphql.Client
}

type ServiceVariablesDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	ServiceId     types.String `tfsdk:"service_id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Variables     types.Map    `tfsdk:"variables"`
}

func (d *ServiceVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_variables"
}

func (d *ServiceVariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read all variables for a Railway service in a given environment. Useful for referencing connection strings and credentials from database or cache services managed outside the current Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the data source.",
				Computed:            true,
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service to read variables from.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment to read variables from.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the service belongs to.",
				Computed:            true,
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Map of variable names to their values.",
				Computed:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *ServiceVariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*graphql.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *graphql.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ServiceVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceVariablesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := data.ServiceId.ValueString()
	environmentId := data.EnvironmentId.ValueString()

	// Look up the project ID from the service
	service, err := getService(ctx, *d.client, serviceId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	projectId := service.Service.ProjectId

	// Fetch all variables for the service
	response, err := getVariables(ctx, *d.client, projectId, environmentId, serviceId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read variables, got error: %s", err))
		return
	}

	// Convert the response map to a Terraform map of strings
	variables := make(map[string]string)
	for key, value := range response.Variables {
		variables[key] = fmt.Sprintf("%v", value)
	}

	variablesMap, diags := types.MapValueFrom(ctx, types.StringType, variables)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("%s:%s", serviceId, environmentId))
	data.ProjectId = types.StringValue(projectId)
	data.Variables = variablesMap

	tflog.Trace(ctx, "read service variables", map[string]interface{}{
		"service_id":     serviceId,
		"environment_id": environmentId,
		"variable_count": len(variables),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
