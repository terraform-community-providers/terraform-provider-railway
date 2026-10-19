package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResourceDefault(t *testing.T) {
	volumeName := fmt.Sprintf("todo-app-volume-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccServiceResourceConfigDefault("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing volume
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("nue-todo-app", volumeName, "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "NEVER"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", volumeName),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing image
			{
				Config: testAccServiceResourceConfigNonDefaultImage("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing repo
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("nue-todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
			// 	),
			// },
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing dockerfile build settings
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsDockerfile("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckResourceAttr("railway_service.test", "dockerfile_path", "packages/api/Dockerfile"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing command build settings
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsCommand("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckResourceAttr("railway_service.test", "build_command", "pnpm --filter api build"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing deploy settings
			{
				Config: testAccServiceResourceConfigNonDefaultDeploySettings("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "nginx:alpine"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_command", "nginx -t"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "start_command", "nginx -g 'daemon off;'"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_path", "/"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ALWAYS"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "20"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "true"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultRepo(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
			// 	),
			// },
			// // ImportState testing
			// {
			// 	ResourceName:      "railway_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
			// // Update with same values
			// {
			// 	Config: testAccServiceResourceConfigNonDefaultRepo("todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo", "railwayapp/blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "source_repo_branch", "main"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "root_directory", "blog"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "config_path", "blog/railway.yaml"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
			// 	),
			// },
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultDeploySettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultDeploySettings("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "nginx:alpine"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_command", "nginx -t"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "start_command", "nginx -g 'daemon off;'"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_path", "/"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ALWAYS"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "20"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "true"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultDeploySettings("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "nginx:alpine"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_command", "nginx -t"),
					resource.TestCheckResourceAttr("railway_service.test", "predeploy_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "start_command", "nginx -g 'daemon off;'"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_path", "/"),
					resource.TestCheckResourceAttr("railway_service.test", "healthcheck_timeout", "30"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ALWAYS"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "20"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "true"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultBuildSettingsDockerfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsDockerfile("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckResourceAttr("railway_service.test", "dockerfile_path", "packages/api/Dockerfile"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsDockerfile("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckResourceAttr("railway_service.test", "dockerfile_path", "packages/api/Dockerfile"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultBuildSettingsCommand(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsCommand("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckResourceAttr("railway_service.test", "build_command", "pnpm --filter api build"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultBuildSettingsCommand("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckResourceAttr("railway_service.test", "build_command", "pnpm --filter api build"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "2"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.0", "packages/api/src/**"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.1", "!packages/api/src/**/*.md"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultRegionsImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultRegionsImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultRegionsImage("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with different values
			{
				Config: testAccServiceResourceConfigNonDefaultRegionsImage2("todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigNonDefaultImage("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckResourceAttr("railway_service.test", "source_image", "hello-world"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "asia-southeast1-eqsg3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceNonDefaultVolume(t *testing.T) {
	suffix := fmt.Sprint(time.Now().UnixNano())
	volumeName := "todo-app-volume-" + suffix
	updatedVolumeName := "data-volume-" + suffix

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", volumeName, "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "NEVER"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", volumeName),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with same values
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", volumeName, "/mnt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "NEVER"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", volumeName),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with different values
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", updatedVolumeName, "/data"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckResourceAttr("railway_service.test", "cron_schedule", "0 0 * * *"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "NEVER"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", updatedVolumeName),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/data"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update with null values
			{
				Config: testAccServiceResourceConfigDefault("nue-todo-app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
					resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
					resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
					resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
					resource.TestCheckNoResourceAttr("railway_service.test", "root_directory"),
					resource.TestCheckNoResourceAttr("railway_service.test", "config_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "build_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "dockerfile_path"),
					resource.TestCheckResourceAttr("railway_service.test", "watch_patterns.#", "0"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "predeploy_timeout"),
					resource.TestCheckNoResourceAttr("railway_service.test", "start_command"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_path"),
					resource.TestCheckNoResourceAttr("railway_service.test", "healthcheck_timeout"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_type", "ON_FAILURE"),
					resource.TestCheckResourceAttr("railway_service.test", "restart_policy_max_retries", "10"),
					resource.TestCheckResourceAttr("railway_service.test", "serverless", "false"),
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccServiceResourceCronScheduleMultipleReplicas(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceResourceConfigCronScheduleMultipleReplicas("todo-app"),
				ExpectError: regexp.MustCompile("(?s)`cron_schedule` can only be set when total number of replicas.*Found 2 replicas"),
			},
		},
	})
}

func TestAccServiceResourceCronScheduleMultipleRegions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceResourceConfigCronScheduleMultipleRegions("todo-app"),
				ExpectError: regexp.MustCompile("(?s)`cron_schedule` can only be set when total number of replicas.*Found 2 replicas"),
			},
		},
	})
}

func testAccServiceResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, name)
}

func testAccServiceResourceConfigNonDefaultBuildSettingsDockerfile(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  dockerfile_path = "packages/api/Dockerfile"

  watch_patterns = [
    "packages/api/src/**",
    "!packages/api/src/**/*.md"
  ]
}
`, name)
}

func testAccServiceResourceConfigNonDefaultBuildSettingsCommand(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  build_command = "pnpm --filter api build"

  watch_patterns = [
    "packages/api/src/**",
    "!packages/api/src/**/*.md"
  ]
}
`, name)
}

func testAccServiceResourceConfigNonDefaultDeploySettings(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
  source_image = "nginx:alpine"

  predeploy_command = "nginx -t"
  predeploy_timeout = 30

  start_command = "nginx -g 'daemon off;'"

  healthcheck_path    = "/"
  healthcheck_timeout = 30

  restart_policy_type        = "ALWAYS"
  restart_policy_max_retries = 20

  serverless = true
}
`, name)
}

func testAccServiceResourceConfigNonDefaultRegionsImage(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  source_image = "hello-world"

  regions = [
    {
      region = "europe-west4-drams3a"
      num_replicas = 3
    },
    {
      region = "us-east4-eqdc4a"
      num_replicas = 2
    }
  ]
}
`, name)
}

func testAccServiceResourceConfigNonDefaultRegionsImage2(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  source_image = "hello-world"

  regions = [
    {
      region = "asia-southeast1-eqsg3a"
      num_replicas = 1
    },
    {
      region = "europe-west4-drams3a"
      num_replicas = 2
    }
  ]
}
`, name)
}

func testAccServiceResourceConfigNonDefaultImage(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  source_image = "hello-world"
}
`, name)
}

func testAccServiceResourceConfigNonDefaultRepo(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  source_repo = "railwayapp/blog"
  source_repo_branch = "main"
  root_directory = "blog"
  config_path = "blog/railway.yaml"
}
`, name)
}

func testAccServiceResourceConfigNonDefaultVolume(name string, volumeName string, path string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  cron_schedule = "0 0 * * *"

  volume = {
    name = "%s"
    mount_path = "%s"
  }
}
`, name, volumeName, path)
}

func testAccServiceResourceConfigCronScheduleMultipleReplicas(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name       = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  cron_schedule = "0 0 * * *"

  regions = [
    {
      region       = "europe-west4-drams3a"
      num_replicas = 2
    }
  ]
}
`, name)
}

func testAccServiceResourceConfigCronScheduleMultipleRegions(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name       = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

  cron_schedule = "0 0 * * *"

  regions = [
    {
      region = "europe-west4-drams3a"
    },
    {
      region = "us-east4-eqdc4a"
    }
  ]
}
`, name)
}
