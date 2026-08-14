package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// The six permission categories and the keys each one carries, read off
// DEFAULT_USER_PERMISSIONS in backend/open_webui/config.py at Open WebUI
// v0.11.0. A group's permissions column is free-form JSON, so the server keeps
// any key it is given. These lists therefore drive the documentation and the
// unknown-key warning; they are not a restriction. A key Open WebUI adds later
// still reaches the server, with a warning telling the practitioner that this
// build of the provider does not know it.
//
// The wire name of the chat import flag is `import`, not `import_`. The backend
// attribute is ChatPermissions.import_ with Field(alias='import') in
// backend/open_webui/routers/users.py.
var (
	groupPermissionsWorkspaceKeys    = []string{"models", "knowledge", "prompts", "tools", "skills", "models_import", "models_export", "prompts_import", "prompts_export", "tools_import", "tools_export", "skills_import", "skills_export"}
	groupPermissionsSharingKeys      = []string{"public_models", "public_knowledge", "public_prompts", "public_tools", "models", "knowledge", "prompts", "tools", "skills", "public_skills", "notes", "public_notes", "folders", "open_chats", "public_chats", "public_calendars"}
	groupPermissionsChatKeys         = []string{"controls", "valves", "system_prompt", "params", "file_upload", "delete", "delete_message", "continue_response", "regenerate_response", "rate_response", "edit", "share", "export", "import", "stt", "tts", "call", "multiple_models", "temporary", "temporary_enforced", "web_upload"}
	groupPermissionsFeaturesKeys     = []string{"direct_tool_servers", "web_search", "image_generation", "code_interpreter", "notes", "memories", "api_keys", "channels", "folders", "automations", "calendar", "webhooks"}
	groupPermissionsAccessGrantsKeys = []string{"allow_users", "allow_groups"}
	groupPermissionsSettingsKeys     = []string{"interface"}

	groupPermissionsAllowedSets = map[string]map[string]struct{}{
		"workspace":     sliceToSet(groupPermissionsWorkspaceKeys),
		"sharing":       sliceToSet(groupPermissionsSharingKeys),
		"chat":          sliceToSet(groupPermissionsChatKeys),
		"features":      sliceToSet(groupPermissionsFeaturesKeys),
		"access_grants": sliceToSet(groupPermissionsAccessGrantsKeys),
		"settings":      sliceToSet(groupPermissionsSettingsKeys),
	}
)

// The default user permissions route takes a typed model, not the free-form
// object a group carries, so its key set is its own. Two differences from the
// group key set, both read off backend/open_webui/routers/users.py at Open WebUI
// v0.11.0:
//
//   - SharingPermissions has no open_chats field, although
//     DEFAULT_USER_PERMISSIONS carries one and chats.py enforces it. The write
//     handler dumps the whole model over the stored blob, so every write through
//     this route deletes sharing.open_chats from the stored config.
//   - A key the model does not declare is dropped, because the model ignores
//     extra fields. A key the provider does not know is therefore an error on
//     this route, where the group resource passes it through with a warning.
var (
	defaultUserPermissionsSharingKeys = withoutPermissionKey(groupPermissionsSharingKeys, "open_chats")

	defaultUserPermissionsAllowedSets = map[string]map[string]struct{}{
		"workspace":     sliceToSet(groupPermissionsWorkspaceKeys),
		"sharing":       sliceToSet(defaultUserPermissionsSharingKeys),
		"chat":          sliceToSet(groupPermissionsChatKeys),
		"features":      sliceToSet(groupPermissionsFeaturesKeys),
		"access_grants": sliceToSet(groupPermissionsAccessGrantsKeys),
		"settings":      sliceToSet(groupPermissionsSettingsKeys),
	}
)

// defaultUserPermissionsKeys returns the keys one category of the default user
// permissions carries.
func defaultUserPermissionsKeys(category string) []string {
	if category == "sharing" {
		return defaultUserPermissionsSharingKeys
	}

	return allowedKeysSlice(category)
}

