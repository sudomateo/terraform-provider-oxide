// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/oxidecomputer/oxide.go/oxide"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = (*imageResource)(nil)
	_ resource.ResourceWithConfigure = (*imageResource)(nil)
)

// NewImageResource is a helper function to simplify the provider implementation.
func NewImageResource() resource.Resource {
	return &imageResource{}
}

// imageResource is the resource implementation.
type imageResource struct {
	client *oxide.Client
}

type imageResourceModel struct {
	BlockSize        types.Int64    `tfsdk:"block_size"`
	Description      types.String   `tfsdk:"description"`
	Digest           types.Object   `tfsdk:"digest"`
	ID               types.String   `tfsdk:"id"`
	Name             types.String   `tfsdk:"name"`
	OS               types.String   `tfsdk:"os"`
	ProjectID        types.String   `tfsdk:"project_id"`
	Size             types.Int64    `tfsdk:"size"`
	SourceSnapshotID types.String   `tfsdk:"source_snapshot_id"`
	SourceURL        types.String   `tfsdk:"source_url"`
	TimeCreated      types.String   `tfsdk:"time_created"`
	TimeModified     types.String   `tfsdk:"time_modified"`
	Version          types.String   `tfsdk:"version"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

type imageResourceDigestModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// Metadata returns the resource type name.
func (r *imageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "oxide_image"
}

// Configure adds the provider configured client to the data source.
func (r *imageResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*oxide.Client)
}

func (r *imageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Schema defines the schema for the resource.
func (r *imageResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the image.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the project that will contain the image.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Description for the image.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os": schema.StringAttribute{
				Required:    true,
				Description: "OS image distribution. Example: alpine",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Required:    true,
				Description: "OS image version. Example: 3.16.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"block_size": schema.Int64Attribute{
				Optional:    true,
				Description: "Size of blocks in bytes.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"source_snapshot_id": schema.StringAttribute{
				Optional:    true,
				Description: "Snapshot ID of the image source if applicable.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("source_url"),
						path.MatchRoot("source_snapshot_id"),
					}...),
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("block_size"),
					}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL source of this image, if applicable.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.Expressions{
						path.MatchRoot("block_size"),
					}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				// TODO: Restore once updates and deletes are enabled
				// Update: true,
				// Delete: true,
			}),
			"digest": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Hash of the image contents, if applicable.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Digest type.",
						Computed:    true,
					},
					"value": schema.StringAttribute{
						Description: "Digest type value.",
						Computed:    true,
					},
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique, immutable, system-controlled identifier of the image.",
			},
			"size": schema.Int64Attribute{
				Computed:    true,
				Description: "Total size in bytes.",
			},
			"time_created": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this image was created.",
			},
			"time_modified": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of when this image was last modified.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *imageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan imageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, defaultTimeout())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	params := oxide.ImageCreateParams{
		Project: oxide.NameOrId(plan.ProjectID.ValueString()),
		Body: &oxide.ImageCreate{
			Description: plan.Description.ValueString(),
			Name:        oxide.Name(plan.Name.ValueString()),
			Os:          plan.OS.ValueString(),
			Version:     plan.Version.ValueString(),
		},
	}

	is := oxide.ImageSource{}
	if !plan.SourceSnapshotID.IsNull() {
		is.Id = plan.SourceSnapshotID.ValueString()
		is.Type = oxide.ImageSourceTypeSnapshot
	} else if !plan.SourceURL.IsNull() {
		is.Id = plan.SourceURL.ValueString()
		is.Type = oxide.ImageSourceTypeUrl
		// TODO: Remove before releasing, for testing purposes only
		if plan.SourceURL.Equal(types.StringValue("you_can_boot_anything_as_long_as_its_alpine")) {
			is.Type = oxide.ImageSourceTypeYouCanBootAnythingAsLongAsItsAlpine
		}
		is.BlockSize = oxide.BlockSize(plan.BlockSize.ValueInt64())
	} else {
		resp.Diagnostics.AddError(
			"Error creating image",
			"One of `source_url` or `source_snapshot_id` must be set",
		)
		return
	}
	params.Body.Source = is

	image, err := r.client.ImageCreate(params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating image",
			"API error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created image with ID: %v", image.Id), map[string]any{"success": true})

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(image.Id)
	plan.Size = types.Int64Value(int64(image.Size))
	plan.TimeCreated = types.StringValue(image.TimeCreated.String())
	plan.TimeModified = types.StringValue(image.TimeModified.String())
	plan.Version = types.StringValue(image.Version)

	// Parse imageResourceDigestModel into types.Object
	dm := imageResourceDigestModel{
		Type:  types.StringValue(string(image.Digest.Type)),
		Value: types.StringValue(image.Digest.Value),
	}
	attributeTypes := map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	}
	digest, diags := types.ObjectValueFrom(ctx, attributeTypes, dm)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Digest = digest

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *imageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state imageResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, defaultTimeout())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	image, err := r.client.ImageView(oxide.ImageViewParams{
		Image: oxide.NameOrId(state.ID.ValueString()),
	})
	if err != nil {
		if is404(err) {
			// Remove resource from state during a refresh
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read image:",
			"API error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("read image with ID: %v", image.Id), map[string]any{"success": true})

	state.BlockSize = types.Int64Value(int64(image.BlockSize))
	state.Description = types.StringValue(image.Description)
	state.ID = types.StringValue(image.Id)
	state.Name = types.StringValue(string(image.Name))
	state.OS = types.StringValue(image.Os)
	state.Size = types.Int64Value(int64(image.Size))
	state.TimeCreated = types.StringValue(image.TimeCreated.String())
	state.TimeModified = types.StringValue(image.TimeModified.String())
	state.Version = types.StringValue(image.Version)

	// Only set ProjectID and SourceURL if they exist to avoid unintentional drift.
	// Some images with silo visibility may not have project IDs, and could be imported.
	if image.ProjectId != "" {
		state.ProjectID = types.StringValue(image.ProjectId)
	}
	if image.Url != "" {
		state.SourceURL = types.StringValue(image.Url)
	}

	// Parse imageResourceDigestModel into types.Object
	dm := imageResourceDigestModel{
		Type:  types.StringValue(string(image.Digest.Type)),
		Value: types.StringValue(image.Digest.Value),
	}
	attributeTypes := map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	}
	digest, diags := types.ObjectValueFrom(ctx, attributeTypes, dm)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Digest = digest

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *imageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Error updating image",
		"the oxide API currently does not support updating images")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *imageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Error deleting image",
		"the oxide API currently does not support deleting images")

	// TODO: Uncomment once image delete is enabled in the API
	//
	//	var state imageResourceModel
	//
	//	// Read Terraform prior state data into the model
	//	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	//	if resp.Diagnostics.HasError() {
	//		return
	//	}
	//
	//	if err := r.client.ImageDelete(oxide.ImageDeleteParams{
	//		Image: oxide.NameOrId(state.ID.ValueString()),
	//	}); err != nil {
	//
	//		resp.Diagnostics.AddError(
	//			"Unable to read image:",
	//			"API error: "+err.Error(),
	//		)
	//		return
	//	}
}
