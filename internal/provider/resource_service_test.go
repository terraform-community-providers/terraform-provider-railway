package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServiceResourceDefault(t *testing.T) {
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckNoResourceAttr("railway_service.test", "regions"),
				),
			},
			// Update and Read testing regions
			{
				Config: testAccServiceResourceConfigNonDefaultRegions("nue-todo-app"),
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "europe-west4-drams3a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "3"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.region", "us-east4-eqdc4a"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.1.num_replicas", "2"),
				),
			},
			// ImportState testing
			// {
			// 	ResourceName:      "railway_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "regions.0.num_replicas", "1"),
			// 	),
			// },
			// ImportState testing
			{
				ResourceName:      "railway_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing volume
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("nue-todo-app", "todo-app-volume", "/mnt"),
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
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.mount_path", "/mnt"),
					resource.TestCheckResourceAttr("railway_service.test", "volume.size", "50000"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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
		Steps:                    []resource.TestStep{
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
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			//		resource.TestCheckResourceAttr("railway_service.test", "region", "us-west1"),
			//		resource.TestCheckResourceAttr("railway_service.test", "num_replicas", "1"),
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
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			//		resource.TestCheckResourceAttr("railway_service.test", "region", "us-west1"),
			//		resource.TestCheckResourceAttr("railway_service.test", "num_replicas", "1"),
			// 	),
			// },
			// Update with null values
			// {
			// 	Config: testAccServiceResourceConfigDefault("nue-todo-app"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestMatchResourceAttr("railway_service.test", "id", uuidRegex()),
			// 		resource.TestCheckResourceAttr("railway_service.test", "name", "nue-todo-app"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "project_id", "0bb01547-570d-4109-a5e8-138691f6a2d1"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "cron_schedule"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_image"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_repo"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "source_repo_branch"),
			// 		resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "region", "us-west1"),
			// 		resource.TestCheckResourceAttr("railway_service.test", "num_replicas", "1"),
			// 	),
			// },
			// ImportState testing
			// {
			// 	ResourceName:      "railway_service.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// },
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
					resource.TestCheckNoResourceAttr("railway_service.test", "volume"),
					resource.TestCheckResourceAttr("railway_service.test", "regions.0.region", "us-west1"),
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

func TestAccServiceResourceNonDefaultVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "todo-app-volume", "/mnt"),
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
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
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
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "todo-app-volume", "/mnt"),
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
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "todo-app-volume"),
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
				Config: testAccServiceResourceConfigNonDefaultVolume("todo-app", "data-volume", "/data"),
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
					resource.TestMatchResourceAttr("railway_service.test", "volume.id", uuidRegex()),
					resource.TestCheckResourceAttr("railway_service.test", "volume.name", "data-volume"),
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

func testAccServiceResourceConfigDefault(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"
}
`, name)
}

func testAccServiceResourceConfigNonDefaultRegions(name string) string {
	return fmt.Sprintf(`
resource "railway_service" "test" {
  name = "%s"
  project_id = "0bb01547-570d-4109-a5e8-138691f6a2d1"

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
