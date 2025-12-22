package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const groupDataSourceFields = "id,name,user_count"

// groupDataSource is the data source implementation.
type groupDataSource struct {
	sdk *v4.LookerSDK
}

// groupModel maps the data source schema data.
// NOTE: RoleIDs has been removed as it cannot be fetched efficiently.
type groupModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	UserCount types.Int64  `tfsdk:"user_count"`
	UserIDs   types.Set    `tfsdk:"user_ids"`
}

// NewGroupDataSource is a helper function to simplify the provider implementation.
func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

// Metadata returns the data source type name.
func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the schema for the data source.
func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about a Looker group and its user membership. Note: Role assignments cannot be read via this data source due to Looker API limitations.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Optional: true, Computed: true},
			"name": schema.StringAttribute{Optional: true, Computed: true},
			"user_count": schema.Int64Attribute{
				Description: "Number of users in the group.",
				Computed:    true,
			},
			"user_ids": schema.SetAttribute{
				Description: "IDs of users in the group.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		d.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var data groupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group v4.Group
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		group, err = d.sdk.Group(data.ID.ValueString(), groupDataSourceFields, nil)
	} else if !data.Name.IsNull() && data.Name.ValueString() != "" {
		name := data.Name.ValueString()
		fields := groupDataSourceFields
		results, e := d.sdk.SearchGroups(v4.RequestSearchGroups{Name: &name, Fields: &fields}, nil)
		err = e
		if err == nil {
			if len(results) == 0 {
				resp.Diagnostics.AddError("Not found", fmt.Sprintf("No group named %q", name))
				return
			}
			group = results[0]
		}
	} else {
		resp.Diagnostics.AddError("Invalid input", "You must provide either `id` or `name`.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Group lookup failed: %v", err))
		return
	}

	data.ID = types.StringPointerValue(group.Id)
	data.Name = types.StringPointerValue(group.Name)
	data.UserCount = types.Int64PointerValue(group.UserCount)

	// Fetch users, which is available directly
	groupUsers, err := d.sdk.AllGroupUsers(v4.RequestAllGroupUsers{GroupId: *group.Id}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to get users for group %s: %v", *group.Id, err))
		return
	}
	var userIDs []string
	for _, user := range groupUsers {
		userIDs = append(userIDs, *user.Id)
	}
	userIdsSet, diags := types.SetValueFrom(ctx, types.StringType, userIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UserIDs = userIdsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
