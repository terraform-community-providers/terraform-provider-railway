package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectDataSourceByName(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-ds-project-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.railway_project.test", "id", "railway_project.test", "id"),
					resource.TestCheckResourceAttrPair("data.railway_project.test", "name", "railway_project.test", "name"),
					resource.TestCheckResourceAttrPair("data.railway_project.test", "workspace_id", "railway_project.test", "workspace_id"),
					resource.TestCheckResourceAttrPair("data.railway_project.test", "default_environment.id", "railway_project.test", "default_environment.id"),
					resource.TestCheckResourceAttrPair("data.railway_project.test", "default_environment.name", "railway_project.test", "default_environment.name"),
				),
			},
		},
	})
}

func testAccProjectDataSourceConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
  %s
}

data "railway_project" "test" {
  name         = railway_project.test.name
  workspace_id = railway_project.test.workspace_id

  depends_on = [railway_project.test]
}
`, projectName, testAccWorkspaceIDAttr())
}
