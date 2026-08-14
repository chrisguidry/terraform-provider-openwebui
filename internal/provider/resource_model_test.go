package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// --- mergeStringAnyMaps ---

func TestMergeStringAnyMaps_BothNil(t *testing.T) {
	result := mergeStringAnyMaps(nil, nil)
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

func TestMergeStringAnyMaps_PrimaryNil(t *testing.T) {
	secondary := map[string]any{"a": "1"}
	result := mergeStringAnyMaps(nil, secondary)
	if result["a"] != "1" {
		t.Fatalf("expected a=1, got %v", result["a"])
	}
}

func TestMergeStringAnyMaps_SecondaryNil(t *testing.T) {
	primary := map[string]any{"a": "1"}
	result := mergeStringAnyMaps(primary, nil)
	if result["a"] != "1" {
		t.Fatalf("expected a=1, got %v", result["a"])
	}
}

// Primary must win when both maps share a key — explicit Terraform attributes
// should never be overwritten by the "additional" JSON blob.
func TestMergeStringAnyMaps_PrimaryWinsOnConflict(t *testing.T) {
	primary := map[string]any{"key": "primary-value", "only-primary": "yes"}
	secondary := map[string]any{"key": "secondary-value", "only-secondary": "yes"}
	result := mergeStringAnyMaps(primary, secondary)

	if result["key"] != "primary-value" {
		t.Fatalf("expected primary-value to win, got %v", result["key"])
	}
	if result["only-primary"] != "yes" {
		t.Fatalf("expected only-primary key present, got %v", result["only-primary"])
	}
	if result["only-secondary"] != "yes" {
		t.Fatalf("expected only-secondary key present, got %v", result["only-secondary"])
	}
}

// --- flattenModelMeta ---

// When the API returns description=null, the key must not appear in
// meta_additional_json — previously it would leak in as "description":null,
// causing the provider to report a plan/apply state mismatch.
func TestFlattenModelMeta_NullDescriptionNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description":       nil,
		"profile_image_url": nil,
		"extra_field":       "should-remain",
	}

	_, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if additionalJSON.IsNull() {
		// No additional at all is fine only if extra_field is also absent, which
		// it shouldn't be — so fail with a clear message.
		t.Fatal("expected non-null additional JSON (extra_field should be present)")
	}

	raw := additionalJSON.ValueString()
	if strings.Contains(raw, "description") {
		t.Errorf("null description leaked into meta_additional_json: %s", raw)
	}
	if strings.Contains(raw, "profile_image_url") {
		t.Errorf("null profile_image_url leaked into meta_additional_json: %s", raw)
	}
	if !strings.Contains(raw, "extra_field") {
		t.Errorf("expected extra_field in meta_additional_json, got: %s", raw)
	}
}

// Sanity check: a non-null description should route to the structured state
// field only — not appear in additional at all.
func TestFlattenModelMeta_NonNullDescriptionNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description": "my model",
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.Description.ValueString() != "my model" {
		t.Fatalf("expected Description=my model, got %q", state.Description.ValueString())
	}

	// No extra keys in the input — additional must be null, not an object
	// containing description. The && in the old guard was a bug: it would
	// silently pass even if a regression put description back in additional.
	if !additionalJSON.IsNull() {
		t.Errorf("expected null additional JSON when no extra keys, got: %s", additionalJSON.ValueString())
	}
}

// Symmetric test for profile_image_url — the fix applied to both fields and
// both need regression coverage.
func TestFlattenModelMeta_NonNullProfileImageURLNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"profile_image_url": "https://example.com/img.png",
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ProfileImageURL.ValueString() != "https://example.com/img.png" {
		t.Fatalf("expected ProfileImageURL=https://example.com/img.png, got %q", state.ProfileImageURL.ValueString())
	}

	if !additionalJSON.IsNull() {
		t.Errorf("expected null additional JSON when no extra keys, got: %s", additionalJSON.ValueString())
	}
}

func TestFlattenModelMeta_Hidden(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": true,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.Hidden.IsNull() {
		t.Fatal("expected non-null hidden")
	}
	if !state.Hidden.ValueBool() {
		t.Fatalf("expected hidden=true, got false")
	}
}

func TestFlattenModelMeta_HiddenNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": true,
	}
	_, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !additionalJSON.IsNull() {
		t.Errorf("hidden must not appear in meta_additional_json, got: %s", additionalJSON.ValueString())
	}
}

