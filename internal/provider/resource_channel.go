package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &channelResource{}
var _ resource.ResourceWithConfigure = &channelResource{}
var _ resource.ResourceWithImportState = &channelResource{}

func init() {
	registeredResources = append(registeredResources, NewChannelResource)
}

// lowercaseValidator rejects a value Open WebUI would rewrite. A channel name
// is lowercased on create and stored verbatim on update, so a name holding an
// uppercase letter applies as one value and updates to another. A user's mail
// address is lowercased on every write.
type lowercaseValidator struct{}

func (v lowercaseValidator) Description(_ context.Context) string {
	return "must be lowercase"
}

func (v lowercaseValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v lowercaseValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == strings.ToLower(value) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Value must be lowercase",
		fmt.Sprintf("Open WebUI lowercases this value when it stores it, so %q would apply as %q. Write the lowercase form.", value, strings.ToLower(value)),
	)
}

// channelResource manages Open WebUI channels.
type channelResource struct {
	client *client.Client
}

// channelResourceModel captures Terraform state for channels.
type channelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsPrivate   types.Bool   `tfsdk:"is_private"`
	DataJSON    types.String `tfsdk:"data_json"`
	MetaJSON    types.String `tfsdk:"meta_json"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	ReadUsers   types.List   `tfsdk:"read_users"`
	WriteUsers  types.List   `tfsdk:"write_users"`
	PublicRead  types.Bool   `tfsdk:"public_read"`
	PublicWrite types.Bool   `tfsdk:"public_write"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
}

// NewChannelResource constructs a new channel resource.
func NewChannelResource() resource.Resource {
	return &channelResource{}
}

// Metadata sets the resource type name.
func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

// Schema defines the channel resource schema.
func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a standard channel in Open WebUI. Every user whose access grants allow it sees a standard channel, and only an admin can create one. " +
			"The group and direct-message rooms users create for themselves are a different kind of channel and are not managed here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "UUID assigned by Open WebUI on create.",
				MarkdownDescription: "UUID assigned by Open WebUI on create.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the channel. Must be lowercase, because Open WebUI lowercases the name it stores on create.",
				MarkdownDescription: "Name of the channel. Must be lowercase, because Open WebUI lowercases the name it stores on create.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					lowercaseValidator{},
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "Description shown under the channel name.",
				MarkdownDescription: "Description shown under the channel name.",
			},
			"is_private": schema.BoolAttribute{
				Optional:            true,
				Description:         "Whether the channel is private.",
				MarkdownDescription: "Whether the channel is private.",
			},
			"data_json": schema.StringAttribute{
				Optional:            true,
				Description:         "Free-form channel data as a JSON object.",
				MarkdownDescription: "Free-form channel data as a JSON object.",
			},
			"meta_json": schema.StringAttribute{
				Optional:            true,
				Description:         "Free-form channel metadata as a JSON object.",
				MarkdownDescription: "Free-form channel metadata as a JSON object.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs whose members can read the channel.",
				MarkdownDescription: "List of group names or IDs whose members can read the channel.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs whose members can post in the channel.",
				MarkdownDescription: "List of group names or IDs whose members can post in the channel.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"read_users": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of user email addresses or IDs allowed to read the channel.",
				MarkdownDescription: "List of user email addresses or IDs allowed to read the channel.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_users": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of user email addresses or IDs allowed to post in the channel. Name the same address in `read_users` as well.",
				MarkdownDescription: "List of user email addresses or IDs allowed to post in the channel. Name the same address in `read_users` as well.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"public_read": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can read the channel. This is the sharing the web UI calls public.",
				MarkdownDescription: "Whether every signed-in user can read the channel. This is the sharing the web UI calls public.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"public_write": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can post in the channel.",
				MarkdownDescription: "Whether every signed-in user can post in the channel.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of the account that created the channel. Set by Open WebUI.",
				MarkdownDescription: "Identifier of the account that created the channel. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Creation timestamp in nanoseconds. Set by Open WebUI.",
				MarkdownDescription: "Creation timestamp in nanoseconds. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Last update timestamp in nanoseconds. Set by Open WebUI.",
				MarkdownDescription: "Last update timestamp in nanoseconds. Set by Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *channelResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a channel.
func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing channels.")
		return
	}

	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := channelFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateChannel(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create channel failed", err.Error())
		return
	}

	state, stateDiags := channelToModel(ctx, r.client, created, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes channel state.
func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing channels.")
		return
	}

	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channel, err := r.client.GetChannel(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read channel failed", err.Error())
		return
	}

	updated, diags := channelToModel(ctx, r.client, channel, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update mutates channel properties.
func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing channels.")
		return
	}

	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := channelFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateChannel(ctx, state.ID.ValueString(), form)
	if err != nil {
		resp.Diagnostics.AddError("Update channel failed", err.Error())
		return
	}

	newState, stateDiags := channelToModel(ctx, r.client, updated, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete removes a channel.
func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing channels.")
		return
	}

	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteChannel(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete channel failed", err.Error())
		return
	}
}

