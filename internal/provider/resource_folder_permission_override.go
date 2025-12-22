package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &folderPermissionOverrideResource{}
	_ resource.ResourceWithConfigure   = &folderPermissionOverrideResource{}
	_ resource.ResourceWithImportState = &folderPermissionOverrideResource{}
)

type folderPermissionOverrideResource struct {
	sdk *v4.LookerSDK
}
type folderPermissionOverrideResourceModel struct {
	ID          types.String `tfsdk:"id"`
	FolderID    types.String `tfsdk:"folder_id"`
	GroupID     types.String `tfsdk:"group_id"`
	AccessLevel types.String `tfsdk:"access_level"`
}

func NewFolderPermissionOverrideResource() resource.Resource {
	return &folderPermissionOverrideResource{}
}

func (r *folderPermissionOverrideResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder_permission_override"
}

func (r *folderPermissionOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a folder permission override. This resource finds an existing, inherited access grant for a group on a folder and updates it to a new, direct access level (e.g., from inherited 'view' to direct 'edit').",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Description: "The unique ID of the access grant that was updated.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"folder_id": schema.StringAttribute{Description: "The ID of the folder (content_metadata_id) whose permissions will be overridden.", Required: true},
			"group_id":  schema.StringAttribute{Description: "The ID of the group whose inherited permission will be overridden.", Required: true},
			"access_level": schema.StringAttribute{
				Description: "The new, direct access level to set. Valid values are: `view` or `edit`.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf("view", "edit")},
			},
		},
	}
}

func (r *folderPermissionOverrideResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	}
}

func (r *folderPermissionOverrideResource) findAccessGrant(_ context.Context, folderID, groupID string) (*v4.ContentMetaGroupUser, error) {
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

func (r *folderPermissionOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan folderPermissionOverrideResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	folderID := plan.FolderID.ValueString()
	groupID := plan.GroupID.ValueString()

	grant, err := r.findAccessGrant(ctx, folderID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("API Error on Find", err.Error())
		return
	}
	if grant == nil {
		resp.Diagnostics.AddError("Cannot Override Permission", fmt.Sprintf("No inherited permission found for group %s on folder %s to override. The group must have parent access first.", groupID, folderID))
		return
	}

	accessLevelString := plan.AccessLevel.ValueString()
	permissionType := v4.PermissionType(accessLevelString)

	updatedGrant, err := r.sdk.UpdateContentMetadataAccess(*grant.Id, v4.ContentMetaGroupUser{PermissionType: &permissionType}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API Error on Update", fmt.Sprintf("Failed to update folder access grant %s: %v", *grant.Id, err))
		return
	}

	plan.ID = types.StringPointerValue(updatedGrant.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *folderPermissionOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state folderPermissionOverrideResourceModel
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
	if grant == nil || grant.PermissionType == nil || string(*grant.PermissionType) != state.AccessLevel.ValueString() {
		tflog.Warn(ctx, "Permission override no longer exists or has been changed externally. Will re-apply on next run.")
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringPointerValue(grant.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderPermissionOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state folderPermissionOverrideResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-use the Create logic as "Update" means finding and updating the inherited grant.
	r.Create(ctx, resource.CreateRequest{Plan: req.Plan}, &resource.CreateResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *folderPermissionOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Warn(ctx, "Deleting a 'looker_folder_permission_override' does not automatically revert the folder to inherited permissions. Please manage permissions in the Looker UI if reversion is needed.")
}

func (r *folderPermissionOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError("Not Implemented", "Import is not supported for this resource.")
}
