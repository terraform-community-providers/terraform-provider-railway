package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWorkspaceMembersDataSourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "railway_workspace_members" "test" {
  workspace_id = "ecb63be7-63fb-47fe-95fc-1585d24e172d"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_workspace_members.test", "id", "ecb63be7-63fb-47fe-95fc-1585d24e172d"),
					resource.TestCheckResourceAttr("data.railway_workspace_members.test", "workspace_id", "ecb63be7-63fb-47fe-95fc-1585d24e172d"),
					resource.TestCheckResourceAttrSet("data.railway_workspace_members.test", "members.0.id"),
					resource.TestCheckResourceAttrSet("data.railway_workspace_members.test", "members.0.email"),
					resource.TestCheckResourceAttrSet("data.railway_workspace_members.test", "members.0.role"),
				),
			},
		},
	})
}
