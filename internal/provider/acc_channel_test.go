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
