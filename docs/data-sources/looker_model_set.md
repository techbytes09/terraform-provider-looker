---
page_title: "looker_model_set Data Source - looker"
description: |-
  Look up a model set by its ID or name.
---

# looker_model_set (Data Source)

Look up a model set by its ID or name.

## Example Usage

```terraform
data "looker_model_set" "finance_models" {
  name = "Finance Models"
}
```

## Schema

### Optional

- `id` (String) The unique identifier of the model set.
- `name` (String) The name of the model set.

### Read-Only

- `all_access` (Boolean) Whether the model set has all access.
- `built_in` (Boolean) Whether the model set is built-in.
- `models` (Set of String) The models in the model set.
- `url` (String) The URL of the model set.
