package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func newOAuthClientTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func decodeOAuthClientRequest(t *testing.T, r *http.Request) map[string]any {
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

// A client_secret in the configuration reaches the registration call, which is
// what selects the static branch and mints a blob an oauth_2.1_static tool
// server can use.
func TestApplyOAuthClientRegistrationSendsStaticCredentials(t *testing.T) {
	var body map[string]any
	apiClient := newOAuthClientTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeOAuthClientRequest(t, r)
		_, _ = w.Write([]byte(`{"status":true,"oauth_client_info":"encrypted-blob"}`))
	})

	state, diags := applyOAuthClientRegistration(context.Background(), apiClient, oauthClientModel{
		URL:            types.StringValue("https://tools.example.invalid"),
		ClientID:       types.StringValue("paperless"),
		ClientName:     types.StringNull(),
		ClientSecret:   types.StringValue("s3cret"),
		OAuthServerURL: types.StringValue("https://idp.example.invalid"),
		OAuthScope:     types.StringNull(),
		Type:           types.StringValue("mcp"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if body["client_secret"] != "s3cret" {
		t.Fatalf("expected client_secret on the wire, got %v", body)
	}
	if body["oauth_server_url"] != "https://idp.example.invalid" {
		t.Fatalf("expected oauth_server_url on the wire, got %v", body)
	}
	if state.OAuthClientInfo.ValueString() != "encrypted-blob" {
		t.Fatalf("expected the registration blob in state, got %v", state.OAuthClientInfo)
	}
	if state.ID.ValueString() != "paperless" {
		t.Fatalf("expected the id to mirror client_id, got %v", state.ID)
	}
}

// Without a secret the request must stay on the dynamic-registration branch.
func TestApplyOAuthClientRegistrationOmitsUnsetCredentials(t *testing.T) {
	var body map[string]any
	apiClient := newOAuthClientTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeOAuthClientRequest(t, r)
		_, _ = w.Write([]byte(`{"status":true,"oauth_client_info":"encrypted-blob"}`))
	})

	_, diags := applyOAuthClientRegistration(context.Background(), apiClient, oauthClientModel{
		URL:            types.StringValue("https://tools.example.invalid"),
		ClientID:       types.StringValue("paperless"),
		ClientName:     types.StringNull(),
		ClientSecret:   types.StringNull(),
		OAuthServerURL: types.StringUnknown(),
		OAuthScope:     types.StringNull(),
		Type:           types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	for _, field := range []string{"client_secret", "oauth_server_url", "client_name", "oauth_scope"} {
		if _, ok := body[field]; ok {
			t.Fatalf("expected no %s key, got %v", field, body)
		}
	}
}