// ImportState maps an import identifier onto the id attribute.
func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// channelFormFromPlan builds the write payload. Every field goes on the wire,
// because the update handler assigns each one from the request it receives.
func channelFormFromPlan(ctx context.Context, apiClient *client.Client, plan channelResourceModel) (client.ChannelForm, diag.Diagnostics) {
	var diags diag.Diagnostics

	data := decodeOptionalJSON(plan.DataJSON, path.Root("data_json"), &diags)
	meta := decodeOptionalJSON(plan.MetaJSON, path.Root("meta_json"), &diags)

	principals := resolveAccessPrincipals(ctx, apiClient, accessPrincipalLists{
		ReadGroups:  plan.ReadGroups,
		WriteGroups: plan.WriteGroups,
		ReadUsers:   plan.ReadUsers,
		WriteUsers:  plan.WriteUsers,
	}, &diags)

	publicRead := !plan.PublicRead.IsNull() && !plan.PublicRead.IsUnknown() && plan.PublicRead.ValueBool()
	publicWrite := !plan.PublicWrite.IsNull() && !plan.PublicWrite.IsUnknown() && plan.PublicWrite.ValueBool()

	accessControl := withPublicAccess(buildAccessControl(principals), publicRead, publicWrite)

	return client.ChannelForm{
		Name:          plan.Name.ValueString(),
		Description:   stringPtr(plan.Description),
		IsPrivate:     boolPtr(plan.IsPrivate),
		Data:          data,
		Meta:          meta,
		AccessControl: accessControl,
	}, diags
}

// channelToModel converts an API channel into Terraform state. The recorded
// model supplies the JSON text already in the plan or state, so that re-encoding
// the server's answer cannot show a difference in key order as drift.
func channelToModel(ctx context.Context, apiClient *client.Client, channel *client.Channel, recorded channelResourceModel) (channelResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if channel == nil {
		diags.AddError("Missing channel response", "Channel details were not returned by the Open WebUI API.")
		return channelResourceModel{}, diags
	}

	shared, sharedDiags := flattenAccessPrincipals(ctx, apiClient, channel.AccessControl)
	diags.Append(sharedDiags...)

	dataJSON, dataDiags := jsonRefreshValue(recorded.DataJSON, channel.Data, "data_json")
	diags.Append(dataDiags...)
	metaJSON, metaDiags := jsonRefreshValue(recorded.MetaJSON, channel.Meta, "meta_json")
	diags.Append(metaDiags...)

	description := types.StringNull()
	if channel.Description != nil {
		description = types.StringValue(*channel.Description)
	}

	isPrivate := types.BoolNull()
	if channel.IsPrivate != nil {
		isPrivate = types.BoolValue(*channel.IsPrivate)
	}

	state := channelResourceModel{
		ID:          types.StringValue(channel.ID),
		Name:        types.StringValue(channel.Name),
		Description: description,
		IsPrivate:   isPrivate,
		DataJSON:    dataJSON,
		MetaJSON:    metaJSON,
		ReadGroups:  shared.ReadGroups,
		WriteGroups: shared.WriteGroups,
		ReadUsers:   shared.ReadUsers,
		WriteUsers:  shared.WriteUsers,
		PublicRead:  types.BoolValue(publicAccessFromControl(channel.AccessControl, "read")),
		PublicWrite: types.BoolValue(publicAccessFromControl(channel.AccessControl, "write")),
		UserID:      types.StringValue(channel.UserID),
		CreatedAt:   types.Int64Value(channel.CreatedAt),
		UpdatedAt:   types.Int64Value(channel.UpdatedAt),
	}

	return state, diags
}

func boolPtr(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}
