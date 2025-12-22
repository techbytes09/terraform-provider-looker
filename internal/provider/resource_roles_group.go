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
	_ resource.Resource                = &roleGroupsResource{}
	_ resource.ResourceWithConfigure   = &roleGroupsResource{}
	_ resource.ResourceWithImportState = &roleGroupsResource{}
)

// roleGroupsResource is the resource implementation.
type roleGroupsResource struct {
	sdk *v4.LookerSDK
}

// roleGroupsResourceModel maps the resource schema data.
type roleGroupsResourceModel struct {
	ID       types.String `tfsdk:"id"`
	RoleID   types.String `tfsdk:"role_id"`
	GroupIDs types.Set    `tfsdk:"group_ids"`
}

// NewRoleGroupsResource is a helper function to simplify the provider implementation.
func NewRoleGroupsResource() resource.Resource {
	return &roleGroupsResource{}
}

// Metadata returns the resource type name.
func (r *roleGroupsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_groups"
}

// Schema defines the schema for the resource.
func (r *roleGroupsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a set of groups to a single Looker role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role.",
				Required:    true,
			},
			"group_ids": schema.SetAttribute{
				Description: "The IDs of the groups to assign to the role.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *roleGroupsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// setRoleGroups is a helper function for Create and Update.
func (r *roleGroupsResource) setRoleGroups(ctx context.Context, plan *roleGroupsResourceModel) error {
	var groupIDs []string
	diags := plan.GroupIDs.ElementsAs(ctx, &groupIDs, false)
	if diags.HasError() {
		return fmt.Errorf("could not get group IDs from plan")
	}

	_, err := r.sdk.SetRoleGroups(plan.RoleID.ValueString(), groupIDs, nil)
	return err
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleGroupsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan roleGroupsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.setRoleGroups(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to set groups for role %s: %v", plan.RoleID.ValueString(), err))
		return
	}

	plan.ID = plan.RoleID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleGroupsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state roleGroupsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	roleID := state.RoleID.ValueString()

	// The SDK method to get groups for a role is RoleGroups.
	groups, err := r.sdk.RoleGroups(roleID, "id", nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to read groups for role %s: %v", roleID, err))
		return
	}

	var groupIDs []string
	for _, group := range groups {
		groupIDs = append(groupIDs, *group.Id)
	}

	groupIDsSet, diags := types.SetValueFrom(ctx, types.StringType, groupIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.GroupIDs = groupIDsSet
	state.ID = state.RoleID

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleGroupsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan roleGroupsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update is the same as create: we just set the complete list of groups.
	err := r.setRoleGroups(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update groups for role %s: %v", plan.RoleID.ValueString(), err))
		return
	}

	plan.ID = plan.RoleID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource. This means setting the groups for the role to an empty list.
func (r *roleGroupsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state roleGroupsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deleting the assignment means setting the list of groups to empty.
	_, err := r.sdk.SetRoleGroups(state.RoleID.ValueString(), []string{}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to clear groups for role %s: %v", state.RoleID.ValueString(), err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *roleGroupsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the role_id
	resource.ImportStatePassthroughID(ctx, path.Root("role_id"), req, resp)
}
