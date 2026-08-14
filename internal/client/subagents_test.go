package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newRecordingClient points a client at a handler the test controls.
func newRecordingClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return c
}

// readJSONBody decodes the request body a handler received.
func readJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	return body
}

func TestGetSubagentsConfigReadsEveryField(t *testing.T) {
	var path string
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"ENABLE_SUBAGENTS":true,"SUBAGENTS_BACKGROUND_ENABLED":false,"SUBAGENTS_MAX_CONCURRENT":3,"SUBAGENTS_MAX_ASYNC":2,"SUBAGENTS_MAX_ITERATIONS":10,"SUBAGENTS_MAX_OUTPUT":8000,"SUBAGENTS_SYSTEM_PROMPT":"be brief"}`))
	})

	config, err := c.GetSubagentsConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSubagentsConfig: %v", err)
	}

	if path != "/api/v1/configs/subagents" {
		t.Fatalf("expected the subagents path, got %s", path)
	}
	if !config.EnableSubagents || config.SubagentsBackgroundEnabled {
		t.Fatalf("expected the two flags to be read apart, got %+v", config)
	}
	if config.SubagentsMaxConcurrent != 3 || config.SubagentsMaxAsync != 2 || config.SubagentsMaxIterations != 10 || config.SubagentsMaxOutput != 8_000 {
		t.Fatalf("expected the four limits to be read, got %+v", config)
	}
	if config.SubagentsSystemPrompt != "be brief" {
		t.Fatalf("expected the system prompt, got %q", config.SubagentsSystemPrompt)
	}
}

// Every field of SubagentsConfigForm is required, so a request that omits one is
// a 422.
func TestSetSubagentsConfigSendsEveryField(t *testing.T) {
	var body map[string]any
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSONBody(t, r)
		_, _ = w.Write([]byte(`{"ENABLE_SUBAGENTS":false,"SUBAGENTS_BACKGROUND_ENABLED":false,"SUBAGENTS_MAX_CONCURRENT":1,"SUBAGENTS_MAX_ASYNC":1,"SUBAGENTS_MAX_ITERATIONS":1,"SUBAGENTS_MAX_OUTPUT":1,"SUBAGENTS_SYSTEM_PROMPT":""}`))
	})

	if _, err := c.SetSubagentsConfig(context.Background(), SubagentsConfigForm{
		SubagentsMaxConcurrent: 1,
		SubagentsMaxAsync:      1,
		SubagentsMaxIterations: 1,
		SubagentsMaxOutput:     1,
	}); err != nil {
		t.Fatalf("SetSubagentsConfig: %v", err)
	}

	for _, field := range []string{
		"ENABLE_SUBAGENTS", "SUBAGENTS_BACKGROUND_ENABLED", "SUBAGENTS_MAX_CONCURRENT",
		"SUBAGENTS_MAX_ASYNC", "SUBAGENTS_MAX_ITERATIONS", "SUBAGENTS_MAX_OUTPUT", "SUBAGENTS_SYSTEM_PROMPT",
	} {
		if _, ok := body[field]; !ok {
			t.Fatalf("expected %s in the request body, got %v", field, body)
		}
	}
}
