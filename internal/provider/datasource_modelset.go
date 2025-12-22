package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const modelSetFields = "id,name,models,built_in,all_access,url"

// modelSetDataSource is the data source implementation.
type modelSetDataSource struct {
	sdk *v4.LookerSDK
}

// modelSetModel maps the data source schema data.
type modelSetModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	BuiltIn   types.Bool   `tfsdk:"built_in"`
	AllAccess types.Bool   `tfsdk:"all_access"`
	Models    types.Set    `tfsdk:"models"`
	URL       types.String `tfsdk:"url"`
}

// NewModelSetDataSource is a helper function to simplify the provider implementation.
func NewModelSetDataSource() datasource.DataSource {
	return &modelSetDataSource{}
}

// Metadata returns the data source type name.
func (d *modelSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_set"
}

// Schema defines the schema for the data source.
func (d *modelSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looker model set (read-only). Provide exactly one of `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Optional: true},
			"name":       schema.StringAttribute{Optional: true},
			"built_in":   schema.BoolAttribute{Computed: true},
			"all_access": schema.BoolAttribute{Computed: true},
			"models":     schema.SetAttribute{ElementType: types.StringType, Computed: true},
			"url":        schema.StringAttribute{Computed: true},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *modelSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		d.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *modelSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var data modelSetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ms v4.ModelSet
	var err error

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		ms, err = d.sdk.ModelSet(data.ID.ValueString(), modelSetFields, nil)
	} else if !data.Name.IsNull() && data.Name.ValueString() != "" {
		name := data.Name.ValueString()
		fields := modelSetFields
		results, e := d.sdk.SearchModelSets(v4.RequestSearchModelSets{
			Name:   &name,
			Fields: &fields,
		}, nil)
		err = e
		if err == nil {
			if len(results) == 0 {
				resp.Diagnostics.AddError("Not found", fmt.Sprintf("No model set named %q", name))
				return
			}
			ms = results[0]
		}
	} else {
		resp.Diagnostics.AddError("Invalid input", "You must provide either `id` or `name`.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Model set lookup failed: %v", err))
		return
	}

	// Map API response to Terraform state
	data.ID = types.StringPointerValue(ms.Id)
	data.Name = types.StringPointerValue(ms.Name)
	data.BuiltIn = types.BoolPointerValue(ms.BuiltIn)
	data.AllAccess = types.BoolPointerValue(ms.AllAccess)
	data.URL = types.StringPointerValue(ms.Url)

	var models []string
	if ms.Models != nil {
		models = *ms.Models
	}
	modelsSet, diags := types.SetValueFrom(ctx, types.StringType, models)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.Models = modelsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
