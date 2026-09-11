package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestBucketInstanceFromConfig(t *testing.T) {
	config := map[string]interface{}{
		"buckets": map[string]interface{}{
			"live":    map[string]interface{}{"region": "iad"},
			"deleted": map[string]interface{}{"isDeleted": true},
		},
	}

	if instance, ok := bucketInstanceFromConfig(config, "live"); !ok || instance["region"] != "iad" {
		t.Fatalf("expected live bucket with region iad, got %v %v", instance, ok)
	}

	if _, ok := bucketInstanceFromConfig(config, "deleted"); ok {
		t.Fatal("expected deleted bucket to be absent")
	}

	if _, ok := bucketInstanceFromConfig(config, "missing"); ok {
		t.Fatal("expected missing bucket to be absent")
	}

	if _, ok := bucketInstanceFromConfig(map[string]interface{}{}, "live"); ok {
		t.Fatal("expected bucket to be absent when config has no buckets")
	}
}

func TestAccBucketResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccBucketResourceConfigDefault("tf-uploads"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_bucket.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "tf-uploads"),
					resource.TestCheckResourceAttr("railway_bucket.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_bucket.test", "environment_id", "d0519b29-5d12-4857-a5dd-76fa7418336c"),
					resource.TestCheckResourceAttr("railway_bucket.test", "region", "sjc"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_bucket.test",
				ImportState:       true,
				ImportStateIdFunc: bucketImportIdFunc,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccBucketResourceConfigDefault("tf-assets"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_bucket.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "tf-assets"),
					resource.TestCheckResourceAttr("railway_bucket.test", "region", "sjc"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccBucketResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_bucket" "test" {
  name           = "%s"
  project_id     = "0bb01547-570d-4109-a5e8-138691f6a2d1"
  environment_id = "d0519b29-5d12-4857-a5dd-76fa7418336c"
}
`, name)
}

func bucketImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_bucket.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["environment_id"], rawState.Primary.Attributes["id"]), nil
}
