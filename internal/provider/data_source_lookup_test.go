package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testLookupWorkspaceId   = "924d0c92-5342-4af6-bc17-92f2f91a0f8c"
	testLookupProjectId     = "0bb01547-570d-4109-a5e8-138691f6a2d1"
	testLookupEnvironmentId = "d0519b29-5d12-4857-a5dd-76fa7418336c"
	testLookupServiceId     = "39da7e07-fa3a-42fd-b695-d229319f2993"
)

type lookupGraphQLRequest struct {
	OperationName string            `json:"operationName"`
	Variables     map[string]string `json:"variables"`
}

func TestProjectDataSourceReadById(t *testing.T) {
	request := lookupRequest(t, `{"data":{"project":{"id":"`+testLookupProjectId+`","name":"charming","workspaceId":"`+testLookupWorkspaceId+`"}}}`)
	response := readLookupDataSource(t, NewProjectDataSource(), request.server, identityConfig(t, NewProjectDataSource(), testLookupProjectId))
	assertNoReadErrors(t, response)
	assertLookupRequest(t, <-request.requests, "readProject", map[string]string{"id": testLookupProjectId})

	var state ProjectDataSourceModel
	getLookupState(t, response, &state)
	want := ProjectDataSourceModel{Id: types.StringValue(testLookupProjectId), Name: types.StringValue("charming"), WorkspaceId: types.StringValue(testLookupWorkspaceId)}
	if !reflect.DeepEqual(state, want) {
		t.Errorf("state = %#v, want %#v", state, want)
	}
}

func TestEnvironmentDataSourceReadById(t *testing.T) {
	request := lookupRequest(t, `{"data":{"environment":{"id":"`+testLookupEnvironmentId+`","name":"production","projectId":"`+testLookupProjectId+`"}}}`)
	response := readLookupDataSource(t, NewEnvironmentDataSource(), request.server, identityConfig(t, NewEnvironmentDataSource(), testLookupEnvironmentId))
	assertNoReadErrors(t, response)
	assertLookupRequest(t, <-request.requests, "readEnvironment", map[string]string{"id": testLookupEnvironmentId})

	var state EnvironmentDataSourceModel
	getLookupState(t, response, &state)
	want := EnvironmentDataSourceModel{Id: types.StringValue(testLookupEnvironmentId), Name: types.StringValue("production"), ProjectId: types.StringValue(testLookupProjectId)}
	if !reflect.DeepEqual(state, want) {
		t.Errorf("state = %#v, want %#v", state, want)
	}
}

func TestServiceDataSourceReadById(t *testing.T) {
	request := lookupRequest(t, `{"data":{"service":{"id":"`+testLookupServiceId+`","name":"server","projectId":"`+testLookupProjectId+`"}}}`)
	response := readLookupDataSource(t, NewServiceDataSource(), request.server, identityConfig(t, NewServiceDataSource(), testLookupServiceId))
	assertNoReadErrors(t, response)
	assertLookupRequest(t, <-request.requests, "readService", map[string]string{"id": testLookupServiceId})

	var state ServiceDataSourceModel
	getLookupState(t, response, &state)
	want := ServiceDataSourceModel{Id: types.StringValue(testLookupServiceId), Name: types.StringValue("server"), ProjectId: types.StringValue(testLookupProjectId)}
	if !reflect.DeepEqual(state, want) {
		t.Errorf("state = %#v, want %#v", state, want)
	}
}

func TestIdentityDataSourcesRejectMissingOrMismatchedIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		new    func() datasource.DataSource
		id     string
		entity string
	}{
		{name: "project", new: NewProjectDataSource, id: testLookupProjectId, entity: "project"},
		{name: "environment", new: NewEnvironmentDataSource, id: testLookupEnvironmentId, entity: "environment"},
		{name: "service", new: NewServiceDataSource, id: testLookupServiceId, entity: "service"},
	} {
		t.Run(test.name+" missing", func(t *testing.T) {
			server := lookupRequest(t, `{"errors":[{"message":"not found"}]}`)
			response := readLookupDataSource(t, test.new(), server.server, identityConfig(t, test.new(), test.id))
			if !response.Diagnostics.HasError() {
				t.Fatal("Read() diagnostics has no error, want absence error")
			}
		})
		t.Run(test.name+" mismatched", func(t *testing.T) {
			server := lookupRequest(t, fmt.Sprintf(`{"data":{"%s":{"id":"wrong-id","name":"wrong","projectId":"%s"}}}`, test.entity, testLookupProjectId))
			response := readLookupDataSource(t, test.new(), server.server, identityConfig(t, test.new(), test.id))
			if !response.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(response.Diagnostics), "received") {
				t.Fatalf("Read() diagnostics = %v, want identity mismatch", response.Diagnostics)
			}
		})
	}
}

