---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "railway_project Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway project.
---

# railway_project (Resource)

Railway project.

## Example Usage

```terraform
resource "railway_project" "example" {
  name = "something"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the project.

### Optional

- `default_environment` (Attributes) Default environment of the project. When multiple exist, the oldest is considered. (see [below for nested schema](#nestedatt--default_environment))
- `description` (String) Description of the project.
- `has_pr_deploys` (Boolean) Whether the project has PR deploys enabled. **Default** `false`.
- `private` (Boolean) Privacy of the project. **Default** `true`.
- `team_id` (String) Identifier of the team the project belongs to.

### Read-Only

- `id` (String) Identifier of the project.

<a id="nestedatt--default_environment"></a>
### Nested Schema for `default_environment`

Optional:

- `name` (String) Name of the default environment.

Read-Only:

- `id` (String) Identifier of the default environment.

## Import

Import is supported using the following syntax:

```shell
terraform import railway_project.example 0bb01547-570d-4109-a5e8-138691f6a2d1
```