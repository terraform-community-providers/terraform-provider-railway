package provider

import (
	"context"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestDeploymentTriggerResourceSchemaShape(t *testing.T) {
	var resp frameworkresource.SchemaResponse
	NewDeploymentTriggerResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	required := []string{"project_id", "environment_id", "service_id", "source_provider", "repository", "branch"}
	for _, name := range required {
		attr, ok := resp.Schema.Attributes[name].(frameworkschema.StringAttribute)
		if !ok {
			t.Fatalf("%s should be a string attribute", name)
		}
		if !attr.Required {
			t.Fatalf("%s should be required", name)
		}
	}

	// branch and repository must be mutable (no RequiresReplace).
	for _, name := range []string{"branch", "repository"} {
		attr := resp.Schema.Attributes[name].(frameworkschema.StringAttribute)
		for _, pm := range attr.PlanModifiers {
			// crude type-name check: RequiresReplace plan modifiers expose
			// a Description starting with "If the value of this attribute"
			d := pm.Description(context.Background())
			if d != "" && (d == "If the value of this attribute changes, Terraform will destroy and recreate the resource." || (len(d) > 8 && d[:8] == "If the v")) {
				t.Fatalf("%s should not have RequiresReplace; updates must be in-place", name)
			}
		}
	}

	// root_directory is optional+computed because Railway does not expose it
	// on read. UseStateForUnknown keeps state stable across refreshes.
	rootDir, ok := resp.Schema.Attributes["root_directory"].(frameworkschema.StringAttribute)
	if !ok {
		t.Fatal("root_directory should be a string attribute")
	}
	if !rootDir.Optional || !rootDir.Computed {
		t.Fatal("root_directory should be Optional+Computed (Railway omits it on read)")
	}
}
