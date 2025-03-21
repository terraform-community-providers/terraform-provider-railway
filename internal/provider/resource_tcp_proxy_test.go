package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTcpProxyResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTcpProxyResourceConfigDefault(6379),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_tcp_proxy.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "application_port", "6379"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "proxy_port"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "domain"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_tcp_proxy.test",
				ImportState:       true,
				ImportStateIdFunc: tcpProxyImportIdFunc,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccTcpProxyResourceConfigDefault(6379),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_tcp_proxy.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "application_port", "6379"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_tcp_proxy.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "proxy_port"),
					resource.TestCheckResourceAttrSet("railway_tcp_proxy.test", "domain"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTcpProxyResourceConfigDefault(port int) string {
	return fmt.Sprintf(`
resource "railway_tcp_proxy" "test" {
  application_port = "%d"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "39da7e07-fa3a-42fd-b695-d229319f2993"
}
`, port)
}

func tcpProxyImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_tcp_proxy.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s:%s", rawState.Primary.Attributes["service_id"], rawState.Primary.Attributes["environment_id"], rawState.Primary.Attributes["id"]), nil
}
