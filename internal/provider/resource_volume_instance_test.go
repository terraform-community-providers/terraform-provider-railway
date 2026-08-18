package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccVolumeOwnershipLifecycle(t *testing.T) {
	projectName := "tf-acc-volume-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	var instanceID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVolumeOwnershipConfig(projectName, "application-data", 0, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_volume.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_volume.test", "name", "application-data"),
					resource.TestCheckResourceAttrPair("railway_volume.test", "project_id", "railway_project.test", "id"),
					testAccCheckVolumeInstance("railway_project.test", "default_environment.id", false, 0),
					testAccCheckVolumeInstance("railway_environment.selected", "id", false, 0),
					testAccCheckVolumeInstance("railway_environment.other", "id", false, 0),
				),
			},
			{
				ResourceName:      "railway_volume.test",
				ImportState:       true,
				ImportStateIdFunc: testAccVolumeImportStateId,
				ImportStateVerify: true,
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 0, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume.test", "name", "renamed-data"),
				),
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 5000, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_volume_instance.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_volume_instance.test", "size_mb", "5000"),
					resource.TestCheckNoResourceAttr("railway_volume_instance.test", "service_id"),
					resource.TestCheckNoResourceAttr("railway_volume_instance.test", "mount_path"),
					testAccCaptureVolumeInstanceID(&instanceID),
					testAccCheckVolumeInstance("railway_environment.selected", "id", true, 5000),
					testAccCheckVolumeInstance("railway_project.test", "default_environment.id", false, 0),
					testAccCheckVolumeInstance("railway_environment.other", "id", false, 0),
				),
			},
			{
				ResourceName:      "railway_volume_instance.test",
				ImportState:       true,
				ImportStateIdFunc: testAccVolumeInstanceImportStateId,
				ImportStateVerify: true,
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 10000, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_volume_instance.test", "size_mb", "10000"),
					testAccCheckVolumeInstanceID(&instanceID),
					testAccCheckVolumeInstance("railway_environment.selected", "id", true, 10000),
				),
			},
			{
				Config:      testAccVolumeOwnershipConfig(projectName, "renamed-data", 9999, ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Volume size cannot be decreased`),
			},
			{
				Config: testAccVolumeOwnershipConfigForEnvironment(
					projectName,
					"renamed-data",
					9999,
					"",
					"railway_environment.other.id",
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 10000, "/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("railway_volume_instance.test", "service_id", "railway_service.test", "id"),
					resource.TestCheckResourceAttr("railway_volume_instance.test", "mount_path", "/data"),
					testAccCheckVolumeInstanceID(&instanceID),
				),
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 10000, "/app/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("railway_volume_instance.test", "service_id", "railway_service.test", "id"),
					resource.TestCheckResourceAttr("railway_volume_instance.test", "mount_path", "/app/data"),
					testAccCheckVolumeInstanceID(&instanceID),
				),
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 10000, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_volume_instance.test", "service_id"),
					resource.TestCheckNoResourceAttr("railway_volume_instance.test", "mount_path"),
					testAccCheckVolumeInstanceID(&instanceID),
				),
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 10000, "/app/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("railway_volume_instance.test", "service_id", "railway_service.test", "id"),
					resource.TestCheckResourceAttr("railway_volume_instance.test", "mount_path", "/app/data"),
					testAccCheckVolumeInstanceID(&instanceID),
				),
			},
			{
				Config: testAccVolumeOwnershipConfig(projectName, "renamed-data", 0, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_volume.test", "id", uuidRegex()),
					testAccCheckVolumeInstance("railway_environment.selected", "id", false, 0),
					testAccCheckVolumeInstance("railway_project.test", "default_environment.id", false, 0),
					testAccCheckVolumeInstance("railway_environment.other", "id", false, 0),
				),
			},
			{
				Config: testAccVolumeProjectConfig(projectName),
				Check:  testAccCheckResourceAbsent("railway_volume.test"),
			},
		},
	})
}

func testAccVolumeProjectConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
}

resource "railway_environment" "selected" {
  project_id = railway_project.test.id
  name       = "selected"
}

resource "railway_environment" "other" {
  project_id = railway_project.test.id
  name       = "other"
}

resource "railway_service" "test" {
  project_id = railway_project.test.id
  name       = "volume-test"
}
	`, projectName)
}

func testAccVolumeOwnershipConfig(projectName string, volumeName string, sizeMB int64, mountPath string) string {
	return testAccVolumeOwnershipConfigForEnvironment(
		projectName,
		volumeName,
		sizeMB,
		mountPath,
		"railway_environment.selected.id",
	)
}

