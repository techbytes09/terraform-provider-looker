package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

var (
	_ resource.Resource                = &folderAccessResource{}
	_ resource.ResourceWithConfigure   = &folderAccessResource{}
	_ resource.ResourceWithImportState = &folderAccessResource{}
)

// folderAccessResource is the resource implementation.
type folderAccessResource struct {
	sdk *v4.LookerSDK
}

// folderAccessResourceModel maps the resource schema data.
type folderAccessResourceModel struct {
	ID          types.String `tfsdk:"id"`
	FolderID    types.String `tfsdk:"folder_id"`
	GroupID     types.String `tfsdk:"group_id"`
	AccessLevel types.String `tfsdk:"access_level"`
}

// NewFolderAccessResource is a helper function to simplify the provider implementation.
func NewFolderAccessResource() resource.Resource {
	return &folderAccessResource{}
}

// Metadata returns the resource type name.
func (r *folderAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder_access"
}

// Schema defines the schema for the resource.
func (r *folderAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages content access grants for a Looker folder (space). This resource links a group to a folder with a specific access level.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique ID of this access grant.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"folder_id": schema.StringAttribute{
				Description: "The ID of the folder (content_metadata_id) to grant access to.",
				Required:    true,
			},
			"group_id": schema.StringAttribute{
				Description: "The ID of the group to grant access to.",
				Required:    true,
			},
			"access_level": schema.StringAttribute{
				Description: "The access level to grant. Valid values are: `view` (View), `edit` (Manage Access, Edit).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("view", "edit"),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *folderAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *folderAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan folderAccessResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessLevelString := plan.AccessLevel.ValueString()
	permissionType := v4.PermissionType(accessLevelString)

	accessGrant, err := r.sdk.CreateContentMetadataAccess(
		v4.ContentMetaGroupUser{
			ContentMetadataId: plan.FolderID.ValueStringPointer(),
			GroupId:           plan.GroupID.ValueStringPointer(),
			PermissionType:    &permissionType,
		},
		false, // sendBoardsNotificationEmail
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to create folder access grant: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(accessGrant.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// findAccessGrant is a helper to locate a specific grant for a folder and group.
// CORRECTED: The unused 'ctx' parameter is renamed to '_' to satisfy the compiler.
func (r *folderAccessResource) findAccessGrant(_ context.Context, folderID, groupID string) (*v4.ContentMetaGroupUser, error) {
	results, err := r.sdk.AllContentMetadataAccesses(folderID, "", nil)
	if err != nil {
		return nil, fmt.Errorf("API error searching for access grants on folder %s: %w", folderID, err)
	}
	for _, grant := range results {
		if grant.GroupId != nil && *grant.GroupId == groupID {
			return &grant, nil
		}
	}
	return nil, nil // Not found
}

// Read refreshes the Terraform state with the latest data.
func (r *folderAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state folderAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant, err := r.findAccessGrant(ctx, state.FolderID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read error", err.Error())
		return
	}
	if grant == nil {
		tflog.Warn(ctx, fmt.Sprintf("Folder access grant for group %s on folder %s not found, removing from state.", state.GroupID.ValueString(), state.FolderID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringPointerValue(grant.Id)
	if grant.PermissionType != nil {
		state.AccessLevel = types.StringValue(string(*grant.PermissionType))
	} else {
		state.AccessLevel = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *folderAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state folderAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessLevelString := plan.AccessLevel.ValueString()
	permissionType := v4.PermissionType(accessLevelString)

	_, err := r.sdk.UpdateContentMetadataAccess(
		state.ID.ValueString(),
		v4.ContentMetaGroupUser{
			PermissionType: &permissionType,
		},
		nil,
	)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update folder access grant %s: %v", state.ID.ValueString(), err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *folderAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state folderAccessResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.sdk.DeleteContentMetadataAccess(state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to delete folder access grant %s: %v", state.ID.ValueString(), err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *folderAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: <folder_id>/<group_id>. Got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("folder_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
}