func TestExpandModelMeta_Hidden(t *testing.T) {
	ctx := context.Background()
	plan := &modelResourceModel{
		Hidden:             types.BoolValue(true),
		SuggestionPrompts:  types.ListNull(types.StringType),
		Tags:               types.ListNull(types.StringType),
		ToolIDs:            types.ListNull(types.StringType),
		DefaultFeatureIDs:  types.ListNull(types.StringType),
		ProfileImageURL:    types.StringNull(),
		Description:        types.StringNull(),
		MetaAdditionalJSON: types.StringNull(),
		IsActive:           types.BoolNull(),
	}
	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if result["hidden"] != true {
		t.Fatalf("expected meta[hidden]=true, got %v", result["hidden"])
	}
}

func TestFlattenModelMeta_HiddenFalse(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": false,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if state.Hidden.IsNull() {
		t.Fatal("expected non-null hidden")
	}
	if state.Hidden.ValueBool() {
		t.Fatalf("expected hidden=false, got true")
	}
}

func TestFlattenModelMeta_HiddenAbsent(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"description": "no hidden key",
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !state.Hidden.IsNull() {
		t.Fatalf("expected null hidden when key absent, got %v", state.Hidden)
	}
}

func TestFlattenModelMeta_HiddenNull(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"hidden": nil,
	}
	state, _, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !state.Hidden.IsNull() {
		t.Fatalf("expected null hidden when API sends null, got %v", state.Hidden)
	}
}

func TestExpandModelMeta_HiddenFalse(t *testing.T) {
	ctx := context.Background()
	plan := &modelResourceModel{
		Hidden:             types.BoolValue(false),
		SuggestionPrompts:  types.ListNull(types.StringType),
		Tags:               types.ListNull(types.StringType),
		ToolIDs:            types.ListNull(types.StringType),
		DefaultFeatureIDs:  types.ListNull(types.StringType),
		ProfileImageURL:    types.StringNull(),
		Description:        types.StringNull(),
		MetaAdditionalJSON: types.StringNull(),
		IsActive:           types.BoolNull(),
	}
	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	val, ok := result["hidden"]
	if !ok {
		t.Fatal("expected hidden key in result")
	}
	if val != false {
		t.Fatalf("expected hidden=false, got %v", val)
	}
}

// --- meta keys the server owns ---

// ModelMeta declares knowledge, so every read carries the key. Left in
// meta_additional_json it fails the apply with an inconsistent result.
func TestFlattenModelMeta_NullKnowledgeNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"knowledge":   nil,
		"extra_field": "should-remain",
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if strings.Contains(additionalJSON.ValueString(), "knowledge") {
		t.Errorf("null knowledge leaked into meta_additional_json: %s", additionalJSON.ValueString())
	}
	if !state.KnowledgeIDs.IsNull() {
		t.Errorf("expected null knowledge_ids for a null knowledge key, got %v", state.KnowledgeIDs)
	}
}

// Open WebUI derives chat_variables_schema from params.system on read, so it is
// never something the configuration can hold.
func TestFlattenModelMeta_ChatVariablesSchemaNotLeakedToAdditional(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"chat_variables_schema": map[string]any{"name": map[string]any{"type": "string"}},
		"extra_field":           "should-remain",
	}

	_, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	raw := additionalJSON.ValueString()
	if strings.Contains(raw, "chat_variables_schema") {
		t.Errorf("chat_variables_schema leaked into meta_additional_json: %s", raw)
	}
	if !strings.Contains(raw, "extra_field") {
		t.Errorf("expected extra_field in meta_additional_json, got: %s", raw)
	}
}

// --- skill_ids ---

func TestFlattenModelMeta_SkillIDs(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"skillIds": []any{"skill-a", "skill-b"},
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var skills []string
	diags.Append(state.SkillIDs.ElementsAs(ctx, &skills, false)...)
	if len(skills) != 2 || skills[0] != "skill-a" || skills[1] != "skill-b" {
		t.Fatalf("expected [skill-a skill-b], got %+v", skills)
	}
	if !additionalJSON.IsNull() {
		t.Errorf("skillIds must not appear in meta_additional_json, got: %s", additionalJSON.ValueString())
	}
}

func TestExpandModelMeta_SkillIDs(t *testing.T) {
	ctx := context.Background()
	skills, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"skill-a"})
	plan := newTestModelPlan()
	plan.SkillIDs = skills

	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	got, ok := result["skillIds"].([]string)
	if !ok || len(got) != 1 || got[0] != "skill-a" {
		t.Fatalf("expected meta[skillIds]=[skill-a], got %#v", result["skillIds"])
	}
}

