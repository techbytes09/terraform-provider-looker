# Terraform Provider for Looker

The Terraform Looker provider allows you to manage your Looker instance resources as code. It provides resources for managing permission sets, model sets, roles, groups, folders, and access controls, enabling you to automate, version control, and reproduce your Looker configuration.

## Requirements

-   Terraform `~> 1.0`
-   Go `~> 1.18` (to build the provider from source)
-   Looker API Credentials (Client ID, Client Secret, and Base URL)


## Terraform Configuration
In your .tf files, declare the provider. The source address must match the one used in your CLI config file.

```sh
terraform {
  required_providers {
    looker = {
      source  = "techbytes09/looker"
      version = "~> 1.0"
    }
  }
}

provider "looker" {
  base_url      = "https://your-instance.looker.com"
  client_id     = "your_api_client_id"
  client_secret = "your_api_client_secret"

}
```
## Example Usage
This example demonstrates a complete workflow: creating a group, creating a new folder, and granting the group access to that folder.

```sh
# 1. Look up the main "Shared" folder, which has a static ID of "1".
data "looker_folder" "shared" {
  id = "1"
}

# 2. Create a new group for the "Data Analytics" team.
resource "looker_group" "data_analytics_team" {
  name = "Data Analytics Team"
  user_emails = [
    "analyst.one@example.com",
    "analyst.two@example.com",
  ]
}

# 3. Create a new folder for the team inside the "Shared" folder.
resource "looker_folder" "data_analytics_folder" {
  name      = "Data Analytics Reports"
  parent_id = data.looker_folder.shared.id
}

# 4. Grant the team "edit" access to their new folder.
resource "looker_folder_access" "analytics_folder_access" {
  # Use the `content_metadata_id` from the folder for access grants.
  folder_id = looker_folder.data_analytics_folder.content_metadata_id

  # Use the ID from the group.
  group_id = looker_group.data_analytics_team.id

  access_level = "edit"
}
```

## Schema Reference

### Resources

### looker_permission_set

Manages a Looker permission set.

##### Example:

```sh
resource "looker_permission_set" "standard_viewer" {
  name = "Standard Viewer"
  permissions = [
    "access_data",
    "see_looks",
    "see_user_dashboards",
  ]
}
```
#### Argument Reference:

- name (Required, String): The name of the permission set.
- permissions (Required, Set of String): A list of permissions to include in the set.



### looker_model_set
Manages a Looker model set.

#### Example:

```sh
resource "looker_model_set" "finance_models" {
  name = "Finance Models"
  models = [
    "finance_model",
    "ga_model",
  ]
}
```

#### Argument Reference:

- name (Required, String): The name of the model set.
- models (Required, Set of String): A list of model names to include in the set.



### looker_role
Manages a Looker role, which connects a permission set and a model set.

#### Example:
```sh
resource "looker_role" "finance_analyst" {
  name              = "Finance Analyst"
  permission_set_id = looker_permission_set.standard_viewer.id
  model_set_id      = looker_model_set.finance_models.id
}
```

#### Argument Reference:

- name (Required, String): The name of the role.
- permission_set_id (Required, String): The ID of the permission set for this role.
- model_set_id (Required, String): The ID of the model set for this role.



### looker_group
Manages a Looker group and its user membership.

#### Example:

```sh
resource "looker_group" "engineering" {
  name = "Engineering Team"
  user_emails = [
    "engineer.one@example.com",
    "engineer.two@example.com",
  ]
}
```

#### Argument Reference:
- name (Required, String): The name of the group.
- user_ids (Optional, Set of String): A set of user IDs to add to the group.
- user_emails (Optional, Set of String): A set of user emails to add to the 
- group. The provider will resolve these to their corresponding user IDs.




### looker_role_groups
Manages the assignment of groups to a single Looker role.

#### Example:

```sh
resource "looker_role_groups" "finance_role_assignment" {
  role_id = looker_role.finance_analyst.id

  group_ids = [
    looker_group.finance_team.id,
    looker_group.executives.id,
  ]
}
```

### Argument Reference:
- role_id (Required, String): The ID of the role.
- group_ids (Required, Set of String): The set of group IDs to assign to the role.



### looker_folder
Manages a Looker folder (space).

#### Example:

```sh
resource "looker_folder" "sales_reports" {
  name      = "Sales Reports"
  parent_id = data.looker_folder.shared.id
}
```

#### Argument Reference:
- name (Required, String): The name of the folder.
- parent_id (Required, String): The ID of the parent folder.



### looker_folder_access
Manages a content access grant for a group on a folder.

#### Example:

```sh 
resource "looker_folder_access" "sales_folder_access" {
  folder_id    = looker_folder.sales_reports.content_metadata_id
  group_id     = looker_group.sales_team.id
  access_level = "view"
}
```

### Argument Reference:
- folder_id (Required, String): The content_metadata_id of the folder.
- group_id (Required, String): The ID of the group to grant access to.
- access_level (Required, String): The level of access to grant. Must be either "view" or "edit".



## Data Sources

Data sources allow you to look up information about existing resources in your Looker instance.

## looker_permission_set

Look up a permission set by its ID or name.

```sh
data "looker_permission_set" "admin" {
  name = "Admin"
}
```

## looker_model_set
Look up a model set by its ID or name.

```sh
data "looker_model_set" "model_sets" {
  name = "Finance Models"
}
```
## looker_role
Look up a role by its ID or name.

```sh
data "looker_role" "admin" {
  name = "Admin"
}
```
## looker_group
Look up a group by its ID or name.

```sh
data "looker_group" "all_users" {
  name = "Engineering Team"
}
```
## looker_folder
Look up a folder by its ID, or by its name and parent folder ID.

```sh
# Look up by ID (for root folders)
data "looker_folder" "shared" {
  id = "1"
}
```

 Look up by name within a parent
```sh
data "looker_folder" "my_folder" {
  name      = "My Folder"
  parent_id = data.looker_folder.shared.id
}
```



