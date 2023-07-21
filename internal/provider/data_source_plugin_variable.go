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

var _ datasource.DataSource = &PluginVariableDataSource{}

func NewPluginVariableDataSource() datasource.DataSource {
	return &PluginVariableDataSource{}
}

type PluginVariableDataSource struct {
	client *graphql.Client
}

type PluginVariableDataSourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Value         types.String `tfsdk:"value"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	PluginId      types.String `tfsdk:"plugin_id"`
	ProjectId     types.String `tfsdk:"project_id"`
}

func (d *PluginVariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_variable"
}

func (d *PluginVariableDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway plugin variable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the variable.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the variable.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the variable.",
				Computed:            true,
				Sensitive:           true,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the variable belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"plugin_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the plugin the variable belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the variable belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
		},
	}
}

func (d *PluginVariableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *PluginVariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *PluginVariableDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	projectId := data.ProjectId.ValueString()
	environmentId := data.EnvironmentId.ValueString()
	pluginId := data.PluginId.ValueString()

	response, err := getPluginVariables(ctx, *d.client, projectId, environmentId, pluginId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read plugin variable, got error: %s", err))
		return
	}

	if value, ok := response.Variables[name]; ok {
		tflog.Trace(ctx, "read a plugin variable")

		data.Id = types.StringValue(fmt.Sprintf("%s:%s:%s", pluginId, environmentId, name))
		data.Name = types.StringValue(name)
		data.Value = types.StringValue(fmt.Sprintf("%v", value))
		data.ProjectId = types.StringValue(projectId)
		data.EnvironmentId = types.StringValue(environmentId)
		data.PluginId = types.StringValue(pluginId)
	} else {
		resp.Diagnostics.AddError("Client Error", "Unable to find plugin variable")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
