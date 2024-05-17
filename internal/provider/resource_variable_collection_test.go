package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableCollectionResourceDefault(t *testing.T) {

	environmentName := "staging"
	environmentId := "d0519b29-5d12-4857-a5dd-76fa7418336c"
	serviceId := "89fa0236-2b1b-4a8c-b12d-ae3634b30d97"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault(environmentId, serviceId, "one", "two", "three"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", fmt.Sprintf("%s:%s:VALUE_A:VALUE_B:VALUE_C", serviceId, environmentId)),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", environmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_A", "one"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_B", "two"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_C", "three"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s:%s:VALUE_A:VALUE_B:VALUE_C", serviceId, environmentName),
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccVariableCollectionResourceConfigDefault(environmentId, serviceId, "one", "two", "three"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", fmt.Sprintf("%s:%s:VALUE_A:VALUE_B:VALUE_C", serviceId, environmentId)),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", environmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_A", "one"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_B", "two"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_C", "three"),
				),
			},
			// Update and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault(environmentId, serviceId, "four", "five", "six"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", fmt.Sprintf("%s:%s:VALUE_A:VALUE_B:VALUE_C", serviceId, environmentId)),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", environmentId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", serviceId),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_A", "four"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_B", "five"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.VALUE_C", "six"),
				),
			},
		},
	})
}

func testAccVariableCollectionResourceConfigDefault(environmentId, serviceId, valueA, valueB, valueC string) string {
	return fmt.Sprintf(`
resource "railway_variable_collection" "test" {
  environment_id = "%s"
  service_id = "%s"
  variables = {
    VALUE_A = "%s"
    VALUE_B = "%s"
    VALUE_C = "%s"
  }
}
`, environmentId, serviceId, valueA, valueB, valueC)
}
