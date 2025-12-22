package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const roleSearchFields = "id,name,permission_set,model_set,url"

// roleDataSource is the data source implementation.
type roleDataSource struct {
	sdk *v4.LookerSDK
}

// roleModel maps the data source schema data.
type roleModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	PermissionSetID types.String `tfsdk:"permission_set_id"`
	ModelSetID      types.String `tfsdk:"model_set_id"`
	URL             types.String `tfsdk:"url"`
}

// NewRoleDataSource is a helper function to simplify the provider implementation.
func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

// Metadata returns the data source type name.
func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the data source.
func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looker role (read-only). Provide exactly one of `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Optional: true},
			"name":              schema.StringAttribute{Optional: true},
			"permission_set_id": schema.StringAttribute{Computed: true},
			"model_set_id":      schema.StringAttribute{Computed: true},
			"url":               schema.StringAttribute{Computed: true},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		d.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var data roleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role v4.Role
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		// CORRECTED: The Role() function does not take a 'fields' argument.
		role, err = d.sdk.Role(data.ID.ValueString(), nil)
	} else if !data.Name.IsNull() && data.Name.ValueString() != "" {
		name := data.Name.ValueString()
		fields := roleSearchFields
		results, e := d.sdk.SearchRoles(v4.RequestSearchRoles{
			Name:   &name,
			Fields: &fields,
		}, nil)
		err = e
		if err == nil {
			if len(results) == 0 {
				resp.Diagnostics.AddError("Not found", fmt.Sprintf("No role named %q", name))
				return
			}
			role = results[0]
		}
	} else {
		resp.Diagnostics.AddError("Invalid input", "You must provide either `id` or `name`.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Role lookup failed: %v", err))
		return
	}

	// Map API response to Terraform state
	data.ID = types.StringPointerValue(role.Id)
	data.Name = types.StringPointerValue(role.Name)
	data.URL = types.StringPointerValue(role.Url)
	if role.PermissionSet != nil {
		data.PermissionSetID = types.StringPointerValue(role.PermissionSet.Id)
	}
	if role.ModelSet != nil {
		data.ModelSetID = types.StringPointerValue(role.ModelSet.Id)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
