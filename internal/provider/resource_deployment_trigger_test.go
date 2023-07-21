package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeploymentTriggerResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeploymentTriggerResourceConfigDefault("nodejs/express"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/express"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "main"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "false"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_deployment_trigger.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccDeploymentTriggerResourceConfigDefault("nodejs/express"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/express"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "main"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "false"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			// {
			// 	Config: testAccDeploymentTriggerResourceConfigNonDefault("nodejs/fastify"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/fastify"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "master"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "true"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 	),
			// },
			// ImportState testing
			{
				ResourceName:      "railway_deployment_trigger.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccDeploymentTriggerResourceNonDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeploymentTriggerResourceConfigNonDefault("nodejs/express"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/express"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "master"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "true"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_deployment_trigger.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging",
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccDeploymentTriggerResourceConfigNonDefault("nodejs/express"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/express"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "master"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "true"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update with null values
			// {
			// 	Config: testAccDeploymentTriggerResourceConfigDefault("nodejs/fastify"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("railway_deployment_trigger.test", "id"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "nodejs/fastify"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "false"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
			// 		resource.TestCheckResourceAttr("railway_deployment_trigger.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 	),
			// },
			// ImportState testing
			{
				ResourceName:      "railway_deployment_trigger.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDeploymentTriggerResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_deployment_trigger" "test" {
  repository = "%s"
  branch = "main"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, value)
}

func testAccDeploymentTriggerResourceConfigNonDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_deployment_trigger" "test" {
  repository = "%s"
  branch = "master"
  check_suites = true
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, value)
}
