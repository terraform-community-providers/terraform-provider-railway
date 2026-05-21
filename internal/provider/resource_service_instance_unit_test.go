package provider

import (
	"context"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestServiceInstanceResourceSchemaRequired(t *testing.T) {
	var resp frameworkresource.SchemaResponse
	NewServiceInstanceResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	for _, name := range []string{"service_id", "environment_id"} {
		attr, ok := resp.Schema.Attributes[name].(frameworkschema.StringAttribute)
		if !ok {
			t.Fatalf("%s should be a string attribute", name)
		}
		if !attr.Required {
			t.Fatalf("%s should be required", name)
		}
		if len(attr.PlanModifiers) == 0 {
			t.Fatalf("%s should declare a RequiresReplace plan modifier", name)
		}
	}
}

func TestServiceInstanceResourceSchemaOptionalEnvScoped(t *testing.T) {
	var resp frameworkresource.SchemaResponse
	NewServiceInstanceResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	// All env-scoped config fields should be Optional+Computed so they can
	// be omitted from config and read back from Railway, but updates are
	// honored. UseStateForUnknown keeps refresh stable for string fields.
	for _, name := range []string{
		"build_command", "builder", "cron_schedule", "healthcheck_path",
		"region", "root_directory", "start_command", "source_image", "source_repo",
	} {
		attr, ok := resp.Schema.Attributes[name].(frameworkschema.StringAttribute)
		if !ok {
			t.Fatalf("%s should be a string attribute", name)
		}
		if !attr.Optional || !attr.Computed {
			t.Fatalf("%s should be Optional+Computed", name)
		}
	}

	for _, name := range []string{"healthcheck_timeout", "num_replicas"} {
		attr, ok := resp.Schema.Attributes[name].(frameworkschema.Int64Attribute)
		if !ok {
			t.Fatalf("%s should be an int64 attribute", name)
		}
		if !attr.Optional || !attr.Computed {
			t.Fatalf("%s should be Optional+Computed", name)
		}
	}
}

func TestServiceInstanceResourceSchemaConflictingSource(t *testing.T) {
	var resp frameworkresource.SchemaResponse
	NewServiceInstanceResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)

	srcImage, _ := resp.Schema.Attributes["source_image"].(frameworkschema.StringAttribute)
	if len(srcImage.Validators) == 0 {
		t.Fatal("source_image should declare a ConflictsWith validator against source_repo")
	}
}