func TestExpandModelMeta_SkillIDsEmpty(t *testing.T) {
	ctx := context.Background()
	empty, _ := types.ListValueFrom(context.Background(), types.StringType, []string{})
	plan := newTestModelPlan()
	plan.SkillIDs = empty

	var diags diag.Diagnostics
	result := expandModelMeta(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	got, ok := result["skillIds"].([]any)
	if !ok || len(got) != 0 {
		t.Fatalf("expected meta[skillIds]=[], got %#v", result["skillIds"])
	}
}

// --- knowledge_ids ---

func TestFlattenModelMeta_KnowledgeIDs(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{
		"knowledge": []any{
			map[string]any{"type": "collection", "id": "kb-1", "name": "Recipes"},
			map[string]any{"type": "file", "id": "file-9", "name": "notes.txt"},
			map[string]any{"type": "collection", "id": "kb-2", "name": "Manuals"},
		},
	}

	state, additionalJSON, diags := flattenModelMeta(ctx, data)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var ids []string
	diags.Append(state.KnowledgeIDs.ElementsAs(ctx, &ids, false)...)
	if len(ids) != 2 || ids[0] != "kb-1" || ids[1] != "kb-2" {
		t.Fatalf("expected [kb-1 kb-2], got %+v", ids)
	}
	if !additionalJSON.IsNull() {
		t.Errorf("knowledge must not appear in meta_additional_json, got: %s", additionalJSON.ValueString())
	}
}

func newKnowledgeTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestApplyKnowledgeToMeta_ResolvesIDs(t *testing.T) {
	ctx := context.Background()
	apiClient := newKnowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"kb-1","user_id":"u1","name":"Recipes","description":"Family recipes","created_at":1,"updated_at":2,"files":[]}`))
	})

	ids, _ := types.ListValueFrom(ctx, types.StringType, []string{"kb-1"})
	var diags diag.Diagnostics
	meta := applyKnowledgeToMeta(ctx, apiClient, ids, map[string]any{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	items, ok := meta["knowledge"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one knowledge entry, got %#v", meta["knowledge"])
	}
	entry, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected a knowledge object, got %T", items[0])
	}
	if entry["type"] != "collection" || entry["id"] != "kb-1" || entry["name"] != "Recipes" {
		t.Fatalf("unexpected knowledge entry: %+v", entry)
	}
}

// An empty list clears meta.knowledge; an unset attribute leaves whatever the
// server holds alone.
func TestApplyKnowledgeToMeta_EmptyListClears(t *testing.T) {
	ctx := context.Background()
	empty, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	var diags diag.Diagnostics
	meta := applyKnowledgeToMeta(ctx, nil, empty, map[string]any{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	items, ok := meta["knowledge"].([]any)
	if !ok || len(items) != 0 {
		t.Fatalf("expected meta[knowledge]=[], got %#v", meta["knowledge"])
	}
}

func TestApplyKnowledgeToMeta_NullLeavesMetaAlone(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	meta := applyKnowledgeToMeta(ctx, nil, types.ListNull(types.StringType), map[string]any{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if _, present := meta["knowledge"]; present {
		t.Fatalf("expected no knowledge key, got %#v", meta["knowledge"])
	}
}

func TestApplyKnowledgeToMeta_UnknownKnowledgeBase(t *testing.T) {
	ctx := context.Background()
	apiClient := newKnowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	ids, _ := types.ListValueFrom(ctx, types.StringType, []string{"missing"})
	var diags diag.Diagnostics
	applyKnowledgeToMeta(ctx, apiClient, ids, map[string]any{}, &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for a knowledge base that does not exist")
	}
}

// --- capabilities ---

func TestModelCapabilities_RoundTrip(t *testing.T) {
	caps := flattenModelCapabilities(map[string]any{
		"file_context":  false,
		"terminal":      false,
		"memory":        true,
		"builtin_tools": false,
	})

	if caps.FileContext.ValueBool() || caps.Terminal.ValueBool() || caps.BuiltinTools.ValueBool() {
		t.Fatalf("expected file_context, terminal and builtin_tools false, got %+v", caps)
	}
	if !caps.Memory.ValueBool() {
		t.Fatalf("expected memory true, got %+v", caps.Memory)
	}

	expanded := expandModelCapabilities(caps)
	for key, want := range map[string]bool{"file_context": false, "terminal": false, "memory": true, "builtin_tools": false} {
		if expanded[key] != want {
			t.Errorf("expected capabilities[%s]=%t, got %#v", key, want, expanded[key])
		}
	}
}

// Every capability in the schema needs a matching attribute type, or the plan
// modifier builds an object the framework cannot decode.
func TestModelCapabilitiesAttrTypes_CoverEveryCapability(t *testing.T) {
	expanded := expandModelCapabilities(&modelCapabilitiesModel{
		Vision:          types.BoolValue(true),
		FileUpload:      types.BoolValue(true),
		FileContext:     types.BoolValue(true),
		WebSearch:       types.BoolValue(true),
		ImageGeneration: types.BoolValue(true),
		CodeInterpreter: types.BoolValue(true),
		Terminal:        types.BoolValue(true),
		Memory:          types.BoolValue(true),
		BuiltinTools:    types.BoolValue(true),
		Citations:       types.BoolValue(true),
		StatusUpdates:   types.BoolValue(true),
		Usage:           types.BoolValue(true),
	})

	if len(expanded) != len(modelCapabilitiesAttrTypes) {
		t.Fatalf("expected %d capabilities, got %d: %+v", len(modelCapabilitiesAttrTypes), len(expanded), expanded)
	}
	for key := range expanded {
		if _, ok := modelCapabilitiesAttrTypes[key]; !ok {
			t.Errorf("capability %q has no attribute type", key)
		}
	}
}

// --- public_read and public_write ---

// A wildcard grant read off the server has to reach state, or a model shared
// publicly in the web UI loses that sharing on the next apply.
func TestModelResponseToModel_PublicFlags(t *testing.T) {
	ctx := context.Background()
	resp := &client.ModelResponse{
		ID:            "m1",
		Name:          "M",
		Meta:          map[string]any{},
		Params:        map[string]any{},
		AccessControl: map[string]any{"public_read": true, "public_write": false},
	}

	state, diags := modelResponseToModel(ctx, nil, resp, "m1")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if !state.PublicRead.ValueBool() {
		t.Error("expected public_read=true")
	}
	if state.PublicWrite.ValueBool() {
		t.Error("expected public_write=false")
	}
}

// A model shared with nobody must read back as not public, never as null.
func TestModelResponseToModel_PublicFlagsDefaultFalse(t *testing.T) {
	ctx := context.Background()
	resp := &client.ModelResponse{ID: "m1", Name: "M", Meta: map[string]any{}, Params: map[string]any{}}

	state, diags := modelResponseToModel(ctx, nil, resp, "m1")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.PublicRead.IsNull() || state.PublicWrite.IsNull() {
		t.Fatalf("expected known public flags, got read=%v write=%v", state.PublicRead, state.PublicWrite)
	}
	if state.PublicRead.ValueBool() || state.PublicWrite.ValueBool() {
		t.Fatalf("expected both public flags false, got read=%v write=%v", state.PublicRead, state.PublicWrite)
	}
}

// --- profile_image_url ---

func TestProfileImageURLValidator(t *testing.T) {
	cases := []struct {
		value   string
		allowed bool
	}{
		{"", true},
		{"/user.png", true},
		{"/favicon.png", true},
		{"/static/favicon.png", true},
		{"/api/v1/users/abc-123/profile/image", true},
		{"https://example.com/img.png", true},
		{"http://example.com/img.png", true},
		{"data:image/png;base64,AAAA", true},
		{"data:image/WEBP;BASE64,AAAA", true},
		{"/img.png", false},
		{"/static/other.png", false},
		{"data:image/svg+xml;base64,AAAA", false},
		{"javascript:alert(1)", false},
		{"//example.com/img.png", false},
		{"http:///img.png", false},
		{"/api/v1/users/abc/profile/image/extra", false},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			resp := &validator.StringResponse{}
			profileImageURLValidator{}.ValidateString(
				context.Background(),
				validator.StringRequest{
					Path:        path.Root("profile_image_url"),
					ConfigValue: types.StringValue(tc.value),
				},
				resp,
			)

			if tc.allowed && resp.Diagnostics.HasError() {
				t.Fatalf("expected %q to be accepted, got %s", tc.value, resp.Diagnostics)
			}
			if !tc.allowed && !resp.Diagnostics.HasError() {
				t.Fatalf("expected %q to be rejected", tc.value)
			}
		})
	}
}

func TestProfileImageURLValidator_SkipsNullAndUnknown(t *testing.T) {
	for name, value := range map[string]types.String{"null": types.StringNull(), "unknown": types.StringUnknown()} {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			profileImageURLValidator{}.ValidateString(
				context.Background(),
				validator.StringRequest{Path: path.Root("profile_image_url"), ConfigValue: value},
				resp,
			)
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no error for a %s value, got %s", name, resp.Diagnostics)
			}
		})
	}
}

// newTestModelPlan builds a plan with every list and string attribute null, so
// a test can set the one attribute it cares about.
func newTestModelPlan() *modelResourceModel {
	return &modelResourceModel{
		Hidden:             types.BoolNull(),
		SuggestionPrompts:  types.ListNull(types.StringType),
		Tags:               types.ListNull(types.StringType),
		ToolIDs:            types.ListNull(types.StringType),
		SkillIDs:           types.ListNull(types.StringType),
		KnowledgeIDs:       types.ListNull(types.StringType),
		DefaultFeatureIDs:  types.ListNull(types.StringType),
		ProfileImageURL:    types.StringNull(),
		Description:        types.StringNull(),
		MetaAdditionalJSON: types.StringNull(),
		IsActive:           types.BoolNull(),
	}
}
