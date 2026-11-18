package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceEdgeRulesResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceEdgeRulesResourceConfigDefault(403),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703:d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "service_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttrSet("railway_service_edge_rules.test", "rules"),
				),
			},
			// ImportState testing, the imported rules include the ids assigned by Railway
			{
				ResourceName:            "railway_service_edge_rules.test",
				ImportState:             true,
				ImportStateIdFunc:       serviceEdgeRulesImportIdFunc,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules"},
			},
			// Update the ruleset
			{
				Config: testAccServiceEdgeRulesResourceConfigDefault(404),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_edge_rules.test", "service_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttrSet("railway_service_edge_rules.test", "rules"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccServiceEdgeRulesResourceConfigDefault(status int) string {
	return fmt.Sprintf(`
resource "railway_service_edge_rules" "test" {
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"
  rules = jsonencode({
    version = 1
    rules = [
      {
        description = "Block known bad IP ranges"
        priority = 10
        enabled = true
        if = { attr = "ipv4.src", op = "in", value = ["192.0.2.0/24"] }
        then = { action = "block", params = { status = %d } }
      }
    ]
  })
}
`, status)
}

func serviceEdgeRulesImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_service_edge_rules.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["service_id"], rawState.Primary.Attributes["environment_id"]), nil
}
