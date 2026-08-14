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

func knowledgeResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r, ok := NewKnowledgeResource().(*knowledgeResource)
	if !ok {
		t.Fatal("NewKnowledgeResource() did not return *knowledgeResource")
	}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

func knowledgeDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	d, ok := NewKnowledgeDataSource().(*knowledgeDataSource)
	if !ok {
		t.Fatal("NewKnowledgeDataSource() did not return *knowledgeDataSource")
	}
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	return resp
}

// TestKnowledgeSchemaDropsInertJSONAttributes guards the removal of data_json
// and meta_json. KnowledgeForm carries neither, so both attributes were written
// to the API and silently discarded.
func TestKnowledgeSchemaDropsInertJSONAttributes(t *testing.T) {
	attributes := knowledgeResourceSchema(t).Schema.Attributes
	for _, name := range []string{"data_json", "meta_json"} {
		if _, exists := attributes[name]; exists {
			t.Errorf("%s is still in the knowledge resource schema", name)
		}
	}
	for _, name := range []string{"public_read", "public_write"} {
		if _, exists := attributes[name]; !exists {
			t.Errorf("%s is missing from the knowledge resource schema", name)
		}
	}
}

func TestKnowledgeDataSourceSchemaDropsInertJSONAttributes(t *testing.T) {
	attributes := knowledgeDataSourceSchema(t).Schema.Attributes
	for _, name := range []string{"data_json", "meta_json"} {
		if _, exists := attributes[name]; exists {
			t.Errorf("%s is still in the knowledge data source schema", name)
		}
	}
	for _, name := range []string{"public_read", "public_write", "file_count"} {
		if _, exists := attributes[name]; !exists {
			t.Errorf("%s is missing from the knowledge data source schema", name)
		}
	}
}

// TestKnowledgeModelMatchesSchema fails when the state struct and the schema
// disagree, which is otherwise only visible at apply time.
func TestKnowledgeModelMatchesSchema(t *testing.T) {
	ctx := context.Background()

	model := knowledgeResourceModel{
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
	}

	state := tfsdk.State{Schema: knowledgeResourceSchema(t).Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("knowledge resource model does not match its schema: %s", diags)
	}

	dataSourceState := tfsdk.State{Schema: knowledgeDataSourceSchema(t).Schema}
	if diags := dataSourceState.Set(ctx, knowledgeDataSourceModel{knowledgeResourceModel: model}); diags.HasError() {
		t.Fatalf("knowledge data source model does not match its schema: %s", diags)
	}
}

func TestKnowledgeResponseToModel_PublicGrants(t *testing.T) {
	ctx := context.Background()
	resp := client.KnowledgeFilesResponse{
		ID:          "k1",
		UserID:      "u1",
		Name:        "K",
		Description: "d",
		AccessControl: map[string]any{
			"read":         map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"write":        map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"public_read":  true,
			"public_write": true,
		},
	}

	model, diags := knowledgeResponseToModel(ctx, nil, resp)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !model.PublicRead.ValueBool() || !model.PublicWrite.ValueBool() {
		t.Fatalf("expected both public flags true, got read=%v write=%v", model.PublicRead, model.PublicWrite)
	}
}
