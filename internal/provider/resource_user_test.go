package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// fakeUserAPI serves the user routes against an in-memory store. It seeds the
// account Open WebUI would have created first, so the primary admin guards have
// something real to protect. sessionID names the account the token belongs to,
// and failLookups makes the two identity reads fail the way a broken instance
// would.
type fakeUserAPI struct {
	mu          sync.Mutex
	users       map[string]map[string]any
	order       []string
	sessionID   string
	failLookups bool
	nextID      int
}

func newFakeUserAPI(t *testing.T) (*httptest.Server, *fakeUserAPI) {
	t.Helper()
	api := &fakeUserAPI{
		users: map[string]map[string]any{
			"u1": {
				"id":                "u1",
				"name":              "Root",
				"email":             "root@example.com",
				"role":              "admin",
				"profile_image_url": "/user.png",
				"last_active_at":    1,
				"updated_at":        1,
				"created_at":        1,
			},
		},
		order:     []string{"u1"},
		sessionID: "u1",
	}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server, api
}

func (a *fakeUserAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	w.Header().Set("Content-Type", "application/json")

	if a.failLookups && (path == "auths/" || path == "users/") {
		http.Error(w, `{"detail":"boom"}`, http.StatusInternalServerError)
		return
	}

	switch {
	case path == "auths/":
		_ = json.NewEncoder(w).Encode(a.users[a.sessionID])
	case path == "users/":
		users := make([]map[string]any, 0, len(a.order))
		for _, id := range a.order {
			users = append(users, a.users[id])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"users": users, "total": len(users)})
	case path == "auths/add":
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.nextID++
		id := fmt.Sprintf("u%d", a.nextID+1)
		role := "pending"
		if value, ok := form["role"].(string); ok {
			role = value
		}
		image := "/user.png"
		if value, ok := form["profile_image_url"].(string); ok {
			image = value
		}
		a.users[id] = map[string]any{
			"id":                id,
			"name":              form["name"],
			"email":             form["email"],
			"role":              role,
			"profile_image_url": image,
			"last_active_at":    1,
			"updated_at":        2,
			"created_at":        2,
		}
		a.order = append(a.order, id)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":             "a-live-jwt",
			"token_type":        "Bearer",
			"id":                id,
			"email":             form["email"],
			"name":              form["name"],
			"role":              role,
			"profile_image_url": fmt.Sprintf("/api/v1/users/%s/profile/image", id),
		})
	case strings.HasSuffix(path, "/update"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "users/"), "/update")
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, ok := a.users[id]
		if !ok {
			http.Error(w, `{"detail":"We could not find what you're looking for :/"}`, http.StatusBadRequest)
			return
		}
		for _, field := range []string{"name", "email", "role", "profile_image_url"} {
			if value, present := form[field]; present {
				user[field] = value
			}
		}
		_ = json.NewEncoder(w).Encode(user)
	case r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "users/")
		delete(a.users, id)
		_ = json.NewEncoder(w).Encode(true)
	default:
		id := strings.TrimPrefix(path, "users/")
		user, ok := a.users[id]
		if !ok {
			http.Error(w, `{"detail":"We could not find what you're looking for :/"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(user)
	}
}

func testUserProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

func testUserResourceConfig(endpoint, name, role, version string) string {
	return fmt.Sprintf(`%s
resource "openwebui_user" "test" {
  name             = %q
  email            = "kid@example.com"
  role             = %q
  password         = "hunter2hunter2"
  password_version = %q
}
`, testUserProviderConfig(endpoint), name, role, version)
}

func TestUserResource_CreateUpdateImport(t *testing.T) {
	server, _ := newFakeUserAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testUserResourceConfig(server.URL, "Kid", "user", "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_user.test", "email", "kid@example.com"),
					resource.TestCheckResourceAttr("openwebui_user.test", "name", "Kid"),
					resource.TestCheckResourceAttr("openwebui_user.test", "role", "user"),
					// The create response reports the profile image as a route;
					// the stored value is what belongs in state.
					resource.TestCheckResourceAttr("openwebui_user.test", "profile_image_url", "/user.png"),
					// A write-only attribute never reaches state.
					resource.TestCheckNoResourceAttr("openwebui_user.test", "password"),
				),
			},
			{
				ResourceName:            "openwebui_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_version"},
			},
			{
				Config: testUserResourceConfig(server.URL, "Kid Renamed", "admin", "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_user.test", "name", "Kid Renamed"),
					resource.TestCheckResourceAttr("openwebui_user.test", "role", "admin"),
				),
			},
		},
	})
}

func TestUserResource_RejectsAnUppercaseEmail(t *testing.T) {
	server, _ := newFakeUserAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_user" "test" {
  name     = "Kid"
  email    = "Kid@Example.com"
  password = "hunter2hunter2"
}
`, testUserProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`must be lowercase`),
			},
		},
	})
}

