---
page_title: "looker_folder Data Source - looker"
description: |-
  Look up a folder by its ID, or by its name and parent folder ID.
---

# looker_folder (Data Source)

Look up a folder by its ID, or by its name and parent folder ID.

## Example Usage

```terraform
# Look up by ID (for root folders)
data "looker_folder" "shared" {
  id = "1"
}

# Look up by name within a parent
data "looker_folder" "my_folder" {
  name      = "My Folder"
  parent_id = data.looker_folder.shared.id
}
```

## Schema

### Optional

- `id` (String) The unique identifier of the folder.
- `name` (String) The name of the folder.
- `parent_id` (String) The ID of the parent folder. Required if looking up by `name`.

### Read-Only

- `content_metadata_id` (String) The ID of the content metadata for this folder.
- `is_personal` (Boolean) Whether the folder is a personal folder.
