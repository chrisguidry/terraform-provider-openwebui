package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKnowledgeResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeResourceConfig(name, "Initial description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge.test", "created_at"),
				),
			},
			{
				ResourceName:      "openwebui_knowledge.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKnowledgeResourceConfig(name, "Updated description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "description", "Updated description"),
				),
			},
		},
	})
}

func TestAccKnowledgeDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeDataSourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_knowledge.test", "name", name),
					resource.TestCheckResourceAttrSet("data.openwebui_knowledge.test", "id"),
					resource.TestCheckResourceAttrSet("data.openwebui_knowledge.test", "knowledge_id"),
					resource.TestCheckResourceAttr("data.openwebui_knowledge.test", "file_count", "0"),
				),
			},
		},
	})
}

func TestAccKnowledgeResource_PublicSharing(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-knowledge-public")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgePublicConfig(name, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "public_read", "true"),
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "public_write", "false"),
				),
			},
			{
				Config: testAccKnowledgePublicConfig(name, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_knowledge.test", "public_read", "false"),
				),
			},
		},
	})
}

func testAccKnowledgePublicConfig(name string, publicRead, publicWrite bool) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name         = %q
  description  = "Public sharing test"
  public_read  = %t
  public_write = %t
}
`, testAccProviderConfig(), name, publicRead, publicWrite)
}

func testAccKnowledgeResourceConfig(name, description string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "test" {
  name        = %q
  description = %q
}
`, testAccProviderConfig(), name, description)
}

func testAccKnowledgeDataSourceConfig(name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "seed" {
  name        = %q
  description = "Data source lookup test"
}

data "openwebui_knowledge" "test" {
  name = openwebui_knowledge.seed.name
}
`, testAccProviderConfig(), name)
}
