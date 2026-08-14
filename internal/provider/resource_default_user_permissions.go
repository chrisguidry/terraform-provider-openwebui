package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &defaultUserPermissionsResource{}
var _ resource.ResourceWithConfigure = &defaultUserPermissionsResource{}
var _ resource.ResourceWithImportState = &defaultUserPermissionsResource{}
var _ resource.ResourceWithValidateConfig = &defaultUserPermissionsResource{}

// defaultUserPermissionCategories lists the categories in the order the schema
// declares them, so diagnostics come out in a stable order.
var defaultUserPermissionCategories = []string{"workspace", "sharing", "chat", "features", "access_grants", "settings"}

func init() {
	registeredResources = append(registeredResources, NewDefaultUserPermissionsResource)
}

// defaultUserPermissionsResource manages the permissions every user carries
// before any group grants more.
type defaultUserPermissionsResource struct {
	client *client.Client
}

type defaultUserPermissionsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Permissions types.Object `tfsdk:"permissions"`
}

// NewDefaultUserPermissionsResource constructs a new default user permissions resource.
func NewDefaultUserPermissionsResource() resource.Resource {
	return &defaultUserPermissionsResource{}
}

// Metadata sets the resource type name.
func (r *defaultUserPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_user_permissions"
}

// Schema defines the default user permissions schema.
func (r *defaultUserPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the default user permissions in Open WebUI: what a user may do before any group grants more.\n\n" +
			"Open WebUI replaces the whole permissions object on every write, and it fills a key the request leaves out with the default " +
			"its Pydantic model carries. Two of those model defaults disagree with the shipped configuration, `sharing.public_tools` and " +
			"`sharing.public_notes`, so this resource requires every key of every category rather than let either value creep in.\n\n" +
			"**Applying this resource clears `sharing.open_chats`.** The write model has no such field, although the shipped configuration " +
			"carries one and Open WebUI enforces it on shared chats. Anything this route writes drops the key. Set `sharing.open_chats` per " +
			"group with `openwebui_group` if you need it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"permissions": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "The six permission categories. All six are required, because Open WebUI rejects a partial write with a 422.",
				Attributes: map[string]schema.Attribute{
					"workspace": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("workspace", "Workspace-level permissions."),
					},
					"sharing": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("sharing", "Sharing permissions. `open_chats` is not among them: this route deletes that key."),
					},
					"chat": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("chat", "Chat-level permissions."),
					},
					"features": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("features", "Feature access permissions."),
					},
					"access_grants": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("access_grants", "Access-grant permissions, controlling whether a user may share a resource with other users or with groups."),
					},
					"settings": schema.MapAttribute{
						Required:            true,
						ElementType:         types.BoolType,
						MarkdownDescription: defaultUserPermissionCategoryDescription("settings", "Settings permissions, controlling whether a user may change their interface settings."),
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *defaultUserPermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// ValidateConfig reports an incomplete or misspelled category at plan time,
// rather than leaving the practitioner to find it during an apply.
func (r *defaultUserPermissionsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config defaultUserPermissionsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Permissions.IsNull() || config.Permissions.IsUnknown() {
		return
	}

	model := objectToPermissionsModel(ctx, config.Permissions, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, category := range defaultUserPermissionCategories {
		value := permissionCategoryMap(model, category)
		if value.IsNull() || value.IsUnknown() {
			continue
		}

		var bools map[string]bool
		if decodeDiags := value.ElementsAs(ctx, &bools, false); decodeDiags.HasError() {
			resp.Diagnostics.Append(decodeDiags...)
			continue
		}

		validateDefaultUserPermissionKeys(category, bools, path.Root("permissions").AtName(category), &resp.Diagnostics)
	}
}

// Create writes the default user permissions.
func (r *defaultUserPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing default user permissions.")
		return
	}

	var plan defaultUserPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyDefaultUserPermissions(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the default user permissions.
func (r *defaultUserPermissionsResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing default user permissions.")
		return
	}

	perms, err := r.client.GetDefaultUserPermissions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read default user permissions failed", err.Error())
		return
	}

	state, diags := defaultUserPermissionsState(ctx, perms)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the default user permissions.
func (r *defaultUserPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing default user permissions.")
		return
	}

	var plan defaultUserPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyDefaultUserPermissions(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves the permissions in place.
func (r *defaultUserPermissionsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *defaultUserPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyDefaultUserPermissions(ctx context.Context, apiClient *client.Client, plan defaultUserPermissionsResourceModel) (defaultUserPermissionsResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := objectToPermissionsModel(ctx, plan.Permissions, &diags)
	if diags.HasError() {
		return defaultUserPermissionsResourceModel{}, diags
	}

	form := expandDefaultUserPermissions(ctx, model, &diags)
	if diags.HasError() {
		return defaultUserPermissionsResourceModel{}, diags
	}

	updated, err := apiClient.SetDefaultUserPermissions(ctx, form)
	if err != nil {
		diags.AddError("Update default user permissions failed", err.Error())
		return defaultUserPermissionsResourceModel{}, diags
	}

	state, stateDiags := defaultUserPermissionsState(ctx, updated)
	diags.Append(stateDiags...)

	return state, diags
}

func defaultUserPermissionsState(ctx context.Context, perms *client.DefaultUserPermissions) (defaultUserPermissionsResourceModel, diag.Diagnostics) {
	model, diags := flattenDefaultUserPermissions(ctx, perms)
	if diags.HasError() {
		return defaultUserPermissionsResourceModel{}, diags
	}

	object, objectDiags := permissionsModelToObject(ctx, model)
	diags.Append(objectDiags...)

	return defaultUserPermissionsResourceModel{
		ID:          types.StringValue("default_user_permissions"),
		Permissions: object,
	}, diags
}

// expandDefaultUserPermissions builds the write payload and holds the
// practitioner to the full key set, because the write replaces the whole object.
func expandDefaultUserPermissions(ctx context.Context, perms groupPermissionsModel, diags *diag.Diagnostics) client.DefaultUserPermissions {
	return client.DefaultUserPermissions{
		Workspace:    expandDefaultUserPermissionCategory(ctx, "workspace", perms.Workspace, diags),
		Sharing:      expandDefaultUserPermissionCategory(ctx, "sharing", perms.Sharing, diags),
		AccessGrants: expandDefaultUserPermissionCategory(ctx, "access_grants", perms.AccessGrants, diags),
		Chat:         expandDefaultUserPermissionCategory(ctx, "chat", perms.Chat, diags),
		Features:     expandDefaultUserPermissionCategory(ctx, "features", perms.Features, diags),
		Settings:     expandDefaultUserPermissionCategory(ctx, "settings", perms.Settings, diags),
	}
}

func expandDefaultUserPermissionCategory(ctx context.Context, category string, value types.Map, diags *diag.Diagnostics) map[string]any {
	attribute := path.Root("permissions").AtName(category)

	if value.IsNull() || value.IsUnknown() {
		diags.AddAttributeError(
			attribute,
			fmt.Sprintf("Missing %s permissions", category),
			fmt.Sprintf("Open WebUI writes all six permission categories at once, so %s needs a value. It carries these keys: %s.",
				attribute.String(), strings.Join(defaultUserPermissionsKeys(category), ", ")),
		)
		return nil
	}

	var bools map[string]bool
	if err := value.ElementsAs(ctx, &bools, false); err != nil {
		diags.AddAttributeError(
			attribute,
			"Invalid permissions value",
			fmt.Sprintf("Unable to decode %s into a map of booleans: %v", attribute.String(), err),
		)
		return nil
	}

	validateDefaultUserPermissionKeys(category, bools, attribute, diags)
	if diags.HasError() {
		return nil
	}

	result := make(map[string]any, len(bools))
	for key, enabled := range bools {
		result[key] = enabled
	}

	return result
}

// validateDefaultUserPermissionKeys holds one category to the exact key set the
// write model declares: a missing key would take the model's own default, and a
// key the model does not declare is dropped without a word.
func validateDefaultUserPermissionKeys(category string, bools map[string]bool, attribute path.Path, diags *diag.Diagnostics) {
	allowed := defaultUserPermissionsAllowedSets[category]

	var unrecognized []string
	for key := range bools {
		if _, ok := allowed[key]; !ok {
			unrecognized = append(unrecognized, key)
		}
	}
	sort.Strings(unrecognized)

	var missing []string
	for _, key := range defaultUserPermissionsKeys(category) {
		if _, ok := bools[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(unrecognized) > 0 {
		detail := fmt.Sprintf("Open WebUI drops a key its model does not declare, so %s would never take effect. The %s keys are: %s.",
			strings.Join(unrecognized, ", "), category, strings.Join(defaultUserPermissionsKeys(category), ", "))
		if category == "sharing" && containsPermissionKey(unrecognized, "open_chats") {
			detail += " `open_chats` is one Open WebUI enforces but does not accept here. Set it per group with `openwebui_group`."
		}
		diags.AddAttributeError(attribute, fmt.Sprintf("Unrecognized %s permission key", category), detail)
	}

	if len(missing) > 0 {
		diags.AddAttributeError(
			attribute,
			fmt.Sprintf("Incomplete %s permissions", category),
			fmt.Sprintf("This write replaces the whole permissions object, and Open WebUI fills a missing key with its own default. Set these keys as well: %s.",
				strings.Join(missing, ", ")),
		)
	}
}

// permissionCategoryMap picks one category out of the permissions model.
func permissionCategoryMap(model groupPermissionsModel, category string) types.Map {
	switch category {
	case "workspace":
		return model.Workspace
	case "sharing":
		return model.Sharing
	case "chat":
		return model.Chat
	case "features":
		return model.Features
	case "access_grants":
		return model.AccessGrants
	case "settings":
		return model.Settings
	}

	return types.MapNull(types.BoolType)
}

// flattenDefaultUserPermissions reads the permissions Open WebUI returns. It
// keeps only the keys the write model declares, because every other key would
// show up in state without ever reaching the server.
func flattenDefaultUserPermissions(ctx context.Context, perms *client.DefaultUserPermissions) (groupPermissionsModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := groupPermissionsModel{}
	sources := map[string]map[string]any{
		"workspace":     perms.Workspace,
		"sharing":       perms.Sharing,
		"chat":          perms.Chat,
		"features":      perms.Features,
		"access_grants": perms.AccessGrants,
		"settings":      perms.Settings,
	}

	converted := make(map[string]types.Map, len(sources))
	for category, nested := range sources {
		bools := filterDefaultUserPermissionResponse(category, nested, &diags)
		tfMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, bools)
		diags.Append(mapDiags...)
		converted[category] = tfMap
	}

	model.Workspace = converted["workspace"]
	model.Sharing = converted["sharing"]
	model.Chat = converted["chat"]
	model.Features = converted["features"]
	model.AccessGrants = converted["access_grants"]
	model.Settings = converted["settings"]

	return model, diags
}

func filterDefaultUserPermissionResponse(category string, nested map[string]any, diags *diag.Diagnostics) map[string]bool {
	allowed, ok := defaultUserPermissionsAllowedSets[category]
	if !ok {
		diags.AddError("Internal provider error", fmt.Sprintf("Unknown permission category %s", category))
		return nil
	}

	filtered := make(map[string]bool, len(nested))
	var unrecognized []string

	for key, raw := range nested {
		if _, known := allowed[key]; !known {
			unrecognized = append(unrecognized, key)
			continue
		}

		enabled, isBool := raw.(bool)
		if !isBool {
			diags.AddWarning(
				"Unexpected permissions response",
				fmt.Sprintf("Open WebUI returned permissions.%s.%s as %T rather than a boolean. The provider drops it.", category, key, raw),
			)
			continue
		}

		filtered[key] = enabled
	}

	if len(unrecognized) > 0 {
		sort.Strings(unrecognized)
		diags.AddWarning(
			"Unrecognized permission key in the response",
			fmt.Sprintf("Open WebUI returned permissions.%s keys this build of the provider does not know: %s. They stay out of state, and the next write through this resource deletes them from the stored configuration.",
				category, strings.Join(unrecognized, ", ")),
		)
	}

	return filtered
}

func containsPermissionKey(keys []string, target string) bool {
	for _, key := range keys {
		if key == target {
			return true
		}
	}

	return false
}
