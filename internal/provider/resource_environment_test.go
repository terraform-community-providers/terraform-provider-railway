package provider

import (
	"context"
	"fmt"
	"testing"

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

func TestAccEnvironmentResourceClone(t *testing.T) {
	projectName := "tf-acc-env-clone-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfigClone(projectName, "integration-clone"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test_clone", "id", uuidRegex()),
					resource.TestMatchResourceAttr("railway_environment.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_environment.test", "name", "integration-clone"),
					resource.TestCheckResourceAttrPair("railway_environment.test", "project_id", "railway_project.test_clone", "id"),
					resource.TestCheckResourceAttrPair("railway_environment.test", "source_environment_id", "railway_project.test_clone", "default_environment.id"),
					resource.TestCheckResourceAttr("railway_environment.test", "skip_initial_deploys", "true"),
					testAccCheckClonedServiceWithoutDeployment,
				),
			},
			{
				ResourceName:            "railway_environment.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccEnvironmentCloneImportStateId,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_environment_id", "skip_initial_deploys"},
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

func testAccEnvironmentResourceConfigClone(projectName string, name string) string {
	return fmt.Sprintf(`
resource "railway_project" "test_clone" {
  name = "%s"
}

resource "railway_service" "clone_source" {
  name         = "clone-source"
  project_id   = railway_project.test_clone.id
  source_image = "traefik/whoami:v1.10"
}

resource "railway_environment" "test" {
  name                  = "%s"
  project_id            = railway_project.test_clone.id
  source_environment_id = railway_project.test_clone.default_environment.id
  skip_initial_deploys  = true

  depends_on = [railway_service.clone_source]
}
`, projectName, name)
}

func testAccCheckClonedServiceWithoutDeployment(state *terraform.State) error {
	environment := state.RootModule().Resources["railway_environment.test"]
	service := state.RootModule().Resources["railway_service.clone_source"]

	response, err := getServiceInstance(
		context.Background(),
		testAccClient(),
		environment.Primary.ID,
		service.Primary.ID,
	)
	if err != nil {
		return fmt.Errorf("read cloned service instance: %w", err)
	}

	instance := response.ServiceInstance
	if instance.Source == nil || instance.Source.Image == nil ||
		*instance.Source.Image != "traefik/whoami:v1.10" {
		return fmt.Errorf("cloned service source image was not preserved")
	}
	if instance.LatestDeployment.Meta != nil {
		return fmt.Errorf("cloned service unexpectedly received an initial deployment")
	}
	return nil
}

func testAccEnvironmentCloneImportStateId(state *terraform.State) (string, error) {
	resourceState := state.RootModule().Resources["railway_environment.test"]
	return fmt.Sprintf(
		"%s:%s",
		resourceState.Primary.Attributes["project_id"],
		resourceState.Primary.Attributes["name"],
	), nil
}
