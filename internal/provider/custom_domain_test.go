package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSetCustomDomainModel(t *testing.T) {
	domain := CustomDomain{
		Id:            "domain-id",
		Domain:        "api.example.com",
		TargetPort:    3000,
		EnvironmentId: "environment-id",
		ServiceId:     "service-id",
		Status: CustomDomainStatus{
			DnsRecords: []CustomDomainStatusDnsRecordsDNSRecords{
				{
					Fqdn:          "api.example.com",
					Hostlabel:     "api",
					Purpose:       DNSRecordPurposeDnsRecordPurposeTrafficRoute,
					RecordType:    DNSRecordTypeDnsRecordTypeCname,
					RequiredValue: "target.railway.app",
					Zone:          "example.com",
				},
			},
			VerificationDnsHost: stringPointer("_railway-verify.api"),
			VerificationToken:   stringPointer("verification-token"),
		},
	}
	model := &CustomDomainModel{}

	err := setCustomDomainModel(model, domain, "project-id")
	if err != nil {
		t.Fatalf("setCustomDomainModel() error = %v", err)
	}

	if model.Id != types.StringValue("domain-id") {
		t.Errorf("Id = %v", model.Id)
	}
	if model.Domain != types.StringValue("api.example.com") {
		t.Errorf("Domain = %v", model.Domain)
	}
	if model.TargetPort != types.Int64Value(3000) {
		t.Errorf("TargetPort = %v", model.TargetPort)
	}
	if model.EnvironmentId != types.StringValue("environment-id") {
		t.Errorf("EnvironmentId = %v", model.EnvironmentId)
	}
	if model.ServiceId != types.StringValue("service-id") {
		t.Errorf("ServiceId = %v", model.ServiceId)
	}
	if model.ProjectId != types.StringValue("project-id") {
		t.Errorf("ProjectId = %v", model.ProjectId)
	}
	if model.HostLabel != types.StringValue("api") {
		t.Errorf("HostLabel = %v", model.HostLabel)
	}
	if model.Zone != types.StringValue("example.com") {
		t.Errorf("Zone = %v", model.Zone)
	}
	if model.DNSRecordValue != types.StringValue("target.railway.app") {
		t.Errorf("DNSRecordValue = %v", model.DNSRecordValue)
	}
	if model.VerificationHostLabel != types.StringValue("_railway-verify.api") {
		t.Errorf("VerificationHostLabel = %v", model.VerificationHostLabel)
	}
	if model.VerificationRecordValue != types.StringValue("verification-token") {
		t.Errorf("VerificationRecordValue = %v", model.VerificationRecordValue)
	}
}

func TestSetCustomDomainModelSelectsTrafficRouteCNAME(t *testing.T) {
	verification := CustomDomainStatusDnsRecordsDNSRecords{Fqdn: "verify.example.com", Purpose: DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge, RecordType: DNSRecordTypeDnsRecordTypeTxt}
	route := CustomDomainStatusDnsRecordsDNSRecords{Fqdn: "api.example.com", Hostlabel: "api", RequiredValue: "target.railway.app", Zone: "example.com", Purpose: DNSRecordPurposeDnsRecordPurposeTrafficRoute, RecordType: DNSRecordTypeDnsRecordTypeCname}
	otherTraffic := CustomDomainStatusDnsRecordsDNSRecords{Fqdn: "other.example.com", Purpose: DNSRecordPurposeDnsRecordPurposeTrafficRoute, RecordType: DNSRecordTypeDnsRecordTypeA}
	for _, records := range [][]CustomDomainStatusDnsRecordsDNSRecords{
		{verification, route, otherTraffic},
		{route, otherTraffic, verification},
	} {
		domain := CustomDomain{Status: CustomDomainStatus{DnsRecords: records}}
		model := &CustomDomainModel{}

		err := setCustomDomainModel(model, domain, "project-id")
		if err != nil {
			t.Fatalf("setCustomDomainModel() error = %v", err)
		}

		if !model.TargetPort.IsNull() {
			t.Errorf("TargetPort = %v, want null", model.TargetPort)
		}
		if model.HostLabel != types.StringValue("api") {
			t.Errorf("HostLabel = %v, want route host label", model.HostLabel)
		}
		if !model.VerificationHostLabel.IsNull() || !model.VerificationRecordValue.IsNull() {
			t.Errorf("verification fields = %v/%v, want null", model.VerificationHostLabel, model.VerificationRecordValue)
		}
	}
}

func TestSetCustomDomainModelRejectsInvalidDNSRecords(t *testing.T) {
	route := CustomDomainStatusDnsRecordsDNSRecords{Purpose: DNSRecordPurposeDnsRecordPurposeTrafficRoute, RecordType: DNSRecordTypeDnsRecordTypeCname}
	for _, test := range []struct {
		name    string
		records []CustomDomainStatusDnsRecordsDNSRecords
		want    string
	}{
		{name: "none", records: nil, want: "has 0"},
		{name: "multiple", records: []CustomDomainStatusDnsRecordsDNSRecords{route, route}, want: "has 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := setCustomDomainModel(&CustomDomainModel{}, CustomDomain{Domain: "api.example.com", Status: CustomDomainStatus{DnsRecords: test.records}}, "project-id")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("setCustomDomainModel() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestSetCustomDomainModelPreservesIndependentVerificationNulls(t *testing.T) {
	route := CustomDomainStatusDnsRecordsDNSRecords{Purpose: DNSRecordPurposeDnsRecordPurposeTrafficRoute, RecordType: DNSRecordTypeDnsRecordTypeCname}
	for _, test := range []struct {
		status    CustomDomainStatus
		wantHost  types.String
		wantValue types.String
	}{
		{status: CustomDomainStatus{DnsRecords: []CustomDomainStatusDnsRecordsDNSRecords{route}, VerificationDnsHost: stringPointer("host")}, wantHost: types.StringValue("host"), wantValue: types.StringNull()},
		{status: CustomDomainStatus{DnsRecords: []CustomDomainStatusDnsRecordsDNSRecords{route}, VerificationToken: stringPointer("token")}, wantHost: types.StringNull(), wantValue: types.StringValue("token")},
	} {
		model := &CustomDomainModel{}
		if err := setCustomDomainModel(model, CustomDomain{Domain: "api.example.com", Status: test.status}, "project-id"); err != nil {
			t.Fatalf("setCustomDomainModel() error = %v", err)
		}
		if model.VerificationHostLabel != test.wantHost || model.VerificationRecordValue != test.wantValue {
			t.Errorf("verification fields = %v/%v, want %v/%v", model.VerificationHostLabel, model.VerificationRecordValue, test.wantHost, test.wantValue)
		}
	}
}

func stringPointer(value string) *string { return &value }
