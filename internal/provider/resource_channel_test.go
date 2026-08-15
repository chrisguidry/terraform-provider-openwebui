package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeChannelAPI serves the channel routes against an in-memory store. It
// lowercases a name on create and stores it verbatim on update, which is what
// Open WebUI does and what the resource's name validator exists for.
type fakeChannelAPI struct {
	mu       sync.Mutex
	channels map[string]map[string]any
	nextID   int
}

// fakeChannelUsers are the accounts the fake API knows. The search route
// matches a mail address, so a user ID reaches an account only through the
// account route.
var fakeChannelUsers = map[string]string{
	"u1": "parent@example.com",
	"u2": "kid@example.com",
}

func newFakeChannelAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeChannelAPI{channels: map[string]map[string]any{}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeChannelAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	w.Header().Set("Content-Type", "application/json")

	switch {
	case path == "users/":
		query := strings.ToLower(r.URL.Query().Get("query"))
		matches := []string{}
		for id, email := range fakeChannelUsers {
			if query != "" && !strings.Contains(email, query) {
				continue
			}
			matches = append(matches, fmt.Sprintf(`{"id":%q,"name":"Account","email":%q,"role":"user","last_active_at":1,"updated_at":2,"created_at":3}`, id, email))
		}
		_, _ = fmt.Fprintf(w, `{"users":[%s],"total":%d}`, strings.Join(matches, ","), len(matches))
	case strings.HasPrefix(path, "users/"):
		id := strings.TrimPrefix(path, "users/")
		email, ok := fakeChannelUsers[id]
		if !ok {
			http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":"Account","email":%q,"role":"user","last_active_at":1,"updated_at":2,"created_at":3}`, id, email)
	case path == "groups/":
		_, _ = w.Write([]byte(`[{"id":"g1","name":"family","description":"","user_ids":[]},{"id":"g2","name":"parents","description":"","user_ids":[]}]`))
	case path == "channels/create":
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.nextID++
		id := fmt.Sprintf("c%d", a.nextID)
		name, _ := form["name"].(string)
		form["name"] = strings.ToLower(name)
		a.channels[id] = a.store(id, form)
		_ = json.NewEncoder(w).Encode(a.channels[id])
	case path == "channels/list":
		list := []map[string]any{}
		for _, channel := range a.channels {
			list = append(list, channel)
		}
		_ = json.NewEncoder(w).Encode(list)
	case strings.HasSuffix(path, "/update"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "channels/"), "/update")
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.channels[id] = a.store(id, form)
		_ = json.NewEncoder(w).Encode(a.channels[id])
	case strings.HasSuffix(path, "/delete"):
		delete(a.channels, strings.TrimSuffix(strings.TrimPrefix(path, "channels/"), "/delete"))
		_ = json.NewEncoder(w).Encode(true)
	default:
		channel, ok := a.channels[strings.TrimPrefix(path, "channels/")]
		if !ok {
			http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
			return
		}
		record := map[string]any{"write_access": true}
		for key, value := range channel {
			record[key] = value
		}
		_ = json.NewEncoder(w).Encode(record)
	}
}

func (a *fakeChannelAPI) store(id string, form map[string]any) map[string]any {
	grants, ok := form["access_grants"].([]any)
	if !ok {
		grants = []any{}
	}

	return map[string]any{
		"id":            id,
		"user_id":       "u1",
		"type":          nil,
		"name":          form["name"],
		"description":   form["description"],
		"is_private":    form["is_private"],
		"data":          form["data"],
		"meta":          form["meta"],
		"access_grants": grants,
		"created_at":    1,
		"updated_at":    2,
	}
}

func testChannelProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

