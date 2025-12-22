package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &permissionSetResource{}
	_ resource.ResourceWithConfigure   = &permissionSetResource{}
	_ resource.ResourceWithImportState = &permissionSetResource{}
)

// permissionSetResource is the resource implementation.
type permissionSetResource struct {
	sdk *v4.LookerSDK
}

// permissionSetResourceModel maps the resource schema data.
type permissionSetResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Permissions types.Set    `tfsdk:"permissions"`
	BuiltIn     types.Bool   `tfsdk:"built_in"`
	AllAccess   types.Bool   `tfsdk:"all_access"`
	URL         types.String `tfsdk:"url"`
}

// NewPermissionSetResource is a helper function to simplify the provider implementation.
func NewPermissionSetResource() resource.Resource {
	return &permissionSetResource{}
}

// Metadata returns the resource type name.
func (r *permissionSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission_set"
}

// Schema defines the schema for the resource.
func (r *permissionSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Looker permission sets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the permission set.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the permission set.",
				Required:    true,
			},
			"permissions": schema.SetAttribute{
				Description: "The permissions of the permission set.",
				Required:    true,
				ElementType: types.StringType,
			},
			"built_in": schema.BoolAttribute{
				Description: "Whether the permission set is built-in.",
				Computed:    true,
			},
			"all_access": schema.BoolAttribute{
				Description: "Whether the permission set has all access.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the permission set.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *permissionSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *permissionSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	// Retrieve values from plan
	var plan permissionSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert permissions from types.Set to []string
	var permissions []string
	diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new permission set
	ps, err := r.sdk.CreatePermissionSet(
		v4.WritePermissionSet{
			Name:        plan.Name.ValueStringPointer(),
			Permissions: &permissions,
		},
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to create permission set: %v", err))
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringPointerValue(ps.Id)
	plan.BuiltIn = types.BoolPointerValue(ps.BuiltIn)
	plan.AllAccess = types.BoolPointerValue(ps.AllAccess)
	plan.URL = types.StringPointerValue(ps.Url)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *permissionSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	// Get current state
	var state permissionSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed permission set value from Looker
	ps, err := r.sdk.PermissionSet(state.ID.ValueString(), "", nil)
	if err != nil {
		// Handle not found error
		resp.State.RemoveResource(ctx)
		return
	}

	// Overwrite items with refreshed state
	state.Name = types.StringPointerValue(ps.Name)
	state.BuiltIn = types.BoolPointerValue(ps.BuiltIn)
	state.AllAccess = types.BoolPointerValue(ps.AllAccess)
	state.URL = types.StringPointerValue(ps.Url)

	var perms []string
	if ps.Permissions != nil {
		perms = *ps.Permissions
	}
	permsSet, diags := types.SetValueFrom(ctx, types.StringType, perms)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.Permissions = permsSet

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *permissionSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	// Retrieve values from plan
	var plan permissionSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state permissionSetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert permissions from types.Set to []string
	var permissions []string
	diags = plan.Permissions.ElementsAs(ctx, &permissions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update existing permission set
	ps, err := r.sdk.UpdatePermissionSet(
		state.ID.ValueString(),
		v4.WritePermissionSet{
			Name:        plan.Name.ValueStringPointer(),
			Permissions: &permissions,
		},
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update permission set: %v", err))
		return
	}

	// Update state with refreshed value
	plan.ID = types.StringPointerValue(ps.Id)
	plan.BuiltIn = types.BoolPointerValue(ps.BuiltIn)
	plan.AllAccess = types.BoolPointerValue(ps.AllAccess)
	plan.URL = types.StringPointerValue(ps.Url)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *permissionSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	// Retrieve values from state
	var state permissionSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing permission set
	_, err := r.sdk.DeletePermissionSet(state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to delete permission set: %v", err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *permissionSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
