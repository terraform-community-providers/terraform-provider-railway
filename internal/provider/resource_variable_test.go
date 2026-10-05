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
					resource.TestCheckResourceAttr("railway_variable.test", "id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:staging:VALUE_A",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccVariableResourceConfigDefault("1234567890"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "1234567890"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			{
				Config: testAccVariableResourceConfigDefault("$${{redis.REDIS_URL}}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.test", "id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable.test", "value", "${{redis.REDIS_URL}}"),
					resource.TestCheckResourceAttr("railway_variable.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable.test", "service_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttr("railway_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable.test",
				ImportState:       true,
				ImportStateId:     "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:staging:VALUE_A",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccVariableResourceConfigDefault(value string) string {
	return fmt.Sprintf(`
resource "railway_variable" "test" {
  name = "VALUE_A"
  value = "%s"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"
}
`, value)
}
