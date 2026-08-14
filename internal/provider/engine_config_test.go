package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func engineResourceSchema(t *testing.T, constructor func() resource.Resource) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	constructor().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema failed: %s", resp.Diagnostics)
	}

	return resp.Schema
}

func nestedAttributes(t *testing.T, parent schema.Schema, name string) map[string]schema.Attribute {
	t.Helper()

	attribute, exists := parent.Attributes[name]
	if !exists {
		t.Fatalf("%s is missing from the schema", name)
	}

	nested, ok := attribute.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("%s is not a single nested attribute", name)
	}

	return nested.Attributes
}

// TestEngineConfigSchemasCoverEveryField counts the attributes against the
// Pydantic forms in backend/open_webui/routers. A field that falls out of a
// schema is otherwise invisible until someone tries to set it.
func TestEngineConfigSchemasCoverEveryField(t *testing.T) {
	ragConfig := engineResourceSchema(t, NewRAGConfigResource)
	audioConfig := engineResourceSchema(t, NewAudioConfigResource)

	cases := []struct {
		name       string
		attributes map[string]schema.Attribute
		want       int
	}{
		// ConfigForm's 64 fields, the web block, and id.
		{"rag_config", ragConfig.Attributes, 66},
		// WebConfig's 74 fields, less YOUTUBE_LOADER_TRANSLATION.
		{"rag_config.web", nestedAttributes(t, ragConfig, "web"), 73},
		// ImagesConfig's 33 fields and id.
		{"images_config", engineResourceSchema(t, NewImagesConfigResource).Attributes, 34},
		// TTSConfigForm's 13 fields.
		{"audio_config.tts", nestedAttributes(t, audioConfig, "tts"), 13},
		// STTConfigForm's 17 fields.
		{"audio_config.stt", nestedAttributes(t, audioConfig, "stt"), 17},
	}

	for _, tc := range cases {
		if len(tc.attributes) != tc.want {
			t.Errorf("%s has %d attributes, want %d", tc.name, len(tc.attributes), tc.want)
		}
	}
}

// TestEngineConfigCredentialsAreSensitive checks every attribute that names a
// credential. Open WebUI reads all of these back in plaintext, so the schema is
// the only thing keeping them out of plan output.
func TestEngineConfigCredentialsAreSensitive(t *testing.T) {
	ragConfig := engineResourceSchema(t, NewRAGConfigResource)
	audioConfig := engineResourceSchema(t, NewAudioConfigResource)
	embedding := engineResourceSchema(t, NewRAGEmbeddingConfigResource)

	groups := map[string]map[string]schema.Attribute{
		"rag_config":       ragConfig.Attributes,
		"rag_config.web":   nestedAttributes(t, ragConfig, "web"),
		"images_config":    engineResourceSchema(t, NewImagesConfigResource).Attributes,
		"audio_config.tts": nestedAttributes(t, audioConfig, "tts"),
		"audio_config.stt": nestedAttributes(t, audioConfig, "stt"),
		"embedding.openai": nestedAttributes(t, embedding, "openai_config"),
		"embedding.ollama": nestedAttributes(t, embedding, "ollama_config"),
		"embedding.azure":  nestedAttributes(t, embedding, "azure_openai_config"),
	}

	suffixes := []string{"api_key", "_key", "_token", "_password", "_secret", "_api_sk", "_subscription_key", "_api_auth"}

	for group, attributes := range groups {
		for name, attribute := range attributes {
			credential := name == "key" || name == "api_key"
			for _, suffix := range suffixes {
				if strings.HasSuffix(name, suffix) {
					credential = true
				}
			}
			if credential && !attribute.IsSensitive() {
				t.Errorf("%s.%s is a credential but is not marked sensitive", group, name)
			}
		}
	}
}

// TestRAGConfigSchemaOmitsUnreachableKeys guards the four keys the retrieval
// routes load but never read or write, and YOUTUBE_LOADER_TRANSLATION, which the
// update route keeps in process memory and loses on restart.
func TestRAGConfigSchemaOmitsUnreachableKeys(t *testing.T) {
	ragConfig := engineResourceSchema(t, NewRAGConfigResource)

	for _, name := range []string{
		"azure_ai_search_api_key",
		"azure_ai_search_endpoint",
		"azure_ai_search_index_name",
		"tiktoken_encoding_name",
		"user_permissions",
		"webui_url",
	} {
		if _, exists := ragConfig.Attributes[name]; exists {
			t.Errorf("%s is in the rag config schema but no retrieval route reads or writes it", name)
		}
	}

	web := nestedAttributes(t, ragConfig, "web")
	if _, exists := web["youtube_loader_translation"]; exists {
		t.Error("youtube_loader_translation is in the web block but has no storage key")
	}
}

