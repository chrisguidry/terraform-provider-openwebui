package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &adminConfigResource{}
var _ resource.ResourceWithConfigure = &adminConfigResource{}
var _ resource.ResourceWithImportState = &adminConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewAdminConfigResource)
}

// jwtExpiryPattern is the expiry format Open WebUI accepts: -1 or 0 for no
// expiry, or a number with a unit. A value that fails this test is dropped on
// write while the request still answers 200.
var jwtExpiryPattern = regexp.MustCompile(`^(-1|0|(-?\d+(\.\d+)?)(ms|s|m|h|d|w))$`)

// countLimitPattern is the format of the three limits Open WebUI stores as
// `int | str`. It casts the value to an int when it is truthy and stores an
// empty string otherwise, so 0 cannot be told apart from unset once written.
var countLimitPattern = regexp.MustCompile(`^([1-9][0-9]*)?$`)

// adminConfigResource manages instance-wide administrative settings.
type adminConfigResource struct {
	client *client.Client
}

type adminConfigModel struct {
	ID                                types.String `tfsdk:"id"`
	ShowAdminDetails                  types.Bool   `tfsdk:"show_admin_details"`
	AdminEmail                        types.String `tfsdk:"admin_email"`
	WebUIURL                          types.String `tfsdk:"webui_url"`
	EnableSignup                      types.Bool   `tfsdk:"enable_signup"`
	EnableAPIKeys                     types.Bool   `tfsdk:"enable_api_keys"`
	EnableAPIKeysEndpointRestrictions types.Bool   `tfsdk:"enable_api_keys_endpoint_restrictions"`
	APIKeysAllowedEndpoints           types.String `tfsdk:"api_keys_allowed_endpoints"`
	DefaultUserRole                   types.String `tfsdk:"default_user_role"`
	DefaultGroupID                    types.String `tfsdk:"default_group_id"`
	JWTExpiresIn                      types.String `tfsdk:"jwt_expires_in"`
	EnableCommunitySharing            types.Bool   `tfsdk:"enable_community_sharing"`
	EnableMessageRating               types.Bool   `tfsdk:"enable_message_rating"`
	EnableFolders                     types.Bool   `tfsdk:"enable_folders"`
	FolderMaxFileCount                types.String `tfsdk:"folder_max_file_count"`
	AutomationMaxCount                types.String `tfsdk:"automation_max_count"`
	AutomationMinInterval             types.String `tfsdk:"automation_min_interval"`
	EnableAutomations                 types.Bool   `tfsdk:"enable_automations"`
	EnableChannels                    types.Bool   `tfsdk:"enable_channels"`
	ChannelModelResponseMode          types.String `tfsdk:"channel_model_response_mode"`
	EnableCalendar                    types.Bool   `tfsdk:"enable_calendar"`
	EnableMemories                    types.Bool   `tfsdk:"enable_memories"`
	EnableMemorySystemContext         types.Bool   `tfsdk:"enable_memory_system_context"`
	EnableNotes                       types.Bool   `tfsdk:"enable_notes"`
	EnableUserWebhooks                types.Bool   `tfsdk:"enable_user_webhooks"`
	EnableUserStatus                  types.Bool   `tfsdk:"enable_user_status"`
	PendingUserOverlayTitle           types.String `tfsdk:"pending_user_overlay_title"`
	PendingUserOverlayContent         types.String `tfsdk:"pending_user_overlay_content"`
	ResponseWatermark                 types.String `tfsdk:"response_watermark"`
}

// NewAdminConfigResource constructs a new admin config resource.
func NewAdminConfigResource() resource.Resource {
	return &adminConfigResource{}
}

// Metadata sets the resource type name.
func (r *adminConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_config"
}

// adminBoolAttribute builds one of the many on/off settings this resource
// carries. Each one is optional, and an attribute the configuration leaves out
// keeps the value the instance already holds.
func adminBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

