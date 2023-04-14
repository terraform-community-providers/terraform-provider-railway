package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPluginResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPluginResourceConfigDefault("auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_plugin.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_plugin.test", "name", "auth"),
					resource.TestCheckResourceAttr("railway_plugin.test", "type", "postgresql"),
					resource.TestCheckResourceAttr("railway_plugin.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_plugin.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccPluginResourceConfigDefault("auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_plugin.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_plugin.test", "name", "auth"),
					resource.TestCheckResourceAttr("railway_plugin.test", "type", "postgresql"),
					resource.TestCheckResourceAttr("railway_plugin.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// Update and Read testing
			{
				Config: testAccPluginResourceConfigDefault("nue-auth"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_plugin.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_plugin.test", "name", "nue-auth"),
					resource.TestCheckResourceAttr("railway_plugin.test", "type", "postgresql"),
					resource.TestCheckResourceAttr("railway_plugin.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_plugin.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPluginResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_plugin" "test" {
  name = "%s"
  type = "postgresql"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, name)
}