func TestServiceDomainDataSourceSelectsExactDomain(t *testing.T) {
	request := lookupRequest(t, `{"data":{"domains":{"serviceDomains":[{"id":"other-id","domain":"other.up.railway.app","suffix":"up.railway.app","environmentId":"`+testLookupEnvironmentId+`","serviceId":"`+testLookupServiceId+`"},{"id":"domain-id","domain":"server.up.railway.app","suffix":"up.railway.app","environmentId":"`+testLookupEnvironmentId+`","serviceId":"`+testLookupServiceId+`"}]}}}`)
	response := readLookupDataSource(t, NewServiceDomainDataSource(), request.server, serviceDomainConfig("server.up.railway.app"))
	assertNoReadErrors(t, response)

	var state ServiceDomainDataSourceModel
	getLookupState(t, response, &state)
	if state.Id != types.StringValue("domain-id") {
		t.Errorf("Id = %v, want domain-id", state.Id)
	}
}

func TestServiceDomainDataSourceRequiresOneExactMatch(t *testing.T) {
	for _, test := range []struct {
		name     string
		response string
		want     string
	}{
		{name: "zero", response: `{"data":{"domains":{"serviceDomains":[{"id":"other","domain":"other.up.railway.app","suffix":"up.railway.app","environmentId":"` + testLookupEnvironmentId + `","serviceId":"` + testLookupServiceId + `"}]}}}`, want: "found 0"},
		{name: "duplicate", response: `{"data":{"domains":{"serviceDomains":[{"id":"one","domain":"server.up.railway.app","suffix":"up.railway.app","environmentId":"` + testLookupEnvironmentId + `","serviceId":"` + testLookupServiceId + `"},{"id":"two","domain":"server.up.railway.app","suffix":"up.railway.app","environmentId":"` + testLookupEnvironmentId + `","serviceId":"` + testLookupServiceId + `"}]}}}`, want: "found 2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := lookupRequest(t, test.response)
			response := readLookupDataSource(t, NewServiceDomainDataSource(), server.server, serviceDomainConfig("server.up.railway.app"))
			if !response.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(response.Diagnostics), test.want) {
				t.Fatalf("Read() diagnostics = %v, want %q", response.Diagnostics, test.want)
			}
		})
	}
}

type lookupTestServer struct {
	server   *httptest.Server
	requests chan lookupGraphQLRequest
}

func lookupRequest(t *testing.T, response string) lookupTestServer {
	t.Helper()
	requests := make(chan lookupGraphQLRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request lookupGraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode GraphQL request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return lookupTestServer{server: server, requests: requests}
}

func identityConfig(t *testing.T, source datasource.DataSource, id string) map[string]tftypes.Value {
	t.Helper()
	var schemaResponse datasource.SchemaResponse
	source.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	config := make(map[string]tftypes.Value, len(schemaResponse.Schema.Attributes))
	for name := range schemaResponse.Schema.Attributes {
		value := interface{}(nil)
		if name == "id" {
			value = id
		}
		config[name] = tftypes.NewValue(tftypes.String, value)
	}
	return config
}

func serviceDomainConfig(domain string) map[string]tftypes.Value {
	return map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, nil), "domain": tftypes.NewValue(tftypes.String, domain), "suffix": tftypes.NewValue(tftypes.String, nil),
		"project_id": tftypes.NewValue(tftypes.String, testLookupProjectId), "environment_id": tftypes.NewValue(tftypes.String, testLookupEnvironmentId), "service_id": tftypes.NewValue(tftypes.String, testLookupServiceId),
	}
}

func readLookupDataSource(t *testing.T, source datasource.DataSource, server *httptest.Server, config map[string]tftypes.Value) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	client := graphql.NewClient(server.URL, server.Client())
	var configureResponse datasource.ConfigureResponse
	source.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: &client}, &configureResponse)
	if configureResponse.Diagnostics.HasError() {
		t.Fatalf("Configure() diagnostics = %v", configureResponse.Diagnostics)
	}
	var schemaResponse datasource.SchemaResponse
	source.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
	request := datasource.ReadRequest{Config: tfsdk.Config{Raw: tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(ctx), config), Schema: schemaResponse.Schema}}
	response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	source.Read(ctx, request, &response)
	return response
}

func assertNoReadErrors(t *testing.T, response datasource.ReadResponse) {
	t.Helper()
	if response.Diagnostics.HasError() {
		t.Fatalf("Read() diagnostics = %v", response.Diagnostics)
	}
}

func assertLookupRequest(t *testing.T, got lookupGraphQLRequest, operationName string, variables map[string]string) {
	t.Helper()
	if got.OperationName != operationName || !reflect.DeepEqual(got.Variables, variables) {
		t.Errorf("request = %#v, want operation %q variables %#v", got, operationName, variables)
	}
}

func getLookupState(t *testing.T, response datasource.ReadResponse, target interface{}) {
	t.Helper()
	if diagnostics := response.State.Get(context.Background(), target); diagnostics.HasError() {
		t.Fatalf("get state diagnostics = %v", diagnostics)
	}
}
