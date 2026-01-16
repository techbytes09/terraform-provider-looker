---
page_title: "Provider: Looker"
description: |-
  The Looker provider allows you to manage Looker resources such as groups, roles, and folder permissions.
---

# Looker Provider

The Looker provider allows you to manage Looker resources such as groups, roles, and folder permissions.

## Example Usage

```terraform
provider "looker" {
  base_url      = "https://myinstance.looker.com:19999"
  client_id     = "my-client-id"
  client_secret = "my-client-secret"
}
```

## Schema

### Optional

- `base_url` (String) Looker host base URL (no `/api/*`). Example: `https://myinstance.looker.com:19999`. Can also be set via the `LOOKER_BASE_URL` environment variable.
- `client_id` (String, Sensitive) Client ID for authentication. Can also be set via the `LOOKER_CLIENT_ID` environment variable.
- `client_secret` (String, Sensitive) Client Secret for authentication. Can also be set via the `LOOKER_CLIENT_SECRET` environment variable.
