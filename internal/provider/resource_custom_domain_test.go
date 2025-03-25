package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomDomainResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCustomDomainResourceConfigDefault("terraform.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_custom_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "terraform.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "terraform"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "dns_record_value"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_custom_domain.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:terraform.example.com",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccCustomDomainResourceConfigDefault("terraform.example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_custom_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "domain", "terraform.example.com"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "host_label", "terraform"),
					resource.TestCheckResourceAttr("railway_custom_domain.test", "zone", "example.com"),
					resource.TestCheckResourceAttrSet("railway_custom_domain.test", "dns_record_value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccCustomDomainResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_custom_domain" "test" {
  domain = "%s"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "39da7e07-fa3a-42fd-b695-d229319f2993"
}
`, name)
}
