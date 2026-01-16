---
page_title: "looker_role Resource - looker"
description: |-
  Manages Looker roles.
---

# looker_role (Resource)

Manages Looker roles.

## Example Usage

```terraform
resource "looker_role" "standard_user" {
  name              = "Standard User"
  permission_set_id = "1"
  model_set_id      = "2"
}
```

## Schema

### Required

- `name` (String) The name of the role.
- `model_set_id` (String) The ID of the model set for this role.
- `permission_set_id` (String) The ID of the permission set for this role.

### Read-Only

- `id` (String) The unique identifier of the role.
- `url` (String) The URL of the role.
