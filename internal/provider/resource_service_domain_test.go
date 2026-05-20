package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
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
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "domain", "terraform-tester.up.railway.app"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service_domain.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:terraform-tester.up.railway.app",
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccServiceDomainResourceConfigDefault("terraform-tester"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_domain.test", "subdomain", "terraform-tester"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
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
					resource.TestCheckResourceAttr("railway_service_domain.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "domain", "terraform-tester-2.up.railway.app"),
					resource.TestCheckResourceAttr("railway_service_domain.test", "suffix", "up.railway.app"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service_domain.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:terraform-tester-2.up.railway.app",
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
  service_id = "39da7e07-fa3a-42fd-b695-d229319f2993"
}
`, name)
}

// TestAccServiceDomainResourceTargetPort exercises target_port end-to-end against
// a disposable Railway project: set, change, clear, then verify Railway's
// readback matches state. Guards the clear semantics fixed in cc4e4db: removing
// target_port from config must actually clear the value on Railway.
func TestAccServiceDomainResourceTargetPort(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-sd-port-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	subdomain := fmt.Sprintf("tf-acc-sd-port-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with target_port set
			{
				Config: testAccServiceDomainResourceConfigTargetPort(projectName, subdomain, "8000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_domain.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_domain.test", "subdomain", subdomain),
					resource.TestCheckResourceAttr("railway_service_domain.test", "target_port", "8000"),
				),
			},
			// Update to a different target_port
			{
				Config: testAccServiceDomainResourceConfigTargetPort(projectName, subdomain, "8080"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_domain.test", "target_port", "8080"),
				),
			},
			// Remove target_port entirely — must actually clear server-side
			// (this is the regression fixed in cc4e4db).
			{
				Config: testAccServiceDomainResourceConfigNoTargetPort(projectName, subdomain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_service_domain.test", "target_port"),
				),
			},
			// Set target_port again to make sure clear didn't leave Railway in a
			// stuck state.
			{
				Config: testAccServiceDomainResourceConfigTargetPort(projectName, subdomain, "9000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_domain.test", "target_port", "9000"),
				),
			},
		},
	})
}

func testAccServiceDomainResourceConfigTargetPort(projectName, subdomain, targetPort string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = "%s"
  %s
}

resource "railway_service" "test" {
  name         = "redis"
  project_id   = railway_project.test.id
  source_image = "redis:7-alpine"
}

resource "railway_service_domain" "test" {
  subdomain      = "%s"
  target_port    = %s
  environment_id = railway_project.test.default_environment.id
  service_id     = railway_service.test.id
}
`, projectName, testAccWorkspaceIDAttr(), subdomain, targetPort)
}

func testAccServiceDomainResourceConfigNoTargetPort(projectName, subdomain string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = "%s"
  %s
}

resource "railway_service" "test" {
  name         = "redis"
  project_id   = railway_project.test.id
  source_image = "redis:7-alpine"
}

resource "railway_service_domain" "test" {
  subdomain      = "%s"
  environment_id = railway_project.test.default_environment.id
  service_id     = railway_service.test.id
}
`, projectName, testAccWorkspaceIDAttr(), subdomain)
}
