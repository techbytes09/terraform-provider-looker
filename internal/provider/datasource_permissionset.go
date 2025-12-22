package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const permissionSetFields = "id,name,permissions,built_in,all_access,url"

type permissionSetDataSource struct {
	sdk *v4.LookerSDK
}

type permissionSetModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	BuiltIn     types.Bool   `tfsdk:"built_in"`
	AllAccess   types.Bool   `tfsdk:"all_access"`
	Permissions types.Set    `tfsdk:"permissions"`
	URL         types.String `tfsdk:"url"`
}

func NewPermissionSetDataSource() datasource.DataSource {
	return &permissionSetDataSource{}
}

func (d *permissionSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permissionset"
}

func (d *permissionSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looker permission set (read-only). Provide exactly one of `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true},
			"name":        schema.StringAttribute{Optional: true},
			"built_in":    schema.BoolAttribute{Computed: true},
			"all_access":  schema.BoolAttribute{Computed: true},
			"permissions": schema.SetAttribute{ElementType: types.StringType, Computed: true},
			"url":         schema.StringAttribute{Computed: true},
		},
	}
}

func (d *permissionSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		d.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

func (d *permissionSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var data permissionSetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ps v4.PermissionSet
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		ps, err = d.sdk.PermissionSet(data.ID.ValueString(), permissionSetFields, nil)

	} else if !data.Name.IsNull() && data.Name.ValueString() != "" {
		name := data.Name.ValueString()
		fields := permissionSetFields
		results, e := d.sdk.SearchPermissionSets(v4.RequestSearchPermissionSets{
			Name:   &name,
			Fields: &fields,
		}, nil)
		err = e
		if err == nil {
			if len(results) == 0 {
				resp.Diagnostics.AddError("Not found", fmt.Sprintf("No permission set named %q", name))
				return
			}
			ps = results[0]
		}

	} else {
		resp.Diagnostics.AddError("Invalid input", "You must provide either `id` or `name`.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Permission set lookup failed: %v", err))
		return
	}

	// Map API response to Terraform state
	data.ID = types.StringPointerValue(ps.Id)
	data.Name = types.StringPointerValue(ps.Name)
	data.BuiltIn = types.BoolPointerValue(ps.BuiltIn)
	data.AllAccess = types.BoolPointerValue(ps.AllAccess)

	var perms []string
	if ps.Permissions != nil {
		perms = *ps.Permissions
	}
	permsSet, diags := types.SetValueFrom(ctx, types.StringType, perms)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.Permissions = permsSet

	data.URL = types.StringPointerValue(ps.Url)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
