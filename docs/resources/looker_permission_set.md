---
page_title: "looker_permission_set Resource - looker"
description: |-
  Manages Looker permission sets.
---

# looker_permission_set (Resource)

Manages Looker permission sets.

## Example Usage

```terraform
resource "looker_permission_set" "standard_viewer" {
  name = "Standard Viewer"
  permissions = [
    "access_data",
    "see_looks",
    "see_user_dashboards",
  ]
}
```

## Schema

### Required

- `name` (String) The name of the permission set.
- `permissions` (Set of String) The permissions of the permission set.

### Read-Only

- `id` (String) The unique identifier of the permission set.
- `built_in` (Boolean) Whether the permission set is built-in.
- `all_access` (Boolean) Whether the permission set has all access.
- `url` (String) The URL of the permission set.
