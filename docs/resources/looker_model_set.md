---
page_title: "looker_model_set Resource - looker"
description: |-
  Manages Looker model sets.
---

# looker_model_set (Resource)

Manages Looker model sets.

## Example Usage

```terraform
resource "looker_model_set" "finance_models" {
  name = "Finance Models"
  models = [
    "finance_model",
    "ga_model",
  ]
}
```

## Schema

### Required

- `name` (String) The name of the model set.
- `models` (Set of String) The models in the model set.

### Read-Only

- `id` (String) The unique identifier of the model set.
- `built_in` (Boolean) Whether the model set is built-in.
- `all_access` (Boolean) Whether the model set has all access.
- `url` (String) The URL of the model set.
