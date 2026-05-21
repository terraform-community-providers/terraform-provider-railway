package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccServiceInstanceLifecycle creates a disposable project + redis
// service, then declares a railway_service_instance binding that overrides
// num_replicas + cron_schedule. Updates num_replicas in place and asserts
// Railway reflects the change.
func TestAccServiceInstanceLifecycle(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-svc-inst-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceInstanceConfig(projectName, "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service_instance.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "1"),
					// Don't pin `builder` — Railway's default has shifted
					// between NIXPACKS and RAILPACK over time; assert the
					// field is populated to one of the valid values.
					resource.TestMatchResourceAttr("railway_service_instance.test", "builder", regexp.MustCompile(`^(NIXPACKS|RAILPACK|PAKETO|HEROKU)$`)),
					resource.TestCheckResourceAttrPair("railway_service_instance.test", "service_id", "railway_service.test", "id"),
					resource.TestCheckResourceAttrPair("railway_service_instance.test", "environment_id", "railway_project.test", "default_environment.id"),
				),
			},
			{
				Config: testAccServiceInstanceConfig(projectName, "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "num_replicas", "2"),
				),
			},
		},
	})
}

func testAccServiceInstanceConfig(projectName, numReplicas string) string {
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

resource "railway_service_instance" "test" {
  service_id     = railway_service.test.id
  environment_id = railway_project.test.default_environment.id
  num_replicas   = %s
}
`, projectName, testAccWorkspaceIDAttr(), numReplicas)
}
