package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newConfigsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func captureRequestBody(t *testing.T, r *http.Request) map[string]any {
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

// POST /configs/models writes every field of the form, so a request that omits
// the two dict fields sets them to null and erases the stored values.
func TestSetModelsConfigSendsEveryField(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"DEFAULT_MODELS":"gpt-4o","DEFAULT_PINNED_MODELS":null,"MODEL_ORDER_LIST":[],"DEFAULT_MODEL_METADATA":{"owner":"platform"},"DEFAULT_MODEL_PARAMS":{"temperature":0.5}}`))
	})

	models := "gpt-4o"
	updated, err := c.SetModelsConfig(context.Background(), ModelsConfigForm{
		DefaultModels:        &models,
		ModelOrderList:       []string{},
		DefaultModelMetadata: map[string]any{"owner": "platform"},
		DefaultModelParams:   map[string]any{"temperature": 0.5},
	})
	if err != nil {
		t.Fatalf("SetModelsConfig: %v", err)
	}

	for _, field := range []string{"DEFAULT_MODELS", "DEFAULT_PINNED_MODELS", "MODEL_ORDER_LIST", "DEFAULT_MODEL_METADATA", "DEFAULT_MODEL_PARAMS"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("expected %s in the request body, got %v", field, body)
		}
	}

	metadata, ok := body["DEFAULT_MODEL_METADATA"].(map[string]any)
	if !ok || metadata["owner"] != "platform" {
		t.Fatalf("expected the metadata to be sent, got %v", body["DEFAULT_MODEL_METADATA"])
	}
	if updated.DefaultModelParams["temperature"] != 0.5 {
		t.Fatalf("expected the params to be read back, got %v", updated.DefaultModelParams)
	}
}

func TestRegisterOAuthClientSendsScope(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"status":true}`))
	})

	scope := "openid profile"
	if _, err := c.RegisterOAuthClient(context.Background(), OAuthClientRegistrationForm{
		URL:        "https://idp.example.invalid",
		ClientID:   "openwebui",
		OAuthScope: &scope,
	}, nil); err != nil {
		t.Fatalf("RegisterOAuthClient: %v", err)
	}

	if body["oauth_scope"] != "openid profile" {
		t.Fatalf("expected oauth_scope in the request body, got %v", body)
	}
}

func TestRegisterOAuthClientOmitsAbsentScope(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"status":true}`))
	})

	if _, err := c.RegisterOAuthClient(context.Background(), OAuthClientRegistrationForm{
		URL:      "https://idp.example.invalid",
		ClientID: "openwebui",
	}, nil); err != nil {
		t.Fatalf("RegisterOAuthClient: %v", err)
	}

	if _, ok := body["oauth_scope"]; ok {
		t.Fatalf("expected no oauth_scope key, got %v", body)
	}
}

// A client_secret takes the static-credentials branch of register_oauth_client,
// which is the only way to register the blob an oauth_2.1_static tool server
// needs.
func TestRegisterOAuthClientSendsStaticCredentials(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"status":true,"oauth_client_info":"encrypted-blob"}`))
	})

	secret := "s3cret"
	serverURL := "https://idp.example.invalid/authorize"
	resp, err := c.RegisterOAuthClient(context.Background(), OAuthClientRegistrationForm{
		URL:            "https://tools.example.invalid",
		ClientID:       "paperless",
		ClientSecret:   &secret,
		OAuthServerURL: &serverURL,
	}, nil)
	if err != nil {
		t.Fatalf("RegisterOAuthClient: %v", err)
	}

	if body["client_secret"] != secret {
		t.Fatalf("expected client_secret in the request body, got %v", body)
	}
	if body["oauth_server_url"] != serverURL {
		t.Fatalf("expected oauth_server_url in the request body, got %v", body)
	}
	if resp["oauth_client_info"] != "encrypted-blob" {
		t.Fatalf("expected the registration blob to be returned, got %v", resp)
	}
}

func TestRegisterOAuthClientOmitsAbsentStaticCredentials(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"status":true}`))
	})

	if _, err := c.RegisterOAuthClient(context.Background(), OAuthClientRegistrationForm{
		URL:      "https://idp.example.invalid",
		ClientID: "openwebui",
	}, nil); err != nil {
		t.Fatalf("RegisterOAuthClient: %v", err)
	}

	for _, field := range []string{"client_secret", "oauth_server_url"} {
		if _, ok := body[field]; ok {
			t.Fatalf("expected no %s key, got %v", field, body)
		}
	}
}

// set_tool_servers_config reads the OAuth client key through
// connection['info']['id'], so info has to survive the round trip.
func TestToolServerConnectionCarriesInfo(t *testing.T) {
	var body map[string]any
	c := newConfigsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = captureRequestBody(t, r)
		_, _ = w.Write([]byte(`{"TOOL_SERVER_CONNECTIONS":[{"url":"https://tools.example.invalid","path":"openapi.json","auth_type":null,"key":null,"config":{},"info":{"id":"tools"}}]}`))
	})

	updated, err := c.SetToolServersConfig(context.Background(), ToolServersConfigForm{
		Connections: []ToolServerConnection{{
			URL:  "https://tools.example.invalid",
			Path: "openapi.json",
			Info: map[string]any{"id": "tools"},
		}},
	})
	if err != nil {
		t.Fatalf("SetToolServersConfig: %v", err)
	}

	connections, ok := body["TOOL_SERVER_CONNECTIONS"].([]any)
	if !ok || len(connections) != 1 {
		t.Fatalf("expected one connection in the request body, got %v", body)
	}
	sent, _ := connections[0].(map[string]any)
	info, ok := sent["info"].(map[string]any)
	if !ok || info["id"] != "tools" {
		t.Fatalf("expected info to be sent, got %v", sent)
	}
	if updated.Connections[0].Info["id"] != "tools" {
		t.Fatalf("expected info to be read back, got %v", updated.Connections[0].Info)
	}
}
