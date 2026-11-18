package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccProjectTokenResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectTokenResourceConfigDefault("test-token"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project_token.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project_token.test", "name", "test-token"),
					resource.TestCheckResourceAttr("railway_project_token.test", "project_id", "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"),
					resource.TestCheckResourceAttr("railway_project_token.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttrSet("railway_project_token.test", "token"),
				),
			},
			// ImportState testing, the token value is not recoverable
			{
				ResourceName:            "railway_project_token.test",
				ImportState:             true,
				ImportStateIdFunc:       projectTokenImportIdFunc,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update with a new name replaces the token
			{
				Config: testAccProjectTokenResourceConfigDefault("test-token-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project_token.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project_token.test", "name", "test-token-renamed"),
					resource.TestCheckResourceAttrSet("railway_project_token.test", "token"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectTokenResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_project_token" "test" {
  name = "%s"
  project_id = "e2bcf8de-5f7c-45d3-9f83-fad52a6b0703"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}
`, name)
}

func projectTokenImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_project_token.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["project_id"], rawState.Primary.Attributes["id"]), nil
}
