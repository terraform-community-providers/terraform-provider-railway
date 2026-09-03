package provider

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CustomDomainModel struct {
	Id                      types.String `tfsdk:"id"`
	Domain                  types.String `tfsdk:"domain"`
	TargetPort              types.Int64  `tfsdk:"target_port"`
	EnvironmentId           types.String `tfsdk:"environment_id"`
	ServiceId               types.String `tfsdk:"service_id"`
	ProjectId               types.String `tfsdk:"project_id"`
	HostLabel               types.String `tfsdk:"host_label"`
	Zone                    types.String `tfsdk:"zone"`
	DNSRecordValue          types.String `tfsdk:"dns_record_value"`
	VerificationHostLabel   types.String `tfsdk:"verification_host_label"`
	VerificationRecordValue types.String `tfsdk:"verification_record_value"`
}

func findCustomDomain(ctx context.Context, client graphql.Client, environmentId string, serviceId string, projectId string, domainHost string) (CustomDomain, error) {
	response, err := listCustomDomains(ctx, client, environmentId, serviceId, projectId)
	if err != nil {
		return CustomDomain{}, fmt.Errorf("unable to list custom domains: %w", err)
	}

	for _, customDomain := range response.Domains.CustomDomains {
		if customDomain.CustomDomain.Domain == domainHost {
			return customDomain.CustomDomain, nil
		}
	}

	return CustomDomain{}, fmt.Errorf("unable to find custom domain %q", domainHost)
}

func readCustomDomain(ctx context.Context, client graphql.Client, environmentId string, serviceId string, projectId string, domainHost string, data *CustomDomainModel) error {
	domain, err := findCustomDomain(ctx, client, environmentId, serviceId, projectId, domainHost)
	if err != nil {
		return err
	}

	return setCustomDomainModel(data, domain, projectId)
}

func setCustomDomainModel(data *CustomDomainModel, domain CustomDomain, projectId string) error {
	if len(domain.Status.DnsRecords) == 0 {
		return fmt.Errorf("custom domain %q has no DNS records", domain.Domain)
	}

	dnsRecord := domain.Status.DnsRecords[0]
	data.Id = types.StringValue(domain.Id)
	data.Domain = types.StringValue(domain.Domain)
	data.EnvironmentId = types.StringValue(domain.EnvironmentId)
	data.ServiceId = types.StringValue(domain.ServiceId)
	data.ProjectId = types.StringValue(projectId)
	data.HostLabel = types.StringValue(dnsRecord.Hostlabel)
	data.Zone = types.StringValue(dnsRecord.Zone)
	data.DNSRecordValue = types.StringValue(dnsRecord.RequiredValue)
	data.VerificationHostLabel = types.StringValue(domain.Status.VerificationDnsHost)
	data.VerificationRecordValue = types.StringValue(domain.Status.VerificationToken)

	if domain.TargetPort == 0 {
		data.TargetPort = types.Int64Null()
	} else {
		data.TargetPort = types.Int64Value(int64(domain.TargetPort))
	}

	return nil
}
