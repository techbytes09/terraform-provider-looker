package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

// folderDataSource is the data source implementation.
type folderDataSource struct {
	sdk *v4.LookerSDK
}

// folderDataSourceModel maps the data source schema data.
type folderDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ParentID          types.String `tfsdk:"parent_id"`
	ContentMetadataID types.String `tfsdk:"content_metadata_id"`
	IsPersonal        types.Bool   `tfsdk:"is_personal"`
}

// NewFolderDataSource is a helper function.
func NewFolderDataSource() datasource.DataSource {
	return &folderDataSource{}
}

// Metadata returns the data source type name.
func (d *folderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

// Schema defines the schema for the data source.
func (d *folderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about a Looker folder (space). Specify `id` to look up by ID, or both `name` and `parent_id` to look up by name within a parent.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Optional: true, Computed: true},
			"name":                schema.StringAttribute{Optional: true, Computed: true},
			"parent_id":           schema.StringAttribute{Optional: true, Computed: true},
			"content_metadata_id": schema.StringAttribute{Computed: true},
			"is_personal":         schema.BoolAttribute{Computed: true},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *folderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		d.sdk = cb.SDK
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *folderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data folderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var folder *v4.Folder
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		f, e := d.sdk.Folder(data.ID.ValueString(), "", nil)
		err = e
		if err == nil {
			folder = &f
		}
	} else if !data.Name.IsNull() && !data.ParentID.IsNull() {
		name := data.Name.ValueString()
		parentID := data.ParentID.ValueString()
		results, e := d.sdk.SearchFolders(v4.RequestSearchFolders{Name: &name, ParentId: &parentID}, nil)
		err = e
		if err == nil {
			if len(results) == 0 {
				resp.Diagnostics.AddError("Not found", fmt.Sprintf("No folder named %q found in parent folder %s", name, parentID))
				return
			}
			if len(results) > 1 {
				resp.Diagnostics.AddError("Multiple found", fmt.Sprintf("Found %d folders named %q in parent folder %s", len(results), name, parentID))
				return
			}
			folder = &results[0]
		}
	} else {
		resp.Diagnostics.AddError("Invalid input", "You must provide either `id` or both `name` and `parent_id`.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Folder lookup failed: %v", err))
		return
	}

	data.ID = types.StringPointerValue(folder.Id)
	// CORRECTED: Use StringValue for the non-pointer 'Name' field.
	data.Name = types.StringValue(folder.Name)
	data.ParentID = types.StringPointerValue(folder.ParentId)
	data.ContentMetadataID = types.StringPointerValue(folder.ContentMetadataId)
	data.IsPersonal = types.BoolPointerValue(folder.IsPersonal)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
