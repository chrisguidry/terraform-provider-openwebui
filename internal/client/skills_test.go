package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordedRequest struct {
	method string
	path   string
	body   map[string]any
}

func newSkillTestClient(t *testing.T, handler func(*recordedRequest, http.ResponseWriter)) (*Client, *recordedRequest) {
	t.Helper()
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.method = r.Method
		recorded.path = r.URL.EscapedPath()
		if body, err := io.ReadAll(r.Body); err == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &recorded.body)
		}
		handler(recorded, w)
	}))
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, recorded
}

func TestCreateSkillSendsAccessGrants(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"S","description":"d","meta":{"tags":["a"]},"is_active":true,"created_at":1,"updated_at":2,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g1","permission":"read"}]}`))
	})

	description := "d"
	active := true
	form := SkillForm{
		ID:          "s1",
		Name:        "S",
		Description: &description,
		Content:     "# Skill",
		Meta:        SkillMeta{Tags: []string{"a"}},
		IsActive:    &active,
		AccessControl: map[string]any{
			"read":  map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"write": map[string]any{"group_ids": []string{}, "user_ids": []string{}},
		},
	}

	out, err := c.CreateSkill(context.Background(), form)
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	if recorded.method != http.MethodPost || recorded.path != "/api/v1/skills/create" {
		t.Fatalf("unexpected request %s %s", recorded.method, recorded.path)
	}
	if _, ok := recorded.body["access_control"]; ok {
		t.Fatalf("access_control must not be sent: %+v", recorded.body)
	}
	grants, ok := recorded.body["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected one access grant, got %+v", recorded.body["access_grants"])
	}
	if recorded.body["content"] != "# Skill" {
		t.Fatalf("expected content in the create payload, got %+v", recorded.body["content"])
	}
	if out.ID != "s1" || out.AccessControl == nil {
		t.Fatalf("unexpected create response: %+v", out)
	}
}

func TestCreateSkillOmitsUnsetOptionalFields(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"S","meta":{"tags":[]},"is_active":true,"created_at":1,"updated_at":2}`))
	})

	if _, err := c.CreateSkill(context.Background(), SkillForm{ID: "s1", Name: "S", Content: "x"}); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	if _, ok := recorded.body["description"]; ok {
		t.Fatalf("description must be omitted when unset: %+v", recorded.body)
	}
	if _, ok := recorded.body["is_active"]; ok {
		t.Fatalf("is_active must be omitted when unset: %+v", recorded.body)
	}
	grants, ok := recorded.body["access_grants"].([]any)
	if !ok || len(grants) != 0 {
		t.Fatalf("expected an empty grant list, got %+v", recorded.body["access_grants"])
	}
}

func TestGetSkillReturnsContentAndGrants(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"S","description":"d","content":"# Body","meta":{"tags":["a","b"]},"is_active":false,"created_at":1,"updated_at":2,"write_access":true,"access_grants":[{"id":"a1","principal_type":"group","principal_id":"g7","permission":"read"},{"id":"a2","principal_type":"user","principal_id":"*","permission":"read"}]}`))
	})

	out, err := c.GetSkill(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}

	if recorded.method != http.MethodGet || recorded.path != "/api/v1/skills/id/s1" {
		t.Fatalf("unexpected request %s %s", recorded.method, recorded.path)
	}
	if out.Content != "# Body" {
		t.Fatalf("expected the skill body, got %q", out.Content)
	}
	if out.IsActive {
		t.Fatalf("expected is_active=false, got true")
	}
	if len(out.Meta.Tags) != 2 || out.Meta.Tags[0] != "a" {
		t.Fatalf("unexpected tags: %+v", out.Meta.Tags)
	}
	if out.WriteAccess == nil || !*out.WriteAccess {
		t.Fatalf("expected write_access=true, got %+v", out.WriteAccess)
	}

	read, ok := out.AccessControl["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected a read section, got %T", out.AccessControl["read"])
	}
	ids, ok := read["group_ids"].([]string)
	if !ok || len(ids) != 1 || ids[0] != "g7" {
		t.Fatalf("expected read.group_ids=[g7], got %+v", out.AccessControl)
	}
	if out.AccessControl["public_read"] != true {
		t.Fatalf("expected public_read=true from the wildcard grant, got %+v", out.AccessControl)
	}
}

func TestGetSkillEscapesID(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"a/b","user_id":"u1","name":"S","content":"x","meta":{"tags":[]},"is_active":true,"created_at":1,"updated_at":2}`))
	})

	if _, err := c.GetSkill(context.Background(), "a/b"); err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if recorded.path != "/api/v1/skills/id/a%2Fb" {
		t.Fatalf("expected the ID to be path-escaped, got %q", recorded.path)
	}
}

func TestUpdateSkillPostsToUpdatePath(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"New","content":"# New","meta":{"tags":[]},"is_active":true,"created_at":1,"updated_at":3,"access_grants":[]}`))
	})

	out, err := c.UpdateSkill(context.Background(), "s1", SkillForm{ID: "s1", Name: "New", Content: "# New"})
	if err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}

	if recorded.method != http.MethodPost || recorded.path != "/api/v1/skills/id/s1/update" {
		t.Fatalf("unexpected request %s %s", recorded.method, recorded.path)
	}
	if out.Content != "# New" || out.Name != "New" {
		t.Fatalf("unexpected update response: %+v", out)
	}
	if out.AccessControl != nil {
		t.Fatalf("expected no access_control from an empty grant list, got %+v", out.AccessControl)
	}
}

func TestDeleteSkillUsesDeleteMethod(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`true`))
	})

	if err := c.DeleteSkill(context.Background(), "s1"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if recorded.method != http.MethodDelete || recorded.path != "/api/v1/skills/id/s1/delete" {
		t.Fatalf("unexpected request %s %s", recorded.method, recorded.path)
	}
}

func TestListSkillsKeepsTrailingSlash(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`[{"id":"s1","user_id":"u1","name":"S","meta":{"tags":[]},"is_active":true,"created_at":1,"updated_at":2}]`))
	})

	out, err := c.ListSkills(context.Background())
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if recorded.path != "/api/v1/skills/" {
		t.Fatalf("expected the trailing slash to survive, got %q", recorded.path)
	}
	if len(out) != 1 || out[0].ID != "s1" {
		t.Fatalf("unexpected list response: %+v", out)
	}
}

func TestSkillFormSendsPublicGrants(t *testing.T) {
	c, recorded := newSkillTestClient(t, func(_ *recordedRequest, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"s1","user_id":"u1","name":"S","meta":{"tags":[]},"is_active":true,"created_at":1,"updated_at":2}`))
	})

	form := SkillForm{
		ID:            "s1",
		Name:          "S",
		Content:       "x",
		AccessControl: map[string]any{"public_read": true, "public_write": false},
	}
	if _, err := c.CreateSkill(context.Background(), form); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	grants, ok := recorded.body["access_grants"].([]any)
	if !ok || len(grants) != 1 {
		t.Fatalf("expected one wildcard grant, got %+v", recorded.body["access_grants"])
	}
	grant, ok := grants[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected grant shape: %+v", grants[0])
	}
	if grant["principal_type"] != "user" || grant["principal_id"] != "*" || grant["permission"] != "read" {
		t.Fatalf("unexpected wildcard grant: %+v", grant)
	}
}
