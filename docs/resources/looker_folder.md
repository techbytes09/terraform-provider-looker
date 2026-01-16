---
page_title: "looker_folder Resource - looker"
description: |-
  Manages Looker folders (spaces).
---

# looker_folder (Resource)

Manages Looker folders (formerly known as spaces).

## Example Usage

```terraform
resource "looker_folder" "marketing" {
  name      = "Marketing"
  parent_id = "1" # Typically "1" is the Shared folder
}

resource "looker_folder" "restricted" {
  name                 = "Restricted"
  parent_id            = "1"
  inherits_permissions = false
}
```

## Schema

### Required

- `name` (String) The name of the folder.
- `parent_id` (String) The ID of the parent folder.

### Optional

- `inherits_permissions` (Boolean) If true, the folder inherits permissions from its parent. If false, the folder has its own explicit permissions. Must be set to `false` to use `looker_folder_access` on this folder.

### Read-Only

- `id` (String) The unique identifier of the folder.
- `content_metadata_id` (String) The ID of the content metadata for this folder, used for access grants.
