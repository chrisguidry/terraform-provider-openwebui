package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTaskConfigResource applies a known task config, checks that an empty
// template stays an empty string rather than becoming null, and re-plans for a
// clean diff. Mutates the target Open WebUI instance.
func TestAccTaskConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccTaskConfig("true", `""`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_task_config.test", "id"),
					resource.TestCheckResourceAttr("openwebui_task_config.test", "enable_title_generation", "true"),
					// An empty template asks Open WebUI for its built-in prompt,
					// and reads back as an empty string.
					resource.TestCheckResourceAttr("openwebui_task_config.test", "title_generation_prompt_template", ""),
					resource.TestCheckNoResourceAttr("openwebui_task_config.test", "task_model"),
				),
			},
			{
				Config:   testAccProviderConfig() + testAccTaskConfig("true", `""`),
				PlanOnly: true,
			},
			{
				ResourceName:      "openwebui_task_config.test",
				ImportState:       true,
				ImportStateId:     "task_config",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + testAccTaskConfig("false", `"Write a title for: {{prompt}}"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_task_config.test", "enable_title_generation", "false"),
					resource.TestCheckResourceAttr("openwebui_task_config.test", "title_generation_prompt_template", "Write a title for: {{prompt}}"),
				),
			},
		},
	})
}

func testAccTaskConfig(enableTitles, titleTemplate string) string {
	return `
resource "openwebui_task_config" "test" {
  enable_title_generation                  = ` + enableTitles + `
  title_generation_prompt_template         = ` + titleTemplate + `
  image_prompt_generation_prompt_template  = ""
  enable_autocomplete_generation           = true
  autocomplete_generation_input_max_length = -1
  autocomplete_generation_prompt_template  = ""
  tags_generation_prompt_template          = ""
  follow_up_generation_prompt_template     = ""
  enable_follow_up_generation              = true
  enable_tags_generation                   = true
  enable_search_query_generation           = true
  enable_retrieval_query_generation        = true
  query_generation_prompt_template         = ""
  tools_function_calling_prompt_template   = ""
  enable_voice_mode_prompt                 = false
}`
}
