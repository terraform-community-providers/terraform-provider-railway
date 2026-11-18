package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ServiceEdgeRulesResource{}
var _ resource.ResourceWithImportState = &ServiceEdgeRulesResource{}

func NewServiceEdgeRulesResource() resource.Resource {
	return &ServiceEdgeRulesResource{}
}

type ServiceEdgeRulesResource struct {
	client *graphql.Client
}

type ServiceEdgeRulesResourceModel struct {
	Id            types.String `tfsdk:"id"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	Rules         types.String `tfsdk:"rules"`
}

func (r *ServiceEdgeRulesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_edge_rules"
}

func (r *ServiceEdgeRulesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service edge rules. The `rules` JSON replaces the whole ruleset of the service, and destroying the resource clears it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the edge rules, in the format `service_id:environment_id`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the edge rules belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the edge rules belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"rules": schema.StringAttribute{
				MarkdownDescription: "JSON ruleset, e.g. built with `jsonencode`. Rules without an `id` get one assigned by Railway, which is ignored when detecting drift.",
				Required:            true,
			},
		},
	}
}

func (r *ServiceEdgeRulesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceEdgeRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created service edge rules")

	data.Id = types.StringValue(edgeRulesId(data))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getServiceEdgeRules(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service edge rules, got error: %s", err))
		return
	}

	remote := response.ServiceInstance.EdgeConfig.EdgeRules

	// Rules were cleared outside of Terraform.
	if remote == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(edgeRulesId(data))

	// Keep the configured JSON unless the rules really changed, otherwise the
	// ids Railway assigns would show up as a diff on every plan.
	// ponytail: ids are ignored for drift, so a changed id alone is not detected.
	var current map[string]interface{}

	if data.Rules.IsNull() || json.Unmarshal([]byte(data.Rules.ValueString()), &current) != nil ||
		edgeRulesWithoutIds(current) != edgeRulesWithoutIds(remote) {
		encoded, err := json.Marshal(remote)

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to encode service edge rules, got error: %s", err))
			return
		}

		data.Rules = types.StringValue(string(encoded))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, data, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated service edge rules")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceEdgeRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceEdgeRulesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// A null ruleset clears the edge rules.
	_, err := updateServiceEdgeRules(ctx, *r.client, UpdateServiceEdgeRulesInput{
		EnvironmentId: data.EnvironmentId.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
		EdgeRules:     nil,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service edge rules, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted service edge rules")
}

func (r *ServiceEdgeRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
}

// apply replaces the whole ruleset of the service with the planned JSON.
func (r *ServiceEdgeRulesResource) apply(ctx context.Context, data *ServiceEdgeRulesResourceModel, diagnostics *diag.Diagnostics) {
	var rules map[string]interface{}

	if err := json.Unmarshal([]byte(data.Rules.ValueString()), &rules); err != nil || rules == nil {
		diagnostics.AddError("Invalid Rules", fmt.Sprintf("`rules` must be a JSON object, got error: %v", err))
		return
	}

	_, err := updateServiceEdgeRules(ctx, *r.client, UpdateServiceEdgeRulesInput{
		EnvironmentId: data.EnvironmentId.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
		EdgeRules:     rules,
	})

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service edge rules, got error: %s", err))
	}
}

func edgeRulesId(data *ServiceEdgeRulesResourceModel) string {
	return data.ServiceId.ValueString() + ":" + data.EnvironmentId.ValueString()
}

// edgeRulesWithoutIds encodes the ruleset with the per-rule ids removed, so two
// rulesets can be compared regardless of ids assigned by Railway.
func edgeRulesWithoutIds(rules map[string]interface{}) string {
	stripped := map[string]interface{}{}

	for key, value := range rules {
		stripped[key] = value
	}

	if list, ok := rules["rules"].([]interface{}); ok {
		withoutIds := make([]interface{}, len(list))

		for i, item := range list {
			rule, ok := item.(map[string]interface{})

			if !ok {
				withoutIds[i] = item
				continue
			}

			copied := map[string]interface{}{}

			for key, value := range rule {
				if key != "id" {
					copied[key] = value
				}
			}

			withoutIds[i] = copied
		}

		stripped["rules"] = withoutIds
	}

	encoded, _ := json.Marshal(stripped)

	return string(encoded)
}
