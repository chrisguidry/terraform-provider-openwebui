package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// An attribute the practitioner leaves out stays out of the request, so Open
// WebUI applies its own default instead of storing a null.
func TestExpandTerminalServerOmitsUnsetAttributes(t *testing.T) {
	var diags diag.Diagnostics
	conn := expandTerminalServer(terminalServerModel{
		ServerID:   types.StringValue("shell"),
		URL:        types.StringValue("http://terminals.invalid"),
		Name:       types.StringNull(),
		Enabled:    types.BoolUnknown(),
		Path:       types.StringUnknown(),
		Key:        types.StringNull(),
		AuthType:   types.StringNull(),
		ConfigJSON: types.StringNull(),
		ServerType: types.StringNull(),
		PolicyID:   types.StringNull(),
	}, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if conn.ID != "shell" || conn.URL != "http://terminals.invalid" {
		t.Fatalf("expected the id and URL to carry through, got %+v", conn)
	}
	if conn.Name != nil || conn.Enabled != nil || conn.Path != nil || conn.Key != nil || conn.AuthType != nil {
		t.Fatalf("expected the unset attributes to stay nil, got %+v", conn)
	}
	if conn.Config != nil || conn.ServerType != nil || conn.PolicyID != nil {
		t.Fatalf("expected the unset attributes to stay nil, got %+v", conn)
	}
}

func TestExpandTerminalServerCarriesTheConfigObject(t *testing.T) {
	var diags diag.Diagnostics
	conn := expandTerminalServer(terminalServerModel{
		ServerID:   types.StringValue("shell"),
		URL:        types.StringValue("http://terminals.invalid"),
		Enabled:    types.BoolValue(false),
		ConfigJSON: types.StringValue(`{"timeout":30}`),
	}, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if conn.Enabled == nil || *conn.Enabled {
		t.Fatalf("expected enabled false to reach the payload, got %+v", conn.Enabled)
	}
	if conn.Config["timeout"] != float64(30) {
		t.Fatalf("expected the config object to be decoded, got %v", conn.Config)
	}
}

func TestExpandTerminalServerReportsInvalidConfigJSON(t *testing.T) {
	var diags diag.Diagnostics
	expandTerminalServer(terminalServerModel{
		ServerID:   types.StringValue("shell"),
		URL:        types.StringValue("http://terminals.invalid"),
		ConfigJSON: types.StringValue("not json"),
	}, &diags)

	if !diags.HasError() {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestFlattenTerminalServerFillsBothIdentifiers(t *testing.T) {
	var diags diag.Diagnostics
	path := "/openapi.json"
	enabled := true
	state := flattenTerminalServer(&client.TerminalServerConnection{
		ID:      "shell",
		URL:     "http://terminals.invalid",
		Path:    &path,
		Enabled: &enabled,
		Config:  map[string]any{"timeout": float64(30)},
	}, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.ID.ValueString() != "shell" || state.ServerID.ValueString() != "shell" {
		t.Fatalf("expected id and server_id to match, got %s and %s", state.ID, state.ServerID)
	}
	if state.ConfigJSON.ValueString() != `{"timeout":30}` {
		t.Fatalf("expected the config to be encoded, got %s", state.ConfigJSON)
	}
	if !state.Name.IsNull() || !state.Key.IsNull() {
		t.Fatalf("expected the absent fields to be null, got %s and %s", state.Name, state.Key)
	}
}
