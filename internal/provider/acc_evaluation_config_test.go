package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEvaluationConfigResource applies arena models and verifies
// idempotency. Mutates the target Open WebUI instance.
func TestAccEvaluationConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_evaluation_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models_json = jsonencode([
    {
      id   = "arena-model"
      name = "Arena Model"
      meta = {
        profile_image_url = "/favicon.png"
        description       = "Two models answer, you pick one."
      }
    }
  ])
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "id", "evaluation"),
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "enable_evaluation_arena_models", "true"),
					resource.TestCheckResourceAttrSet("openwebui_evaluation_config.test", "evaluation_arena_models_json"),
				),
			},
			{
				ResourceName:      "openwebui_evaluation_config.test",
				ImportState:       true,
				ImportStateId:     "evaluation",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_evaluation_config" "test" {
  enable_evaluation_arena_models = false
  evaluation_arena_models_json   = jsonencode([])
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "enable_evaluation_arena_models", "false"),
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "evaluation_arena_models_json", "[]"),
				),
			},
		},
	})
}
