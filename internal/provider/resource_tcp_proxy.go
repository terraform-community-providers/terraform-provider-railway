package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TcpProxyResource{}
var _ resource.ResourceWithImportState = &TcpProxyResource{}

func NewTcpProxyResource() resource.Resource {
	return &TcpProxyResource{}
}

type TcpProxyResource struct {
	client *graphql.Client
}

type TcpProxyResourceModel struct {
	Id              types.String `tfsdk:"id"`
	ApplicationPort types.Int64  `tfsdk:"application_port"`
	EnvironmentId   types.String `tfsdk:"environment_id"`
	ServiceId       types.String `tfsdk:"service_id"`
	ProxyPort       types.Int64  `tfsdk:"proxy_port"`
	Domain          types.String `tfsdk:"domain"`
}

func (r *TcpProxyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_proxy"
}

func (r *TcpProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Railway TCP proxy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the TCP proxy.",
				Computed:            true,
			},
			"application_port": schema.Int64Attribute{
				MarkdownDescription: "Port of the application the TCP proxy points to.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtMost(65535),
				},
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the TCP proxy belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the TCP proxy belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"proxy_port": schema.Int64Attribute{
				MarkdownDescription: "Port of the TCP proxy.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain of the TCP proxy.",
				Computed:            true,
			},
		},
	}
}

func (r *TcpProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TcpProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *TcpProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := TCPProxyCreateInput{
		ApplicationPort: int(data.ApplicationPort.ValueInt64()),
		ServiceId:       data.ServiceId.ValueString(),
		EnvironmentId:   data.EnvironmentId.ValueString(),
	}

	response, err := createTcpProxy(ctx, *r.client, input)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tcp proxy, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a tcp proxy")

	proxy := response.TcpProxyCreate.TCPProxy

	data.Id = types.StringValue(proxy.Id)
	data.ApplicationPort = types.Int64Value(int64(proxy.ApplicationPort))
	data.EnvironmentId = types.StringValue(proxy.EnvironmentId)
	data.ServiceId = types.StringValue(proxy.ServiceId)
	data.ProxyPort = types.Int64Value(int64(proxy.ProxyPort))
	data.Domain = types.StringValue(proxy.Domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *TcpProxyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	response, err := getTcpProxy(ctx, *r.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tcp proxy, got error: %s", err))
		return
	}

	for _, proxy := range response.TcpProxies {
		if proxy.Id == data.Id.ValueString() {
			data.Id = types.StringValue(proxy.Id)
			data.ApplicationPort = types.Int64Value(int64(proxy.ApplicationPort))
			data.EnvironmentId = types.StringValue(proxy.EnvironmentId)
			data.ServiceId = types.StringValue(proxy.ServiceId)
			data.ProxyPort = types.Int64Value(int64(proxy.ProxyPort))
			data.Domain = types.StringValue(proxy.Domain)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *TcpProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *TcpProxyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := deleteTcpProxy(ctx, *r.client, data.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tcp proxy, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a tcp proxy")
}

func (r *TcpProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service_id:environment_id:tcp_proxy_id. Got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
}
