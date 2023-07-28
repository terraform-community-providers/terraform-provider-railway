package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceDomainResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceDomainResourceConfigDefault("terraform-tester"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_domain.test", "subdomain", "terraform-tester"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "domain", "terraform-tester.up.railway.app"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service_domain.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging:terraform-tester.up.railway.app",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccServiceDomainResourceConfigDefault("terraform-tester"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_domain.test", "subdomain", "terraform-tester"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "domain", "terraform-tester.up.railway.app"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// Update with default values
			{
				Config: testAccServiceDomainResourceConfigDefault("terraform-tester-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_domain.test", "subdomain", "terraform-tester-2"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "domain", "terraform-tester-2.up.railway.app"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service_domain.test",
				ImportState:       true,
				ImportStateId:     "89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging:terraform-tester-2.up.railway.app",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccServiceDomainResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_service_domain" "test" {
  subdomain = "%s"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"
}
`, name)
}
