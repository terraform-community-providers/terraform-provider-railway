package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVariableResource(t *testing.T) {
	projectName := "tf-acc-variable-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVariableResourceConfig(
					projectName,
					"1234567890",
					"first-secret",
					1,
					false,
					"readable-value",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.readable", "value", "1234567890"),
					resource.TestCheckNoResourceAttr("railway_variable.sealed", "value"),
					resource.TestCheckNoResourceAttr("railway_variable.sealed", "value_wo"),
					resource.TestCheckResourceAttr("railway_variable.sealed", "value_wo_version", "1"),
					resource.TestCheckResourceAttr("railway_variable.transition", "value", "readable-value"),
				),
			},
			{
				ResourceName:      "railway_variable.readable",
				ImportState:       true,
				ImportStateIdFunc: testAccVariableImportStateId("railway_variable.readable"),
				ImportStateVerify: true,
			},
			{
				Config: testAccVariableResourceConfig(
					projectName,
					"$${{redis.REDIS_URL}}",
					"second-secret",
					2,
					true,
					"sealed-value",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("railway_variable.transition", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable.readable", "value", "${{redis.REDIS_URL}}"),
					resource.TestCheckNoResourceAttr("railway_variable.sealed", "value_wo"),
					resource.TestCheckResourceAttr("railway_variable.sealed", "value_wo_version", "2"),
					resource.TestCheckNoResourceAttr("railway_variable.transition", "value"),
					resource.TestCheckNoResourceAttr("railway_variable.transition", "value_wo"),
					resource.TestCheckResourceAttr("railway_variable.transition", "value_wo_version", "1"),
				),
			},
			{
				ResourceName:            "railway_variable.sealed",
				ImportState:             true,
				ImportStateIdFunc:       testAccVariableImportStateId("railway_variable.sealed"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value_wo", "value_wo_version"},
			},
			{
				Config: testAccVariableResourceConfig(
					projectName,
					"$${{redis.REDIS_URL}}",
					"second-secret",
					2,
					false,
					"new-readable-value",
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("railway_variable.transition", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("railway_variable.transition", "value", "new-readable-value"),
			},
		},
	})
}

func testAccVariableResourceConfig(
	projectName string,
	readableValue string,
	sealedValue string,
	sealedVersion int64,
	transitionSealed bool,
	transitionValue string,
) string {
	transitionConfig := fmt.Sprintf(`
resource "railway_variable" "transition" {
  name           = "TERRAFORM_VARIABLE_TRANSITION_TEST"
  value          = "%s"
  environment_id = railway_project.test_variable.default_environment.id
  service_id     = railway_service.test_variable.id
}
`, transitionValue)
	if transitionSealed {
		transitionConfig = fmt.Sprintf(`
resource "railway_variable" "transition" {
  name             = "TERRAFORM_VARIABLE_TRANSITION_TEST"
  value_wo         = "%s"
  value_wo_version = 1
  environment_id   = railway_project.test_variable.default_environment.id
  service_id       = railway_service.test_variable.id
}
`, transitionValue)
	}

	return fmt.Sprintf(`
resource "railway_project" "test_variable" {
  name = "%s"
}

resource "railway_service" "test_variable" {
  name       = "terraform-provider-variable-test"
  project_id = railway_project.test_variable.id
}

resource "railway_variable" "readable" {
  name           = "REDIS_URL"
  value          = "%s"
  environment_id = railway_project.test_variable.default_environment.id
  service_id     = railway_service.test_variable.id
}

resource "railway_variable" "sealed" {
  name             = "TERRAFORM_SEALED_TEST"
  value_wo         = "%s"
  value_wo_version = %d
  environment_id   = railway_project.test_variable.default_environment.id
  service_id       = railway_service.test_variable.id
}

%s
`, projectName, readableValue, sealedValue, sealedVersion, transitionConfig)
}

func testAccVariableImportStateId(resourceName string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		resourceState := state.RootModule().Resources[resourceName]
		return fmt.Sprintf(
			"%s:production:%s",
			resourceState.Primary.Attributes["service_id"],
			resourceState.Primary.Attributes["name"],
		), nil
	}
}
