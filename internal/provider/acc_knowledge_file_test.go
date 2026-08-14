package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKnowledgeFileResource(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello from terraform acc test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	name := acctest.RandomWithPrefix("tf-acc-kf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccKnowledgeFileResourceConfig(name, filePath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "knowledge_id"),
					resource.TestCheckResourceAttrSet("openwebui_knowledge_file.test", "file_id"),
					// The knowledge file listing defers the extracted content, so
					// file_json holds metadata only.
					resource.TestMatchResourceAttr("openwebui_knowledge_file.test", "file_json", regexp.MustCompile(`"filename"`)),
					func(s *terraform.State) error {
						attrs := s.RootModule().Resources["openwebui_knowledge_file.test"].Primary.Attributes
						var file map[string]any
						if err := json.Unmarshal([]byte(attrs["file_json"]), &file); err != nil {
							return fmt.Errorf("decode file_json: %w", err)
						}
						if _, ok := file["data"]; ok {
							return fmt.Errorf("file_json must not carry a data key: %s", attrs["file_json"])
						}
						return nil
					},
				),
			},
			{
				ResourceName:            "openwebui_knowledge_file.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_file"},
			},
		},
	})
}

func testAccKnowledgeFileResourceConfig(name, filePath string) string {
	return fmt.Sprintf(`%s
resource "openwebui_knowledge" "kb" {
  name        = %q
  description = "Knowledge file acc test"
}

resource "openwebui_file" "doc" {
  source_path = %q
  process     = true
}

resource "openwebui_knowledge_file" "test" {
  knowledge_id = openwebui_knowledge.kb.id
  file_id      = openwebui_file.doc.id
  delete_file  = true
}
`, testAccProviderConfig(), name, filePath)
}
