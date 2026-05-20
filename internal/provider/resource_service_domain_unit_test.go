package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestServiceDomainResourceSchemaTargetPort(t *testing.T) {
	var resp frameworkresource.SchemaResponse

	NewServiceDomainResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["target_port"].(frameworkschema.Int64Attribute)
	if !ok {
		t.Fatal("target_port should be an int64 attribute")
	}

	if !attr.Optional {
		t.Fatal("target_port should be optional")
	}

	if attr.Required {
		t.Fatal("target_port should not be required")
	}

	if len(attr.Validators) == 0 {
		t.Fatal("target_port should declare at least one validator (port range)")
	}
}

// TestServiceDomainUpdateInputClearsTargetPort guards the clear semantics:
// when the user removes target_port from config, the update mutation must
// send explicit JSON `null` (not omit the field), otherwise Railway keeps
// the previous port and Terraform diverges from config.
func TestServiceDomainUpdateInputClearsTargetPort(t *testing.T) {
	input := ServiceDomainUpdateInput{
		ServiceDomainId: "sd-id",
		Domain:          "example.up.railway.app",
		ServiceId:       "svc-id",
		EnvironmentId:   "env-id",
		// TargetPort intentionally left nil — user removed it from config.
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(encoded), `"targetPort":null`) {
		t.Fatalf("expected explicit null for targetPort to clear server-side value, got: %s", encoded)
	}
}

// TestServiceDomainCreateInputOmitsTargetPort guards the create path: when
// target_port is unset, the create mutation should omit the field so Railway
// applies its own default rather than receiving an explicit null.
func TestServiceDomainCreateInputOmitsTargetPort(t *testing.T) {
	input := ServiceDomainCreateInput{
		ServiceId:     "svc-id",
		EnvironmentId: "env-id",
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(encoded), "targetPort") {
		t.Fatalf("expected create payload to omit targetPort when unset, got: %s", encoded)
	}
}
