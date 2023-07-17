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
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:8ebab3fe-1368-46a8-bedd-ec0b064c12db:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     "0bb01547-570d-4109-a5e8-138691f6a2d1:8ebab3fe-1368-46a8-bedd-ec0b064c12db:API_KEY",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccSharedVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:8ebab3fe-1368-46a8-bedd-ec0b064c12db:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			{
				Config: testAccSharedVariableResourceConfigDefault("nice"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_shared_variable.test", "id", "0bb01547-570d-4109-a5e8-138691f6a2d1:8ebab3fe-1368-46a8-bedd-ec0b064c12db:API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "name", "API_KEY"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "value", "nice"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_shared_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_shared_variable.test",
				ImportState:       true,
				ImportStateId:     "0bb01547-570d-4109-a5e8-138691f6a2d1:8ebab3fe-1368-46a8-bedd-ec0b064c12db:API_KEY",
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
  environment_id = "8ebab3fe-1368-46a8-bedd-ec0b064c12db"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, value)
}
