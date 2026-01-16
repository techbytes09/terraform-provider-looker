---
page_title: "looker_group Resource - looker"
description: |-
  Manages Looker groups and their user memberships.
---

# looker_group (Resource)

Manages Looker groups and their user memberships.

## Example Usage

```terraform
resource "looker_group" "wizards" {
  name = "Wizards"
  
  # specific user ids
  user_ids = ["1", "2"]
  
  # or by email (provider resolves to IDs)
  user_emails = ["gandalf@middleearth.com"]
}
```

## Schema

### Required

- `name` (String) The name of the group.

### Optional

- `user_emails` (Set of String) Emails of users to be added to the group. The provider will resolve these to user IDs. Use this or `user_ids`, but not both.
- `user_ids` (Set of String) IDs of users to be added to the group.

### Read-Only

- `id` (String) The unique identifier of the group.
