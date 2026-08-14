package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testSkillDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "seed" {
  skill_id    = "release-notes"
  name        = "Release Notes"
  description = "Writes release notes"
  content     = "# Release notes\n"
  tags        = ["docs"]
}

data "openwebui_skill" "test" {
  skill_id = openwebui_skill.seed.skill_id
}
`, testSkillProviderConfig(endpoint))
}

func TestSkillDataSource_ReadsByID(t *testing.T) {
	server := newFakeSkillAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testSkillDataSourceConfig(server.URL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "skill_id", "release-notes"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "id", "release-notes"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "name", "Release Notes"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "description", "Writes release notes"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "content", "# Release notes\n"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "tags.0", "docs"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "is_active", "true"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "public_read", "false"),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "write_access", "true"),
				),
			},
		},
	})
}

func TestSkillDataSource_MissingSkill(t *testing.T) {
	server := newFakeSkillAPI(t)

	config := fmt.Sprintf(`%s
data "openwebui_skill" "test" {
  skill_id = "nothing-here"
}
`, testSkillProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Skill not found|Read skill failed`),
			},
		},
	})
}
