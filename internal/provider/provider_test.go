package provider

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRailwayProviderDataSources(t *testing.T) {
	provider := New("test")().(*RailwayProvider)
	var names []string
	for _, newDataSource := range provider.DataSources(context.Background()) {
		var response datasource.MetadataResponse
		newDataSource().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "railway"}, &response)
		names = append(names, response.TypeName)
	}

	want := []string{
		"railway_project",
		"railway_environment",
		"railway_service",
		"railway_custom_domain",
		"railway_service_domain",
	}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("data source names = %#v, want %#v", names, want)
	}
}

func TestRailwayProviderAuthSchemaIsSensitive(t *testing.T) {
	var response provider.SchemaResponse
	New("test")().Schema(context.Background(), provider.SchemaRequest{}, &response)
	for _, name := range []string{"token", "project_token"} {
		attribute, ok := response.Schema.Attributes[name].(providerschema.StringAttribute)
		if !ok || !attribute.Sensitive {
			t.Errorf("attribute %q = %#v, want sensitive string", name, response.Schema.Attributes[name])
		}
	}
}

func TestRailwayProviderConfigureSupportsOneAuthKind(t *testing.T) {
	for _, test := range []struct {
		name         string
		token        interface{}
		projectToken interface{}
		tokenEnv     string
		projectEnv   string
	}{
		{name: "configured bearer", token: "bearer-secret"},
		{name: "bearer environment", tokenEnv: "bearer-secret"},
		{name: "configured project", projectToken: "project-secret"},
		{name: "project environment", projectEnv: "project-secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(tokenEnvVar, test.tokenEnv)
			t.Setenv(projectTokenEnvVar, test.projectEnv)
			response := configureTestProvider(t, test.token, test.projectToken)
			if response.Diagnostics.HasError() {
				t.Fatalf("Configure() diagnostics = %v", response.Diagnostics)
			}
			if _, ok := response.DataSourceData.(*graphql.Client); !ok {
				t.Errorf("DataSourceData = %T, want *graphql.Client", response.DataSourceData)
			}
		})
	}
	if railwayGraphQLEndpoint != "https://backboard.railway.com/graphql/v2?source=terraform_provider_railway" {
		t.Errorf("endpoint = %q", railwayGraphQLEndpoint)
	}
}

func TestRailwayProviderConfigureRejectsMissingOrConflictingAuthWithoutLeakingSecrets(t *testing.T) {
	for _, test := range []struct {
		name         string
		token        interface{}
		projectToken interface{}
		want         string
	}{
		{name: "missing", want: "Required token"},
		{name: "conflicting", token: "bearer-secret", projectToken: "project-secret", want: "Exactly one"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(tokenEnvVar, "")
			t.Setenv(projectTokenEnvVar, "")
			response := configureTestProvider(t, test.token, test.projectToken)
			diagnostics := fmt.Sprint(response.Diagnostics)
			if !response.Diagnostics.HasError() || !strings.Contains(diagnostics, test.want) {
				t.Fatalf("Configure() diagnostics = %v, want %q", response.Diagnostics, test.want)
			}
			if strings.Contains(diagnostics, "bearer-secret") || strings.Contains(diagnostics, "project-secret") {
				t.Fatalf("Configure() diagnostics leaked a secret: %s", diagnostics)
			}
		})
	}
}

func TestRailwayProviderConfigureDefersUnknownAuth(t *testing.T) {
	for _, test := range []struct {
		name         string
		token        interface{}
		projectToken interface{}
	}{
		{name: "bearer", token: tftypes.UnknownValue},
		{name: "project", projectToken: tftypes.UnknownValue},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(tokenEnvVar, "")
			t.Setenv(projectTokenEnvVar, "")
			response := configureTestProvider(t, test.token, test.projectToken)
			if response.Diagnostics.HasError() {
				t.Fatalf("Configure() diagnostics = %v", response.Diagnostics)
			}
			if response.DataSourceData != nil || response.ResourceData != nil {
				t.Fatalf("Configure() data = %T/%T, want nil while auth is unknown", response.DataSourceData, response.ResourceData)
			}
		})
	}
}

func configureTestProvider(t *testing.T, token interface{}, projectToken interface{}) provider.ConfigureResponse {
	t.Helper()
	ctx := context.Background()
	providerUnderTest := New("test")()
	var schemaResponse provider.SchemaResponse
	providerUnderTest.Schema(ctx, provider.SchemaRequest{}, &schemaResponse)
	config := tfsdk.Config{
		Raw: tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
			"token":         tftypes.NewValue(tftypes.String, token),
			"project_token": tftypes.NewValue(tftypes.String, projectToken),
		}),
		Schema: schemaResponse.Schema,
	}
	var response provider.ConfigureResponse
	providerUnderTest.Configure(ctx, provider.ConfigureRequest{Config: config}, &response)
	return response
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"railway": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RAILWAY_TOKEN"); v == "" {
		t.Fatal("RAILWAY_TOKEN must be set for acceptance tests")
	}
}
