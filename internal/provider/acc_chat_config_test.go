package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccChatConfigResource applies context compaction settings and verifies
// idempotency. Mutates the target Open WebUI instance.
func TestAccChatConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_chat_config" "test" {
  enable_context_compaction               = true
  context_compaction_token_threshold      = 40000
  context_compaction_token_cap            = 60000
  context_compaction_retention_percentage = 30
  context_compaction_prompt_template      = "Summarise the conversation so far."
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "id", "chat"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "enable_context_compaction", "true"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_token_threshold", "40000"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_token_cap", "60000"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_retention_percentage", "30"),
				),
			},
			{
				ResourceName:      "openwebui_chat_config.test",
				ImportState:       true,
				ImportStateId:     "chat",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_chat_config" "test" {
  enable_context_compaction          = false
  context_compaction_token_threshold = 80000
  context_compaction_prompt_template = "Summarise the conversation so far."
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "enable_context_compaction", "false"),
					// The cap and the retention keep the values already stored.
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_token_cap", "60000"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_retention_percentage", "30"),
				),
			},
		},
	})
}