// adminStringAttribute builds a text setting, with optional validators.
func adminStringAttribute(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		Validators:          validators,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// Schema defines the admin config schema.
func (r *adminConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the instance-wide administrative settings of Open WebUI: signup, the default user role, API keys, token expiry, and the feature switches for folders, automations, channels, calendar, memories, and notes.\n\n" +
			"Open WebUI drops a value it considers invalid and still answers 200, so the default user role, the channel response mode, and the token expiry are validated here instead. A rejected value fails the plan rather than applying as a silent no-op.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"show_admin_details":                    adminBoolAttribute("Whether the sign-in page shows the administrator's contact details."),
			"admin_email":                           adminStringAttribute("Contact address shown to users waiting for approval."),
			"webui_url":                             adminStringAttribute("Public URL of this Open WebUI instance."),
			"enable_signup":                         adminBoolAttribute("Whether visitors can create their own accounts."),
			"enable_api_keys":                       adminBoolAttribute("Whether users can mint API keys."),
			"enable_api_keys_endpoint_restrictions": adminBoolAttribute("Whether API keys are limited to the endpoints named in `api_keys_allowed_endpoints`."),
			"api_keys_allowed_endpoints":            adminStringAttribute("Comma-separated endpoints an API key may call. Applies when `enable_api_keys_endpoint_restrictions` is true."),
			"default_user_role": adminStringAttribute(
				"Role given to a new account: `pending`, `user`, or `admin`.",
				stringvalidator.OneOf("pending", "user", "admin"),
			),
			"default_group_id": adminStringAttribute("Identifier of the group every new account joins. Empty for none."),
			"jwt_expires_in": adminStringAttribute(
				"Session token lifetime, e.g. `4h` or `30d`. Use `-1` for no expiry.",
				stringvalidator.RegexMatches(jwtExpiryPattern, "must be -1, 0, or a number with a unit of ms, s, m, h, d, or w"),
			),
			"enable_community_sharing": adminBoolAttribute("Whether users can share chats to the Open WebUI community site."),
			"enable_message_rating":    adminBoolAttribute("Whether users can rate model responses."),
			"enable_folders":           adminBoolAttribute("Whether users can organise chats into folders."),
			"folder_max_file_count": adminStringAttribute(
				"Maximum number of files in a folder, as a string. Empty means no limit; Open WebUI stores 0 as empty, so it is not a value this attribute can hold.",
				stringvalidator.RegexMatches(countLimitPattern, "must be empty or a positive whole number"),
			),
			"automation_max_count": adminStringAttribute(
				"Maximum number of automations per user, as a string. Empty means no limit.",
				stringvalidator.RegexMatches(countLimitPattern, "must be empty or a positive whole number"),
			),
			"automation_min_interval": adminStringAttribute(
				"Shortest interval an automation may run on, in seconds, as a string. Empty means no floor.",
				stringvalidator.RegexMatches(countLimitPattern, "must be empty or a positive whole number"),
			),
			"enable_automations": adminBoolAttribute("Whether users can create automations."),
			"enable_channels":    adminBoolAttribute("Whether the channels feature is available. The `openwebui_channel` resource needs this on."),
			"channel_model_response_mode": adminStringAttribute(
				"Where a model answers in a channel: `thread` or `channel`.",
				stringvalidator.OneOf("thread", "channel"),
			),
			"enable_memories":              adminBoolAttribute("Whether the memories feature is available."),
			"enable_memory_system_context": adminBoolAttribute("Whether memories are added to the system context of a chat."),
			"enable_calendar":              adminBoolAttribute("Whether the calendar feature is available."),
			"enable_notes":                 adminBoolAttribute("Whether the notes feature is available."),
			"enable_user_webhooks":         adminBoolAttribute("Whether users can register their own webhooks."),
			"enable_user_status":           adminBoolAttribute("Whether users publish an online status."),
			"pending_user_overlay_title":   adminStringAttribute("Heading shown to an account that is still pending approval."),
			"pending_user_overlay_content": adminStringAttribute("Body text shown to an account that is still pending approval."),
			"response_watermark":           adminStringAttribute("Watermark text added to copied model responses."),
		},
	}
}

// Configure assigns the API client.
func (r *adminConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create writes the admin config.
func (r *adminConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	var plan adminConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAdminConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the admin config.
func (r *adminConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	config, err := r.client.GetAdminConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read admin config failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, adminConfigToModel(config))...)
}

// Update writes the admin config.
func (r *adminConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}

	var plan adminConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyAdminConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *adminConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing admin config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *adminConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyAdminConfig writes the plan over the settings the instance holds today.
// The route takes all 28 keys at once and most of them refuse a null, so the
// current values fill in for every attribute the plan does not name.
func applyAdminConfig(ctx context.Context, apiClient *client.Client, plan adminConfigModel) (adminConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := apiClient.GetAdminConfig(ctx)
	if err != nil {
		diags.AddError("Read admin config failed", err.Error())
		return adminConfigModel{}, diags
	}

	config := *current
	overlayBoolValue(&config.ShowAdminDetails, plan.ShowAdminDetails)
	overlayOptionalStringValue(&config.AdminEmail, plan.AdminEmail)
	overlayStringValue(&config.WebUIURL, plan.WebUIURL)
	overlayBoolValue(&config.EnableSignup, plan.EnableSignup)
	overlayBoolValue(&config.EnableAPIKeys, plan.EnableAPIKeys)
	overlayBoolValue(&config.EnableAPIKeysEndpointRestrictions, plan.EnableAPIKeysEndpointRestrictions)
	overlayStringValue(&config.APIKeysAllowedEndpoints, plan.APIKeysAllowedEndpoints)
	overlayStringValue(&config.DefaultUserRole, plan.DefaultUserRole)
	overlayStringValue(&config.DefaultGroupID, plan.DefaultGroupID)
	overlayStringValue(&config.JWTExpiresIn, plan.JWTExpiresIn)
	overlayBoolValue(&config.EnableCommunitySharing, plan.EnableCommunitySharing)
	overlayBoolValue(&config.EnableMessageRating, plan.EnableMessageRating)
	overlayBoolValue(&config.EnableFolders, plan.EnableFolders)
	overlayFlexValue(&config.FolderMaxFileCount, plan.FolderMaxFileCount)
	overlayFlexValue(&config.AutomationMaxCount, plan.AutomationMaxCount)
	overlayFlexValue(&config.AutomationMinInterval, plan.AutomationMinInterval)
	overlayBoolValue(&config.EnableAutomations, plan.EnableAutomations)
	overlayBoolValue(&config.EnableChannels, plan.EnableChannels)
	overlayStringValue(&config.ChannelModelResponseMode, plan.ChannelModelResponseMode)
	overlayBoolValue(&config.EnableCalendar, plan.EnableCalendar)
	overlayBoolValue(&config.EnableMemories, plan.EnableMemories)
	overlayBoolValue(&config.EnableMemorySystemContext, plan.EnableMemorySystemContext)
	overlayBoolValue(&config.EnableNotes, plan.EnableNotes)
	overlayBoolValue(&config.EnableUserWebhooks, plan.EnableUserWebhooks)
	overlayBoolValue(&config.EnableUserStatus, plan.EnableUserStatus)
	overlayOptionalStringValue(&config.PendingUserOverlayTitle, plan.PendingUserOverlayTitle)
	overlayOptionalStringValue(&config.PendingUserOverlayContent, plan.PendingUserOverlayContent)
	overlayOptionalStringValue(&config.ResponseWatermark, plan.ResponseWatermark)

	updated, err := apiClient.SetAdminConfig(ctx, config)
	if err != nil {
		diags.AddError("Update admin config failed", err.Error())
		return adminConfigModel{}, diags
	}

	return adminConfigToModel(updated), diags
}

