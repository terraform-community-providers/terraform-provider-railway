package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ datasource.DataSource = &CustomDomainDataSource{}

func NewCustomDomainDataSource() datasource.DataSource {
	return &CustomDomainDataSource{}
}

type CustomDomainDataSource struct {
	client *graphql.Client
}

func (d *CustomDomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_domain"
}

func (d *CustomDomainDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing Railway custom domain and its required DNS records.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the custom domain.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Custom domain to look up.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"target_port": schema.Int64Attribute{
				MarkdownDescription: "Target port of the service for the custom domain.",
				Computed:            true,
			},
			"environment_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the environment the custom domain belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service the custom domain belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the project the custom domain belongs to.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(uuidRegex(), "must be an id"),
				},
			},
			"host_label": schema.StringAttribute{
				MarkdownDescription: "CNAME host label of the custom domain.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "DNS zone of the custom domain.",
				Computed:            true,
			},
			"dns_record_value": schema.StringAttribute{
				MarkdownDescription: "CNAME record value of the custom domain.",
				Computed:            true,
			},
			"verification_host_label": schema.StringAttribute{
				MarkdownDescription: "TXT host label for custom domain verification.",
				Computed:            true,
			},
			"verification_record_value": schema.StringAttribute{
				MarkdownDescription: "TXT record value for custom domain verification.",
				Computed:            true,
			},
		},
	}
}

func (d *CustomDomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CustomDomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CustomDomainModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := readCustomDomain(ctx, *d.client, data.EnvironmentId.ValueString(), data.ServiceId.ValueString(), data.ProjectId.ValueString(), data.Domain.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read custom domain, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
