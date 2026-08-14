package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newModelTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestCreateModelSendsAccessGrants(t *testing.T) {
	var sent map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}]}`))
	})

	form := ModelForm{
		ID:   "m1",
		Name: "M",
		Meta: map[string]any{}, Params: map[string]any{},
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}
	out, err := c.CreateModel(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	if _, ok := sent["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", sent)
	}
	grants, ok := sent["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected 1 access_grant, got %+v", sent["access_grants"])
	}

	read, ok := out.AccessControl["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read to be map[string]any, got %T", out.AccessControl["read"])
	}
	ids, ok := read["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected group_ids to be []string, got %T", read["group_ids"])
	}
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected read.group_ids=[g1], got %+v", out.AccessControl)
	}
}

func TestGetModelParsesAccessGrants(t *testing.T) {
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g9","permission":"write"}]}`))
	})
	out, err := c.GetModel(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	write, ok := out.AccessControl["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write to be map[string]any, got %T", out.AccessControl["write"])
	}
	ids, ok := write["group_ids"].([]string)
	if !ok {
		t.Fatalf("expected group_ids to be []string, got %T", write["group_ids"])
	}
	if len(ids) != 1 || ids[0] != "g9" {
		t.Fatalf("expected write.group_ids=[g9], got %+v", out.AccessControl)
	}
}

// Open WebUI carries the stored base_model_id forward when the key is missing
// from an update, so a model whose base_model_id was removed from the config
// must still send the key with a null value.
func TestUpdateModelSendsNullBaseModelID(t *testing.T) {
	var sent map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2}`))
	})

	form := ModelForm{ID: "m1", Name: "M", Meta: map[string]any{}, Params: map[string]any{}}
	if _, err := c.UpdateModel(context.Background(), "m1", form); err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}

	value, present := sent["base_model_id"]
	if !present {
		t.Fatalf("expected base_model_id to be present, got %+v", sent)
	}
	if value != nil {
		t.Fatalf("expected base_model_id to be null, got %#v", value)
	}
}

func TestUpdateModelSendsBaseModelID(t *testing.T) {
	var sent map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2}`))
	})

	base := "llama3.2"
	form := ModelForm{ID: "m1", Name: "M", Meta: map[string]any{}, Params: map[string]any{}, BaseModelID: &base}
	if _, err := c.UpdateModel(context.Background(), "m1", form); err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}

	if sent["base_model_id"] != "llama3.2" {
		t.Fatalf("expected base_model_id=llama3.2, got %#v", sent["base_model_id"])
	}
}

// A wildcard user grant is Open WebUI's public sharing. The model form has to
// carry the public_read and public_write booleans through to access_grants.
func TestCreateModelSendsWildcardGrants(t *testing.T) {
	var sent map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &sent)
		_, _ = w.Write([]byte(`{"id":"m1","user_id":"u1","name":"M","meta":{},"params":{},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`))
	})

	form := ModelForm{
		ID:   "m1",
		Name: "M",
		Meta: map[string]any{}, Params: map[string]any{},
		AccessControl: map[string]any{"public_read": true, "public_write": false},
	}
	out, err := c.CreateModel(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}

	grants, ok := sent["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected 1 access_grant, got %+v", sent["access_grants"])
	}
	grant, ok := grants[0].(map[string]any)
	if !ok {
		t.Fatalf("expected grant object, got %T", grants[0])
	}
	if grant["principal_type"] != "user" || grant["principal_id"] != "*" || grant["permission"] != "read" {
		t.Fatalf("expected a user/*/read grant, got %+v", grant)
	}

	if out.AccessControl["public_read"] != true {
		t.Fatalf("expected public_read=true, got %+v", out.AccessControl)
	}
	if out.AccessControl["public_write"] != false {
		t.Fatalf("expected public_write=false, got %+v", out.AccessControl)
	}
}

func TestDeleteModelSendsBody(t *testing.T) {
	var gotMethod, gotQuery string
	var gotBody map[string]any
	c := newModelTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		_, _ = w.Write([]byte(`true`))
	})
	if err := c.DeleteModel(context.Background(), "m1"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotQuery != "" {
		t.Fatalf("expected no query string, got %q", gotQuery)
	}
	if gotBody["id"] != "m1" {
		t.Fatalf("expected body id=m1, got %+v", gotBody)
	}
}
