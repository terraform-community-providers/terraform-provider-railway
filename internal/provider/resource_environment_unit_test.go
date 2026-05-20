package provider

import (
	"context"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestEnvironmentResourceSchemaSourceEnvironmentId(t *testing.T) {
	var resp frameworkresource.SchemaResponse

	NewEnvironmentResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["source_environment_id"].(frameworkschema.StringAttribute)
	if !ok {
		t.Fatal("source_environment_id should be a string attribute")
	}

	if !attr.Optional {
		t.Fatal("source_environment_id should be optional")
	}

	if !attr.Computed {
		t.Fatal("source_environment_id should be computed so imported resources can omit it")
	}

	if !hasStringPlanModifierDescription(attr, "destroy and recreate") {
		t.Fatal("source_environment_id should require replacement when changed")
	}

	if !hasStringPlanModifierDescription(attr, "state will not change") {
		t.Fatal("source_environment_id should preserve state for unknown plans")
	}

	if len(attr.Validators) == 0 {
		t.Fatal("source_environment_id should validate values")
	}
}

func hasStringPlanModifierDescription(attr frameworkschema.StringAttribute, want string) bool {
	for _, modifier := range attr.PlanModifiers {
		if strings.Contains(modifier.Description(context.Background()), want) {
			return true
		}
	}

	return false
}
