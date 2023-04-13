package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", ""),
					resource.TestCheckResourceAttr("railway_project.test", "private", "true"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccProjectResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", ""),
					resource.TestCheckResourceAttr("railway_project.test", "private", "true"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "false"),
				),
			},
			// Update and Read testing
			{
				Config: testAccProjectResourceConfigNonDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", "nice project"),
					resource.TestCheckResourceAttr("railway_project.test", "private", "false"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccProjectResourceNonDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfigNonDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", "nice project"),
					resource.TestCheckResourceAttr("railway_project.test", "private", "false"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccProjectResourceConfigNonDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", "nice project"),
					resource.TestCheckResourceAttr("railway_project.test", "private", "false"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "true"),
				),
			},
			// Update with null values
			{
				Config: testAccProjectResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_project.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_project.test", "description", ""),
					resource.TestCheckResourceAttr("railway_project.test", "private", "true"),
					resource.TestCheckResourceAttr("railway_project.test", "has_pr_deploys", "false"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = "%s"
}
`, name)
}

func testAccProjectResourceConfigNonDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = "%s"
  description = "nice project"
  private = false
  has_pr_deploys = true
}
`, name)
}
