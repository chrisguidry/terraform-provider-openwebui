package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAdminConfigResource applies a few administrative settings and verifies
// idempotency. The settings it names are cosmetic, and the ones it does not
// name keep the values the instance already holds. Mutates the target Open
// WebUI instance.
func TestAccAdminConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_admin_config" "test" {
  pending_user_overlay_title   = "Waiting for approval"
  pending_user_overlay_content = "An administrator reviews new accounts."
  folder_max_file_count        = "50"
  jwt_expires_in               = "4h"
  default_user_role            = "pending"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "id", "admin"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "pending_user_overlay_title", "Waiting for approval"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "folder_max_file_count", "50"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "jwt_expires_in", "4h"),
					// A setting the configuration does not name still reads back, so
					// the resource covers the whole form. WEBUI_URL is empty on a
					// fresh instance, so it cannot carry this check.
					resource.TestCheckResourceAttrSet("openwebui_admin_config.test", "channel_model_response_mode"),
				),
			},
			{
				ResourceName:      "openwebui_admin_config.test",
				ImportState:       true,
				ImportStateId:     "admin",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_admin_config" "test" {
  pending_user_overlay_title   = "Almost there"
  pending_user_overlay_content = "An administrator reviews new accounts."
  folder_max_file_count        = ""
  jwt_expires_in               = "4h"
  default_user_role            = "pending"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "pending_user_overlay_title", "Almost there"),
					resource.TestCheckResourceAttr("openwebui_admin_config.test", "folder_max_file_count", ""),
				),
			},
		},
	})
}

// Open WebUI drops a token expiry it cannot parse and still answers 200, which
// would leave a permanent diff with nothing to explain it.
func TestAccAdminConfigResource_RejectsAnUnparsableExpiry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_admin_config" "invalid" {
  jwt_expires_in = "a fortnight"
}
`,
				ExpectError: regexp.MustCompile(`must be -1, 0, or a number`),
			},
		},
	})
}
