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

var _ resource.Resource = &CustomDomainResource{}
var _ resource.ResourceWithImportState = &CustomDomainResource{}

func NewCustomDomainResource() resource.Resource {
	return &CustomDomainResource{}
}

type CustomDomainResource struct {
	client *graphql.Client
}

type CustomDomainResourceModel struct {
	Id             types.String `tfsdk:"id"`
	Domain         types.String `tfsdk:"domain"`
	EnvironmentId  types.String `tfsdk:"environment_id"`
	ServiceId      types.String `tfsdk:"service_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	HostLabel      types.String `tfsdk:"host_label"`
	Zone           types.String `tfsdk:"zone"`
	DNSRecordValue types.String `tfsdk:"dns_record_value"`
}

func (r *CustomDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_domain"
}

func (r *CustomDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway custom domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the custom domain.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Custom domain.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the custom domain belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the custom domain belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the custom domain belongs to.",
				Computed:            true,
			},
			"host_label": schema.StringAttribute{
				MarkdownDescription: "Host label of the custom domain.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "Zone of the custom domain.",
				Computed:            true,
			},
			"dns_record_value": schema.StringAttribute{
				MarkdownDescription: "DNS record value of the custom domain.",
				Computed:            true,
			},
		},
	}
}

func (r *CustomDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CustomDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *CustomDomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	service, err := getService(ctx, *r.client, data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	input := CustomDomainCreateInput{
		Domain:        data.Domain.ValueString(),
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		ProjectId:     service.Service.ProjectId,
	}

	response, err := createCustomDomain(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create custom domain, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a custom domain")

	domain := response.CustomDomainCreate.CustomDomain

	data.Id = types.StringValue(domain.Id)
	data.Domain = types.StringValue(domain.Domain)
	data.EnvironmentId = types.StringValue(domain.EnvironmentId)
	data.ServiceId = types.StringValue(domain.ServiceId)
	data.ProjectId = types.StringValue(service.Service.ProjectId)
	data.HostLabel = types.StringValue(domain.Status.DnsRecords[0].Hostlabel)
	data.Zone = types.StringValue(domain.Status.DnsRecords[0].Zone)
	data.DNSRecordValue = types.StringValue(domain.Status.DnsRecords[0].RequiredValue)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *CustomDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var domain CustomDomain

	response, err := listCustomDomains(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.ProjectId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list custom domains, got error: %s", err))
		return
	}

	for _, customDomain := range response.Domains.CustomDomains {
		if customDomain.CustomDomain.Domain == data.Domain.ValueString() {
			domain = customDomain.CustomDomain
			break
		}
	}

	if domain.Id == "" {
		resp.Diagnostics.AddError("Client Error", "Unable to find custom domain")
		return
	}

	data.Id = types.StringValue(domain.Id)
	data.Domain = types.StringValue(domain.Domain)
	data.EnvironmentId = types.StringValue(domain.EnvironmentId)
	data.ServiceId = types.StringValue(domain.ServiceId)
	data.HostLabel = types.StringValue(domain.Status.DnsRecords[0].Hostlabel)
	data.Zone = types.StringValue(domain.Status.DnsRecords[0].Zone)
	data.DNSRecordValue = types.StringValue(domain.Status.DnsRecords[0].RequiredValue)

	service, err := getService(ctx, *r.client, domain.ServiceId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	data.ProjectId = types.StringValue(service.Service.ProjectId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *CustomDomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *CustomDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteCustomDomain(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete custom domain, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a custom domain")
}

func (r *CustomDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_name:domain. Got: %q", req.ID),
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

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), environmentId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
}
