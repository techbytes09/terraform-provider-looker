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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

// groupResource is the resource implementation.
type groupResource struct {
	sdk *v4.LookerSDK
}

// groupResourceModel maps the resource schema data.
type groupResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	UserIDs    types.Set    `tfsdk:"user_ids"`
	UserEmails types.Set    `tfsdk:"user_emails"`
}

// NewGroupResource is a helper function to simplify the provider implementation.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

// Metadata returns the resource type name.
func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the schema for the resource.
func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Looker groups and their user memberships.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"user_ids": schema.SetAttribute{
				Description: "IDs of users to be added to the group.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"user_emails": schema.SetAttribute{
				Description: "Emails of users to be added to the group. The provider will resolve these to user IDs. Use this or `user_ids`, but not both.",
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Helper function to resolve emails to IDs
func (r *groupResource) resolveUserEmailsToIDs(ctx context.Context, emails []string) ([]string, error) {
	var resolvedIDs []string
	for _, email := range emails {
		// Search for the user by email
		results, err := r.sdk.SearchUsers(v4.RequestSearchUsers{Email: &email}, nil)
		if err != nil {
			return nil, fmt.Errorf("API error searching for user with email %s: %w", email, err)
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("no user found with email %s", email)
		}
		if len(results) > 1 {
			return nil, fmt.Errorf("multiple users found with email %s", email)
		}
		resolvedIDs = append(resolvedIDs, *results[0].Id)
	}
	return resolvedIDs, nil
}

// Create creates the resource and sets the initial Terraform state.
func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan groupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.sdk.CreateGroup(v4.WriteGroup{Name: plan.Name.ValueStringPointer()}, "", nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to create group: %v", err))
		return
	}
	plan.ID = types.StringPointerValue(group.Id)
	groupID := *group.Id

	// MODIFIED: Combine user IDs and resolved user emails
	var finalUserIDs []string
	if !plan.UserIDs.IsNull() {
		var userIDs []string
		resp.Diagnostics.Append(plan.UserIDs.ElementsAs(ctx, &userIDs, false)...)
		finalUserIDs = append(finalUserIDs, userIDs...)
	}
	if !plan.UserEmails.IsNull() {
		var userEmails []string
		resp.Diagnostics.Append(plan.UserEmails.ElementsAs(ctx, &userEmails, false)...)
		resolvedIDs, err := r.resolveUserEmailsToIDs(ctx, userEmails)
		if err != nil {
			resp.Diagnostics.AddError("User resolution failed", err.Error())
			return
		}
		finalUserIDs = append(finalUserIDs, resolvedIDs...)
	}

	for _, userID := range finalUserIDs {
		_, err := r.sdk.AddGroupUser(groupID, v4.GroupIdForGroupUserInclusion{UserId: &userID}, nil)
		if err != nil {
			resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to add user %s to group %s: %v", userID, groupID, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	groupID := state.ID.ValueString()

	group, err := r.sdk.Group(groupID, "id,name", nil)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Group %s not found, removing from state", groupID))
		resp.State.RemoveResource(ctx)
		return
	}
	state.Name = types.StringPointerValue(group.Name)

	groupUsers, err := r.sdk.AllGroupUsers(v4.RequestAllGroupUsers{GroupId: groupID}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to get users for group %s: %v", groupID, err))
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
	state.UserIDs = userIdsSet
	// NOTE: We only populate user_ids in the state, as this reflects the remote resource.
	// user_emails is treated as a write-only convenience attribute.
	state.UserEmails = types.SetNull(types.StringType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan, state groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	groupID := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		_, err := r.sdk.UpdateGroup(groupID, v4.WriteGroup{Name: plan.Name.ValueStringPointer()}, "", nil)
		if err != nil {
			resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update group name for %s: %v", groupID, err))
			return
		}
	}

	// MODIFIED: Resolve planned emails to IDs for diffing
	var planUserIDs []string
	if !plan.UserIDs.IsNull() {
		var userIDs []string
		resp.Diagnostics.Append(plan.UserIDs.ElementsAs(ctx, &userIDs, false)...)
		planUserIDs = append(planUserIDs, userIDs...)
	}
	if !plan.UserEmails.IsNull() {
		var userEmails []string
		resp.Diagnostics.Append(plan.UserEmails.ElementsAs(ctx, &userEmails, false)...)
		resolvedIDs, err := r.resolveUserEmailsToIDs(ctx, userEmails)
		if err != nil {
			resp.Diagnostics.AddError("User resolution failed", err.Error())
			return
		}
		planUserIDs = append(planUserIDs, resolvedIDs...)
	}

	var stateUserIDs []string
	resp.Diagnostics.Append(state.UserIDs.ElementsAs(ctx, &stateUserIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Standard diffing logic
	planUsers := make(map[string]bool)
	for _, id := range planUserIDs {
		planUsers[id] = true
	}
	stateUsers := make(map[string]bool)
	for _, id := range stateUserIDs {
		stateUsers[id] = true
	}

	for userID := range planUsers {
		if !stateUsers[userID] {
			_, err := r.sdk.AddGroupUser(groupID, v4.GroupIdForGroupUserInclusion{UserId: &userID}, nil)
			if err != nil {
				resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to add user %s to group %s: %v", userID, groupID, err))
				return
			}
		}
	}

	for userID := range stateUsers {
		if !planUsers[userID] {
			err := r.sdk.DeleteGroupUser(groupID, userID, nil)
			if err != nil {
				resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to remove user %s from group %s: %v", userID, groupID, err))
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state groupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	groupID := state.ID.ValueString()

	_, err := r.sdk.DeleteGroup(groupID, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to delete group %s: %v", groupID, err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
