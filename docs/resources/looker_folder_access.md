---
page_title: "looker_folder_access Resource - looker"
description: |-
  Manages content access grants for a Looker folder (space).
---

# looker_folder_access (Resource)

Manages content access grants for a Looker folder (space). This resource links a group to a folder with a specific access level.

## Example Usage

```terraform
resource "looker_folder_access" "sales_folder_access" {
  folder_id    = looker_folder.sales_reports.content_metadata_id
  group_id     = looker_group.sales_team.id
  access_level = "view"
}
```

## Schema

### Required

- `folder_id` (String) The ID of the folder (content_metadata_id) to grant access to.
- `group_id` (String) The ID of the group to grant access to.
- `access_level` (String) The access level to grant. Valid values are: `view` (View), `edit` (Manage Access, Edit).

### Read-Only

- `id` (String) The unique ID of this access grant.
