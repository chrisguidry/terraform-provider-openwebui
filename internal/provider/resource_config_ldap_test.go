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

// fakeLDAPConfigAPI serves the two LDAP routes: the directory connection and
// the sign-in switch. The switch is spelled enable_ldap on the way in and
// ENABLE_LDAP on the way out, as Open WebUI spells it.
type fakeLDAPConfigAPI struct {
	mu      sync.Mutex
	server  map[string]any
	enabled bool
}

func newFakeLDAPConfigAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeLDAPConfigAPI{server: map[string]any{
		"label":                   "Directory",
		"host":                    "ldap.example.com",
		"port":                    nil,
		"attribute_for_mail":      "mail",
		"attribute_for_username":  "uid",
		"app_dn":                  "cn=svc,dc=example,dc=com",
		"app_dn_password":         "bind-secret",
		"search_base":             "dc=example,dc=com",
		"search_filters":          "",
		"use_tls":                 true,
		"certificate_path":        nil,
		"validate_cert":           true,
		"ciphers":                 "ALL",
		"enable_group_management": false,
		"enable_group_creation":   false,
		"attribute_for_groups":    "memberOf",
	}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeLDAPConfigAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if strings.HasSuffix(r.URL.Path, "/ldap/server") {
		if r.Method == http.MethodPost {
			var form map[string]any
			if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, key := range []string{"label", "host", "attribute_for_mail", "attribute_for_username", "search_base"} {
				if text, _ := form[key].(string); text == "" {
					http.Error(w, fmt.Sprintf(`{"detail":"%s is required"}`, key), http.StatusBadRequest)
					return
				}
			}
			for key, value := range form {
				a.server[key] = value
			}
		}
		_ = json.NewEncoder(w).Encode(a.server)
		return
	}

	if r.Method == http.MethodPost {
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		enabled, _ := form["enable_ldap"].(bool)
		a.enabled = enabled
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"ENABLE_LDAP": a.enabled})
}

func testLDAPConfigProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

func TestLDAPConfigResource_WritesTheConnectionAndTheSwitch(t *testing.T) {
	server := newFakeLDAPConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_ldap_config" "test" {
  enable_ldap     = true
  label           = "Household directory"
  host            = "ldap.internal.example"
  port            = 636
  app_dn          = "cn=openwebui,dc=example,dc=com"
  app_dn_password = "bind-secret"
  search_base     = "ou=people,dc=example,dc=com"
}
`, testLDAPConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "id", "ldap"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "enable_ldap", "true"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "label", "Household directory"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "port", "636"),
					// The bind password comes back in plain text, so state converges.
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "app_dn_password", "bind-secret"),
					// Fields the configuration does not name keep their values.
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "attribute_for_mail", "mail"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "ciphers", "ALL"),
				),
			},
			{
				ResourceName:      "openwebui_ldap_config.test",
				ImportState:       true,
				ImportStateId:     "ldap",
				ImportStateVerify: true,
			},
		},
	})
}

// Open WebUI refuses group management with no attribute to read groups from.
// The refusal belongs in the plan, where it names the attribute at fault.
func TestLDAPConfigResource_RejectsGroupManagementWithoutAnAttribute(t *testing.T) {
	server := newFakeLDAPConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_ldap_config" "test" {
  label                   = "Household directory"
  host                    = "ldap.internal.example"
  app_dn                  = "cn=openwebui,dc=example,dc=com"
  app_dn_password         = "bind-secret"
  search_base             = "ou=people,dc=example,dc=com"
  enable_group_management = true
  attribute_for_groups    = ""
}
`, testLDAPConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Group attribute required`),
			},
		},
	})
}

func TestLDAPConfigResource_RejectsAnEmptyLabel(t *testing.T) {
	server := newFakeLDAPConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_ldap_config" "test" {
  label = ""
  host  = "ldap.internal.example"
}
`, testLDAPConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`string length must be at least 1`),
			},
		},
	})
}
