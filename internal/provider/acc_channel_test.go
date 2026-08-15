package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The channel routes answer 403 while the channels feature is off, so the
// configuration turns it on first. Mutates the target Open WebUI instance.
func TestAccChannelResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-channel")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelResourceConfig(name, "House announcements"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_channel.test", "description", "House announcements"),
					resource.TestCheckResourceAttrSet("openwebui_channel.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_channel.test", "user_id"),
				),
			},
			{
				ResourceName:      "openwebui_channel.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccChannelResourceConfig(name, "House announcements, revised"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "description", "House announcements, revised"),
				),
			},
		},
	})
}

// A name Open WebUI would lowercase on create never converges, because the
// update path stores the name verbatim.
func TestAccChannelResource_RejectsAnUppercaseName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_channel" "invalid" {
  name = "Announcements"
}
`,
				ExpectError: regexp.MustCompile(`must be lowercase`),
			},
		},
	})
}

func TestAccChannelDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-channel-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelResourceConfig(name, "House announcements") + `
data "openwebui_channel" "found" {
  channel_id = openwebui_channel.test.id
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_channel.found", "name", name),
					resource.TestCheckResourceAttrSet("data.openwebui_channel.found", "user_id"),
				),
			},
		},
	})
}

func testAccChannelResourceConfig(name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_admin_config" "channels" {
  enable_channels = true
}

resource "openwebui_channel" "test" {
  name        = %q
  description = %q
  is_private  = false

  depends_on = [openwebui_admin_config.channels]
}
`, testAccProviderConfig(), name, description)
}

// A channel shared with one account by its mail address. The read-back names
// the same address, so a second plan is empty. Mutates the target Open WebUI
// instance.
func TestAccChannelResource_UserGrants(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-channel-users")
	email := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-channel-member"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccChannelWithUserConfig(name, email, `[openwebui_user.member.email]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.0", email),
					resource.TestCheckResourceAttr("openwebui_channel.test", "public_read", "false"),
				),
			},
			{
				// read_users is Optional and Computed, so an empty list is the
				// only way to revoke the grant. Dropping the attribute keeps the
				// value that is already in state.
				Config: testAccChannelWithUserConfig(name, email, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_channel.test", "read_users.#", "0"),
				),
			},
		},
	})
}

func testAccChannelWithUserConfig(name, email, readUsers string) string {
	return fmt.Sprintf(`%s
resource "openwebui_admin_config" "channels" {
  enable_channels = true
}

resource "openwebui_user" "member" {
  name             = "Channel Member"
  email            = %q
  role             = "user"
  password         = "correct-horse-battery"
  password_version = "1"
}

resource "openwebui_channel" "test" {
  name        = %q
  description = "Shared with one account"
  is_private  = false
  read_users  = %s

  depends_on = [openwebui_admin_config.channels]
}
`, testAccProviderConfig(), email, name, readUsers)
}
