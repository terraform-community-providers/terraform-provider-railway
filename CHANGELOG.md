## 0.5.1

### Bug fixes
* Fixes issue with updating service regions defaulting to `us-west1`

## 0.5.0

### BREAKING
* Remove `railway_deployment_trigger` resource
* Remove `region` from `railway_service` in favor of `regions` multi region support

### Bug fixes
* Fixes issue with tcp proxy not being used in service after creating
* Fixes issue with updating project

## 0.4.6

### Bug fixes
* Fix issue with `targetPort` for custom domains being set to `0` if not provided
* Changed reading custom domain from service instance instead of using id

## 0.4.5

### Bug fixes
* Fix issue with optional `team_id` in `resource_project`
* Fix issue with `region` and `volume` in `resource_service`

## 0.4.4

### Bug fixes
* Fix issue with setting `source_image_registry_*` in `resource_service`

## 0.4.3

### Enhancements
* Added `railway_variable_collection` resource

## 0.4.2

### Bug fixes
* Fix issue with root directory and config path not being read correctly

## 0.4.1

### Enhancements
* Added `region` to `railway_service`
* Added `num_replicas` to `railway_service`
* Added registry credentials to `railway_service`

## 0.4.0

### BREAKING
* Add required `source_repo_branch` to `railway_service` when `source_repo` is present

## 0.3.1

### Bug fixes
* Fix issue with replicas of a service being set to `0`

## 0.3.0

### BREAKING
* Remove `railway_plugin` resource
* Remove `railway_plugin_variable` data source

### Enhancements
* Add `railway_tcp_proxy` resource
* Support `volume` in `railway_service` resource

## 0.2.0

### Enhancements
* Add `railway_service_domain` resource
* Add `railway_custom_domain` resource

### BREAKING
* Remove `project_id` input from `railway_deployment_trigger`
* Remove `project_id` input from `railway_variable`

## 0.1.2

### Enhancements
* Add `railway_deployment_trigger` resource

## 0.1.1

### Enhancements
* Add support for more service settings

## 0.1.0 (First release)
