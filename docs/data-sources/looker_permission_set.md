---
page_title: "looker_permission_set Data Source - looker"
description: |-
  Look up a permission set by its ID or name.
---

# looker_permission_set (Data Source)

Look up a permission set by its ID or name.

## Example Usage

```terraform
data "looker_permission_set" "admin" {
  name = "Admin"
}
```

## Schema

### Optional

- `id` (String) The unique identifier of the permission set.
- `name` (String) The name of the permission set.

### Read-Only

- `all_access` (Boolean) Whether the permission set has all access.
- `built_in` (Boolean) Whether the permission set is built-in.
- `permissions` (Set of String) The permissions of the permission set.
- `url` (String) The URL of the permission set.
