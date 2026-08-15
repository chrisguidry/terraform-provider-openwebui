package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func toolResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r, ok := NewToolResource().(*toolResource)
	if !ok {
		t.Fatal("NewToolResource() did not return *toolResource")
	}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

func toolDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	d, ok := NewToolDataSource().(*toolDataSource)
	if !ok {
		t.Fatal("NewToolDataSource() did not return *toolDataSource")
	}
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	return resp
}

func TestToolSchemaCarriesPublicAndUserValves(t *testing.T) {
	wanted := []string{"public_read", "public_write", "has_user_valves"}

	resourceAttributes := toolResourceSchema(t).Schema.Attributes
	for _, name := range wanted {
		if _, exists := resourceAttributes[name]; !exists {
			t.Errorf("%s is missing from the tool resource schema", name)
		}
	}

	dataSourceAttributes := toolDataSourceSchema(t).Schema.Attributes
	for _, name := range wanted {
		if _, exists := dataSourceAttributes[name]; !exists {
			t.Errorf("%s is missing from the tool data source schema", name)
		}
	}
}

func TestToolModelMatchesSchema(t *testing.T) {
	ctx := context.Background()

	state := tfsdk.State{Schema: toolResourceSchema(t).Schema}
	model := toolResourceModel{
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
		ReadUsers:   types.ListNull(types.StringType),
		WriteUsers:  types.ListNull(types.StringType),
	}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("tool resource model does not match its schema: %s", diags)
	}

	dataSourceState := tfsdk.State{Schema: toolDataSourceSchema(t).Schema}
	dataSourceModel := toolDataSourceModel{
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
		ReadUsers:   types.ListNull(types.StringType),
		WriteUsers:  types.ListNull(types.StringType),
	}
	if diags := dataSourceState.Set(ctx, dataSourceModel); diags.HasError() {
		t.Fatalf("tool data source model does not match its schema: %s", diags)
	}
}

func TestToolResponseToModel_UserValvesAndPublicGrants(t *testing.T) {
	ctx := context.Background()
	hasUserValves := true
	access := &client.ToolAccessResponse{
		ID:     "t1",
		UserID: "u1",
		Name:   "T",
		Meta:   client.ToolMeta{HasUserValves: &hasUserValves},
		AccessControl: map[string]any{
			"read":         map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"write":        map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"public_read":  true,
			"public_write": false,
		},
	}

	state, diags := toolResponseToModel(ctx, nil, access, "content", nil, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !state.HasUserValves.ValueBool() {
		t.Fatalf("expected has_user_valves=true, got %v", state.HasUserValves)
	}
	if !state.PublicRead.ValueBool() {
		t.Fatalf("expected public_read=true, got %v", state.PublicRead)
	}
	if state.PublicWrite.ValueBool() {
		t.Fatalf("expected public_write=false, got %v", state.PublicWrite)
	}
}

// TestToolResponseToModel_MissingUserValves records the older servers that omit
// the key: has_user_valves reads as false rather than null.
func TestToolResponseToModel_MissingUserValves(t *testing.T) {
	ctx := context.Background()
	access := &client.ToolAccessResponse{ID: "t2", UserID: "u1", Name: "T"}

	state, diags := toolResponseToModel(ctx, nil, access, "content", nil, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.HasUserValves.IsNull() || state.HasUserValves.ValueBool() {
		t.Fatalf("expected has_user_valves=false, got %v", state.HasUserValves)
	}
}
