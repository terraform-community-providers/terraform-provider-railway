---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "railway_variable Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway variable. Any changes in collection triggers service redeployment.
---

# railway_variable (Resource)

Railway variable. Any changes in collection triggers service redeployment.

## Example Usage

```terraform
resource "railway_variable" "example" {
  name           = "SENTRY_KEY"
  value          = "1234567890"
  environment_id = railway_project.example.default_environment.id
  service_id     = railway_service.example.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `environment_id` (String) Identifier of the environment the variable belongs to.
- `name` (String) Name of the variable.
- `service_id` (String) Identifier of the service the variable belongs to.
- `value` (String, Sensitive) Value of the variable.

### Read-Only

- `id` (String) Identifier of the variable.
- `project_id` (String) Identifier of the project the variable belongs to.

## Import

Import is supported using the following syntax:

```shell
terraform import railway_variable.sentry 89fa0236-2b1b-4a8c-b12d-ae3634b30d97:staging:SENTRY_KEY
```
