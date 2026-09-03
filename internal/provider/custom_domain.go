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
	var routeRecords []CustomDomainStatusDnsRecordsDNSRecords
	for _, record := range domain.Status.DnsRecords {
		if record.Purpose == DNSRecordPurposeDnsRecordPurposeTrafficRoute && record.RecordType == DNSRecordTypeDnsRecordTypeCname {
			routeRecords = append(routeRecords, record)
		}
	}
	if len(routeRecords) != 1 {
		return fmt.Errorf("custom domain %q has %d TRAFFIC_ROUTE CNAME records; expected exactly one", domain.Domain, len(routeRecords))
	}
	dnsRecord := routeRecords[0]
	data.Id = types.StringValue(domain.Id)
	data.Domain = types.StringValue(domain.Domain)
	data.EnvironmentId = types.StringValue(domain.EnvironmentId)
	data.ServiceId = types.StringValue(domain.ServiceId)
	data.ProjectId = types.StringValue(projectId)
	data.HostLabel = types.StringValue(dnsRecord.Hostlabel)
	data.Zone = types.StringValue(dnsRecord.Zone)
	data.DNSRecordValue = types.StringValue(dnsRecord.RequiredValue)
	if domain.Status.VerificationDnsHost == nil {
		data.VerificationHostLabel = types.StringNull()
	} else {
		data.VerificationHostLabel = types.StringValue(*domain.Status.VerificationDnsHost)
	}
	if domain.Status.VerificationToken == nil {
		data.VerificationRecordValue = types.StringNull()
	} else {
		data.VerificationRecordValue = types.StringValue(*domain.Status.VerificationToken)
	}

	if domain.TargetPort == 0 {
		data.TargetPort = types.Int64Null()
	} else {
		data.TargetPort = types.Int64Value(int64(domain.TargetPort))
	}

	return nil
}
