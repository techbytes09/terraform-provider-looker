---
page_title: "looker_group Data Source - looker"
description: |-
  Provides information about a Looker group and its user membership.
---

# looker_group (Data Source)

Provides information about a Looker group and its user membership.

## Example Usage

```terraform
data "looker_group" "admins" {
  name = "Admins"
}
```

## Schema

### Optional

- `id` (String) The unique identifier of the group.
- `name` (String) The name of the group.

### Read-Only

- `user_count` (Number) Number of users in the group.
- `user_ids` (Set of String) IDs of users in the group.