func testAccVolumeOwnershipConfigForEnvironment(
	projectName string,
	volumeName string,
	sizeMB int64,
	mountPath string,
	environmentId string,
) string {
	instance := ""
	if sizeMB > 0 {
		mount := ""
		if mountPath != "" {
			mount = fmt.Sprintf(`
  service_id = railway_service.test.id
  mount_path = %q`, mountPath)
		}
		instance = fmt.Sprintf(`
resource "railway_volume_instance" "test" {
  volume_id      = railway_volume.test.id
  environment_id = %s
  size_mb        = %d%s
}
`, environmentId, sizeMB, mount)
	}

	return fmt.Sprintf(`
resource "railway_project" "test" {
  name = %q
}

resource "railway_environment" "selected" {
  project_id = railway_project.test.id
  name       = "selected"
}

resource "railway_environment" "other" {
  project_id = railway_project.test.id
  name       = "other"
}

resource "railway_service" "test" {
  project_id = railway_project.test.id
  name       = "volume-test"
}

resource "railway_volume" "test" {
  project_id = railway_project.test.id
  name       = %q
}
%s`, projectName, volumeName, instance)
}

func testAccCaptureVolumeInstanceID(instanceID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources["railway_volume_instance.test"]
		if !ok {
			return fmt.Errorf("railway_volume_instance.test not found in state")
		}
		*instanceID = resourceState.Primary.Attributes["id"]
		if *instanceID == "" {
			return fmt.Errorf("railway_volume_instance.test has no id")
		}
		return nil
	}
}

func testAccCheckVolumeInstanceID(instanceID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources["railway_volume_instance.test"]
		if !ok {
			return fmt.Errorf("railway_volume_instance.test not found in state")
		}
		got := resourceState.Primary.Attributes["id"]
		if got != *instanceID {
			return fmt.Errorf("volume instance id changed from %q to %q", *instanceID, got)
		}
		return nil
	}
}

func testAccCheckVolumeInstance(
	environmentResource string,
	environmentAttribute string,
	wantPresent bool,
	wantSizeMB int,
) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		volumeState, ok := state.RootModule().Resources["railway_volume.test"]
		if !ok {
			return fmt.Errorf("railway_volume.test not found in state")
		}
		environmentState, ok := state.RootModule().Resources[environmentResource]
		if !ok {
			return fmt.Errorf("%s not found in state", environmentResource)
		}

		client := testAccRailwayClient()
		instance, found, err := getManagedVolumeInstance(
			context.Background(),
			client,
			environmentState.Primary.Attributes[environmentAttribute],
			volumeState.Primary.Attributes["id"],
		)
		if err != nil {
			return fmt.Errorf("read Railway volume instance: %w", err)
		}
		if found != wantPresent {
			return fmt.Errorf(
				"volume instance presence in %s was %t, want %t",
				environmentResource,
				found,
				wantPresent,
			)
		}
		if found && instance.SizeMB != wantSizeMB {
			return fmt.Errorf("volume instance size was %d MB, want %d MB", instance.SizeMB, wantSizeMB)
		}
		return nil
	}
}

func testAccRailwayClient() graphql.Client {
	httpClient := http.Client{
		Transport: &authedTransport{
			token:   os.Getenv(envVarName),
			wrapped: http.DefaultTransport,
		},
	}
	return graphql.NewClient("https://backboard.railway.app/graphql/v2?source=terraform_provider_railway_acceptance", &httpClient)
}

func testAccCheckResourceAbsent(resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if _, ok := state.RootModule().Resources[resourceName]; ok {
			return fmt.Errorf("%s remains in Terraform state", resourceName)
		}
		return nil
	}
}

func testAccVolumeImportStateId(state *terraform.State) (string, error) {
	resourceState := state.RootModule().Resources["railway_volume.test"]
	return fmt.Sprintf(
		"%s:%s",
		resourceState.Primary.Attributes["project_id"],
		resourceState.Primary.Attributes["id"],
	), nil
}

func testAccVolumeInstanceImportStateId(state *terraform.State) (string, error) {
	resourceState := state.RootModule().Resources["railway_volume_instance.test"]
	return fmt.Sprintf(
		"%s:%s",
		resourceState.Primary.Attributes["environment_id"],
		resourceState.Primary.Attributes["volume_id"],
	), nil
}