func adminConfigToModel(config *client.AdminConfig) adminConfigModel {
	return adminConfigModel{
		ID:                                types.StringValue("admin"),
		ShowAdminDetails:                  types.BoolValue(config.ShowAdminDetails),
		AdminEmail:                        stringValueOrNull(config.AdminEmail),
		WebUIURL:                          types.StringValue(config.WebUIURL),
		EnableSignup:                      types.BoolValue(config.EnableSignup),
		EnableAPIKeys:                     types.BoolValue(config.EnableAPIKeys),
		EnableAPIKeysEndpointRestrictions: types.BoolValue(config.EnableAPIKeysEndpointRestrictions),
		APIKeysAllowedEndpoints:           types.StringValue(config.APIKeysAllowedEndpoints),
		DefaultUserRole:                   types.StringValue(config.DefaultUserRole),
		DefaultGroupID:                    types.StringValue(config.DefaultGroupID),
		JWTExpiresIn:                      types.StringValue(config.JWTExpiresIn),
		EnableCommunitySharing:            types.BoolValue(config.EnableCommunitySharing),
		EnableMessageRating:               types.BoolValue(config.EnableMessageRating),
		EnableFolders:                     types.BoolValue(config.EnableFolders),
		FolderMaxFileCount:                types.StringValue(string(config.FolderMaxFileCount)),
		AutomationMaxCount:                types.StringValue(string(config.AutomationMaxCount)),
		AutomationMinInterval:             types.StringValue(string(config.AutomationMinInterval)),
		EnableAutomations:                 types.BoolValue(config.EnableAutomations),
		EnableChannels:                    types.BoolValue(config.EnableChannels),
		ChannelModelResponseMode:          types.StringValue(config.ChannelModelResponseMode),
		EnableCalendar:                    types.BoolValue(config.EnableCalendar),
		EnableMemories:                    types.BoolValue(config.EnableMemories),
		EnableMemorySystemContext:         types.BoolValue(config.EnableMemorySystemContext),
		EnableNotes:                       types.BoolValue(config.EnableNotes),
		EnableUserWebhooks:                types.BoolValue(config.EnableUserWebhooks),
		EnableUserStatus:                  types.BoolValue(config.EnableUserStatus),
		PendingUserOverlayTitle:           stringValueOrNull(config.PendingUserOverlayTitle),
		PendingUserOverlayContent:         stringValueOrNull(config.PendingUserOverlayContent),
		ResponseWatermark:                 stringValueOrNull(config.ResponseWatermark),
	}
}

// overlayBoolValue writes a planned value over a current one. A null or unknown
// attribute leaves the current value in place.
func overlayBoolValue(target *bool, planned types.Bool) {
	if planned.IsNull() || planned.IsUnknown() {
		return
	}
	*target = planned.ValueBool()
}

// overlayStringValue writes a planned value over a current one.
func overlayStringValue(target *string, planned types.String) {
	if planned.IsNull() || planned.IsUnknown() {
		return
	}
	*target = planned.ValueString()
}

// overlayOptionalStringValue writes a planned value over a current one that the
// API allows to be null.
func overlayOptionalStringValue(target **string, planned types.String) {
	if planned.IsNull() || planned.IsUnknown() {
		return
	}
	value := planned.ValueString()
	*target = &value
}

// overlayFlexValue writes a planned value over a current one the API types as
// `int | str | None`.
func overlayFlexValue(target *client.FlexString, planned types.String) {
	if planned.IsNull() || planned.IsUnknown() {
		return
	}
	*target = client.FlexString(planned.ValueString())
}
