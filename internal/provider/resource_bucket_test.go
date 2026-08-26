package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccBucketResourceDefault(t *testing.T) {
	projectName := "tf-acc-bucket-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBucketResourceConfig(projectName, "terraform-provider-bucket-test", "ams", "https://example.com", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "terraform-provider-bucket-test"),
					resource.TestCheckResourceAttr("railway_bucket.test", "region", "ams"),
					resource.TestCheckResourceAttr("railway_bucket_cors_configuration.test", "cors_rules.#", "1"),
					resource.TestCheckTypeSetElemAttr("railway_bucket_cors_configuration.test", "cors_rules.0.allowed_origins.*", "https://example.com"),
					resource.TestCheckResourceAttr("railway_bucket_cors_configuration.test", "cors_rules.0.max_age_seconds", "3600"),
					testAccCheckBucketEnvironmentIsolation,
				),
			},
			{
				ResourceName:      "railway_bucket.test",
				ImportState:       true,
				ImportStateIdFunc: bucketImportIDFunc,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "railway_bucket_cors_configuration.test",
				ImportState:       true,
				ImportStateIdFunc: bucketCorsConfigurationImportIDFunc,
				ImportStateVerify: true,
			},
			{
				Config: testAccBucketResourceConfig(projectName, "terraform-provider-bucket-renamed", "ams", "https://updated.example.com", 1800),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "terraform-provider-bucket-renamed"),
					resource.TestCheckTypeSetElemAttr("railway_bucket_cors_configuration.test", "cors_rules.0.allowed_origins.*", "https://updated.example.com"),
					resource.TestCheckResourceAttr("railway_bucket_cors_configuration.test", "cors_rules.0.max_age_seconds", "1800"),
					testAccCheckBucketEnvironmentIsolation,
				),
			},
			{
				Config: testAccBucketResourceConfig(projectName, "terraform-provider-bucket-renamed", "ams", "", 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "name", "terraform-provider-bucket-renamed"),
				),
			},
			{
				Config: testAccBucketResourceConfig(projectName, "terraform-provider-bucket-renamed", "iad", "", 0),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("railway_bucket.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_bucket.test", "region", "iad"),
					testAccCheckBucketEnvironmentIsolation,
				),
			},
		},
	})
}

func testAccBucketResourceConfig(projectName string, name string, region string, corsOrigin string, maxAgeSeconds int) string {
	corsConfiguration := ""
	if corsOrigin != "" {
		corsConfiguration = fmt.Sprintf(`
resource "railway_bucket_cors_configuration" "test" {
  project_id     = railway_project.test.id
  environment_id = railway_project.test.default_environment.id
  bucket_id      = railway_bucket.test.id

  cors_rules = [{
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD", "PUT"]
    allowed_origins = ["%s"]
    expose_headers  = ["ETag"]
    max_age_seconds = %d
  }]
}
`, corsOrigin, maxAgeSeconds)
	}

	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = "%s"

  default_environment = {
    name = "test"
  }
}

resource "railway_environment" "other" {
  name       = "other"
  project_id = railway_project.test.id
}

resource "railway_bucket" "test" {
  name           = "%s"
  project_id     = railway_project.test.id
  environment_id = railway_project.test.default_environment.id
  region         = "%s"
}
%s
`, projectName, name, region, corsConfiguration)
}

func testAccCheckBucketEnvironmentIsolation(state *terraform.State) error {
	bucket := state.RootModule().Resources["railway_bucket.test"]
	otherEnvironment := state.RootModule().Resources["railway_environment.other"]

	_, deployed, err := getManagedBucketDeployment(
		context.Background(),
		testAccClient(),
		bucket.Primary.Attributes["environment_id"],
		bucket.Primary.ID,
	)
	if err != nil {
		return fmt.Errorf("read selected bucket deployment: %w", err)
	}
	if !deployed {
		return fmt.Errorf("bucket was not deployed to the selected environment")
	}

	_, deployed, err = getManagedBucketDeployment(
		context.Background(),
		testAccClient(),
		otherEnvironment.Primary.ID,
		bucket.Primary.ID,
	)
	if err != nil {
		return fmt.Errorf("read other bucket deployment: %w", err)
	}
	if deployed {
		return fmt.Errorf("bucket was unexpectedly deployed to another environment")
	}
	return nil
}

func bucketImportIDFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_bucket.test"]
	if !ok {
		return "", fmt.Errorf("resource not found")
	}

	return fmt.Sprintf(
		"%s:%s:%s",
		rawState.Primary.Attributes["project_id"],
		rawState.Primary.Attributes["environment_id"],
		rawState.Primary.Attributes["id"],
	), nil
}

func bucketCorsConfigurationImportIDFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["railway_bucket_cors_configuration.test"]
	if !ok {
		return "", fmt.Errorf("resource not found")
	}

	return fmt.Sprintf(
		"%s:%s:%s",
		rawState.Primary.Attributes["project_id"],
		rawState.Primary.Attributes["environment_id"],
		rawState.Primary.Attributes["bucket_id"],
	), nil
}
