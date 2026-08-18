package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

func TestExpandServiceInstanceEnvironmentPatchPreservesRegistryAndRemovesRegions(t *testing.T) {
	data := ServiceInstanceResourceModel{
		ServiceId:                          types.StringValue("service-id"),
		SourceImagePrivateRegistryUsername: types.StringValue("registry-user"),
		SourceImagePrivateRegistryPassword: types.StringValue("registry-password"),
	}
	regions := map[string]ServiceInstanceRegionModel{
		"europe-west4": {
			NumReplicas: types.Int64Value(1),
		},
	}

	got := expandServiceInstanceEnvironmentPatch(data, regions, railway.EnvironmentConfig{
		Services: map[string]railway.ServiceConfig{
			"service-id": {
				Deploy: &railway.ServiceDeployConfig{
					MultiRegionConfig: map[string]*railway.ServiceRegionConfig{
						"us-west2": {NumReplicas: 1},
					},
				},
			},
		},
	})
	service := got.Services["service-id"]
	credentials := service.Deploy.RegistryCredentials
	if credentials == nil ||
		credentials.Username != "registry-user" ||
		credentials.Password != "registry-password" {
		t.Fatalf("unexpected registry credentials: %#v", credentials)
	}
	if region := service.Deploy.MultiRegionConfig["europe-west4"]; region == nil || region.NumReplicas != 1 {
		t.Fatalf("unexpected managed region: %#v", region)
	}
	if region, exists := service.Deploy.MultiRegionConfig["us-west2"]; !exists || region != nil {
		t.Fatalf("expected previous region to be removed, got %#v", region)
	}
}

func TestAccServiceInstanceResourceDefault(t *testing.T) {
	projectName := "tf-acc-service-instance-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceInstanceResourceConfig(projectName, "/health", 0.25),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_project.test_instance", "id", uuidRegex()),
					resource.TestCheckResourceAttrPair(
						"railway_service.test_instance",
						"project_id",
						"railway_project.test_instance",
						"id",
					),
					resource.TestCheckNoResourceAttr("railway_service.test_instance", "source_image"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", "traefik/whoami:v1.10"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/health"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "vcpus", "0.25"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "memory_gb", "0.5"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "regions.europe-west4-drams3a.num_replicas", "1"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "effective_regions.europe-west4-drams3a.num_replicas", "1"),
					testAccCheckSiblingServiceInstance,
				),
			},
			{
				ResourceName:      "railway_service_instance.test",
				ImportState:       true,
				ImportStateIdFunc: testAccServiceInstanceImportStateId,
				ImportStateVerify: true,
			},
			{
				Config: testAccServiceInstanceResourceConfig(projectName, "/healthz", 0.5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_service.test_instance", "source_image"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/healthz"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "vcpus", "0.5"),
					testAccCheckSiblingServiceInstance,
					testAccDetachServiceInstanceRegion,
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccServiceInstanceResourceConfig(projectName, "/healthz", 0.5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", "traefik/whoami:v1.10"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", "/healthz"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "vcpus", "0.5"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "memory_gb", "0.5"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "regions.europe-west4-drams3a.num_replicas", "1"),
					testAccCheckSiblingServiceInstance,
				),
			},
			{
				Config: testAccServiceInstanceResourceBuildConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "root_directory", "/services/api"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "config_path", "/railway.json"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "regions.europe-west4-drams3a.num_replicas", "1"),
					testAccCheckSiblingServiceInstance,
				),
			},
			{
				Config: testAccServiceInstanceResourceResetConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "cron_schedule", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "root_directory", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "config_path", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "vcpus", "0"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "memory_gb", "0"),
					resource.TestCheckNoResourceAttr("railway_service_instance.test", "regions"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "effective_regions.%", "1"),
					testAccCheckSiblingServiceInstance,
				),
			},
			{
				ResourceName:      "railway_service_instance.test",
				ImportState:       true,
				ImportStateIdFunc: testAccServiceInstanceImportStateId,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"regions",
				},
			},
			{
				Config: testAccServiceInstanceResourceResetConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("railway_service_instance.test", "regions"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "effective_regions.%", "1"),
				),
			},
			{
				Config: testAccServiceInstanceResourceWithoutSiblingConfig(projectName),
				Check:  testAccCheckSiblingServiceInstanceAbsent,
			},
		},
	})
}

func TestAccServiceInstanceResourceRecreateClearsRetainedSettings(t *testing.T) {
	projectName := "tf-acc-recreate-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceInstanceRetainedSettingsConfig(projectName),
			},
			{
				Config: testAccServiceInstanceProjectConfig(projectName),
			},
			{
				Config: testAccServiceInstanceResourceResetConfig(projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("railway_service_instance.test", "source_image", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "healthcheck_path", ""),
					resource.TestCheckResourceAttr("railway_service_instance.test", "vcpus", "0"),
					resource.TestCheckResourceAttr("railway_service_instance.test", "memory_gb", "0"),
				),
			},
		},
	})
}

func testAccServiceInstanceProjectResourcesConfig(projectName string) string {
	return fmt.Sprintf(`
resource "railway_project" "test_instance" {
  name = "%s"
}

resource "railway_service" "test_instance" {
  name       = "terraform-provider-service-instance-test"
  project_id = railway_project.test_instance.id
}

resource "railway_environment" "sibling" {
  name       = "sibling"
  project_id = railway_project.test_instance.id

  depends_on = [railway_service.test_instance]
}
`, projectName)
}

