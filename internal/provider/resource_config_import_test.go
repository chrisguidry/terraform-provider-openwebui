package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNestedConfigKeys_FlatExport(t *testing.T) {
	nested := nestedConfigKeys(map[string]any{
		"ui.banners":        []any{},
		"audio.stt.engine":  "openai",
		"webhook_url":       "https://example.invalid/hook",
		"models.default_id": nil,
	})
	if len(nested) != 0 {
		t.Fatalf("expected a v0.11.0 export to look flat, got %v", nested)
	}
}

func TestNestedConfigKeys_LegacyExport(t *testing.T) {
	nested := nestedConfigKeys(map[string]any{
		"ui":             map[string]any{"banners": []any{}},
		"code_execution": map[string]any{"enable": true},
		"webhook_url":    "https://example.invalid/hook",
	})
	if len(nested) != 2 || nested[0] != "code_execution" || nested[1] != "ui" {
		t.Fatalf("expected the nested keys sorted, got %v", nested)
	}
}

func TestRefreshedConfigJSON_NoDriftKeepsTheConfiguredText(t *testing.T) {
	current := types.StringValue(`{"ui.banners": [], "webhook_url": "https://example.invalid/hook"}`)
	managed := map[string]any{
		"ui.banners":  []any{},
		"webhook_url": "https://example.invalid/hook",
	}
	exported := map[string]any{
		"ui.banners":       []any{},
		"webhook_url":      "https://example.invalid/hook",
		"audio.stt.engine": "openai",
	}

	result, diags := refreshedConfigJSON(current, managed, exported)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !result.Equal(current) {
		t.Fatalf("expected the configured text to survive a clean refresh, got %s", result)
	}
}

func TestRefreshedConfigJSON_DriftReportsManagedKeysOnly(t *testing.T) {
	current := types.StringValue(`{"webhook_url":"https://example.invalid/hook"}`)
	managed := map[string]any{"webhook_url": "https://example.invalid/hook"}
	exported := map[string]any{
		"webhook_url":      "https://changed.invalid/hook",
		"audio.stt.engine": "openai",
	}

	result, diags := refreshedConfigJSON(current, managed, exported)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.ValueString() != `{"webhook_url":"https://changed.invalid/hook"}` {
		t.Fatalf("expected the drifted managed key alone, got %s", result.ValueString())
	}
}

func TestRefreshedConfigJSON_DeletedKeyIsDrift(t *testing.T) {
	current := types.StringValue(`{"webhook_url":"https://example.invalid/hook"}`)
	managed := map[string]any{"webhook_url": "https://example.invalid/hook"}
	exported := map[string]any{"audio.stt.engine": "openai"}

	result, _ := refreshedConfigJSON(current, managed, exported)
	if result.ValueString() != "{}" {
		t.Fatalf("expected a key the server dropped to show as drift, got %s", result.ValueString())
	}
}

// An empty object manages no keys, which is not the same as managing none yet.
func TestRefreshedConfigJSON_EmptyObjectManagesNothing(t *testing.T) {
	current := types.StringValue("{}")
	result, diags := refreshedConfigJSON(current, map[string]any{}, map[string]any{"audio.stt.engine": "openai"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !result.Equal(current) {
		t.Fatalf("expected an empty object to stay empty, got %s", result)
	}
}

// An imported resource has no config_json yet, so the whole export is the only
// sensible starting point.
func TestRefreshedConfigJSON_NoManagedKeysTakesTheWholeExport(t *testing.T) {
	exported := map[string]any{
		"webhook_url":      "https://example.invalid/hook",
		"audio.stt.engine": "openai",
	}

	result, diags := refreshedConfigJSON(types.StringNull(), nil, exported)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.ValueString() != `{"audio.stt.engine":"openai","webhook_url":"https://example.invalid/hook"}` {
		t.Fatalf("expected the whole export, got %s", result.ValueString())
	}
}
