---
page_title: "looker_folder_permission_override Resource - looker"
description: |-
  Manages a folder permission override.
---

# looker_folder_permission_override (Resource)

Manages a folder permission override. This resource finds an existing, inherited access grant for a group on a folder and updates it to a new, direct access level (e.g., from inherited 'view' to direct 'edit').

## Example Usage

```terraform
resource "looker_folder_permission_override" "override_edit" {
  folder_id    = looker_folder.sub_folder.content_metadata_id
  group_id     = looker_group.all_users.id
  access_level = "edit"
}
```

## Schema

### Required

- `folder_id` (String) The ID of the folder (content_metadata_id) whose permissions will be overridden.
- `group_id` (String) The ID of the group whose inherited permission will be overridden.
- `access_level` (String) The new, direct access level to set. Valid values are: `view` or `edit`.

### Read-Only

- `id` (String) The unique ID of the access grant that was updated.