func testAccServiceInstanceProjectConfig(projectName string) string {
	return fmt.Sprintf(`
%s
resource "railway_service_instance" "sibling" {
  environment_id   = railway_environment.sibling.id
  service_id       = railway_service.test_instance.id
  source_image     = "nginx:alpine"
  healthcheck_path = "/sibling"
}
`, testAccServiceInstanceProjectResourcesConfig(projectName))
}

func testAccServiceInstanceResourceConfig(projectName, healthcheckPath string, vcpus float64) string {
	return fmt.Sprintf(`
%s

resource "railway_service_instance" "test" {
  environment_id   = railway_project.test_instance.default_environment.id
  service_id       = railway_service.test_instance.id
  source_image     = "traefik/whoami:v1.10"
  healthcheck_path = "%s"
  vcpus             = %g
  memory_gb         = 0.5

  regions = {
    "europe-west4-drams3a" = {
      num_replicas = 1
    }
  }
}
`, testAccServiceInstanceProjectConfig(projectName), healthcheckPath, vcpus)
}

func testAccServiceInstanceRetainedSettingsConfig(projectName string) string {
	return fmt.Sprintf(`
%s

resource "railway_service_instance" "test" {
  environment_id   = railway_project.test_instance.default_environment.id
  service_id       = railway_service.test_instance.id
  source_image     = "traefik/whoami:v1.10"
  healthcheck_path = "/health"
  vcpus             = 0.25
  memory_gb         = 0.5
}
`, testAccServiceInstanceProjectConfig(projectName))
}

func testAccServiceInstanceResourceBuildConfig(projectName string) string {
	return fmt.Sprintf(`
%s

resource "railway_service_instance" "test" {
  environment_id = railway_project.test_instance.default_environment.id
  service_id     = railway_service.test_instance.id
  cron_schedule  = "0 0 * * *"
  root_directory = "/services/api"
  config_path    = "/railway.json"

  regions = {
    "europe-west4-drams3a" = {
      num_replicas = 1
    }
  }
}
`, testAccServiceInstanceProjectConfig(projectName))
}

func testAccServiceInstanceResourceResetConfig(projectName string) string {
	return fmt.Sprintf(`
%s

resource "railway_service_instance" "test" {
  environment_id = railway_project.test_instance.default_environment.id
  service_id     = railway_service.test_instance.id
}
`, testAccServiceInstanceProjectConfig(projectName))
}

func testAccServiceInstanceResourceWithoutSiblingConfig(projectName string) string {
	return fmt.Sprintf(`
%s

resource "railway_service_instance" "test" {
  environment_id = railway_project.test_instance.default_environment.id
  service_id     = railway_service.test_instance.id
}
`, testAccServiceInstanceProjectResourcesConfig(projectName))
}

func testAccServiceInstanceImportStateId(state *terraform.State) (string, error) {
	resourceState := state.RootModule().Resources["railway_service_instance.test"]
	return resourceState.Primary.ID, nil
}

func testAccCheckSiblingServiceInstance(state *terraform.State) error {
	environment := state.RootModule().Resources["railway_environment.sibling"]
	service := state.RootModule().Resources["railway_service.test_instance"]

	response, err := getManagedServiceInstance(
		context.Background(),
		testAccClient(),
		environment.Primary.ID,
		service.Primary.ID,
	)
	if err != nil {
		return fmt.Errorf("read sibling service instance: %w", err)
	}

	instance := response.ServiceInstance
	if instance.Source == nil || instance.Source.Image == nil || *instance.Source.Image != "nginx:alpine" {
		return fmt.Errorf("sibling service source image changed")
	}
	if instance.HealthcheckPath == nil || *instance.HealthcheckPath != "/sibling" {
		return fmt.Errorf("sibling service healthcheck path changed")
	}
	return nil
}

func testAccDetachServiceInstanceRegion(state *terraform.State) error {
	instance := state.RootModule().Resources["railway_service_instance.test"]
	environmentID := instance.Primary.Attributes["environment_id"]
	serviceID := instance.Primary.Attributes["service_id"]

	if err := commitAndWaitForEnvironmentPatch(
		context.Background(),
		testAccClient(),
		environmentID,
		railway.DeleteServiceInstancePatch(serviceID),
		"Externally detach Terraform acceptance test service instance",
	); err != nil {
		return fmt.Errorf("externally detach service instance: %w", err)
	}

	response, err := getManagedServiceInstance(
		context.Background(),
		testAccClient(),
		environmentID,
		serviceID,
	)
	if err != nil {
		return fmt.Errorf("read externally detached service instance: %w", err)
	}
	if _, exists := response.Environment.Config.Services[serviceID]; exists {
		return fmt.Errorf("externally detached service remains in environment config")
	}
	if response.ServiceInstance.Source == nil ||
		response.ServiceInstance.Source.Image == nil ||
		*response.ServiceInstance.Source.Image != "traefik/whoami:v1.10" {
		return fmt.Errorf("external detach did not retain the service image")
	}

	return nil
}

func testAccCheckSiblingServiceInstanceAbsent(state *terraform.State) error {
	project := state.RootModule().Resources["railway_project.test_instance"]
	environment := state.RootModule().Resources["railway_environment.sibling"]
	service := state.RootModule().Resources["railway_service.test_instance"]

	response, err := getEnvironmentServiceInstances(
		context.Background(),
		testAccClient(),
		project.Primary.ID,
		environment.Primary.ID,
		nil,
	)
	if err != nil {
		return fmt.Errorf("list sibling environment service instances: %w", err)
	}
	for _, edge := range response.Environment.ServiceInstances.Edges {
		if edge.Node.ServiceId == service.Primary.ID {
			return fmt.Errorf("sibling service instance %s still exists", edge.Node.ServiceId)
		}
	}

	return nil
}