// TestImagesConfigSchemaOmitsUserPermissions guards IMAGE_CONFIG_KEYS' one entry
// that ImagesConfig has no field for. The router drops it from both directions.
func TestImagesConfigSchemaOmitsUserPermissions(t *testing.T) {
	attributes := engineResourceSchema(t, NewImagesConfigResource).Attributes
	if _, exists := attributes["user_permissions"]; exists {
		t.Error("user_permissions is in the images config schema but the router never touches it")
	}
}

// TestRAGConfigFileLimitsAreStrings guards the four keys that read the empty
// string as "no limit". An integer attribute cannot express that.
func TestRAGConfigFileLimitsAreStrings(t *testing.T) {
	attributes := engineResourceSchema(t, NewRAGConfigResource).Attributes

	for _, name := range []string{
		"file_max_size",
		"file_max_count",
		"file_image_compression_width",
		"file_image_compression_height",
	} {
		if _, ok := attributes[name].(schema.StringAttribute); !ok {
			t.Errorf("%s is %T, want a string attribute so it can carry the empty string", name, attributes[name])
		}
	}
}

// TestEngineConfigModelsMatchTheirSchemas fails when a state struct and its
// schema disagree, which is otherwise only visible at apply time.
func TestEngineConfigModelsMatchTheirSchemas(t *testing.T) {
	ctx := context.Background()

	ragModel := ragConfigModel{
		ContentExtractionSupportedMediaMimeTypes: types.ListNull(types.StringType),
		MineruFileExtensions:                     types.ListNull(types.StringType),
		AllowedFileExtensions:                    types.ListNull(types.StringType),
		Web: &ragWebConfigModel{
			WebSearchDomainFilterList: types.ListNull(types.StringType),
			YoutubeLoaderLanguage:     types.ListNull(types.StringType),
		},
	}

	audioModel := audioConfigModel{
		TTS: &audioTTSConfigModel{},
		STT: &audioSTTConfigModel{
			SupportedContentTypes: types.ListNull(types.StringType),
			AllowedExtensions:     types.ListNull(types.StringType),
		},
	}

	cases := []struct {
		name   string
		schema schema.Schema
		model  any
	}{
		{"rag_config", engineResourceSchema(t, NewRAGConfigResource), ragModel},
		{"images_config", engineResourceSchema(t, NewImagesConfigResource), imagesConfigModel{}},
		{"audio_config", engineResourceSchema(t, NewAudioConfigResource), audioModel},
		{"rag_embedding_config", engineResourceSchema(t, NewRAGEmbeddingConfigResource), ragEmbeddingConfigModel{}},
	}

	for _, tc := range cases {
		state := tfsdk.State{Schema: tc.schema}
		if diags := state.Set(ctx, tc.model); diags.HasError() {
			t.Errorf("%s model does not match its schema: %s", tc.name, diags)
		}
	}
}

// TestImageSizeValidator mirrors the check the update route runs, so a bad size
// fails the plan instead of the apply.
func TestImageSizeValidator(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"512x512", true},
		{"1024x1536", true},
		{"auto", true},
		{"", true},
		{"512", false},
		{"512 x 512", false},
		{"large", false},
	}

	for _, tc := range cases {
		if got := imageSizePattern.MatchString(tc.value); got != tc.valid {
			t.Errorf("image size %q: matched=%v, want %v", tc.value, got, tc.valid)
		}
	}
}

func TestEngineStringKeepsTheStoredValue(t *testing.T) {
	stored := "http://comfy:8188"

	if got := engineString(types.StringNull(), &stored); got != &stored {
		t.Fatalf("a null plan value should keep the stored value, got %v", got)
	}

	if got := engineString(types.StringUnknown(), &stored); got != &stored {
		t.Fatalf("an unknown plan value should keep the stored value, got %v", got)
	}

	got := engineString(types.StringValue("http://other:8188"), &stored)
	if got == nil || *got != "http://other:8188" {
		t.Fatalf("a planned value should win, got %v", got)
	}
}

