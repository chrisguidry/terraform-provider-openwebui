package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJSONStateValue_PlannedTextWins(t *testing.T) {
	planned := types.StringValue(`{"b": 2, "a": 1}`)
	result, diags := jsonStateValue(planned, map[string]any{"a": float64(1), "b": float64(2)}, "default_model_params_json")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !result.Equal(planned) {
		t.Fatalf("expected the planned text to survive, got %s", result)
	}
}

func TestJSONStateValue_UnknownPlanTakesTheServerValue(t *testing.T) {
	result, diags := jsonStateValue(types.StringUnknown(), map[string]any{"a": float64(1)}, "default_model_params_json")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.ValueString() != `{"a":1}` {
		t.Fatalf("expected the server value, got %s", result.ValueString())
	}
}

func TestJSONStateValue_NullServerValueIsNull(t *testing.T) {
	result, _ := jsonStateValue(types.StringUnknown(), nil, "default_model_metadata_json")
	if !result.IsNull() {
		t.Fatalf("expected null for an absent server value, got %s", result)
	}
}

func TestJSONRefreshValue_NoDriftKeepsTheRecordedText(t *testing.T) {
	recorded := types.StringValue(`{ "owner": "platform" }`)
	result, diags := jsonRefreshValue(recorded, map[string]any{"owner": "platform"}, "default_model_metadata_json")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !result.Equal(recorded) {
		t.Fatalf("expected the recorded text to survive a clean refresh, got %s", result)
	}
}

func TestJSONRefreshValue_DriftTakesTheServerValue(t *testing.T) {
	recorded := types.StringValue(`{"owner":"platform"}`)
	result, diags := jsonRefreshValue(recorded, map[string]any{"owner": "someone-else"}, "default_model_metadata_json")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result.ValueString() != `{"owner":"someone-else"}` {
		t.Fatalf("expected the server value, got %s", result.ValueString())
	}
}