func TestUserResource_RejectsAnUnknownRole(t *testing.T) {
	server, _ := newFakeUserAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_user" "test" {
  name     = "Kid"
  email    = "kid@example.com"
  role     = "superuser"
  password = "hunter2hunter2"
}
`, testUserProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func TestUserResource_CreateNeedsAPassword(t *testing.T) {
	server, _ := newFakeUserAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_user" "test" {
  name  = "Kid"
  email = "kid@example.com"
}
`, testUserProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Password required`),
			},
		},
	})
}

func newGuardedUserResource(t *testing.T, server *httptest.Server) *userResource {
	t.Helper()
	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return &userResource{client: apiClient}
}

func rolePointer(value string) *string {
	return &value
}

// The primary admin cannot lose its admin role, whoever asks.
func TestUserGuard_RefusesToDemoteThePrimaryAdmin(t *testing.T) {
	server, _ := newFakeUserAPI(t)
	r := newGuardedUserResource(t, server)

	diags := r.guardPrimaryAdminUpdate(context.Background(), "u1", rolePointer("user"))

	if !diags.HasError() {
		t.Fatalf("expected the demotion to be refused")
	}
	if !strings.Contains(diags.Errors()[0].Summary(), "Refusing to demote the primary admin") {
		t.Fatalf("unexpected diagnostic: %s", diags.Errors()[0].Summary())
	}
}

// Renaming the primary admin is allowed while the provider's own token is that
// account. Open WebUI itself permits it.
func TestUserGuard_AllowsThePrimaryAdminToChangeItself(t *testing.T) {
	server, _ := newFakeUserAPI(t)
	r := newGuardedUserResource(t, server)

	diags := r.guardPrimaryAdminUpdate(context.Background(), "u1", nil)

	if diags.HasError() {
		t.Fatalf("expected the update to be allowed, got %v", diags.Errors())
	}
}

// A token belonging to any other admin cannot touch the primary admin at all.
func TestUserGuard_RefusesAnotherAdminsUpdate(t *testing.T) {
	server, api := newFakeUserAPI(t)
	api.mu.Lock()
	api.users["u2"] = map[string]any{
		"id": "u2", "name": "Deputy", "email": "deputy@example.com", "role": "admin",
		"profile_image_url": "/user.png", "last_active_at": 1, "updated_at": 1, "created_at": 5,
	}
	api.order = append(api.order, "u2")
	api.sessionID = "u2"
	api.mu.Unlock()

	r := newGuardedUserResource(t, server)

	diags := r.guardPrimaryAdminUpdate(context.Background(), "u1", rolePointer("admin"))

	if !diags.HasError() {
		t.Fatalf("expected the update to be refused")
	}
	if !strings.Contains(diags.Errors()[0].Summary(), "Refusing to update the primary admin") {
		t.Fatalf("unexpected diagnostic: %s", diags.Errors()[0].Summary())
	}
}

// An account that is not the primary admin is not guarded.
func TestUserGuard_LeavesOtherAccountsAlone(t *testing.T) {
	server, api := newFakeUserAPI(t)
	api.mu.Lock()
	api.users["u2"] = map[string]any{
		"id": "u2", "name": "Kid", "email": "kid@example.com", "role": "user",
		"profile_image_url": "/user.png", "last_active_at": 1, "updated_at": 1, "created_at": 5,
	}
	api.order = append(api.order, "u2")
	api.mu.Unlock()

	r := newGuardedUserResource(t, server)

	if diags := r.guardPrimaryAdminUpdate(context.Background(), "u2", rolePointer("pending")); diags.HasError() {
		t.Fatalf("expected the update to be allowed, got %v", diags.Errors())
	}
	if diags := r.guardPrimaryAdminDelete(context.Background(), "u2"); diags.HasError() {
		t.Fatalf("expected the delete to be allowed, got %v", diags.Errors())
	}
}

func TestUserGuard_RefusesToDeleteThePrimaryAdmin(t *testing.T) {
	server, _ := newFakeUserAPI(t)
	r := newGuardedUserResource(t, server)

	diags := r.guardPrimaryAdminDelete(context.Background(), "u1")

	if !diags.HasError() {
		t.Fatalf("expected the delete to be refused")
	}
	summaries := []string{}
	for _, d := range diags.Errors() {
		summaries = append(summaries, d.Summary())
	}
	if !strings.Contains(strings.Join(summaries, "|"), "Refusing to delete the primary admin") {
		t.Fatalf("unexpected diagnostics: %v", summaries)
	}
}

// Open WebUI refuses to let an account delete itself, primary admin or not.
func TestUserGuard_RefusesToDeleteTheProvidersOwnAccount(t *testing.T) {
	server, api := newFakeUserAPI(t)
	api.mu.Lock()
	api.users["u2"] = map[string]any{
		"id": "u2", "name": "Deputy", "email": "deputy@example.com", "role": "admin",
		"profile_image_url": "/user.png", "last_active_at": 1, "updated_at": 1, "created_at": 5,
	}
	api.order = append(api.order, "u2")
	api.sessionID = "u2"
	api.mu.Unlock()

	r := newGuardedUserResource(t, server)

	diags := r.guardPrimaryAdminDelete(context.Background(), "u2")

	if !diags.HasError() {
		t.Fatalf("expected the delete to be refused")
	}
	if !strings.Contains(diags.Errors()[0].Summary(), "Refusing to delete the provider's own account") {
		t.Fatalf("unexpected diagnostic: %s", diags.Errors()[0].Summary())
	}
}

// A guard that cannot tell who the primary admin is stops the write.
func TestUserGuard_FailsClosedWhenTheLookupFails(t *testing.T) {
	server, api := newFakeUserAPI(t)
	api.mu.Lock()
	api.failLookups = true
	api.mu.Unlock()

	r := newGuardedUserResource(t, server)

	if diags := r.guardPrimaryAdminUpdate(context.Background(), "u2", rolePointer("user")); !diags.HasError() {
		t.Fatalf("expected the update to be refused")
	}
	if diags := r.guardPrimaryAdminDelete(context.Background(), "u2"); !diags.HasError() {
		t.Fatalf("expected the delete to be refused")
	}
}
