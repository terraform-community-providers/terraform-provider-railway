package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &WorkspaceMembersDataSource{}
var _ datasource.DataSourceWithConfigure = &WorkspaceMembersDataSource{}

func NewWorkspaceMembersDataSource() datasource.DataSource {
	return &WorkspaceMembersDataSource{}
}

type WorkspaceMembersDataSource struct {
	client *graphql.Client
}

type WorkspaceMembersDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Members     types.List   `tfsdk:"members"`
	IdsByEmail  types.Map    `tfsdk:"ids_by_email"`
}

type WorkspaceMembersDataSourceMemberModel struct {
	Id    types.String `tfsdk:"id"`
	Email types.String `tfsdk:"email"`
	Name  types.String `tfsdk:"name"`
	Role  types.String `tfsdk:"role"`
}

var workspaceMemberAttrTypes = map[string]attr.Type{
	"id":    types.StringType,
	"email": types.StringType,
	"name":  types.StringType,
	"role":  types.StringType,
}

func (d *WorkspaceMembersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_members"
}

func (d *WorkspaceMembersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Members of a Railway workspace. Pending invitations are not included.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the data source, same as `workspace_id`.",
				Computed:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "Members of the workspace.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Identifier of the user.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "Email of the user.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the user.",
							Computed:            true,
						},
						"role": schema.StringAttribute{
							MarkdownDescription: "Role of the user in the workspace.",
							Computed:            true,
						},
					},
				},
			},
			"ids_by_email": schema.MapAttribute{
				MarkdownDescription: "Map of member email to user identifier, for looking up a member by email. Emails are as stored by Railway.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *WorkspaceMembersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkspaceMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *WorkspaceMembersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getWorkspaceMembers(ctx, *d.client, data.WorkspaceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workspace members, got error: %s", err))
		return
	}

	members := make([]WorkspaceMembersDataSourceMemberModel, 0, len(response.Workspace.Members))
	idsByEmail := make(map[string]string, len(response.Workspace.Members))

	for _, member := range response.Workspace.Members {
		name := types.StringNull()

		if member.Name != nil {
			name = types.StringValue(*member.Name)
		}

		members = append(members, WorkspaceMembersDataSourceMemberModel{
			Id:    types.StringValue(member.Id),
			Email: types.StringValue(member.Email),
			Name:  name,
			Role:  types.StringValue(string(member.Role)),
		})

		idsByEmail[member.Email] = member.Id
	}

	membersValue, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: workspaceMemberAttrTypes}, members)
	resp.Diagnostics.Append(diags...)

	idsByEmailValue, diags := types.MapValueFrom(ctx, types.StringType, idsByEmail)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Id = data.WorkspaceId
	data.Members = membersValue
	data.IdsByEmail = idsByEmailValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