func TestEngineStringSendsTheEmptyString(t *testing.T) {
	stored := "something"

	got := engineString(types.StringValue(""), &stored)
	if got == nil || *got != "" {
		t.Fatalf("the empty string is a value, not an absence, got %v", got)
	}
}

func TestEngineNumericStringValueKeepsTheClearedSentinel(t *testing.T) {
	got := engineNumericStringValue(types.StringValue(""), nil)
	if got.IsNull() || got.ValueString() != "" {
		t.Fatalf("a cleared limit should stay the empty string, got %s", got)
	}

	got = engineNumericStringValue(types.StringNull(), nil)
	if !got.IsNull() {
		t.Fatalf("an unset limit should stay null, got %s", got)
	}

	stored := client.NumericString("1024")
	got = engineNumericStringValue(types.StringValue(""), &stored)
	if got.ValueString() != "1024" {
		t.Fatalf("a stored limit should win over the recorded sentinel, got %s", got)
	}
}

func TestEngineTrimmedURLDropsTheTrailingSlash(t *testing.T) {
	value := "http://comfy:8188/"

	got := engineTrimmedURL(&value)
	if got == nil || *got != "http://comfy:8188" {
		t.Fatalf("expected the slash to go, got %v", got)
	}

	if engineTrimmedURL(nil) != nil {
		t.Fatal("a missing URL should stay missing")
	}
}

// Terraform fails an apply whose result differs from the plan, so a URL the
// configuration wrote with a slash has to stay in state exactly as written
// while it still names the server Open WebUI stored.
func TestEngineTrimmedURLValueKeepsTheConfiguredForm(t *testing.T) {
	stored := "http://comfy:8188"

	got := engineTrimmedURLValue(types.StringValue("http://comfy:8188/"), &stored)
	if got.ValueString() != "http://comfy:8188/" {
		t.Fatalf("expected the configured form to survive, got %q", got.ValueString())
	}
}

func TestEngineTrimmedURLValueTakesADifferentServer(t *testing.T) {
	stored := "http://elsewhere:8188"

	got := engineTrimmedURLValue(types.StringValue("http://comfy:8188/"), &stored)
	if got.ValueString() != "http://elsewhere:8188" {
		t.Fatalf("expected the server's value, got %q", got.ValueString())
	}

	if !engineTrimmedURLValue(types.StringNull(), nil).IsNull() {
		t.Fatal("a missing URL should stay missing")
	}
}

func TestEngineJSONValueKeepsTheRecordedText(t *testing.T) {
	recorded := types.StringValue(`{ "b": 2, "a": 1 }`)
	diags := &diag.Diagnostics{}

	got := engineJSONValue(recorded, map[string]any{"a": float64(1), "b": float64(2)}, "docling_params", diags)
	if !got.Equal(recorded) {
		t.Fatalf("expected the recorded text to survive a clean refresh, got %s", got)
	}

	got = engineJSONValue(recorded, map[string]any{"a": float64(9)}, "docling_params", diags)
	if got.ValueString() != `{"a":9}` {
		t.Fatalf("expected drift to take the server value, got %s", got)
	}

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestEngineJSONValueCarriesAnArray(t *testing.T) {
	diags := &diag.Diagnostics{}

	got := engineJSONValue(types.StringNull(), []any{map[string]any{"node": "4"}}, "comfyui_workflow_nodes", diags)
	if got.ValueString() != `[{"node":"4"}]` {
		t.Fatalf("expected the array to survive, got %s", got)
	}

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func TestEngineJSONRejectsInvalidText(t *testing.T) {
	diags := &diag.Diagnostics{}

	engineJSON(types.StringValue("{not json"), nil, path.Root("docling_params"), diags)
	if !diags.HasError() {
		t.Fatal("expected invalid JSON to produce an error")
	}
}

func TestEngineStringListValueDistinguishesEmptyFromMissing(t *testing.T) {
	ctx := context.Background()
	diags := &diag.Diagnostics{}

	if got := engineStringListValue(ctx, nil, diags); !got.IsNull() {
		t.Fatalf("a missing list should be null, got %s", got)
	}

	empty := []string{}
	got := engineStringListValue(ctx, &empty, diags)
	if got.IsNull() || len(got.Elements()) != 0 {
		t.Fatalf("an empty list should stay an empty list, got %s", got)
	}

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}
