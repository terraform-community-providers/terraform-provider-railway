package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentDataSourceByName(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-ds-env-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentDataSourceConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.railway_environment.test", "id", "railway_environment.test", "id"),
					resource.TestCheckResourceAttrPair("data.railway_environment.test", "name", "railway_environment.test", "name"),
					resource.TestCheckResourceAttrPair("data.railway_environment.test", "project_id", "railway_project.test", "id"),
				),
			},
		},
	})
}

func testAccEnvironmentDataSourceConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
  %s
}

resource "railway_environment" "test" {
  name       = "staging"
  project_id = railway_project.test.id
}

data "railway_environment" "test" {
  name       = railway_environment.test.name
  project_id = railway_project.test.id

  depends_on = [railway_environment.test]
}
`, projectName, testAccWorkspaceIDAttr())
}
