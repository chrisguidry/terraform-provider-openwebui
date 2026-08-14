package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// accSkillID builds an identifier Open WebUI stores unchanged: lowercase, no
// spaces.
func accSkillID(prefix string) string {
	return strings.ToLower(acctest.RandomWithPrefix(prefix))
}

func accSkillContent(body string) string {
	return fmt.Sprintf(`# Acceptance skill

%s
`, body)
}

func testAccSkillResourceConfig(skillID, name, description, body string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id    = %q
  name        = %q
  description = %q
  content     = <<-MD
%s
  MD
  tags        = ["acceptance", "terraform"]
}
`, testAccProviderConfig(), skillID, name, description, accSkillContent(body))
}

func testAccSkillWithGroupConfig(skillID, name, groupName, readGroups string) string {
	return fmt.Sprintf(`%s
resource "openwebui_group" "test" {
  name        = %q
  description = "Skill sharing test"
}

resource "openwebui_skill" "test" {
  skill_id    = %q
  name        = %q
  description = "Shared with a group"
  content     = <<-MD
%s
  MD
  tags        = ["acceptance", "terraform"]
  read_groups = %s
}
`, testAccProviderConfig(), groupName, skillID, name, accSkillContent("Shared body."), readGroups)
}

func testAccSkillDataSourceConfig(skillID, name string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "seed" {
  skill_id    = %q
  name        = %q
  description = "Data source lookup test"
  content     = <<-MD
%s
  MD
  tags        = ["acceptance"]
}

data "openwebui_skill" "test" {
  skill_id = openwebui_skill.seed.skill_id
}
`, testAccProviderConfig(), skillID, name, accSkillContent("Lookup body."))
}

func TestAccSkillResource(t *testing.T) {
	skillID := accSkillID("tfaccskill")
	name := acctest.RandomWithPrefix("Acc Skill")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSkillResourceConfig(skillID, name, "Initial description", "Initial body."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "skill_id", skillID),
					resource.TestCheckResourceAttr("openwebui_skill.test", "id", skillID),
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", name),
					resource.TestCheckResourceAttr("openwebui_skill.test", "description", "Initial description"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.#", "2"),
					resource.TestMatchResourceAttr("openwebui_skill.test", "content", regexp.MustCompile(`Initial body\.`)),
					resource.TestCheckResourceAttrSet("openwebui_skill.test", "user_id"),
				),
			},
			{
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateId:     skillID,
				ImportStateVerify: true,
			},
			{
				Config: testAccSkillResourceConfig(skillID, name+" v2", "Updated description", "Updated body."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "skill_id", skillID),
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", name+" v2"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "description", "Updated description"),
					resource.TestMatchResourceAttr("openwebui_skill.test", "content", regexp.MustCompile(`Updated body\.`)),
				),
			},
		},
	})
}

func TestAccSkillResource_ReadGroups(t *testing.T) {
	skillID := accSkillID("tfaccskillgrp")
	name := acctest.RandomWithPrefix("Acc Skill Group")
	groupName := acctest.RandomWithPrefix("tf-acc-skill-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSkillWithGroupConfig(skillID, name, groupName, `[openwebui_group.test.name]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.#", "1"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.0", groupName),
					resource.TestCheckResourceAttr("openwebui_skill.test", "public_read", "false"),
				),
			},
			{
				// read_groups is Optional and Computed, so an empty list is the
				// only way to revoke the grant. Dropping the attribute keeps the
				// value that is already in state.
				Config: testAccSkillWithGroupConfig(skillID, name, groupName, `[]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.#", "0"),
				),
			},
			{
				Config: testAccSkillResourceConfig(skillID, name, "Sharing removed", "Shared body."),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "read_groups.#", "0"),
				),
			},
		},
	})
}

func TestAccSkillResource_PublicRead(t *testing.T) {
	skillID := accSkillID("tfaccskillpub")
	name := acctest.RandomWithPrefix("Acc Skill Public")

	config := fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id    = %q
  name        = %q
  description = "Public sharing test"
  content     = <<-MD
%s
  MD
  public_read = true
}
`, testAccProviderConfig(), skillID, name, accSkillContent("Public body."))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "public_read", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "public_write", "false"),
				),
			},
			{
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateId:     skillID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSkillDataSource(t *testing.T) {
	skillID := accSkillID("tfaccskillds")
	name := acctest.RandomWithPrefix("Acc Skill DS")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSkillDataSourceConfig(skillID, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "skill_id", skillID),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "name", name),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "description", "Data source lookup test"),
					resource.TestMatchResourceAttr("data.openwebui_skill.test", "content", regexp.MustCompile(`Lookup body\.`)),
					resource.TestCheckResourceAttr("data.openwebui_skill.test", "tags.0", "acceptance"),
				),
			},
		},
	})
}