func testChannelResourceConfig(endpoint, name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_channel" "test" {
  name        = %q
  description = %q
  is_private  = false
  meta_json   = jsonencode({ icon = "megaphone" })
  read_groups = ["family"]
  public_read = true
}
`, testChannelProviderConfig(endpoint), name, description)
}

func TestChannelResource_CreateUpdateImport(t *testing.T) {
	server := newFakeChannelAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testChannelResourceConfig(server.URL, "announcements", "House news"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "id", "c1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "name", "announcements"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "description", "House news"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "is_private", "false"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_groups.0", "family"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "public_read", "true"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "public_write", "false"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "user_id", "u1"),
				),
			},
			{
				ResourceName:      "openwebui_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testChannelResourceConfig(server.URL, "announcements", "House news, revised"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "description", "House news, revised"),
				),
			},
		},
	})
}

// A name Open WebUI would lowercase on create and keep verbatim on update never
// converges, so the plan rejects it.
func TestChannelResource_RejectsAnUppercaseName(t *testing.T) {
	server := newFakeChannelAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_channel" "test" {
  name = "Announcements"
}
`, testChannelProviderConfig(server.URL))

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

func TestChannelDataSource_ByName(t *testing.T) {
	server := newFakeChannelAPI(t)

	config := fmt.Sprintf(`%s

data "openwebui_channel" "found" {
  name       = openwebui_channel.test.name
  depends_on = [openwebui_channel.test]
}
`, testChannelResourceConfig(server.URL, "announcements", "House news"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "id", "c1"),
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "description", "House news"),
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "read_groups.0", "family"),
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "public_read", "true"),
				),
			},
		},
	})
}

func testChannelUserGrantConfig(endpoint, grants string) string {
	return fmt.Sprintf(`%s
resource "openwebui_channel" "test" {
  name        = "announcements"
  description = "House news"
%s
}
`, testChannelProviderConfig(endpoint), grants)
}

// A mail address in the configuration becomes a user grant, and the read-back
// names the same address rather than the ID Open WebUI stores.
func TestChannelResource_ReadUsers(t *testing.T) {
	server := newFakeChannelAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testChannelUserGrantConfig(server.URL, `  read_users = ["parent@example.com"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.0", "parent@example.com"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "write_users.#", "0"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "public_read", "false"),
				),
			},
			{
				ResourceName:      "openwebui_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// read_users is Optional and Computed, so an empty list is the
				// only way to revoke the grant.
				Config: testChannelUserGrantConfig(server.URL, `  read_users = []`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "0"),
				),
			},
		},
	})
}

// A user named for writing can read the channel too, so the grant appears in
// both lists.
func TestChannelResource_WriteUsersImplyRead(t *testing.T) {
	server := newFakeChannelAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testChannelUserGrantConfig(server.URL, `  write_users = ["kid@example.com"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "write_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "write_users.0", "kid@example.com"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.0", "kid@example.com"),
				),
			},
		},
	})
}

// Group grants and user grants land on one channel without either one losing
// the other.
func TestChannelResource_GroupAndUserGrantsTogether(t *testing.T) {
	server := newFakeChannelAPI(t)

	grants := `  read_groups = ["family"]
  read_users  = ["parent@example.com", "kid@example.com"]`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testChannelUserGrantConfig(server.URL, grants),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_groups.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_groups.0", "family"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "2"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.0", "parent@example.com"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.1", "kid@example.com"),
				),
			},
		},
	})
}

// The data source reports the user grants a channel carries.
func TestChannelDataSource_ReportsUserGrants(t *testing.T) {
	server := newFakeChannelAPI(t)

	config := testChannelUserGrantConfig(server.URL, `  read_users = ["parent@example.com"]`) + `
data "openwebui_channel" "found" {
  channel_id = openwebui_channel.test.id
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "read_users.#", "1"),
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "read_users.0", "parent@example.com"),
				),
			},
		},
	})
}

// A configuration that names one account for reading and another for writing
// has to name the writer in read_users too, because Open WebUI stores a read
// grant for every writer.
func TestChannelResource_ReadUsersNamesEveryWriter(t *testing.T) {
	server := newFakeChannelAPI(t)

	grants := `  read_users  = ["parent@example.com", "kid@example.com"]
  write_users = ["kid@example.com"]`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testChannelUserGrantConfig(server.URL, grants),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "2"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.0", "parent@example.com"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.1", "kid@example.com"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "write_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "write_users.0", "kid@example.com"),
				),
			},
		},
	})
}