// defaultUserPermissionCategoryDescription builds the schema description for one
// category of the default user permissions, so the documentation follows the key
// lists rather than a hand-written copy of them.
func defaultUserPermissionCategoryDescription(category, summary string) string {
	keys := defaultUserPermissionsKeys(category)
	quoted := make([]string, 0, len(keys))
	for _, key := range keys {
		quoted = append(quoted, "`"+key+"`")
	}

	return fmt.Sprintf(
		"%s Every key is required: %s. Open WebUI replaces the whole permissions object on each write, so a key left out would take the value its Pydantic model defaults to.",
		summary, strings.Join(quoted, ", "),
	)
}

func withoutPermissionKey(keys []string, drop string) []string {
	remaining := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == drop {
			continue
		}
		remaining = append(remaining, key)
	}

	return remaining
}

// permissionsAttrTypes returns the framework attribute types for the permissions object.
// Used to construct a types.Object that the framework can hold as unknown/null.
func permissionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"workspace":     types.MapType{ElemType: types.BoolType},
		"sharing":       types.MapType{ElemType: types.BoolType},
		"chat":          types.MapType{ElemType: types.BoolType},
		"features":      types.MapType{ElemType: types.BoolType},
		"access_grants": types.MapType{ElemType: types.BoolType},
		"settings":      types.MapType{ElemType: types.BoolType},
	}
}

// permissionsModelToObject converts a groupPermissionsModel struct to a types.Object.
func permissionsModelToObject(ctx context.Context, model groupPermissionsModel) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, permissionsAttrTypes(), model)
}

// objectToPermissionsModel converts a types.Object to a groupPermissionsModel struct.
// Returns a null-filled model if the object is null or unknown.
func objectToPermissionsModel(ctx context.Context, obj types.Object, diags *diag.Diagnostics) groupPermissionsModel {
	null := groupPermissionsModel{
		Workspace:    types.MapNull(types.BoolType),
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}
	if obj.IsNull() || obj.IsUnknown() {
		return null
	}
	var model groupPermissionsModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	return model
}

// permissionsObjectSpecified returns true when the object is not null/unknown
// and at least one map within it is populated.
func permissionsObjectSpecified(ctx context.Context, obj types.Object, diags *diag.Diagnostics) bool {
	if obj.IsNull() || obj.IsUnknown() {
		return false
	}
	model := objectToPermissionsModel(ctx, obj, diags)
	return permissionsSpecified(model)
}

func permissionsSpecified(perms groupPermissionsModel) bool {
	return mapProvided(perms.Workspace) || mapProvided(perms.Sharing) ||
		mapProvided(perms.Chat) || mapProvided(perms.Features) ||
		mapProvided(perms.AccessGrants) || mapProvided(perms.Settings)
}

func mapProvided(value types.Map) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func sliceToSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}

	return set
}

func expandPermissions(ctx context.Context, perms groupPermissionsModel, diags *diag.Diagnostics) map[string]any {
	result := make(map[string]any)

	add := func(category string, value types.Map, attribute path.Path) {
		if value.IsNull() || value.IsUnknown() {
			return
		}

		var bools map[string]bool
		if err := value.ElementsAs(ctx, &bools, false); err != nil {
			diags.AddAttributeError(
				attribute,
				"Invalid permissions value",
				fmt.Sprintf("Unable to decode %s into a map of booleans: %v", attribute.String(), err),
			)
			return
		}

		filtered := filterPermissionKeys(category, bools, attribute, diags)
		if len(filtered) == 0 {
			return
		}

		nested := make(map[string]any, len(filtered))
		for k, v := range filtered {
			nested[k] = v
		}

		result[category] = nested
	}

	add("workspace", perms.Workspace, path.Root("permissions").AtName("workspace"))
	add("sharing", perms.Sharing, path.Root("permissions").AtName("sharing"))
	add("chat", perms.Chat, path.Root("permissions").AtName("chat"))
	add("features", perms.Features, path.Root("permissions").AtName("features"))
	add("access_grants", perms.AccessGrants, path.Root("permissions").AtName("access_grants"))
	add("settings", perms.Settings, path.Root("permissions").AtName("settings"))

	if len(result) == 0 {
		return nil
	}

	return result
}

