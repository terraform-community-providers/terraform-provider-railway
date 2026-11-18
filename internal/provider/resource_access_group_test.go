package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAccessGroupResourceDefault(t *testing.T) {
	members := "[data.railway_workspace_members.test.members[0].id]"
	projects := "[railway_project.test.id]"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccAccessGroupResourceConfig("test-group", "MEMBER", members, projects),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group.test", "workspace_id", "ecb63be7-63fb-47fe-95fc-1585d24e172d"),
					resource.TestCheckResourceAttr("railway_access_group.test", "name", "test-group"),
					resource.TestCheckResourceAttr("railway_access_group.test", "role", "MEMBER"),
					resource.TestCheckResourceAttr("railway_access_group.test", "member_ids.#", "1"),
					resource.TestCheckResourceAttr("railway_access_group.test", "project_ids.#", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_access_group.test",
				ImportState:       true,
				ImportStateIdFunc: accessGroupImportIdFunc,
				ImportStateVerify: true,
			},
			// Update name and role
			{
				Config: testAccAccessGroupResourceConfig("test-group-renamed", "VIEWER", members, projects),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_access_group.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_access_group.test", "name", "test-group-renamed"),
					resource.TestCheckResourceAttr("railway_access_group.test", "role", "VIEWER"),
					resource.TestCheckResourceAttr("railway_access_group.test", "member_ids.#", "1"),
					resource.TestCheckResourceAttr("railway_access_group.test", "project_ids.#", "1"),
				),
			},
			// Remove all members and projects
			{
				Config: testAccAccessGroupResourceConfig("test-group-renamed", "VIEWER", "[]", "[]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_access_group.test", "member_ids.#", "0"),
					resource.TestCheckResourceAttr("railway_access_group.test", "project_ids.#", "0"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccAccessGroupResourceConfig(name string, role string, members string, projects string) string {
	return fmt.Sprintf(`
data "railway_workspace_members" "test" {
  workspace_id = "ecb63be7-63fb-47fe-95fc-1585d24e172d"
}

resource "railway_project" "test" {
  name = "access-group-test"
  workspace_id = "ecb63be7-63fb-47fe-95fc-1585d24e172d"
}

resource "railway_access_group" "test" {
  workspace_id = "ecb63be7-63fb-47fe-95fc-1585d24e172d"
  name = "%s"
  role = "%s"
  member_ids = %s
  project_ids = %s
}
`, name, role, members, projects)
}

func accessGroupImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_access_group.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["workspace_id"], rawState.Primary.Attributes["id"]), nil
}
