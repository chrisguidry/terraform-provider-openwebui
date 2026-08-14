package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigExportDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigExportDataSourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openwebui_config_export.current", "config_json"),
				),
			},
		},
	})
}

func TestAccConfigImportResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_TEST_CONFIG_IMPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigImportResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_config_import.restore", "id", "config_import"),
					resource.TestCheckResourceAttrSet("openwebui_config_import.restore", "config_json"),
				),
			},
		},
	})
}

// A config_json holding fewer keys than the server has must apply and re-plan
// clean. POST /configs/import answers with all 394 keys, and recording that
// answer used to fail the apply with an inconsistent result.
func TestAccConfigImportResource_PartialConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_TEST_CONFIG_IMPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_config_import" "partial" {
  config_json = jsonencode({
    "ui.banners" = []
  })
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_config_import.partial", "id", "config_import"),
					resource.TestCheckResourceAttr("openwebui_config_import.partial", "config_json", `{"ui.banners":[]}`),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_config_import" "partial" {
  config_json = jsonencode({
    "ui.banners" = []
  })
}`,
				PlanOnly: true,
			},
		},
	})
}

// A configuration exported from Open WebUI v0.9.x is a nested tree. Importing it
// into v0.11.0 writes rows that nothing reads, so the provider refuses it.
func TestAccConfigImportResource_RejectsNestedConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccRequireEnv(t, "OPENWEBUI_TEST_CONFIG_IMPORT")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_config_import" "nested" {
  config_json = jsonencode({
    ui = { banners = [] }
  })
}`,
				ExpectError: regexp.MustCompile(`Nested configuration keys`),
			},
		},
	})
}

func testAccConfigExportDataSourceConfig() string {
	return testAccProviderConfig() + `
data "openwebui_config_export" "current" {}
`
}

func testAccConfigImportResourceConfig() string {
	return testAccProviderConfig() + `
data "openwebui_config_export" "current" {}

resource "openwebui_config_import" "restore" {
  config_json = data.openwebui_config_export.current.config_json
}
`
}
