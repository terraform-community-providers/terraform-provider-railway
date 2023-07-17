package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:8ebab3fe-1368-46a8-bedd-ec0b064c12db:REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:8ebab3fe-1368-46a8-bedd-ec0b064c12db:REDIS_URL",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:8ebab3fe-1368-46a8-bedd-ec0b064c12db:REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			{
				Config: testAccVariableResourceConfigDefault("$${{redis.REDIS_URL}}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:8ebab3fe-1368-46a8-bedd-ec0b064c12db:REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "REDIS_URL"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "${{redis.REDIS_URL}}"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "8ebab3fe-1368-46a8-bedd-ec0b064c12db"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:8ebab3fe-1368-46a8-bedd-ec0b064c12db:REDIS_URL",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccVariableResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_variable" "test" {
  name = "REDIS_URL"
  value = "%s"
  service_id = "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"
  environment_id = "8ebab3fe-1368-46a8-bedd-ec0b064c12db"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, value)
}
