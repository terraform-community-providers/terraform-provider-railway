package provider

import (
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
					Hostlabel:     "api",
					RequiredValue: "target.railway.app",
					Zone:          "example.com",
				},
			},
			VerificationDnsHost: "_railway-verify.api",
			VerificationToken:   "verification-token",
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

func TestSetCustomDomainModelWithoutTargetPort(t *testing.T) {
	domain := CustomDomain{
		Status: CustomDomainStatus{
			DnsRecords: []CustomDomainStatusDnsRecordsDNSRecords{{}},
		},
	}
	model := &CustomDomainModel{}

	err := setCustomDomainModel(model, domain, "project-id")
	if err != nil {
		t.Fatalf("setCustomDomainModel() error = %v", err)
	}

	if !model.TargetPort.IsNull() {
		t.Errorf("TargetPort = %v, want null", model.TargetPort)
	}
}

func TestSetCustomDomainModelWithoutDNSRecord(t *testing.T) {
	model := &CustomDomainModel{}

	err := setCustomDomainModel(model, CustomDomain{}, "project-id")
	if err == nil {
		t.Fatal("setCustomDomainModel() error = nil, want error")
	}
}
