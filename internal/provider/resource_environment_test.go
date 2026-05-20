package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccEnvironmentResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEnvironmentResourceConfigDefault("integration"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_environment.test",
				ImportState:       true,
				ImportStateId:     "0bb01547-570d-4109-a5e8-138691f6a2d1:integration",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccEnvironmentResourceConfigDefault("integration"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration"),
					resource.TestCheckResourceAttr("railway_environment.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccEnvironmentResourceSourceEnvironmentId(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-env-source-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfigSourceEnvironmentId(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_environment.clone", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.clone", "name", "clone"),
					resource.TestCheckResourceAttrPair("railway_environment.clone", "project_id", "railway_project.test", "id"),
					resource.TestCheckResourceAttrPair("railway_environment.clone", "source_environment_id", "railway_project.test", "default_environment.id"),
					testAccCheckSharedVariableExists("railway_environment.clone", "SOURCE_ENV_TEST", "cloned"),
				),
			},
			{
				ResourceName:            "railway_environment.clone",
				ImportState:             true,
				ImportStateIdFunc:       testAccEnvironmentImportStateId("railway_project.test", "clone"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_environment_id"},
			},
		},
	})
}

func testAccEnvironmentResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_environment" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, name)
}

func testAccEnvironmentResourceConfigSourceEnvironmentId(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
  %s
}

resource "railway_shared_variable" "source" {
  name           = "SOURCE_ENV_TEST"
  value          = "cloned"
  project_id     = railway_project.test.id
  environment_id = railway_project.test.default_environment.id
}

resource "railway_environment" "clone" {
  name                  = "clone"
  project_id            = railway_project.test.id
  source_environment_id = railway_project.test.default_environment.id

  depends_on = [railway_shared_variable.source]
}
`, projectName, testAccWorkspaceIDAttr())
}

func testAccEnvironmentImportStateId(projectResourceName string, environmentName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		project, ok := state.RootModule().Resources[projectResourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found", projectResourceName)
		}

		projectID := project.Primary.ID
		if projectID == "" {
			return "", fmt.Errorf("resource %s has no ID", projectResourceName)
		}

		return fmt.Sprintf("%s:%s", projectID, environmentName), nil
	}
}

func testAccCheckSharedVariableExists(environmentResourceName string, name string, expectedValue string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		environment, ok := state.RootModule().Resources[environmentResourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", environmentResourceName)
		}

		projectID := environment.Primary.Attributes["project_id"]
		environmentID := environment.Primary.ID

		if projectID == "" {
			return fmt.Errorf("resource %s has no project_id", environmentResourceName)
		}

		if environmentID == "" {
			return fmt.Errorf("resource %s has no ID", environmentResourceName)
		}

		var data SharedVariableResourceModel

		err := getSharedVariable(context.Background(), testAccGraphQLClient(), projectID, environmentID, name, &data)
		if err != nil {
			return err
		}

		if data.Id.IsNull() || data.Id.IsUnknown() {
			return fmt.Errorf("shared variable %q was not cloned into environment %s", name, environmentID)
		}

		if data.Value.ValueString() != expectedValue {
			return fmt.Errorf("shared variable %q has value %q, want %q", name, data.Value.ValueString(), expectedValue)
		}

		return nil
	}
}

func testAccGraphQLClient() graphql.Client {
	httpClient := http.Client{
		Transport: &authedTransport{
			token:   os.Getenv(envVarName),
			wrapped: http.DefaultTransport,
		},
	}

	return graphql.NewClient("https://backboard.railway.app/graphql/v2?source=terraform_provider_railway", &httpClient)
}
