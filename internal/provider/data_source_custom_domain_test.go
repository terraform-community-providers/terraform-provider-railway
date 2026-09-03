package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testCustomDomainProjectId     = "0bb01547-570d-4109-a5e8-138691f6a2d1"
	testCustomDomainEnvironmentId = "d0519b29-5d12-4857-a5dd-76fa7418336c"
	testCustomDomainServiceId     = "39da7e07-fa3a-42fd-b695-d229319f2993"
)

type customDomainGraphQLRequest struct {
	OperationName string            `json:"operationName"`
	Variables     map[string]string `json:"variables"`
}

func TestCustomDomainDataSourceRead(t *testing.T) {
	requests := make(chan customDomainGraphQLRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request customDomainGraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode GraphQL request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"domains":{"customDomains":[{"id":"other-id","domain":"other.example.com","targetPort":8080,"status":{"dnsRecords":[{"hostlabel":"other","requiredValue":"other.railway.app","zone":"example.com"}],"verificationDnsHost":"_verify.other","verificationToken":"other-token"},"environmentId":"other-environment","serviceId":"other-service"},{"id":"domain-id","domain":"api.example.com","targetPort":3000,"status":{"dnsRecords":[{"hostlabel":"api","requiredValue":"target.railway.app","zone":"example.com"}],"verificationDnsHost":"_railway-verify.api","verificationToken":"verification-token"},"environmentId":"` + testCustomDomainEnvironmentId + `","serviceId":"` + testCustomDomainServiceId + `"}]}}}`))
	}))
	t.Cleanup(server.Close)

	response := readCustomDomainDataSource(t, server, "api.example.com")
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}

	request := <-requests
	if request.OperationName != "listCustomDomains" {
		t.Errorf("operationName = %q, want listCustomDomains", request.OperationName)
	}
	wantVariables := map[string]string{
		"projectId":     testCustomDomainProjectId,
		"environmentId": testCustomDomainEnvironmentId,
		"serviceId":     testCustomDomainServiceId,
	}
	if !reflect.DeepEqual(request.Variables, wantVariables) {
		t.Errorf("variables = %#v, want %#v", request.Variables, wantVariables)
	}

	var state CustomDomainModel
	if diagnostics := response.State.Get(context.Background(), &state); diagnostics.HasError() {
		t.Fatalf("get state diagnostics = %v", diagnostics)
	}

	assertCustomDomainState(t, state)
}

func TestCustomDomainDataSourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"domains":{"customDomains":[]}}}`))
	}))
	t.Cleanup(server.Close)

	response := readCustomDomainDataSource(t, server, "missing.example.com")
	if !response.Diagnostics.HasError() {
		t.Fatal("Read() diagnostics has no error, want not-found error")
	}
}

func readCustomDomainDataSource(t *testing.T, server *httptest.Server, domain string) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	dataSource := NewCustomDomainDataSource().(*CustomDomainDataSource)
	client := graphql.NewClient(server.URL, server.Client())
	dataSource.client = &client

	var schemaResponse datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
	terraformType := schemaResponse.Schema.Type().TerraformType(ctx)
	configValue := tftypes.NewValue(terraformType, map[string]tftypes.Value{
		"id":                        tftypes.NewValue(tftypes.String, nil),
		"domain":                    tftypes.NewValue(tftypes.String, domain),
		"target_port":               tftypes.NewValue(tftypes.Number, nil),
		"environment_id":            tftypes.NewValue(tftypes.String, testCustomDomainEnvironmentId),
		"service_id":                tftypes.NewValue(tftypes.String, testCustomDomainServiceId),
		"project_id":                tftypes.NewValue(tftypes.String, testCustomDomainProjectId),
		"host_label":                tftypes.NewValue(tftypes.String, nil),
		"zone":                      tftypes.NewValue(tftypes.String, nil),
		"dns_record_value":          tftypes.NewValue(tftypes.String, nil),
		"verification_host_label":   tftypes.NewValue(tftypes.String, nil),
		"verification_record_value": tftypes.NewValue(tftypes.String, nil),
	})

	request := datasource.ReadRequest{
		Config: tfsdk.Config{Raw: configValue, Schema: schemaResponse.Schema},
	}
	response := datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResponse.Schema},
	}
	dataSource.Read(ctx, request, &response)

	return response
}

func assertCustomDomainState(t *testing.T, state CustomDomainModel) {
	t.Helper()
	want := CustomDomainModel{
		Id:                      types.StringValue("domain-id"),
		Domain:                  types.StringValue("api.example.com"),
		TargetPort:              types.Int64Value(3000),
		EnvironmentId:           types.StringValue(testCustomDomainEnvironmentId),
		ServiceId:               types.StringValue(testCustomDomainServiceId),
		ProjectId:               types.StringValue(testCustomDomainProjectId),
		HostLabel:               types.StringValue("api"),
		Zone:                    types.StringValue("example.com"),
		DNSRecordValue:          types.StringValue("target.railway.app"),
		VerificationHostLabel:   types.StringValue("_railway-verify.api"),
		VerificationRecordValue: types.StringValue("verification-token"),
	}

	if !reflect.DeepEqual(state, want) {
		t.Errorf("state = %#v, want %#v", state, want)
	}
}
