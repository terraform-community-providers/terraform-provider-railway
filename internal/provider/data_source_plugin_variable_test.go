package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPluginVariableDataSourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccPluginVariableDataSourceConfigDefault(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "id", "a0055c10-3f27-4d81-94d4-5825f08fe40c:d2b367de-b363-4455-b842-e60cda62a385:REDISHOST"),
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "name", "REDISHOST"),
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "value", "containers-us-west-11.railway.app"),
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "environment_id", "d2b367de-b363-4455-b842-e60cda62a385"),
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "plugin_id", "a0055c10-3f27-4d81-94d4-5825f08fe40c"),
					resource.TestCheckResourceAttr("data.railway_plugin_variable.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
				),
			},
		},
	})
}

func testAccPluginVariableDataSourceConfigDefault() string {
	return `
data "railway_plugin_variable" "test" {
  name = "REDISHOST"
  environment_id = "d2b367de-b363-4455-b842-e60cda62a385"
  plugin_id = "a0055c10-3f27-4d81-94d4-5825f08fe40c"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`
}
