package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newUsersTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestSearchUsersOmitsLimit(t *testing.T) {
	var gotQuery, gotLimit string
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		gotLimit = r.URL.Query().Get("limit")
		_, _ = w.Write([]byte(`{"users":[{"id":"u1","name":"Alice","email":"alice@example.com","role":"user","last_active_at":1,"updated_at":2,"created_at":3}],"total":1}`))
	})
	users, err := c.SearchUsers(context.Background(), "alice", 50)
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if gotQuery != "alice" {
		t.Fatalf("expected query=alice, got %q", gotQuery)
	}
	if gotLimit != "" {
		t.Fatalf("expected no limit param, got %q", gotLimit)
	}
	if len(users) != 1 || users[0].ID != "u1" {
		t.Fatalf("unexpected users: %+v", users)
	}
}

// The create response carries a live JWT for the new account. Nothing but the
// identity may survive the call, and the profile image it reports is a route
// rather than the stored value, so the account is read back.
func TestAddUserReadsTheAccountBackAndDropsTheToken(t *testing.T) {
	var paths []string
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/v1/auths/add" {
			_, _ = w.Write([]byte(`{"token":"secret-jwt","token_type":"Bearer","id":"u9","email":"kid@example.com","name":"Kid","role":"user","profile_image_url":"/api/v1/users/u9/profile/image"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"u9","name":"Kid","email":"kid@example.com","role":"user","profile_image_url":"/user.png","last_active_at":1,"updated_at":2,"created_at":3}`))
	})

	user, err := c.AddUser(context.Background(), AddUserForm{Name: "Kid", Email: "kid@example.com", Password: "hunter2hunter2"})
	if err != nil {
		t.Fatalf("AddUser: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/auths/add" || paths[1] != "/api/v1/users/u9" {
		t.Fatalf("expected a create then a read, got %v", paths)
	}
	if user.ProfileImage != "/user.png" {
		t.Fatalf("expected the stored profile image, got %q", user.ProfileImage)
	}
}

// UserUpdateForm is applied field by field, so an omitted field must not reach
// the wire as a null.
func TestUpdateUserSendsOnlyTheNamedFields(t *testing.T) {
	var body map[string]any
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"id":"u9","name":"Kid","email":"kid@example.com","role":"admin","profile_image_url":"/user.png","last_active_at":1,"updated_at":2,"created_at":3}`))
	})

	role := "admin"
	if _, err := c.UpdateUser(context.Background(), "u9", UserUpdateForm{Role: &role}); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	if len(body) != 1 || body["role"] != "admin" {
		t.Fatalf("expected only the role in the request body, got %v", body)
	}
}

// The primary admin is whichever account was created first, which is the rule
// the server's own check uses.
func TestGetPrimaryAdminAsksForTheEarliestAccount(t *testing.T) {
	var query string
	c := newUsersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"users":[{"id":"u1","name":"Root","email":"root@example.com","role":"admin","last_active_at":1,"updated_at":2,"created_at":3},{"id":"u2","name":"Kid","email":"kid@example.com","role":"user","last_active_at":1,"updated_at":2,"created_at":9}],"total":2}`))
	})

	primary, err := c.GetPrimaryAdmin(context.Background())
	if err != nil {
		t.Fatalf("GetPrimaryAdmin: %v", err)
	}

	for _, expected := range []string{"order_by=created_at", "direction=asc", "page=1"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("expected %s in the query, got %q", expected, query)
		}
	}
	if primary == nil || primary.ID != "u1" {
		t.Fatalf("expected the earliest account, got %+v", primary)
	}
}

func TestGetPrimaryAdminOnAnEmptyInstance(t *testing.T) {
	c := newUsersTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"users":[],"total":0}`))
	})

	primary, err := c.GetPrimaryAdmin(context.Background())
	if err != nil {
		t.Fatalf("GetPrimaryAdmin: %v", err)
	}
	if primary != nil {
		t.Fatalf("expected no primary admin, got %+v", primary)
	}
}
