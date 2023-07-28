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

var _ resource.Resource = &ServiceDomainResource{}
var _ resource.ResourceWithImportState = &ServiceDomainResource{}

func NewServiceDomainResource() resource.Resource {
	return &ServiceDomainResource{}
}

type ServiceDomainResource struct {
	client *graphql.Client
}

type ServiceDomainResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Subdomain     types.String `tfsdk:"subdomain"`
	EnvironmentId types.String `tfsdk:"environment_id"`
	ServiceId     types.String `tfsdk:"service_id"`
	ProjectId     types.String `tfsdk:"project_id"`
	Suffix        types.String `tfsdk:"suffix"`
	Domain        types.String `tfsdk:"domain"`
}

func (r *ServiceDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_domain"
}

func (r *ServiceDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway service domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service domain.",
				Computed:            true,
			},
			"subdomain": schema.StringAttribute{
				MarkdownDescription: "Subdomain of the service domain.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the service domain belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the service domain belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the service domain belongs to.",
				Computed:            true,
			},
			"suffix": schema.StringAttribute{
				MarkdownDescription: "Suffix of the service domain.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Full domain of the service domain.",
				Computed:            true,
			},
		},
	}
}

func (r *ServiceDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ServiceDomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ServiceDomainCreateInput{
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	response, err := createServiceDomain(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service domain, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a service domain")

	domain := response.ServiceDomainCreate.ServiceDomain
	domainName := data.Subdomain.ValueString() + "." + domain.Suffix

	updateInput := ServiceDomainUpdateInput{
		Domain:        domainName,
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	updateResponse, err := updateServiceDomain(ctx, *r.client, updateInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service domain, got error: %s", err))
		return
	}

	if !updateResponse.ServiceDomainUpdate {
		resp.Diagnostics.AddError("Client Error", "Unable to update service domain, got false as response")
		return
	}

	tflog.Trace(ctx, "updated a service domain")

	service, err := getService(ctx, *r.client, domain.ServiceId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}

	err = getAndBuildServiceDomain(ctx, *r.client, service.Service.ProjectId, domain.EnvironmentId, domain.ServiceId, domainName, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service domain, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ServiceDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := getAndBuildServiceDomain(ctx, *r.client, data.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.Domain.ValueString(), data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service domain, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ServiceDomainResourceModel
	var state *ServiceDomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	domainName := data.Subdomain.ValueString() + "." + state.Suffix.ValueString()

	updateInput := ServiceDomainUpdateInput{
		Domain:        domainName,
		ServiceId:     data.ServiceId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
	}

	response, err := updateServiceDomain(ctx, *r.client, updateInput)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service domain, got error: %s", err))
		return
	}

	if !response.ServiceDomainUpdate {
		resp.Diagnostics.AddError("Client Error", "Unable to update service domain, got false as response")
		return
	}

	tflog.Trace(ctx, "updated a service domain")

	err = getAndBuildServiceDomain(ctx, *r.client, state.ProjectId.ValueString(), data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), domainName, data)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service domain, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ServiceDomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteServiceDomain(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service domain, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a service domain")
}

func (r *ServiceDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func findServiceDomain(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, domain string) (*ServiceDomain, error) {
	response, err := listServiceDomains(ctx, client, environmentId, serviceId, projectId)

	if err != nil {
		return nil, err
	}

	for _, serviceDomain := range response.Domains.ServiceDomains {
		if serviceDomain.ServiceDomain.Domain == domain {
			return &serviceDomain.ServiceDomain, nil
		}
	}

	return nil, fmt.Errorf("service domain doesn't exist")
}

func getAndBuildServiceDomain(ctx context.Context, client graphql.Client, projectId string, environmentId string, serviceId string, domain string, data *ServiceDomainResourceModel) error {
	serviceDomain, err := findServiceDomain(ctx, client, projectId, environmentId, serviceId, domain)

	if err != nil {
		return err
	}

	data.Id = types.StringValue(serviceDomain.Id)
	data.EnvironmentId = types.StringValue(serviceDomain.EnvironmentId)
	data.ServiceId = types.StringValue(serviceDomain.ServiceId)
	data.Suffix = types.StringValue(serviceDomain.Suffix)
	data.Domain = types.StringValue(serviceDomain.Domain)

	data.Subdomain = types.StringValue(serviceDomain.Domain[:len(serviceDomain.Domain)-len(serviceDomain.Suffix)-1])
	data.ProjectId = types.StringValue(projectId)

	return nil
}
