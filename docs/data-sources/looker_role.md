---
page_title: "looker_role Data Source - looker"
description: |-
  Look up a role by its ID or name.
---

# looker_role (Data Source)

Look up a role by its ID or name.

## Example Usage

```terraform
data "looker_role" "admin" {
  name = "Admin"
}
```

## Schema

### Optional

- `id` (String) The unique identifier of the role.
- `name` (String) The name of the role.

### Read-Only

- `model_set_id` (String) The ID of the model set for this role.
- `permission_set_id` (String) The ID of the permission set for this role.
- `url` (String) The URL of the role.
