package provider

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fullPermissionMap builds a category map carrying every key this route accepts.
func fullPermissionMap(t *testing.T, category string, value bool) types.Map {
	t.Helper()

	keys := make(map[string]bool, len(defaultUserPermissionsKeys(category)))
	for _, key := range defaultUserPermissionsKeys(category) {
		keys[key] = value
	}

	tfMap, diags := types.MapValueFrom(context.Background(), types.BoolType, keys)
	if diags.HasError() {
		t.Fatalf("build the %s map: %s", category, diags)
	}

	return tfMap
}

func permissionMap(t *testing.T, keys map[string]bool) types.Map {
	t.Helper()

	tfMap, diags := types.MapValueFrom(context.Background(), types.BoolType, keys)
	if diags.HasError() {
		t.Fatalf("build the map: %s", diags)
	}

	return tfMap
}

// SharingPermissions on the default user permissions route has no open_chats
// field, although a group's free-form permissions object carries one.
func TestDefaultUserPermissionsSharingDropsOpenChats(t *testing.T) {
	for _, key := range defaultUserPermissionsSharingKeys {
		if key == "open_chats" {
			t.Fatal("expected open_chats to stay out of the default user permission keys")
		}
	}

	if len(defaultUserPermissionsSharingKeys) != len(groupPermissionsSharingKeys)-1 {
		t.Fatalf("expected exactly one key fewer than a group carries, got %d against %d",
			len(defaultUserPermissionsSharingKeys), len(groupPermissionsSharingKeys))
	}
}

// Every other category matches the group key set.
func TestDefaultUserPermissionsShareTheOtherCategories(t *testing.T) {
	for _, category := range []string{"workspace", "chat", "features", "access_grants", "settings"} {
		if len(defaultUserPermissionsKeys(category)) != len(allowedKeysSlice(category)) {
			t.Fatalf("expected %s to carry the group key set, got %v", category, defaultUserPermissionsKeys(category))
		}
	}
}

func TestExpandDefaultUserPermissionCategoryAcceptsTheFullKeySet(t *testing.T) {
	var diags diag.Diagnostics
	result := expandDefaultUserPermissionCategory(context.Background(), "sharing", fullPermissionMap(t, "sharing", true), &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(result) != len(defaultUserPermissionsSharingKeys) {
		t.Fatalf("expected every key to reach the payload, got %v", result)
	}
	if result["public_tools"] != true {
		t.Fatalf("expected public_tools to carry the configured value, got %v", result["public_tools"])
	}
}

// A missing key would take the default its Pydantic model carries, and two of
// those defaults disagree with the shipped configuration, so an incomplete
// category is an error rather than a partial write.
func TestExpandDefaultUserPermissionCategoryRejectsMissingKeys(t *testing.T) {
	keys := make(map[string]bool)
	for _, key := range defaultUserPermissionsSharingKeys {
		if key == "public_tools" || key == "public_notes" {
			continue
		}
		keys[key] = false
	}

	var diags diag.Diagnostics
	expandDefaultUserPermissionCategory(context.Background(), "sharing", permissionMap(t, keys), &diags)

	if !diags.HasError() {
		t.Fatal("expected an error for the missing keys")
	}
	detail := diags.Errors()[0].Detail()
	for _, key := range []string{"public_tools", "public_notes"} {
		if !strings.Contains(detail, key) {
			t.Fatalf("expected %s to be named in the error, got %q", key, detail)
		}
	}
}

// open_chats reaches this route only to be dropped, so the provider names the
// resource that can still set it.
func TestExpandDefaultUserPermissionCategoryRejectsOpenChats(t *testing.T) {
	keys := make(map[string]bool, len(defaultUserPermissionsSharingKeys)+1)
	for _, key := range defaultUserPermissionsSharingKeys {
		keys[key] = false
	}
	keys["open_chats"] = true

	var diags diag.Diagnostics
	expandDefaultUserPermissionCategory(context.Background(), "sharing", permissionMap(t, keys), &diags)

	if !diags.HasError() {
		t.Fatal("expected an error for open_chats")
	}
	detail := diags.Errors()[0].Detail()
	if !strings.Contains(detail, "open_chats") || !strings.Contains(detail, "openwebui_group") {
		t.Fatalf("expected the error to name open_chats and openwebui_group, got %q", detail)
	}
}

func TestExpandDefaultUserPermissionCategoryRejectsAnAbsentCategory(t *testing.T) {
	var diags diag.Diagnostics
	expandDefaultUserPermissionCategory(context.Background(), "settings", types.MapNull(types.BoolType), &diags)

	if !diags.HasError() {
		t.Fatal("expected an error for an absent category")
	}
}

func TestFilterDefaultUserPermissionResponseDropsUnknownKeys(t *testing.T) {
	var diags diag.Diagnostics
	filtered := filterDefaultUserPermissionResponse("sharing", map[string]any{
		"public_tools": true,
		"open_chats":   true,
	}, &diags)

	if _, present := filtered["open_chats"]; present {
		t.Fatalf("expected open_chats to stay out of state, got %v", filtered)
	}
	if filtered["public_tools"] != true {
		t.Fatalf("expected the known key to survive, got %v", filtered)
	}
	if len(diags.Warnings()) != 1 {
		t.Fatalf("expected one warning about the unknown key, got %s", diags)
	}
}

func TestFilterDefaultUserPermissionResponseDropsNonBooleans(t *testing.T) {
	var diags diag.Diagnostics
	filtered := filterDefaultUserPermissionResponse("settings", map[string]any{"interface": "yes"}, &diags)

	if len(filtered) != 0 {
		t.Fatalf("expected the non-boolean to be dropped, got %v", filtered)
	}
	if len(diags.Warnings()) != 1 {
		t.Fatalf("expected one warning about the value type, got %s", diags)
	}
}

// The key checks run at plan time, so a practitioner sees them before an apply
// touches the permissions of every user. These two steps reach no API.
func TestDefaultUserPermissionsRejectAnIncompleteCategoryAtPlanTime(t *testing.T) {
	t.Setenv("OPENWEBUI_ENDPOINT", "http://fake.example.com")
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDefaultUserPermissionsConfig(nil, []string{"sharing.public_notes"}),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Incomplete sharing permissions`),
			},
		},
	})
}

func TestDefaultUserPermissionsRejectOpenChatsAtPlanTime(t *testing.T) {
	t.Setenv("OPENWEBUI_ENDPOINT", "http://fake.example.com")
	t.Setenv("OPENWEBUI_TOKEN", "test-token")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccDefaultUserPermissionsConfig(map[string]bool{"sharing.open_chats": true}, nil),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Unrecognized sharing permission key`),
			},
		},
	})
}

func TestDefaultUserPermissionCategoryDescriptionListsEveryKey(t *testing.T) {
	description := defaultUserPermissionCategoryDescription("access_grants", "Access-grant permissions.")

	for _, key := range defaultUserPermissionsKeys("access_grants") {
		if !strings.Contains(description, "`"+key+"`") {
			t.Fatalf("expected %s in the description, got %q", key, description)
		}
	}
}
