package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AccessGroupResource{}
var _ resource.ResourceWithImportState = &AccessGroupResource{}

func NewAccessGroupResource() resource.Resource {
	return &AccessGroupResource{}
}

type AccessGroupResource struct {
	client *graphql.Client
}

type AccessGroupResourceModel struct {
	Id          types.String `tfsdk:"id"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
	Name        types.String `tfsdk:"name"`
	Role        types.String `tfsdk:"role"`
	MemberIds   types.Set    `tfsdk:"member_ids"`
	ProjectIds  types.Set    `tfsdk:"project_ids"`
}

func (r *AccessGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_group"
}

func (r *AccessGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptySet := types.SetValueMust(types.StringType, []attr.Value{})

	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway access group. Members and projects are managed exclusively by Terraform: any member or project added outside of Terraform is removed on the next apply. Access groups are a beta Railway feature that is only available on eligible plans, see the [Railway docs](https://docs.railway.com/enterprise/access-groups).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the access group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace the access group belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the access group.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role the access group grants on its projects. Possible values are `ADMIN`, `MEMBER` and `VIEWER`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ADMIN", "MEMBER", "VIEWER"),
				},
			},
			"member_ids": schema.SetAttribute{
				MarkdownDescription: "Identifiers of the users in the access group. They must be members of the workspace, see the `railway_workspace_members` data source. **Default** `[]`.",
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(emptySet),
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.UTF8LengthAtLeast(1)),
				},
			},
			"project_ids": schema.SetAttribute{
				MarkdownDescription: "Identifiers of the projects attached to the access group. **Default** `[]`.",
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(emptySet),
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(uuidRegex(), "must be an id")),
				},
			},
		},
	}
}

func (r *AccessGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := AccessGroupCreateInput{
		Name:        data.Name.ValueString(),
		Role:        ProjectRole(data.Role.ValueString()),
		WorkspaceId: data.WorkspaceId.ValueString(),
	}

	response, err := createAccessGroup(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created an access group")

	data.Id = types.StringValue(response.AccessGroupCreate.Id)

	// Save the group before adding members and projects, so it is tracked even if one of them fails.
	// If one of them fails, Terraform marks the group tainted and replaces it on the next apply,
	// so a half-built group is never left behind untracked.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.reconcile(ctx, data, nil, nil)...)
}

func (r *AccessGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	details, err := findAccessGroup(ctx, *r.client, data.WorkspaceId.ValueString(), data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access group, got error: %s", err))
		return
	}

	// Access group was deleted outside of Terraform.
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(setAccessGroupModel(ctx, data, details)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *AccessGroupResourceModel
	var state *AccessGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Name.Equal(state.Name) || !data.Role.Equal(state.Role) {
		input := AccessGroupUpdateInput{
			Name: data.Name.ValueString(),
			Role: ProjectRole(data.Role.ValueString()),
		}

		_, err := updateAccessGroup(ctx, *r.client, data.Id.ValueString(), input)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update access group, got error: %s", err))
			return
		}
	}

	haveMembers, diags := stringSetValues(ctx, state.MemberIds)
	resp.Diagnostics.Append(diags...)

	haveProjects, diags := stringSetValues(ctx, state.ProjectIds)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.reconcile(ctx, data, haveMembers, haveProjects)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated an access group")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *AccessGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteAccessGroup(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an access group")
}

func (r *AccessGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_id:access_group_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// reconcile makes the members and projects of the access group match the planned ones,
// given the ones it currently has. Removals are done first, then additions.
func (r *AccessGroupResource) reconcile(ctx context.Context, data *AccessGroupResourceModel, haveMembers []string, haveProjects []string) diag.Diagnostics {
	var diags diag.Diagnostics

	wantMembers, d := stringSetValues(ctx, data.MemberIds)
	diags.Append(d...)

	wantProjects, d := stringSetValues(ctx, data.ProjectIds)
	diags.Append(d...)

	if diags.HasError() {
		return diags
	}

	groupId := data.Id.ValueString()
	addMembers, removeMembers := accessGroupIdsDiff(wantMembers, haveMembers)
	addProjects, removeProjects := accessGroupIdsDiff(wantProjects, haveProjects)

	for _, userId := range removeMembers {
		_, err := removeAccessGroupMember(ctx, *r.client, AccessGroupMemberInput{AccessGroupId: groupId, UserId: userId})

		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to remove user %s from access group, got error: %s", userId, err))
			return diags
		}
	}

	for _, projectId := range removeProjects {
		_, err := detachAccessGroupProject(ctx, *r.client, AccessGroupProjectInput{AccessGroupId: groupId, ProjectId: projectId})

		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to detach project %s from access group, got error: %s", projectId, err))
			return diags
		}
	}

	for _, userId := range addMembers {
		_, err := addAccessGroupMember(ctx, *r.client, AccessGroupMemberInput{AccessGroupId: groupId, UserId: userId})

		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to add user %s to access group, got error: %s", userId, err))
			return diags
		}
	}

	for _, projectId := range addProjects {
		_, err := attachAccessGroupProject(ctx, *r.client, AccessGroupProjectInput{AccessGroupId: groupId, ProjectId: projectId})

		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to attach project %s to access group, got error: %s", projectId, err))
			return diags
		}
	}

	return diags
}

// accessGroupDetails is an access group with all of its members and projects.
type accessGroupDetails struct {
	group      AccessGroup
	memberIds  []string
	projectIds []string
}

// findAccessGroup returns the access group with the given id along with every one of its
// members and projects, or nil if the workspace has no such group. All pages are followed.
func findAccessGroup(ctx context.Context, client graphql.Client, workspaceId string, id string) (*accessGroupDetails, error) {
	var group *AccessGroup

	// Cursor of the group right before the one we want, empty if it is the first one.
	// Used to address that single group when paging its members and projects.
	previousCursor := ""
	after := ""

	for group == nil {
		response, err := getAccessGroups(ctx, client, workspaceId, after)

		if err != nil {
			return nil, err
		}

		groups := response.Workspace.AccessGroups

		for _, edge := range groups.Edges {
			if edge.Node.AccessGroup.Id == id {
				found := edge.Node.AccessGroup
				group = &found
				break
			}

			previousCursor = edge.Cursor
		}

		if group != nil || !groups.PageInfo.HasNextPage {
			break
		}

		if groups.PageInfo.EndCursor == "" {
			return nil, fmt.Errorf("access groups reported another page without a cursor")
		}

		after = groups.PageInfo.EndCursor
	}

	if group == nil {
		return nil, nil
	}

	memberIds, err := listAccessGroupMemberIds(ctx, client, workspaceId, id, previousCursor)

	if err != nil {
		return nil, err
	}

	projectIds, err := listAccessGroupProjectIds(ctx, client, workspaceId, id, previousCursor)

	if err != nil {
		return nil, err
	}

	return &accessGroupDetails{group: *group, memberIds: memberIds, projectIds: projectIds}, nil
}

// listAccessGroupMemberIds returns the user ids of every member of the group, following all pages.
// groupAfter is the cursor of the group before it, which addresses the group in the workspace list.
func listAccessGroupMemberIds(ctx context.Context, client graphql.Client, workspaceId string, groupId string, groupAfter string) ([]string, error) {
	ids := make([]string, 0)
	after := ""

	for {
		response, err := getAccessGroupMembersPage(ctx, client, workspaceId, groupAfter, after)

		if err != nil {
			return nil, err
		}

		edges := response.Workspace.AccessGroups.Edges

		if len(edges) != 1 || edges[0].Node.Id != groupId {
			return nil, fmt.Errorf("access group %s changed position while reading its members, please try again", groupId)
		}

		members := edges[0].Node.Members

		for _, edge := range members.Edges {
			ids = append(ids, edge.Node.UserId)
		}

		if !members.PageInfo.HasNextPage {
			return ids, nil
		}

		if members.PageInfo.EndCursor == "" {
			return nil, fmt.Errorf("access group members reported another page without a cursor")
		}

		after = members.PageInfo.EndCursor
	}
}

// listAccessGroupProjectIds returns the ids of every project attached to the group, following all pages.
func listAccessGroupProjectIds(ctx context.Context, client graphql.Client, workspaceId string, groupId string, groupAfter string) ([]string, error) {
	ids := make([]string, 0)
	after := ""

	for {
		response, err := getAccessGroupProjectsPage(ctx, client, workspaceId, groupAfter, after)

		if err != nil {
			return nil, err
		}

		edges := response.Workspace.AccessGroups.Edges

		if len(edges) != 1 || edges[0].Node.Id != groupId {
			return nil, fmt.Errorf("access group %s changed position while reading its projects, please try again", groupId)
		}

		projects := edges[0].Node.Projects

		for _, edge := range projects.Edges {
			ids = append(ids, edge.Node.ProjectId)
		}

		if !projects.PageInfo.HasNextPage {
			return ids, nil
		}

		if projects.PageInfo.EndCursor == "" {
			return nil, fmt.Errorf("access group projects reported another page without a cursor")
		}

		after = projects.PageInfo.EndCursor
	}
}

func setAccessGroupModel(ctx context.Context, data *AccessGroupResourceModel, details *accessGroupDetails) diag.Diagnostics {
	var diags diag.Diagnostics

	members, d := types.SetValueFrom(ctx, types.StringType, details.memberIds)
	diags.Append(d...)

	projects, d := types.SetValueFrom(ctx, types.StringType, details.projectIds)
	diags.Append(d...)

	data.Id = types.StringValue(details.group.Id)
	data.WorkspaceId = types.StringValue(details.group.WorkspaceId)
	data.Name = types.StringValue(details.group.Name)
	data.Role = types.StringValue(string(details.group.Role))
	data.MemberIds = members
	data.ProjectIds = projects

	return diags
}

func stringSetValues(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	values := make([]string, 0, len(set.Elements()))
	diags := set.ElementsAs(ctx, &values, false)

	return values, diags
}

// accessGroupIdsDiff returns the ids only in want (to add) and the ones only in have (to remove).
func accessGroupIdsDiff(want []string, have []string) ([]string, []string) {
	wantMap := make(map[string]bool, len(want))
	haveMap := make(map[string]bool, len(have))

	for _, id := range want {
		wantMap[id] = true
	}

	for _, id := range have {
		haveMap[id] = true
	}

	add := make([]string, 0)
	remove := make([]string, 0)

	for _, id := range want {
		if !haveMap[id] {
			add = append(add, id)
		}
	}

	for _, id := range have {
		if !wantMap[id] {
			remove = append(remove, id)
		}
	}

	return add, remove
}
