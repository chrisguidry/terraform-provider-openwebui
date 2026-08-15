package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModelResource(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfig(modelID, "Initial Name", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttr("openwebui_model.test", "name", "Initial Name"),
					resource.TestCheckResourceAttr("openwebui_model.test", "is_active", "false"),
					resource.TestCheckResourceAttrSet("openwebui_model.test", "id"),
				),
			},
			{
				ResourceName:      "openwebui_model.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccModelResourceConfig(modelID, "Updated Name", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "name", "Updated Name"),
					resource.TestCheckResourceAttr("openwebui_model.test", "is_active", "true"),
				),
			},
		},
	})
}

func TestAccModelDataSource(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelDataSourceConfig(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttrSet("data.openwebui_model.test", "id"),
				),
			},
		},
	})
}

// TestAccModelResourceNoCapabilities verifies that omitting the capabilities
// block entirely does not produce an "unknown value" conversion error.
func TestAccModelResourceNoCapabilities(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-nocaps")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigNoCaps(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttrSet("openwebui_model.test", "id"),
				),
			},
		},
	})
}

func testAccModelResourceConfigNoCaps(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "No Caps Model"
  base_model_id = "llama3.2"

  params = {}
}
`, testAccProviderConfig(), modelID)
}

func testAccModelResourceConfig(modelID, name string, active bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = %q
  base_model_id = "llama3.2"
  is_active     = %t

  params = {}

  capabilities = {}
}
`, testAccProviderConfig(), modelID, name, active)
}

// TestAccModelResourceNullDescriptionUpdate is a regression test for the bug
// where setting description on a model that previously had no description
// (API returns null) would cause a "planned/actual state mismatch" provider
// error. The fix ensures null meta fields are never leaked into
// meta_additional_json during flattenModelMeta.
func TestAccModelResourceNullDescriptionUpdate(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-desc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// Step 1: create without description — API will return null for it.
				Config: testAccModelResourceConfigNoDescription(modelID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckNoResourceAttr("openwebui_model.test", "description"),
				),
			},
			{
				// Step 2: add description — must apply cleanly with no state mismatch.
				Config: testAccModelResourceConfigWithDescription(modelID, "hello from terraform"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "description", "hello from terraform"),
				),
			},
			{
				ResourceName:      "openwebui_model.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccModelResourceConfigNoDescription(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Desc Test Model"
  base_model_id = "llama3.2"

  params = {}
}
`, testAccProviderConfig(), modelID)
}

func testAccModelResourceConfigWithDescription(modelID, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Desc Test Model"
  base_model_id = "llama3.2"
  description   = %q

  params = {}
}
`, testAccProviderConfig(), modelID, description)
}

// TestAccModelResourceHidden verifies that the hidden field round-trips through
// create and update operations correctly.
func TestAccModelResourceHidden(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-hidden")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelResourceConfigHidden(modelID, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "model_id", modelID),
					resource.TestCheckResourceAttr("openwebui_model.test", "hidden", "true"),
				),
			},
			{
				Config: testAccModelResourceConfigHidden(modelID, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "hidden", "false"),
				),
			},
		},
	})
}

func testAccModelResourceConfigHidden(modelID string, hidden bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Hidden Test Model"
  base_model_id = "llama3.2"
  hidden        = %t

  params = {}
}
`, testAccProviderConfig(), modelID, hidden)
}

func testAccModelDataSourceConfig(modelID string) string {
	return fmt.Sprintf(`%s
resource "openwebui_model" "seed" {
  model_id      = %q
  name          = "Acc DS Model"
  base_model_id = "llama3.2"

  params = {}

  capabilities = {}
}

data "openwebui_model" "test" {
  model_id = openwebui_model.seed.model_id
}
`, testAccProviderConfig(), modelID)
}

// A model shared with one account for reading and one for writing. A user named
// for writing reads the model too, so both lists name that account. Mutates the
// target Open WebUI instance.
func TestAccModelResource_UserGrants(t *testing.T) {
	modelID := acctest.RandomWithPrefix("tf-acc-model-users")
	email := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-model-editor"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccModelWithUserConfig(modelID, email, `[openwebui_user.editor.email]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "write_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_model.test", "write_users.0", email),
					resource.TestCheckResourceAttr("openwebui_model.test", "read_users.#", "1"),
					resource.TestCheckResourceAttr("openwebui_model.test", "read_users.0", email),
					resource.TestCheckResourceAttr("openwebui_model.test", "public_read", "false"),
				),
			},
			{
				ResourceName:      "openwebui_model.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccModelWithUserConfig(modelID, email, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_model.test", "write_users.#", "0"),
					resource.TestCheckResourceAttr("openwebui_model.test", "read_users.#", "0"),
				),
			},
		},
	})
}

func testAccModelWithUserConfig(modelID, email, writeUsers string) string {
	return fmt.Sprintf(`%s
resource "openwebui_user" "editor" {
  name             = "Model Editor"
  email            = %q
  role             = "user"
  password         = "correct-horse-battery"
  password_version = "1"
}

resource "openwebui_model" "test" {
  model_id      = %q
  name          = "Shared Model"
  base_model_id = "llama3.2"

  params = {}

  write_users = %s
}
`, testAccProviderConfig(), email, modelID, writeUsers)
}
