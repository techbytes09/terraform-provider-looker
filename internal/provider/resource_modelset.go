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
	_ resource.Resource                = &modelSetResource{}
	_ resource.ResourceWithConfigure   = &modelSetResource{}
	_ resource.ResourceWithImportState = &modelSetResource{}
)

// modelSetResource is the resource implementation.
type modelSetResource struct {
	sdk *v4.LookerSDK
}

// modelSetResourceModel maps the resource schema data.
type modelSetResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Models    types.Set    `tfsdk:"models"`
	BuiltIn   types.Bool   `tfsdk:"built_in"`
	AllAccess types.Bool   `tfsdk:"all_access"`
	URL       types.String `tfsdk:"url"`
}

// NewModelSetResource is a helper function to simplify the provider implementation.
func NewModelSetResource() resource.Resource {
	return &modelSetResource{}
}

// Metadata returns the resource type name.
func (r *modelSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_set"
}

// Schema defines the schema for the resource.
func (r *modelSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Looker model sets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the model set.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the model set.",
				Required:    true,
			},
			"models": schema.SetAttribute{
				Description: "The models in the model set.",
				Required:    true,
				ElementType: types.StringType,
			},
			"built_in": schema.BoolAttribute{
				Description: "Whether the model set is built-in.",
				Computed:    true,
			},
			"all_access": schema.BoolAttribute{
				Description: "Whether the model set has all access.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the model set.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *modelSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	} else if req.ProviderData != nil {
		resp.Diagnostics.AddError("Unexpected provider data", "Missing Looker SDK client")
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *modelSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan modelSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var models []string
	diags = plan.Models.ElementsAs(ctx, &models, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ms, err := r.sdk.CreateModelSet(v4.WriteModelSet{
		Name:   plan.Name.ValueStringPointer(),
		Models: &models,
	}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to create model set: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(ms.Id)
	plan.BuiltIn = types.BoolPointerValue(ms.BuiltIn)
	plan.AllAccess = types.BoolPointerValue(ms.AllAccess)
	plan.URL = types.StringPointerValue(ms.Url)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *modelSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state modelSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ms, err := r.sdk.ModelSet(state.ID.ValueString(), "", nil)
	if err != nil {
		// Handle not found error
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringPointerValue(ms.Name)
	state.BuiltIn = types.BoolPointerValue(ms.BuiltIn)
	state.AllAccess = types.BoolPointerValue(ms.AllAccess)
	state.URL = types.StringPointerValue(ms.Url)

	var models []string
	if ms.Models != nil {
		models = *ms.Models
	}
	modelsSet, diags := types.SetValueFrom(ctx, types.StringType, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Models = modelsSet

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *modelSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var plan modelSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelSetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var models []string
	diags = plan.Models.ElementsAs(ctx, &models, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ms, err := r.sdk.UpdateModelSet(state.ID.ValueString(), v4.WriteModelSet{
		Name:   plan.Name.ValueStringPointer(),
		Models: &models,
	}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to update model set: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(ms.Id)
	plan.BuiltIn = types.BoolPointerValue(ms.BuiltIn)
	plan.AllAccess = types.BoolPointerValue(ms.AllAccess)
	plan.URL = types.StringPointerValue(ms.Url)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *modelSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.sdk == nil {
		resp.Diagnostics.AddError("Unconfigured client", "Provider did not set Looker SDK client")
		return
	}

	var state modelSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.sdk.DeleteModelSet(state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("API error", fmt.Sprintf("Failed to delete model set: %v", err))
		return
	}
}

// ImportState imports the resource into the Terraform state.
func (r *modelSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
