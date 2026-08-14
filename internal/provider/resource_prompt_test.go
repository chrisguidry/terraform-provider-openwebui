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

func promptResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r, ok := NewPromptResource().(*promptResource)
	if !ok {
		t.Fatal("NewPromptResource() did not return *promptResource")
	}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

func promptDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	d, ok := NewPromptDataSource().(*promptDataSource)
	if !ok {
		t.Fatal("NewPromptDataSource() did not return *promptDataSource")
	}
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	return resp
}

// TestPromptSchemaDropsIsActive guards the removal of is_active. PromptForm has
// no such field, so the attribute never reached the server.
func TestPromptSchemaDropsIsActive(t *testing.T) {
	resourceAttributes := promptResourceSchema(t).Schema.Attributes
	if _, exists := resourceAttributes["is_active"]; exists {
		t.Error("is_active is still in the prompt resource schema")
	}
	for _, name := range []string{"public_read", "public_write"} {
		if _, exists := resourceAttributes[name]; !exists {
			t.Errorf("%s is missing from the prompt resource schema", name)
		}
	}

	dataSourceAttributes := promptDataSourceSchema(t).Schema.Attributes
	if _, exists := dataSourceAttributes["is_active"]; exists {
		t.Error("is_active is still in the prompt data source schema")
	}
}

func TestPromptModelMatchesSchema(t *testing.T) {
	ctx := context.Background()
	model := promptResourceModel{
		Tags:        types.ListNull(types.StringType),
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
	}

	state := tfsdk.State{Schema: promptResourceSchema(t).Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("prompt resource model does not match its schema: %s", diags)
	}

	dataSourceState := tfsdk.State{Schema: promptDataSourceSchema(t).Schema}
	if diags := dataSourceState.Set(ctx, model); diags.HasError() {
		t.Fatalf("prompt data source model does not match its schema: %s", diags)
	}
}

func TestPromptResponseToModel_TagsAndJSON(t *testing.T) {
	ctx := context.Background()
	resp := &client.PromptModel{
		ID:      "p1",
		Command: "/greet",
		Name:    "Greet",
		Content: "Hello!",
		Tags:    []string{"util", "test"},
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	var tags []string
	if err := state.Tags.ElementsAs(ctx, &tags, false); err != nil {
		t.Fatalf("ElementsAs tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "util" || tags[1] != "test" {
		t.Fatalf("unexpected tags: %v", tags)
	}

	if !state.DataJSON.IsNull() {
		t.Fatalf("expected data_json to be null when Data is nil, got %v", state.DataJSON)
	}
	if !state.MetaJSON.IsNull() {
		t.Fatalf("expected meta_json to be null when Meta is nil, got %v", state.MetaJSON)
	}
}

func TestPromptResponseToModel_NilTags(t *testing.T) {
	ctx := context.Background()
	resp := &client.PromptModel{
		ID:      "p2",
		Command: "/test",
		Name:    "Test",
		Content: "content",
		Tags:    nil,
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if !state.Tags.IsNull() {
		t.Fatalf("expected nil tags to produce null list, got %v", state.Tags)
	}
}

func TestPromptResponseToModel_PublicGrants(t *testing.T) {
	ctx := context.Background()
	resp := &client.PromptModel{
		ID:      "p3",
		Command: "/public",
		Name:    "Public",
		Content: "content",
		AccessControl: map[string]any{
			"read":         map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"write":        map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"public_read":  true,
			"public_write": false,
		},
	}

	state, diags := promptResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if !state.PublicRead.ValueBool() {
		t.Fatalf("expected public_read=true, got %v", state.PublicRead)
	}
	if state.PublicWrite.ValueBool() {
		t.Fatalf("expected public_write=false, got %v", state.PublicWrite)
	}
}
