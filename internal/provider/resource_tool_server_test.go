package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// toolServerTestClient stands in for a configured client. The mapping under
// test resolves no groups, so it makes no request.
func toolServerTestClient(t *testing.T) *client.Client {
	t.Helper()

	c, err := client.NewClient("https://openwebui.invalid", "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return c
}

func TestToolServerEntryFromPlanOwnsOnlyItsOwnKeys(t *testing.T) {
	var diags diag.Diagnostics

	plan := toolServerResourceModel{
		ServerID: types.StringValue("paperless"),
		URL:      types.StringValue("https://paperless.mcp.example"),
		Path:     types.StringNull(),
		Type:     types.StringValue("mcp"),
		AuthType: types.StringValue("oauth_2.1"),
		Enabled:  types.BoolValue(true),
	}

	entry := toolServerEntryFromPlan(context.Background(), toolServerTestClient(t), plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if entry["path"] != "" {
		t.Fatalf("expected an unset path to be written as empty, got %v", entry["path"])
	}

	info, ok := entry["info"].(map[string]any)
	if !ok {
		t.Fatalf("expected an info object, got %v", entry["info"])
	}
	if info["id"] != "paperless" {
		t.Fatalf("expected info.id to carry the server_id, got %v", info)
	}

	// A nil tells the client to remove the key, which is how an attribute the
	// operator does not set leaves the stored info alone.
	for _, key := range []string{"name", "description", "oauth_client_id", "oauth_client_secret", "oauth_client_info"} {
		value, present := info[key]
		if !present || value != nil {
			t.Fatalf("expected info.%s to be present and nil, got %v", key, value)
		}
	}

	config, ok := entry["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected a config object, got %v", entry["config"])
	}
	if config["enable"] != true {
		t.Fatalf("expected config.enable to be true, got %v", config)
	}

	grants, ok := config["access_grants"].([]any)
	if !ok || len(grants) != 0 {
		t.Fatalf("expected an empty access_grants list, got %v", config["access_grants"])
	}

	if _, present := entry["spec_type"]; present {
		t.Fatalf("expected an unset spec_type to be left out of the write, got %v", entry)
	}
}

func TestToolServerStateFromEntryCarriesTheOAuthBlob(t *testing.T) {
	entry := client.ToolServerEntry{
		"url":       "https://paperless.mcp.example",
		"path":      "",
		"type":      "mcp",
		"auth_type": "oauth_2.1",
		"key":       nil,
		"config":    map[string]any{"enable": true, "oauth_scope": "read"},
		"info": map[string]any{
			"id":                "paperless",
			"name":              "Paperless",
			"oauth_client_info": "gAAAAABlciphertext",
		},
	}

	state, diags := toolServerStateFromEntry(context.Background(), toolServerTestClient(t), entry, toolServerResourceModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.OAuthClientInfo.ValueString() != "gAAAAABlciphertext" {
		t.Fatalf("expected the OAuth blob in state, got %v", state.OAuthClientInfo)
	}
	if state.ServerID.ValueString() != "paperless" {
		t.Fatalf("expected server_id to come from info.id, got %v", state.ServerID)
	}
	if state.Name.ValueString() != "Paperless" {
		t.Fatalf("expected name to come from info.name, got %v", state.Name)
	}
	if !state.AuthType.Equal(types.StringValue("oauth_2.1")) {
		t.Fatalf("expected auth_type oauth_2.1, got %v", state.AuthType)
	}
	if !state.Enabled.ValueBool() {
		t.Fatalf("expected enabled to come from config.enable, got %v", state.Enabled)
	}
	if state.Key.ValueString() != "" || !state.Key.IsNull() {
		t.Fatalf("expected a null key, got %v", state.Key)
	}
}

// Open WebUI reads oauth_scope from info first and from config second, so the
// provider reports the value in force.
func TestToolServerStateFromEntryPrefersInfoOverConfig(t *testing.T) {
	entry := client.ToolServerEntry{
		"url":    "https://paperless.mcp.example",
		"config": map[string]any{"oauth_scope": "read", "oauth_resource_parameter": "auto"},
		"info":   map[string]any{"id": "paperless", "oauth_scope": "read write"},
	}

	state, diags := toolServerStateFromEntry(context.Background(), toolServerTestClient(t), entry, toolServerResourceModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.OAuthScope.ValueString() != "read write" {
		t.Fatalf("expected the info scope to win, got %v", state.OAuthScope)
	}
	if state.OAuthResourceParameter.ValueString() != "auto" {
		t.Fatalf("expected the config value as the fallback, got %v", state.OAuthResourceParameter)
	}
	if state.Type.ValueString() != "openapi" {
		t.Fatalf("expected a missing type to read as openapi, got %v", state.Type)
	}
}

func TestToolServerStateFromEntryFallsBackToTheStatedServerID(t *testing.T) {
	entry := client.ToolServerEntry{"url": "https://tools.example", "path": "openapi.json"}
	fallback := toolServerResourceModel{ServerID: types.StringValue("weather")}

	state, diags := toolServerStateFromEntry(context.Background(), toolServerTestClient(t), entry, fallback)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ServerID.ValueString() != "weather" {
		t.Fatalf("expected the fallback server_id, got %v", state.ServerID)
	}
}

func TestParseToolServerImportID(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		serverID string
		index    int
		fails    bool
	}{
		{name: "plain id", raw: "paperless", serverID: "paperless", index: -1},
		{name: "id and position", raw: "weather@2", serverID: "weather", index: 2},
		{name: "empty", raw: "  ", fails: true},
		{name: "no id", raw: "@2", fails: true},
		{name: "no position", raw: "weather@", fails: true},
		{name: "negative position", raw: "weather@-1", fails: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serverID, index, err := parseToolServerImportID(tc.raw)
			if tc.fails {
				if err == nil {
					t.Fatalf("expected %q to be rejected", tc.raw)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseToolServerImportID(%q): %v", tc.raw, err)
			}
			if serverID != tc.serverID || index != tc.index {
				t.Fatalf("expected %q and %d, got %q and %d", tc.serverID, tc.index, serverID, index)
			}
		})
	}
}

func TestToolServerPreservedJSONKeepsEquivalentText(t *testing.T) {
	configured := types.StringValue(`{"b":"2","a":"1"}`)
	stored := types.StringValue(`{"a":"1","b":"2"}`)

	if got := toolServerPreservedJSON(configured, stored); got.ValueString() != configured.ValueString() {
		t.Fatalf("expected the configured text to survive a key reorder, got %v", got)
	}

	different := types.StringValue(`{"a":"9"}`)
	if got := toolServerPreservedJSON(configured, different); got.ValueString() != different.ValueString() {
		t.Fatalf("expected a real difference to report the stored text, got %v", got)
	}
}
