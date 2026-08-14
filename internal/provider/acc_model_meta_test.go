package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Open WebUI answers every model read with meta.knowledge, whether or not
// anything set it. A model that also carries meta_additional_json fails its
// apply if the provider lets that key through.
func TestAccModelResourceMetaAdditionalJSON(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-meta")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigMetaAdditional(modelID, `jsonencode({ info = "custom" })`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "meta_additional_json", `{"info":"custom"}`),
				),
			},
		},
	})
}

// Open WebUI derives meta.chat_variables_schema from the system prompt when it
// uses the {{ chat.variables.<key> }} syntax. It is server-computed, so it must
// stay out of meta_additional_json.
func TestAccModelResourceChatVariablesSchema(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-chatvars")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigChatVariables(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "meta_additional_json", `{"info":"custom"}`),
				),
			},
		},
	})
}

// Open WebUI keeps the stored base_model_id when the key is absent from an
// update, so removing the attribute has to send an explicit null.
func TestAccModelResourceBaseModelIDRemoval(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-base")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigBaseModel(modelID, `base_model_id = "llama3.2"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "base_model_id", "llama3.2"),
				),
			},
			{
				Config: testAccModelResourceConfigBaseModel(modelID, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("openwebui_model.test", "base_model_id"),
				),
			},
		},
	})
}

func TestAccModelResourceProfileImageURL(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-image")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Open WebUI drops a bare relative path and stores nothing, so
				// the provider has to refuse it while planning. This step comes
				// first because the post-test destroy plans the last config, and
				// a config the validator rejects fails that destroy too.
				Config:      testAccModelResourceConfigProfileImage(modelID, "/img.png"),
				ExpectError: regexp.MustCompile("Invalid profile image URL"),
			},
			{
				Config: testAccModelResourceConfigProfileImage(modelID, "https://example.com/img.png"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "profile_image_url", "https://example.com/img.png"),
				),
			},
		},
	})
}

func TestAccModelResourceSkillIDs(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-skills")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigSkillIDs(modelID, `["research", "summarise"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.#", "2"),
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.0", "research"),
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.1", "summarise"),
				),
			},
			{
				Config: testAccModelResourceConfigSkillIDs(modelID, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.#", "0"),
				),
			},
		},
	})
}

// meta.skillIds carries the raw skill id, so a model has to be able to take it
// straight from a skill resource in the same configuration.
func TestAccModelResourceSkillIDsFromSkillResource(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-skillref")
	skillID := accSkillID("tfaccmodelskill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigSkillResource(modelID, skillID, `[openwebui_skill.test.id]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"openwebui_model.test", "skill_ids.0",
						"openwebui_skill.test", "id",
					),
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.0", skillID),
				),
			},
			{
				Config: testAccModelResourceConfigSkillResource(modelID, skillID, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "skill_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccModelResourceKnowledgeIDs(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-knowledge")
	knowledgeName := acctest.RandomWithPrefix("tf-acc-model-kb")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigKnowledgeIDs(modelID, knowledgeName, `[openwebui_knowledge.test.id]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "knowledge_ids.#", "1"),
					resource.TestCheckResourceAttrPair(
						"openwebui_model.test", "knowledge_ids.0",
						"openwebui_knowledge.test", "id",
					),
				),
			},
			{
				Config: testAccModelResourceConfigKnowledgeIDs(modelID, knowledgeName, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "knowledge_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccModelResourceCapabilities(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-caps")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigCapabilities(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "capabilities.file_context", "false"),
					resource.TestCheckResourceAttr("openwebui_model.test", "capabilities.terminal", "false"),
					resource.TestCheckResourceAttr("openwebui_model.test", "capabilities.memory", "true"),
					resource.TestCheckResourceAttr("openwebui_model.test", "capabilities.builtin_tools", "false"),
				),
			},
		},
	})
}

// A wildcard grant is how Open WebUI shares a model with every signed-in user.
// Without the public_read attribute the provider revokes that sharing on the
// next apply, with no diff to warn about it.
func TestAccModelResourcePublicSharing(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-public")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigPublic(modelID, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "public_read", "true"),
					resource.TestCheckResourceAttr("openwebui_model.test", "public_write", "false"),
				),
			},
			{
				Config: testAccModelResourceConfigPublic(modelID, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "public_read", "false"),
				),
			},
		},
	})
}

func testAccModelResourceConfigMetaAdditional(modelID, metaJSON string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Meta Test Model"
  base_model_id = "llama3.2"

  meta_additional_json = %s

  params = {}
}
`, testAccProviderConfig(), modelID, metaJSON)
}

func testAccModelResourceConfigChatVariables(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Chat Variables Model"
  base_model_id = "llama3.2"

  meta_additional_json = jsonencode({ info = "custom" })

  params = {
    system = "Hello {{ chat.variables.name }}"
  }
}
`, testAccProviderConfig(), modelID)
}

func testAccModelResourceConfigBaseModel(modelID, baseModel string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id = %q
  name     = "Base Model Test"
  %s

  params = {}
}
`, testAccProviderConfig(), modelID, baseModel)
}

func testAccModelResourceConfigProfileImage(modelID, imageURL string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id          = %q
  name              = "Profile Image Model"
  base_model_id     = "llama3.2"
  profile_image_url = %q

  params = {}
}
`, testAccProviderConfig(), modelID, imageURL)
}

func testAccModelResourceConfigSkillIDs(modelID, skillIDs string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Skill Model"
  base_model_id = "llama3.2"
  skill_ids     = %s

  params = {}
}
`, testAccProviderConfig(), modelID, skillIDs)
}

func testAccModelResourceConfigSkillResource(modelID, skillID, skillIDs string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id    = %q
  name        = %q
  description = "Attached to a model"
  content     = <<-MD
# Model skill

Body for the coupling test.
  MD
}

resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Skill Reference Model"
  base_model_id = "llama3.2"
  skill_ids     = %s

  params = {}
}
`, testAccProviderConfig(), skillID, "Model Skill "+skillID, modelID, skillIDs)
}

func testAccModelResourceConfigKnowledgeIDs(modelID, knowledgeName, knowledgeIDs string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name        = %q
  description = "Model knowledge test"
}

resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Knowledge Model"
  base_model_id = "llama3.2"
  knowledge_ids = %s

  params = {}
}
`, testAccProviderConfig(), knowledgeName, modelID, knowledgeIDs)
}

func testAccModelResourceConfigCapabilities(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Capabilities Model"
  base_model_id = "llama3.2"

  capabilities = {
    file_context  = false
    terminal      = false
    memory        = true
    builtin_tools = false
  }

  params = {}
}
`, testAccProviderConfig(), modelID)
}

func testAccModelResourceConfigPublic(modelID string, publicRead, publicWrite bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Public Model"
  base_model_id = "llama3.2"
  public_read   = %t
  public_write  = %t

  params = {}
}
`, testAccProviderConfig(), modelID, publicRead, publicWrite)
}
