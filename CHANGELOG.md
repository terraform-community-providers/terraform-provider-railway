## master

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
