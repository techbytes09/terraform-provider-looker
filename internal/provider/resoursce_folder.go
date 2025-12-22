package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

var (
	_ resource.Resource                = &folderResource{}
	_ resource.ResourceWithConfigure   = &folderResource{}
	_ resource.ResourceWithImportState = &folderResource{}
)

type folderResource struct {
	sdk *v4.LookerSDK
}

type folderResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	ParentID            types.String `tfsdk:"parent_id"`
	ContentMetadataID   types.String `tfsdk:"content_metadata_id"`
	InheritsPermissions types.Bool   `tfsdk:"inherits_permissions"`
}

func NewFolderResource() resource.Resource {
	return &folderResource{}
}

func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Looker folders (spaces).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":      schema.StringAttribute{Required: true},
			"parent_id": schema.StringAttribute{Description: "The ID of the parent folder.", Required: true},
			"content_metadata_id": schema.StringAttribute{
				Description: "The ID of the content metadata for this folder, used for access grants.",
				Computed:    true,
			},
			"inherits_permissions": schema.BoolAttribute{
				Description: "If true, the folder inherits permissions from its parent. If false, the folder has its own explicit permissions. Must be set to `false` to use `looker_folder_access` on this folder.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *folderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if cb, ok := req.ProviderData.(*clientBundle); ok && cb.SDK != nil {
		r.sdk = cb.SDK
	}
}

func (r *folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan folderResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.sdk.CreateFolder(v4.CreateFolder{
		Name:     plan.Name.ValueString(),
		ParentId: plan.ParentID.ValueString(),
	}, nil)
	if err != nil {
		resp.Diagnostics.AddError("API error on CreateFolder", fmt.Sprintf("Failed to create folder: %v", err))
		return
	}

	plan.ID = types.StringPointerValue(folder.Id)
	plan.ContentMetadataID = types.StringPointerValue(folder.ContentMetadataId)

	if !plan.InheritsPermissions.IsNull() && !plan.InheritsPermissions.ValueBool() {
		_, err := r.sdk.UpdateContentMetadata(
			*folder.ContentMetadataId,
			v4.WriteContentMeta{Inherits: types.BoolValue(false).ValueBoolPointer()},
			nil,
		)
		if err != nil {
			resp.Diagnostics.AddError("API error on UpdateContentMetadata", fmt.Sprintf("Failed to set inherits_permissions=false on folder %s: %v", *folder.Id, err))
			return
		}
		plan.InheritsPermissions = types.BoolValue(false)
	} else {
		plan.InheritsPermissions = types.BoolValue(true)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state folderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	folder, err := r.sdk.Folder(state.ID.ValueString(), "id,name,parent_id,content_metadata_id", nil)
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	contentMeta, err := r.sdk.ContentMetadata(*folder.ContentMetadataId, "inherits", nil)
	if err != nil {
		resp.Diagnostics.AddError("API error on ContentMetadata", fmt.Sprintf("Failed to read content metadata for folder %s: %v", state.ID.ValueString(), err))
		return
	}

	state.Name = types.StringValue(folder.Name)
	state.ParentID = types.StringPointerValue(folder.ParentId)
	state.ContentMetadataID = types.StringPointerValue(folder.ContentMetadataId)
	state.InheritsPermissions = types.BoolPointerValue(contentMeta.Inherits)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) || !plan.ParentID.Equal(state.ParentID) {
		_, err := r.sdk.UpdateFolder(plan.ID.ValueString(), v4.UpdateFolder{
			Name:     plan.Name.ValueStringPointer(),
			ParentId: plan.ParentID.ValueStringPointer(),
		}, nil)
		if err != nil {
			resp.Diagnostics.AddError("API error on UpdateFolder", fmt.Sprintf("Failed to update folder %s: %v", plan.ID.ValueString(), err))
			return
		}
	}

	if !plan.InheritsPermissions.Equal(state.InheritsPermissions) {
		_, err := r.sdk.UpdateContentMetadata(plan.ContentMetadataID.ValueString(),
			v4.WriteContentMeta{Inherits: plan.InheritsPermissions.ValueBoolPointer()},
			nil,
		)
		if err != nil {
			resp.Diagnostics.AddError("API error on UpdateContentMetadata", fmt.Sprintf("Failed to update inherits_permissions on folder %s: %v", plan.ID.ValueString(), err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state folderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.sdk.DeleteFolder(state.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("API error on DeleteFolder", fmt.Sprintf("Failed to delete folder %s: %v", state.ID.ValueString(), err))
		return
	}
}

func (r *folderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
