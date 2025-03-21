package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableCollectionResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault("one", "two", "three"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", "39da7e07-fa3a-42fd-b695-d229319f2993:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_A:VALUE_B:VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "one"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "two"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "three"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:VALUE_A:VALUE_B:VALUE_C",
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccVariableCollectionResourceConfigDefault("four", "five", "six"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", "39da7e07-fa3a-42fd-b695-d229319f2993:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_A:VALUE_B:VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_A"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "four"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "five"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "six"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:VALUE_A:VALUE_B:VALUE_C",
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccVariableCollectionResourceConfigNonDefault("four", "five", "six"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_variable_collection.test", "id", "39da7e07-fa3a-42fd-b695-d229319f2993:d0519b29-5d12-4857-a5dd-76fa7418336c:VALUE_B:VALUE_C:VALUE_D"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "service_id", "39da7e07-fa3a-42fd-b695-d229319f2993"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.name", "VALUE_B"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.0.value", "four"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.name", "VALUE_C"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.1.value", "five"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.name", "VALUE_D"),
					resource.TestCheckResourceAttr("railway_variable_collection.test", "variables.2.value", "six"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_variable_collection.test",
				ImportState:       true,
				ImportStateId:     "39da7e07-fa3a-42fd-b695-d229319f2993:staging:VALUE_B:VALUE_C:VALUE_D",
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccVariableCollectionResourceConfigDefault(valueA, valueB, valueC string) string {
	return fmt.Sprintf(`
resource "railway_variable_collection" "test" {
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "39da7e07-fa3a-42fd-b695-d229319f2993"

  variables = [
    {
      name = "VALUE_A"
      value = "%s"
    },
    {
      name = "VALUE_B"
      value = "%s"
    },
    {
      name = "VALUE_C"
      value = "%s"
    }
  ]
}
`, valueA, valueB, valueC)
}

func testAccVariableCollectionResourceConfigNonDefault(valueB, valueC, valueD string) string {
	return fmt.Sprintf(`
resource "railway_variable_collection" "test" {
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
  service_id = "39da7e07-fa3a-42fd-b695-d229319f2993"

  variables = [
    {
      name = "VALUE_B"
      value = "%s"
    },
    {
      name = "VALUE_C"
      value = "%s"
    },
    {
      name = "VALUE_D"
      value = "%s"
    }
  ]
}
`, valueB, valueC, valueD)
}
