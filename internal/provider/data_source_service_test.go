package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceDataSourceByName(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-ds-service-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceDataSourceConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.railway_service.test", "id", "railway_service.test", "id"),
					resource.TestCheckResourceAttrPair("data.railway_service.test", "name", "railway_service.test", "name"),
					resource.TestCheckResourceAttrPair("data.railway_service.test", "project_id", "railway_project.test", "id"),
				),
			},
		},
	})
}

func testAccServiceDataSourceConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
  %s
}

resource "railway_service" "test" {
  name         = "redis"
  project_id   = railway_project.test.id
  source_image = "redis:7-alpine"
}

data "railway_service" "test" {
  name       = railway_service.test.name
  project_id = railway_project.test.id

  depends_on = [railway_service.test]
}
`, projectName, testAccWorkspaceIDAttr())
}
