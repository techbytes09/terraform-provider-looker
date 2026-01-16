---
page_title: "looker_role_groups Resource - looker"
description: |-
  Manages the assignment of a set of groups to a single Looker role.
---

# looker_role_groups (Resource)

Manages the assignment of a set of groups to a single Looker role.

## Example Usage

```terraform
resource "looker_role_groups" "finance_role_assignment" {
  role_id = looker_role.finance_analyst.id

  group_ids = [
    looker_group.finance_team.id,
    looker_group.executives.id,
  ]
}
```

## Schema

### Required

- `role_id` (String) The ID of the role.
- `group_ids` (Set of String) The IDs of the groups to assign to the role.

### Read-Only

- `id` (String) The unique identifier of the resource.
