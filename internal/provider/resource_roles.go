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

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

// roleResource is the resource implementation.
type roleResource struct {
	sdk *v4.LookerSDK
}

// roleResourceModel maps the resource schema data.
type roleResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	PermissionSetID types.String `tfsdk:"permission_set_id"`
	ModelSetID      types.String `tfsdk:"model_set_id"`
	URL             types.String `tfsdk:"url"`
}

// NewRoleResource is a helper function to simplify the provider implementation.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

// Metadata returns the resource type name.
func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Looker roles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the role.",
				Required:    true,
			},
			"permission_set_id": schema.StringAttribute{
				Description: "The ID of the permission set for this role.",
				Required:    true,
			},
			"model_set_id": schema.StringAttribute{
				Description: "The ID of the model set for this role.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the role.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.sdk.CreateRole(v4.WriteRole{
		Name:            plan.Name.ValueStringPointer(),
		PermissionSetId: plan.PermissionSetID.ValueStringPointer(),
		ModelSetId:      plan.ModelSetID.ValueStringPointer(),
	}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to create role: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(role.Id)
	plan.URL = types.StringPointerValue(role.Url)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// CORRECTED: The Role() function does not take a 'fields' argument.
	role, err := r.sdk.Role(state.ID.ValueString(), nil)
	if err != nil {
		// Handle not found error by removing the resource from state
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringPointerValue(role.Name)
	state.URL = types.StringPointerValue(role.Url)
	if role.PermissionSet != nil {
		state.PermissionSetID = types.StringPointerValue(role.PermissionSet.Id)
	}
	if role.ModelSet != nil {
		state.ModelSetID = types.StringPointerValue(role.ModelSet.Id)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state roleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.sdk.UpdateRole(state.ID.ValueString(), v4.WriteRole{
		Name:            plan.Name.ValueStringPointer(),
		PermissionSetId: plan.PermissionSetID.ValueStringPointer(),
		ModelSetId:      plan.ModelSetID.ValueStringPointer(),
	}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update role: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(role.Id)
	plan.URL = types.StringPointerValue(role.Url)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.sdk.DeleteRole(state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to delete role: %v", err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