func flattenPermissions(ctx context.Context, perms map[string]any) (groupPermissionsModel, diag.Diagnostics) {
	model := groupPermissionsModel{
		Workspace:    types.MapNull(types.BoolType),
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}

	var diags diag.Diagnostics

	if len(perms) == 0 {
		return model, diags
	}

	convert := func(category string) types.Map {
		raw, ok := perms[category]
		if !ok || raw == nil {
			return types.MapNull(types.BoolType)
		}

		nested, ok := raw.(map[string]any)
		if !ok {
			diags.AddError(
				"Unexpected permissions response",
				fmt.Sprintf("Expected permissions.%s to be an object", category),
			)
			return types.MapNull(types.BoolType)
		}

		bools := filterPermissionResponse(category, nested, &diags)
		tfMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, bools)
		diags.Append(mapDiags...)
		return tfMap
	}

	model.Workspace = convert("workspace")
	model.Sharing = convert("sharing")
	model.Chat = convert("chat")
	model.Features = convert("features")
	model.AccessGrants = convert("access_grants")
	model.Settings = convert("settings")

	return model, diags
}

// filterPermissionKeys passes every key through to the server. A key this build
// of the provider does not know earns a warning, because Open WebUI stores group
// permissions as a free-form object and a newer server may well accept it.
func filterPermissionKeys(category string, bools map[string]bool, attribute path.Path, diags *diag.Diagnostics) map[string]bool {
	allowed, ok := groupPermissionsAllowedSets[category]
	if !ok {
		diags.AddError(
			"Internal provider error",
			fmt.Sprintf("Unknown permission category %s", category),
		)
		return nil
	}

	filtered := make(map[string]bool, len(bools))
	for key, value := range bools {
		if _, exists := allowed[key]; !exists {
			diags.AddAttributeWarning(
				attribute,
				fmt.Sprintf("Unrecognized %s permission key", category),
				fmt.Sprintf(
					"The provider knows these %s keys: %s. It sends %q as well, because Open WebUI stores group permissions as a free-form object. Check the spelling if the key has no effect.",
					category, allowedKeysList(category), key,
				),
			)
		}

		filtered[key] = value
	}

	return filtered
}

// filterPermissionResponse reads the permissions the server returns. It keeps
// unknown keys, so that a key written through filterPermissionKeys round-trips
// into state instead of showing up as a permanent diff. A value that is not a
// boolean cannot live in the map of booleans, so it is dropped with a warning.
func filterPermissionResponse(category string, nested map[string]any, diags *diag.Diagnostics) map[string]bool {
	if _, ok := groupPermissionsAllowedSets[category]; !ok {
		diags.AddError(
			"Internal provider error",
			fmt.Sprintf("Unknown permission category %s", category),
		)
		return nil
	}

	filtered := make(map[string]bool, len(nested))
	for key, raw := range nested {
		boolVal, ok := raw.(bool)
		if !ok {
			diags.AddWarning(
				"Unexpected permissions response",
				fmt.Sprintf("Open WebUI returned permissions.%s.%s as %T rather than a boolean. The provider drops it.", category, key, raw),
			)
			continue
		}

		filtered[key] = boolVal
	}

	return filtered
}

// permissionCategoryDescription builds the schema description for one permission
// category from the key lists above, so the documentation cannot drift from the
// keys the provider knows.
func permissionCategoryDescription(category, summary string) string {
	return fmt.Sprintf(
		"%s Keys Open WebUI v0.11.0 defines: %s. Any other key is sent to the server with a warning, because Open WebUI stores group permissions as a free-form object.",
		summary, permissionKeysMarkdown(category),
	)
}

// permissionKeysMarkdown renders a category's known keys as an inline-code list
// for the schema descriptions, so the documentation follows the lists above.
func permissionKeysMarkdown(category string) string {
	keys := allowedKeysSlice(category)
	quoted := make([]string, 0, len(keys))
	for _, key := range keys {
		quoted = append(quoted, "`"+key+"`")
	}

	return strings.Join(quoted, ", ")
}

func allowedKeysList(category string) string {
	return strings.Join(allowedKeysSlice(category), ", ")
}

func allowedKeysSlice(category string) []string {
	var keys []string

	switch category {
	case "workspace":
		keys = groupPermissionsWorkspaceKeys
	case "sharing":
		keys = groupPermissionsSharingKeys
	case "chat":
		keys = groupPermissionsChatKeys
	case "features":
		keys = groupPermissionsFeaturesKeys
	case "access_grants":
		keys = groupPermissionsAccessGrantsKeys
	case "settings":
		keys = groupPermissionsSettingsKeys
	}

	return keys
}
