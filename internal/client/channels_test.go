package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newChannelsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// decodeJSONBody reads a request body as a JSON object.
func decodeJSONBody(t *testing.T, r *http.Request) map[string]any {
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

// The update handler assigns every field from the form it receives, so a field
// left out of the request clears the stored value.
func TestChannelFormSendsEveryField(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"id":"c1","user_id":"u1","name":"general","description":null,"is_private":null,"data":null,"meta":null,"access_grants":[],"created_at":1,"updated_at":2}`))
	})

	if _, err := c.UpdateChannel(context.Background(), "c1", ChannelForm{Name: "general"}); err != nil {
		t.Fatalf("UpdateChannel: %v", err)
	}

	for _, field := range []string{"name", "description", "is_private", "data", "meta", "access_grants"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("expected %s in the request body, got %v", field, body)
		}
	}
}

// The form carries no type, group_ids, or user_ids: those belong to the group
// and dm channels Open WebUI creates for its users.
func TestChannelFormOmitsMembershipFields(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"id":"c1","user_id":"u1","name":"general","access_grants":[],"created_at":1,"updated_at":2}`))
	})

	if _, err := c.CreateChannel(context.Background(), ChannelForm{Name: "general"}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	for _, field := range []string{"type", "group_ids", "user_ids"} {
		if _, ok := body[field]; ok {
			t.Fatalf("expected no %s in the request body, got %v", field, body)
		}
	}
}

func TestChannelFormWritesAccessGrants(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"id":"c1","user_id":"u1","name":"general","access_grants":[{"principal_type":"group","principal_id":"g1","permission":"read"},{"principal_type":"user","principal_id":"*","permission":"read"}],"created_at":1,"updated_at":2}`))
	})

	created, err := c.CreateChannel(context.Background(), ChannelForm{
		Name: "general",
		AccessControl: map[string]any{
			"read":        map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"public_read": true,
		},
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	grants, ok := body["access_grants"].([]any)
	if !ok || len(grants) != 2 {
		t.Fatalf("expected two grants on the wire, got %v", body["access_grants"])
	}

	public, _ := created.AccessControl["public_read"].(bool)
	if !public {
		t.Fatalf("expected the wildcard grant to read back as public_read, got %v", created.AccessControl)
	}
	readIDs, _ := created.AccessControl["read"].(map[string]any)
	groupIDs, _ := readIDs["group_ids"].([]string)
	if len(groupIDs) != 1 || groupIDs[0] != "g1" {
		t.Fatalf("expected the group grant to survive the round trip, got %v", created.AccessControl)
	}
}

func TestGetChannelUsesTheBareIDPath(t *testing.T) {
	var path string
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"c1","user_id":"u1","name":"general","access_grants":[],"created_at":1,"updated_at":2,"write_access":true}`))
	})

	channel, err := c.GetChannel(context.Background(), "c1")
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}

	if path != "/api/v1/channels/c1" {
		t.Fatalf("expected the read path, got %q", path)
	}
	if channel.WriteAccess == nil || !*channel.WriteAccess {
		t.Fatalf("expected write_access from the read route, got %+v", channel.WriteAccess)
	}
}
