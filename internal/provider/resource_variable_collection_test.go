package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableCollectionResourceDefault(t *testing.T) {

	environmentName := "production"
	environmentId := "3050e612-087b-40dd-bdf2-4b52a8290900"
	serviceId := "8faa58d7-8a06-4b9a-8e9a-4257c91c8b15"

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
