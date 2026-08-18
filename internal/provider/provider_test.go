package provider

import (
	"net/http"
	"os"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

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

func testAccClient() graphql.Client {
	httpClient := http.Client{
		Transport: &authedTransport{
			token:   os.Getenv("RAILWAY_TOKEN"),
			wrapped: http.DefaultTransport,
		},
	}
	return graphql.NewClient("https://backboard.railway.app/graphql/v2?source=terraform_provider_railway", &httpClient)
}
