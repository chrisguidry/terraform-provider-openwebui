package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccLDAPConfigResource applies an LDAP directory connection and verifies
// idempotency. It leaves LDAP sign-in off, so the instance keeps answering to
// its local accounts. Mutates the target Open WebUI instance.
func TestAccLDAPConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_config" "test" {
  enable_ldap            = false
  label                  = "Household directory"
  host                   = "ldap.invalid"
  port                   = 636
  attribute_for_mail     = "mail"
  attribute_for_username = "uid"
  app_dn                 = "cn=openwebui,dc=example,dc=com"
  app_dn_password        = "bind-secret"
  search_base            = "ou=people,dc=example,dc=com"
  use_tls                = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "id", "ldap"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "enable_ldap", "false"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "label", "Household directory"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "port", "636"),
					// The bind password reads back unmasked, so state converges.
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "app_dn_password", "bind-secret"),
				),
			},
			{
				ResourceName:      "openwebui_ldap_config.test",
				ImportState:       true,
				ImportStateId:     "ldap",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_config" "test" {
  enable_ldap            = false
  label                  = "Household directory"
  host                   = "ldap.invalid"
  port                   = 636
  attribute_for_mail     = "mail"
  attribute_for_username = "uid"
  app_dn                 = "cn=openwebui,dc=example,dc=com"
  app_dn_password        = "bind-secret"
  search_base            = "ou=people,dc=example,dc=com"
  use_tls                = true

  enable_group_management = true
  attribute_for_groups    = "memberOf"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "enable_group_management", "true"),
					resource.TestCheckResourceAttr("openwebui_ldap_config.test", "attribute_for_groups", "memberOf"),
				),
			},
		},
	})
}

// Open WebUI refuses group management with no attribute to read groups from.
func TestAccLDAPConfigResource_RejectsGroupManagementWithoutAnAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_ldap_config" "invalid" {
  label                   = "Household directory"
  host                    = "ldap.invalid"
  app_dn                  = "cn=openwebui,dc=example,dc=com"
  app_dn_password         = "bind-secret"
  search_base             = "ou=people,dc=example,dc=com"
  enable_group_management = true
  attribute_for_groups    = ""
}
`,
				ExpectError: regexp.MustCompile(`Group attribute required`),
			},
		},
	})
}
