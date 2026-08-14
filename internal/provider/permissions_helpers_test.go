package provider

import (
	"context"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFilterPermissionKeys_KnownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"models": true, "tools": false},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["models"] != true {
		t.Fatalf("expected models=true, got %v", result["models"])
	}
	if result["tools"] != false {
		t.Fatalf("expected tools=false, got %v", result["tools"])
	}
}

func TestFilterPermissionKeys_UnknownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"bad_key": true},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("an unknown key must not be an error: %s", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning for the unknown key, got %d", diags.WarningsCount())
	}
	if result["bad_key"] != true {
		t.Fatalf("expected the unknown key to be sent through, got %v", result)
	}
}

func TestFilterPermissionKeys_KnownKeysWarnNothing(t *testing.T) {
	var diags diag.Diagnostics
	filterPermissionKeys(
		"features",
		map[string]bool{"webhooks": true},
		path.Root("permissions").AtName("features"),
		&diags,
	)
	if diags.WarningsCount() != 0 {
		t.Fatalf("expected no warnings for a known key, got %s", diags)
	}
}

func TestFilterPermissionKeys_ChatKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"chat",
		map[string]bool{"file_upload": true, "delete": false},
		path.Root("permissions").AtName("chat"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["file_upload"] != true || result["delete"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionResponse_ValidBools(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("features", map[string]any{
		"web_search":       true,
		"image_generation": false,
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["web_search"] != true || result["image_generation"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionResponse_NonBool(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("features", map[string]any{
		"web_search": "yes", // wrong type
	}, &diags)
	if diags.HasError() {
		t.Fatalf("a non-boolean value must not be an error: %s", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning for the non-boolean value, got %d", diags.WarningsCount())
	}
	if len(result) != 0 {
		t.Fatalf("expected the non-boolean value to be dropped, got %v", result)
	}
}

func TestFilterPermissionResponse_UnknownKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("workspace", map[string]any{
		"unknown_key": true,
	}, &diags)
	if diags.HasError() {
		t.Fatalf("an unknown key must not be an error: %s", diags)
	}
	if result["unknown_key"] != true {
		t.Fatalf("expected the unknown key to reach state, got %v", result)
	}
}

// A key the config sets must come back out of the response, or the write and the
// read disagree and the practitioner sees a permanent diff.
func TestFilterPermissionResponse_MixedKnownAndUnknown(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionResponse("workspace", map[string]any{
		"models":     true,
		"future_key": false,
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !result["models"] {
		t.Fatalf("expected models=true, got %v", result)
	}
	value, ok := result["future_key"]
	if !ok || value {
		t.Fatalf("expected future_key=false to round-trip, got %v", result)
	}
}

func TestExpandPermissions_Workspace(t *testing.T) {
	ctx := context.Background()
	wsMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true, "tools": false})
	if mapDiags.HasError() {
		t.Fatalf("setup: %s", mapDiags)
	}
	model := groupPermissionsModel{
		Workspace:    wsMap,
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	result := expandPermissions(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	ws, ok := result["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected workspace map, got %T", result["workspace"])
	}
	if ws["models"] != true {
		t.Fatalf("expected models=true, got %v", ws["models"])
	}
}

func TestExpandPermissions_AllNull(t *testing.T) {
	ctx := context.Background()
	model := groupPermissionsModel{
		Workspace:    types.MapNull(types.BoolType),
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	result := expandPermissions(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result != nil {
		t.Fatalf("expected nil for all-null model, got %+v", result)
	}
}

func TestFlattenPermissions_Nil(t *testing.T) {
	ctx := context.Background()
	model, diags := flattenPermissions(ctx, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() || !model.AccessGrants.IsNull() || !model.Settings.IsNull() {
		t.Fatal("expected all null for nil input")
	}
}

func TestFlattenPermissions_WithWorkspace(t *testing.T) {
	ctx := context.Background()
	perms := map[string]any{
		"workspace": map[string]any{
			"models": true,
			"tools":  false,
		},
	}
	model, diags := flattenPermissions(ctx, perms)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if model.Workspace.IsNull() {
		t.Fatal("expected non-null workspace")
	}
	var bools map[string]bool
	if err := model.Workspace.ElementsAs(ctx, &bools, false); err != nil {
		t.Fatalf("ElementsAs: %v", err)
	}
	if !bools["models"] {
		t.Fatalf("expected models=true, got %v", bools)
	}
	if bools["tools"] {
		t.Fatalf("expected tools=false, got %v", bools)
	}
}

func TestExpandFlattenPermissionsRoundTrip(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true})
	chatMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"file_upload": true, "delete": false})
	original := groupPermissionsModel{
		Workspace:    wsMap,
		Sharing:      types.MapNull(types.BoolType),
		Chat:         chatMap,
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}

	var expandDiags diag.Diagnostics
	expanded := expandPermissions(ctx, original, &expandDiags)
	if expandDiags.HasError() {
		t.Fatalf("expand: %s", expandDiags)
	}

	flattened, flatDiags := flattenPermissions(ctx, expanded)
	if flatDiags.HasError() {
		t.Fatalf("flatten: %s", flatDiags)
	}

	var wsBools map[string]bool
	if err := flattened.Workspace.ElementsAs(ctx, &wsBools, false); err != nil {
		t.Fatalf("workspace ElementsAs: %v", err)
	}
	if !wsBools["models"] {
		t.Fatalf("round-trip: expected models=true, got %v", wsBools)
	}

	var chatBools map[string]bool
	if err := flattened.Chat.ElementsAs(ctx, &chatBools, false); err != nil {
		t.Fatalf("chat ElementsAs: %v", err)
	}
	if !chatBools["file_upload"] || chatBools["delete"] {
		t.Fatalf("round-trip: unexpected chat values: %v", chatBools)
	}
}

func TestFilterPermissionKeys_SharingKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"sharing",
		map[string]bool{"public_models": true},
		path.Root("permissions").AtName("sharing"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["public_models"] != true {
		t.Fatalf("expected public_models=true, got %v", result["public_models"])
	}
}

func TestFilterPermissionKeys_FeaturesKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"features",
		map[string]bool{"web_search": true},
		path.Root("permissions").AtName("features"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["web_search"] != true {
		t.Fatalf("expected web_search=true, got %v", result["web_search"])
	}
}

// A key added by a future Open WebUI release must reach the server. Hard-failing
// on it made the provider unusable against a newer backend.
func TestExpandPermissions_UnknownKeyReachesTheServer(t *testing.T) {
	ctx := context.Background()
	futureMap, mapDiags := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"future_key": true})
	if mapDiags.HasError() {
		t.Fatalf("setup: %s", mapDiags)
	}
	model := groupPermissionsModel{
		Workspace:    futureMap,
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}
	var diags diag.Diagnostics
	result := expandPermissions(ctx, model, &diags)
	if diags.HasError() {
		t.Fatalf("an unknown key must not be an error: %s", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning for the unknown key, got %d", diags.WarningsCount())
	}
	workspace, ok := result["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected a workspace map, got %T", result["workspace"])
	}
	if workspace["future_key"] != true {
		t.Fatalf("expected future_key=true to be sent, got %v", workspace)
	}
}

func TestFlattenPermissions_EmptyMap(t *testing.T) {
	ctx := context.Background()
	model, diags := flattenPermissions(ctx, map[string]any{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() || !model.AccessGrants.IsNull() || !model.Settings.IsNull() {
		t.Fatal("expected all null for empty map input")
	}
}

func TestObjectToPermissionsModel_NullObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectNull(permissionsAttrTypes())
	model := objectToPermissionsModel(ctx, obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() || !model.AccessGrants.IsNull() || !model.Settings.IsNull() {
		t.Fatal("expected null-filled model for null object")
	}
}

func TestObjectToPermissionsModel_UnknownObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectUnknown(permissionsAttrTypes())
	model := objectToPermissionsModel(ctx, obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.Workspace.IsNull() || !model.Sharing.IsNull() || !model.Chat.IsNull() || !model.Features.IsNull() || !model.AccessGrants.IsNull() || !model.Settings.IsNull() {
		t.Fatal("expected null-filled model for unknown object")
	}
}

func TestPermissionsModelToObjectRoundTrip(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true, "tools": false})
	chatMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"file_upload": true})
	original := groupPermissionsModel{
		Workspace:    wsMap,
		Sharing:      types.MapNull(types.BoolType),
		Chat:         chatMap,
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}

	obj, objDiags := permissionsModelToObject(ctx, original)
	if objDiags.HasError() {
		t.Fatalf("permissionsModelToObject: %s", objDiags)
	}

	var roundDiags diag.Diagnostics
	result := objectToPermissionsModel(ctx, obj, &roundDiags)
	if roundDiags.HasError() {
		t.Fatalf("objectToPermissionsModel: %s", roundDiags)
	}

	if !result.Workspace.Equal(original.Workspace) {
		t.Fatalf("workspace mismatch: got %v, want %v", result.Workspace, original.Workspace)
	}
	if !result.Sharing.Equal(original.Sharing) {
		t.Fatalf("sharing mismatch: got %v, want %v", result.Sharing, original.Sharing)
	}
	if !result.Chat.Equal(original.Chat) {
		t.Fatalf("chat mismatch: got %v, want %v", result.Chat, original.Chat)
	}
	if !result.Features.Equal(original.Features) {
		t.Fatalf("features mismatch: got %v, want %v", result.Features, original.Features)
	}
}

func TestPermissionsObjectSpecified_NullObject(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	obj := types.ObjectNull(permissionsAttrTypes())
	if permissionsObjectSpecified(ctx, obj, &diags) {
		t.Fatal("expected false for null object")
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestPermissionsObjectSpecified_PopulatedObject(t *testing.T) {
	ctx := context.Background()
	wsMap, _ := types.MapValueFrom(ctx, types.BoolType, map[string]bool{"models": true})
	model := groupPermissionsModel{
		Workspace:    wsMap,
		Sharing:      types.MapNull(types.BoolType),
		Chat:         types.MapNull(types.BoolType),
		Features:     types.MapNull(types.BoolType),
		AccessGrants: types.MapNull(types.BoolType),
		Settings:     types.MapNull(types.BoolType),
	}
	obj, objDiags := permissionsModelToObject(ctx, model)
	if objDiags.HasError() {
		t.Fatalf("setup: %s", objDiags)
	}

	var diags diag.Diagnostics
	if !permissionsObjectSpecified(ctx, obj, &diags) {
		t.Fatal("expected true for populated object")
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestFilterPermissionKeys_NewWorkspaceKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"skills": true, "models_import": false, "tools_export": true},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["skills"] != true || result["models_import"] != false || result["tools_export"] != true {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionKeys_NewSharingKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"sharing",
		map[string]bool{"models": true, "skills": false, "public_skills": true, "notes": false, "public_chats": true, "public_calendars": false},
		path.Root("permissions").AtName("sharing"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["models"] != true || result["skills"] != false || result["public_skills"] != true || result["public_chats"] != true || result["public_calendars"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionKeys_NewFeaturesKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"features",
		map[string]bool{"memories": true, "api_keys": false, "channels": true, "folders": false, "automations": true, "calendar": false},
		path.Root("permissions").AtName("features"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["memories"] != true || result["api_keys"] != false || result["channels"] != true || result["folders"] != false || result["automations"] != true || result["calendar"] != false {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestFilterPermissionKeys_NewChatKeys(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"chat",
		map[string]bool{"web_upload": true},
		path.Root("permissions").AtName("chat"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["web_upload"] != true {
		t.Fatalf("expected web_upload=true, got %v", result["web_upload"])
	}
}

func TestFilterPermissionKeys_AccessGrantsKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"access_grants",
		map[string]bool{"allow_users": true},
		path.Root("permissions").AtName("access_grants"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["allow_users"] != true {
		t.Fatalf("expected allow_users=true, got %v", result["allow_users"])
	}
}

func TestFilterPermissionKeys_SettingsKey(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"settings",
		map[string]bool{"interface": true},
		path.Root("permissions").AtName("settings"),
		&diags,
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["interface"] != true {
		t.Fatalf("expected interface=true, got %v", result["interface"])
	}
}

// The permission tree exactly as DEFAULT_USER_PERMISSIONS declares it in
// backend/open_webui/config.py at Open WebUI v0.11.0, lines 1919 to 1999. This
// test is the record that the provider's lists are complete. When Open WebUI
// changes the tree, update this literal first and let the failure show what
// moved.
var backendPermissionKeys = map[string][]string{
	"workspace": {
		"models", "knowledge", "prompts", "tools", "skills",
		"models_import", "models_export", "prompts_import", "prompts_export",
		"tools_import", "tools_export", "skills_import", "skills_export",
	},
	"sharing": {
		"models", "public_models", "knowledge", "public_knowledge",
		"prompts", "public_prompts", "tools", "public_tools",
		"skills", "public_skills", "notes", "public_notes",
		"folders", "public_chats", "open_chats", "public_calendars",
	},
	"access_grants": {"allow_users", "allow_groups"},
	"chat": {
		"controls", "valves", "system_prompt", "params", "file_upload",
		"web_upload", "delete", "delete_message", "continue_response",
		"regenerate_response", "rate_response", "edit", "share", "export",
		"import", "stt", "tts", "call", "multiple_models", "temporary",
		"temporary_enforced",
	},
	"features": {
		"api_keys", "notes", "folders", "channels", "direct_tool_servers",
		"web_search", "image_generation", "code_interpreter", "memories",
		"automations", "calendar", "webhooks",
	},
	"settings": {"interface"},
}

func TestPermissionKeysMatchTheBackend(t *testing.T) {
	for category, expected := range backendPermissionKeys {
		t.Run(category, func(t *testing.T) {
			got := append([]string(nil), allowedKeysSlice(category)...)
			want := append([]string(nil), expected...)
			sort.Strings(got)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Fatalf("permission keys for %s do not match the backend:\n got: %v\nwant: %v", category, got, want)
			}
		})
	}

	if len(groupPermissionsAllowedSets) != len(backendPermissionKeys) {
		t.Fatalf("expected %d permission categories, got %d", len(backendPermissionKeys), len(groupPermissionsAllowedSets))
	}
}

// The seven keys v0.11.0 added that the provider refused to send. The wire name
// of the chat flag is `import`, not `import_`.
func TestFilterPermissionKeys_V011Additions(t *testing.T) {
	cases := map[string]string{
		"workspace":     "skills_import",
		"sharing":       "folders",
		"access_grants": "allow_groups",
		"chat":          "import",
		"features":      "webhooks",
	}

	for category, key := range cases {
		t.Run(category+"."+key, func(t *testing.T) {
			var diags diag.Diagnostics
			result := filterPermissionKeys(
				category,
				map[string]bool{key: true},
				path.Root("permissions").AtName(category),
				&diags,
			)
			if diags.HasError() || diags.WarningsCount() != 0 {
				t.Fatalf("expected %s.%s to be a known key: %s", category, key, diags)
			}
			if result[key] != true {
				t.Fatalf("expected %s.%s=true, got %v", category, key, result)
			}
		})
	}
}

func TestFilterPermissionKeys_MoreV011Additions(t *testing.T) {
	var diags diag.Diagnostics
	result := filterPermissionKeys(
		"workspace",
		map[string]bool{"skills_export": true},
		path.Root("permissions").AtName("workspace"),
		&diags,
	)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected workspace.skills_export to be a known key: %s", diags)
	}
	if result["skills_export"] != true {
		t.Fatalf("expected skills_export=true, got %v", result)
	}

	diags = nil
	sharing := filterPermissionKeys(
		"sharing",
		map[string]bool{"open_chats": true},
		path.Root("permissions").AtName("sharing"),
		&diags,
	)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected sharing.open_chats to be a known key: %s", diags)
	}
	if sharing["open_chats"] != true {
		t.Fatalf("expected open_chats=true, got %v", sharing)
	}
}

func TestPermissionCategoryDescriptionNamesEveryKey(t *testing.T) {
	description := permissionCategoryDescription("features", "Feature access permissions.")
	for _, key := range groupPermissionsFeaturesKeys {
		if !strings.Contains(description, "`"+key+"`") {
			t.Fatalf("expected the features description to name %q: %s", key, description)
		}
	}
}

func TestFlattenPermissions_WithAccessGrantsAndSettings(t *testing.T) {
	ctx := context.Background()
	perms := map[string]any{
		"access_grants": map[string]any{"allow_users": true},
		"settings":      map[string]any{"interface": false},
	}
	model, diags := flattenPermissions(ctx, perms)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if model.AccessGrants.IsNull() {
		t.Fatal("expected non-null access_grants")
	}
	if model.Settings.IsNull() {
		t.Fatal("expected non-null settings")
	}
	var ag map[string]bool
	if err := model.AccessGrants.ElementsAs(ctx, &ag, false); err != nil {
		t.Fatalf("ElementsAs access_grants: %v", err)
	}
	if !ag["allow_users"] {
		t.Fatalf("expected allow_users=true, got %v", ag)
	}
}
