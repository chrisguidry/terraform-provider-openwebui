package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSubagentsConfigResource applies a known subagents config, checks the
// read-back, and re-plans for a clean diff. Mutates the target Open WebUI
// instance.
func TestAccSubagentsConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccSubagentsConfig("2", "be brief"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_subagents_config.test", "id"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "enable_subagents", "true"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_concurrent", "2"),
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_system_prompt", "be brief"),
				),
			},
			{
				Config:   testAccProviderConfig() + testAccSubagentsConfig("2", "be brief"),
				PlanOnly: true,
			},
			{
				ResourceName:      "openwebui_subagents_config.test",
				ImportState:       true,
				ImportStateId:     "subagents",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + testAccSubagentsConfig("4", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_max_concurrent", "4"),
					// An empty system prompt is a real value, not an absent one.
					resource.TestCheckResourceAttr("openwebui_subagents_config.test", "subagents_system_prompt", ""),
				),
			},
		},
	})
}

func testAccSubagentsConfig(maxConcurrent, systemPrompt string) string {
	return `
resource "openwebui_subagents_config" "test" {
  enable_subagents             = true
  subagents_background_enabled = false
  subagents_max_concurrent     = ` + maxConcurrent + `
  subagents_max_async          = 1
  subagents_max_iterations     = 8
  subagents_max_output         = 4000
  subagents_system_prompt      = "` + systemPrompt + `"
}`
}
