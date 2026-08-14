package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// AddPipelineForm declares url and urlIdx only. The pipeline server's API key
// comes from the OpenAI connection at urlIdx, so a key in the body is dropped.
func TestAddPipelineSendsOnlyURLAndIndex(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"id":"pipe1","name":"Pipe"}`))
	}))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.AddPipeline(context.Background(), "http://pipelines.invalid:9099", 0); err != nil {
		t.Fatalf("AddPipeline: %v", err)
	}

	if len(body) != 2 {
		t.Fatalf("expected url and urlIdx alone, got %v", body)
	}
	if body["url"] != "http://pipelines.invalid:9099" {
		t.Fatalf("expected the pipeline url, got %v", body["url"])
	}
	if _, ok := body["key"]; ok {
		t.Fatalf("expected no key field, got %v", body)
	}
}
