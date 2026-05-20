package provider

import (
	"fmt"
	"os"
	"testing"

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

// testAccWorkspaceIDAttr returns `workspace_id = "..."` if
// RAILWAY_TEST_WORKSPACE_ID is set, otherwise an empty string. Railway
// tokens with access to multiple workspaces require workspace_id on
// projectCreate; tokens scoped to a single workspace don't. Threading this
// through env keeps the disposable acceptance tests usable in both modes.
func testAccWorkspaceIDAttr() string {
	if id := os.Getenv("RAILWAY_TEST_WORKSPACE_ID"); id != "" {
		return fmt.Sprintf("workspace_id = %q", id)
	}
	return ""
}
