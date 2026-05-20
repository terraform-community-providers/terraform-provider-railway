package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDeploymentTriggerLifecycle exercises the full lifecycle of a
// railway_deployment_trigger against a disposable project: create with a
// branch, update the branch (and check_suites) in place, and confirm Railway
// returns the new values.
func TestAccDeploymentTriggerLifecycle(t *testing.T) {
	projectName := fmt.Sprintf("tf-acc-dep-trigger-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	branchA := fmt.Sprintf("tf-acc-branch-a-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	branchB := fmt.Sprintf("tf-acc-branch-b-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentTriggerConfig(projectName, branchA, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_deployment_trigger.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", branchA),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "source_provider", "github"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "repository", "techinterviewcoaching/techinterview.coach"),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "false"),
					resource.TestCheckResourceAttrPair("railway_deployment_trigger.test", "service_id", "railway_service.test", "id"),
					resource.TestCheckResourceAttrPair("railway_deployment_trigger.test", "environment_id", "railway_project.test", "default_environment.id"),
				),
			},
			// In-place update: change branch + check_suites; the existing
			// trigger should be mutated, not recreated.
			{
				Config: testAccDeploymentTriggerConfig(projectName, branchB, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "branch", branchB),
					resource.TestCheckResourceAttr("railway_deployment_trigger.test", "check_suites", "true"),
				),
			},
		},
	})
}

func testAccDeploymentTriggerConfig(projectName, branch, checkSuites string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
  %s
}

resource "railway_service" "test" {
  name         = "trigger-anchor"
  project_id   = railway_project.test.id
  source_image = "redis:7-alpine"
}

resource "railway_deployment_trigger" "test" {
  project_id     = railway_project.test.id
  environment_id = railway_project.test.default_environment.id
  service_id     = railway_service.test.id
  source_provider = "github"
  repository     = "techinterviewcoaching/techinterview.coach"
  branch         = %q
  check_suites   = %s
}
`, projectName, testAccWorkspaceIDAttr(), branch, checkSuites)
}
