package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeAdminConfigAPI serves the admin config routes. It stores every key it
// receives and casts the three count limits the way Open WebUI does: an int
// when truthy, an empty string otherwise.
type fakeAdminConfigAPI struct {
	mu     sync.Mutex
	values map[string]any
}

func newFakeAdminConfigAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeAdminConfigAPI{values: map[string]any{
		"SHOW_ADMIN_DETAILS":                    true,
		"ADMIN_EMAIL":                           nil,
		"WEBUI_URL":                             "https://chat.example.com",
		"ENABLE_SIGNUP":                         true,
		"ENABLE_API_KEYS":                       true,
		"ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS": false,
		"API_KEYS_ALLOWED_ENDPOINTS":            "",
		"DEFAULT_USER_ROLE":                     "pending",
		"DEFAULT_GROUP_ID":                      "",
		"JWT_EXPIRES_IN":                        "4h",
		"ENABLE_COMMUNITY_SHARING":              true,
		"ENABLE_MESSAGE_RATING":                 true,
		"ENABLE_FOLDERS":                        true,
		"FOLDER_MAX_FILE_COUNT":                 "",
		"AUTOMATION_MAX_COUNT":                  "",
		"AUTOMATION_MIN_INTERVAL":               "",
		"ENABLE_AUTOMATIONS":                    true,
		"ENABLE_CHANNELS":                       false,
		"CHANNEL_MODEL_RESPONSE_MODE":           "thread",
		"ENABLE_CALENDAR":                       true,
		"ENABLE_MEMORIES":                       true,
		"ENABLE_MEMORY_SYSTEM_CONTEXT":          true,
		"ENABLE_NOTES":                          true,
		"ENABLE_USER_WEBHOOKS":                  true,
		"ENABLE_USER_STATUS":                    true,
		"PENDING_USER_OVERLAY_TITLE":            nil,
		"PENDING_USER_OVERLAY_CONTENT":          nil,
		"RESPONSE_WATERMARK":                    nil,
	}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeAdminConfigAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for key, value := range form {
			a.values[key] = value
		}
		for _, key := range []string{"FOLDER_MAX_FILE_COUNT", "AUTOMATION_MAX_COUNT", "AUTOMATION_MIN_INTERVAL"} {
			text, _ := form[key].(string)
			if text == "" || text == "0" {
				a.values[key] = ""
				continue
			}
			a.values[key] = json.Number(text)
		}
	}

	_ = json.NewEncoder(w).Encode(a.values)
}

func testAdminConfigProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

// The route writes all 28 keys at once, so the settings the configuration does
// not name have to carry the values the instance already holds.
func TestAdminConfigResource_KeepsUnnamedSettings(t *testing.T) {
	server := newFakeAdminConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_admin_config" "test" {
  enable_signup         = false
  enable_channels       = true
  default_user_role     = "user"
  folder_max_file_count = "50"
}
`, testAdminConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "id", "admin"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_signup", "false"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_channels", "true"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "default_user_role", "user"),
					// A number stored for a key typed `int | str` reads back as text.
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "folder_max_file_count", "50"),
					// Settings the configuration never mentions keep their values.
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "webui_url", "https://chat.example.com"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "jwt_expires_in", "4h"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "enable_notes", "true"),
					resource.TestCheckNoResourceAttr("openwebui_admin_config.test", "admin_email"),
				),
			},
			{
				ResourceName:      "openwebui_admin_config.test",
				ImportState:       true,
				ImportStateId:     "admin",
				ImportStateVerify: true,
			},
		},
	})
}

// Open WebUI drops an unusable value and still answers 200, so the plan is
// where a wrong one has to fail.
func TestAdminConfigResource_RejectsValuesTheServerWouldDrop(t *testing.T) {
	server := newFakeAdminConfigAPI(t)

	cases := map[string]struct {
		body    string
		message *regexp.Regexp
	}{
		"default user role":     {body: `default_user_role = "superuser"`, message: regexp.MustCompile(`value must be one of`)},
		"channel response mode": {body: `channel_model_response_mode = "sidebar"`, message: regexp.MustCompile(`value must be one of`)},
		"jwt expiry":            {body: `jwt_expires_in = "forever"`, message: regexp.MustCompile(`must be -1, 0, or a number`)},
		"folder file count":     {body: `folder_max_file_count = "many"`, message: regexp.MustCompile(`must be empty or a positive whole number`)},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			config := fmt.Sprintf(`%s
resource "openwebui_admin_config" "test" {
  %s
}
`, testAdminConfigProviderConfig(server.URL), testCase.body)

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: testCase.message,
					},
				},
			})
		})
	}
}
