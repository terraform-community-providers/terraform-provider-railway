package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSharedVariableResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSharedVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:d0519b29-5d12-4857-a5dd-76fa7418336c:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     "0bb01547-570d-4109-a5e8-138691f6a2d1:staging:API_KEY",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccSharedVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:d0519b29-5d12-4857-a5dd-76fa7418336c:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			{
				Config: testAccSharedVariableResourceConfigDefault("nice"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:d0519b29-5d12-4857-a5dd-76fa7418336c:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "nice"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     "0bb01547-570d-4109-a5e8-138691f6a2d1:staging:API_KEY",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccSharedVariableResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_shared_variable" "test" {
  name = "API_KEY"
  value = "%s"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, value)
}
