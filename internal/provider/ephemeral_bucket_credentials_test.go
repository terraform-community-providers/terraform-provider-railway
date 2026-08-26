package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBucketCredentialsEphemeralResourceDefault(t *testing.T) {
	projectName := "tf-acc-bucket-creds-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "railway_project" "credentials_test" {
  name = "%s"
}

resource "railway_service" "credentials_test" {
  name       = "terraform-provider-bucket-credentials-test"
  project_id = railway_project.credentials_test.id
}

resource "railway_bucket" "credentials_test" {
  name       = "terraform-provider-credentials-test"
  project_id = railway_project.credentials_test.id
}

resource "railway_bucket_instance" "credentials_test" {
  bucket_id      = railway_bucket.credentials_test.id
  environment_id = railway_project.credentials_test.default_environment.id
  region         = "ams"
}

ephemeral "railway_bucket_credentials" "test" {
  project_id     = railway_bucket.credentials_test.project_id
  environment_id = railway_bucket_instance.credentials_test.environment_id
  bucket_id      = railway_bucket_instance.credentials_test.bucket_id
}

resource "railway_variable" "credentials_test" {
  name             = "TERRAFORM_EPHEMERAL_BUCKET_SECRET_TEST"
  value_wo         = ephemeral.railway_bucket_credentials.test.secret_access_key
  value_wo_version = 1
  environment_id   = railway_bucket_instance.credentials_test.environment_id
  service_id       = railway_service.credentials_test.id
}
`, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_variable.credentials_test", "value_wo"),
				),
			},
		},
	})
}
