// internal/provider/provider.go
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

var _ provider.Provider = &lookerProvider{}

type lookerProvider struct{ version string }

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &lookerProvider{version: version} }
}

type providerModel struct {
	BaseURL      types.String `tfsdk:"base_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

type clientBundle struct {
	SDK *v4.LookerSDK
}

func (p *lookerProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "looker"
	resp.Version = p.version
}

func (p *lookerProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for Looker (Google Cloud core) API 4.0",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Looker host base URL (no `/api/*`). Example: `https://myinstance.looker.com:19999`",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(8),
				},
			},
			"client_id": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *lookerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := os.Getenv("LOOKER_BASE_URL")
	clientID := os.Getenv("LOOKER_CLIENT_ID")
	clientSecret := os.Getenv("LOOKER_CLIENT_SECRET")

	if !cfg.BaseURL.IsNull() {
		baseURL = cfg.BaseURL.ValueString()
	}
	if !cfg.ClientID.IsNull() {
		clientID = cfg.ClientID.ValueString()
	}
	if !cfg.ClientSecret.IsNull() {
		clientSecret = cfg.ClientSecret.ValueString()
	}

	if baseURL == "" || clientID == "" || clientSecret == "" {
		resp.Diagnostics.AddError("Missing configuration",
			"base_url, client_id, and client_secret must be set (or via LOOKER_* env).")
		return
	}

	settings := &rtl.ApiSettings{
		BaseUrl:      baseURL,
		ClientId:     clientID,
		ClientSecret: clientSecret,
	}

	authSession := rtl.NewAuthSession(*settings)

	// Initialize the SDK
	sdk := v4.NewLookerSDK(authSession)

	// optional: quick ping to fail-fast on bad creds
	if _, err := sdk.Me("", nil); err != nil {
		resp.Diagnostics.AddError("Looker authentication failed",
			fmt.Sprintf("Failed calling /me with provided credentials: %v", err))
		return
	}

	resp.DataSourceData = &clientBundle{SDK: sdk}
	resp.ResourceData = &clientBundle{SDK: sdk}
}

func (p *lookerProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPermissionSetDataSource,
		NewModelSetDataSource,
		NewRoleDataSource,
		NewGroupDataSource,
		NewFolderDataSource,
	}
}

func (p *lookerProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPermissionSetResource,
		NewModelSetResource,
		NewRoleResource,
		NewGroupResource,
		NewRoleGroupsResource,
		NewFolderResource,
		NewFolderAccessResource,
		NewFolderPermissionOverrideResource,
	}

}
