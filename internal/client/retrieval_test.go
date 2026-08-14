package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// The retrieval config stores rag.mineru_api_timeout from an environment
// variable, so a fresh install reads it back as a string although the form
// declares it an integer.
func TestNumericIntAcceptsBothShapes(t *testing.T) {
	cases := []struct {
		raw  string
		want NumericInt
	}{
		{`300`, 300},
		{`"300"`, 300},
	}

	for _, tc := range cases {
		var got NumericInt
		if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Errorf("unmarshal %s: got %d, want %d", tc.raw, got, tc.want)
		}
	}

	var got NumericInt
	if err := json.Unmarshal([]byte(`"five minutes"`), &got); err == nil {
		t.Error("expected text that is not a number to fail")
	}
}

func TestNumericStringAcceptsBothShapes(t *testing.T) {
	cases := []struct {
		raw  string
		want NumericString
	}{
		{`1024`, "1024"},
		{`"1024"`, "1024"},
		{`""`, ""},
	}

	for _, tc := range cases {
		var got NumericString
		if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Errorf("unmarshal %s: got %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// A whole number goes back as a number, so the stored value keeps the type the
// rest of Open WebUI reads it as. The empty string goes back as it stands
// because that is what clears the limit.
func TestNumericStringMarshalsWholeNumbersAsNumbers(t *testing.T) {
	cases := []struct {
		value NumericString
		want  string
	}{
		{"1024", `1024`},
		{"", `""`},
		{"unlimited", `"unlimited"`},
	}

	for _, tc := range cases {
		encoded, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatalf("marshal %q: %v", tc.value, err)
		}
		if string(encoded) != tc.want {
			t.Errorf("marshal %q: got %s, want %s", tc.value, encoded, tc.want)
		}
	}
}

// The update route answers without five of the keys the read route returns, so
// the client reads the config back rather than trusting the response.
func TestSetRAGConfigReadsTheConfigBack(t *testing.T) {
	var paths []string
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"status":true,"CHUNK_SIZE":1000}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":true,"CHUNK_SIZE":1000,"MINERU_FILE_EXTENSIONS":["pdf"],"RAG_RERANKING_BATCH_SIZE":32}`))
	})

	updated, err := c.SetRAGConfig(context.Background(), RAGConfigForm{})
	if err != nil {
		t.Fatalf("SetRAGConfig: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/retrieval/config/update" || paths[1] != "/api/v1/retrieval/config" {
		t.Fatalf("expected a write then a read, got %v", paths)
	}

	if updated.RAGRerankingBatchSize == nil || *updated.RAGRerankingBatchSize != 32 {
		t.Fatalf("expected the read to supply the key the write left out, got %v", updated.RAGRerankingBatchSize)
	}
}

// A key the plan does not name must not reach the request, because the
// retrieval route keeps the stored value only for keys the form leaves out.
func TestSetRAGConfigOmitsUnsetKeys(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body = captureRequestBody(t, r)
		}
		_, _ = w.Write([]byte(`{"status":true}`))
	})

	size := int64(1000)
	if _, err := c.SetRAGConfig(context.Background(), RAGConfigForm{ChunkSize: &size}); err != nil {
		t.Fatalf("SetRAGConfig: %v", err)
	}

	if len(body) != 1 {
		t.Fatalf("expected only CHUNK_SIZE in the request body, got %v", body)
	}
	if body["CHUNK_SIZE"] != float64(1000) {
		t.Fatalf("expected CHUNK_SIZE 1000, got %v", body["CHUNK_SIZE"])
	}
}

// An empty list is a value the form must carry, and a missing list must stay
// missing: WEB_SEARCH_DOMAIN_FILTER_LIST and ALLOWED_FILE_EXTENSIONS both
// reject an explicit null.
func TestSetRAGConfigCarriesAnEmptyList(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body = captureRequestBody(t, r)
		}
		_, _ = w.Write([]byte(`{"status":true}`))
	})

	empty := []string{}
	if _, err := c.SetRAGConfig(context.Background(), RAGConfigForm{AllowedFileExtensions: &empty}); err != nil {
		t.Fatalf("SetRAGConfig: %v", err)
	}

	extensions, ok := body["ALLOWED_FILE_EXTENSIONS"].([]any)
	if !ok || len(extensions) != 0 {
		t.Fatalf("expected an empty ALLOWED_FILE_EXTENSIONS array, got %v", body)
	}
}

func TestGetRAGEmbeddingConfigReadsTheEngineBlocks(t *testing.T) {
	c := newConfigsTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":true,"RAG_EMBEDDING_ENGINE":"openai","RAG_EMBEDDING_MODEL":"text-embedding-3-small",` +
			`"openai_config":{"url":"https://api.openai.com/v1","key":"sk-test"},` +
			`"ollama_config":{"url":"","key":""},` +
			`"azure_openai_config":{"url":"","key":"","version":"2024-02-01"}}`))
	})

	config, err := c.GetRAGEmbeddingConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRAGEmbeddingConfig: %v", err)
	}

	if config.OpenAIConfig == nil || config.OpenAIConfig.Key == nil || *config.OpenAIConfig.Key != "sk-test" {
		t.Fatalf("expected the openai key to read back in plaintext, got %v", config.OpenAIConfig)
	}
	if config.AzureOpenAIConfig == nil || config.AzureOpenAIConfig.Version == nil || *config.AzureOpenAIConfig.Version != "2024-02-01" {
		t.Fatalf("expected the azure api version, got %v", config.AzureOpenAIConfig)
	}
}

// An engine block the configuration leaves out must not reach the request: the
// update route keeps the stored url and key only when the block is absent.
func TestSetRAGEmbeddingConfigOmitsAbsentEngineBlocks(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"status":true,"RAG_EMBEDDING_ENGINE":"openai"}`))
	})

	engine := "openai"
	url := "https://api.openai.com/v1"
	key := "sk-test"
	_, err := c.SetRAGEmbeddingConfig(context.Background(), RAGEmbeddingConfigForm{
		RAGEmbeddingEngine: &engine,
		OpenAIConfig:       &RAGEngineConfigForm{URL: &url, Key: &key},
	})
	if err != nil {
		t.Fatalf("SetRAGEmbeddingConfig: %v", err)
	}

	if _, exists := body["openai_config"]; !exists {
		t.Fatalf("expected openai_config in the request body, got %v", body)
	}
	for _, name := range []string{"ollama_config", "azure_openai_config"} {
		if _, exists := body[name]; exists {
			t.Errorf("%s reached the request although the configuration names no such block", name)
		}
	}

	openai, ok := body["openai_config"].(map[string]any)
	if !ok {
		t.Fatalf("openai_config is not an object: %v", body["openai_config"])
	}
	if _, exists := openai["version"]; exists {
		t.Error("version reached an openai block, which has no such field")
	}
}
