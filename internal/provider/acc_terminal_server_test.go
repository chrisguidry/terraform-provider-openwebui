package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccTerminalServerResource creates one connection, checks the server-side
// defaults come back, imports it, and edits it. Mutates the target Open WebUI
// instance. The URLs never have to answer: Open WebUI stores a connection it
// cannot reach.
func TestAccTerminalServerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "test" {
  server_id = "tf-acc-shell"
  url       = "http://terminals.invalid:8080"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "id", "tf-acc-shell"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "url", "http://terminals.invalid:8080"),
					// Open WebUI fills these when the request leaves them out.
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "path", "/openapi.json"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "auth_type", "bearer"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "enabled", "true"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "test" {
  server_id = "tf-acc-shell"
  url       = "http://terminals.invalid:8080"
}`,
				PlanOnly: true,
			},
			{
				ResourceName:      "openwebui_terminal_server.test",
				ImportState:       true,
				ImportStateId:     "tf-acc-shell",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "test" {
  server_id   = "tf-acc-shell"
  name        = "Acceptance shell"
  url         = "http://terminals.invalid:9090"
  enabled     = false
  auth_type   = "session"
  key         = "secret"
  config_json = jsonencode({ timeout = 30 })
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "url", "http://terminals.invalid:9090"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "name", "Acceptance shell"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "enabled", "false"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.test", "key", "secret"),
				),
			},
		},
	})
}

// TestAccTerminalServerResource_TwoInOneApply is the regression test for the
// read-modify-write race. Terraform applies both resources concurrently, and
// each one rewrites the whole list, so a lost write would leave one connection
// missing from state on the re-plan.
func TestAccTerminalServerResource_TwoInOneApply(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "first" {
  server_id = "tf-acc-first"
  url       = "http://first.invalid:8080"
}

resource "openwebui_terminal_server" "second" {
  server_id = "tf-acc-second"
  url       = "http://second.invalid:8080"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_terminal_server.first", "url", "http://first.invalid:8080"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.second", "url", "http://second.invalid:8080"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "first" {
  server_id = "tf-acc-first"
  url       = "http://first.invalid:8080"
}

resource "openwebui_terminal_server" "second" {
  server_id = "tf-acc-second"
  url       = "http://second.invalid:8080"
}`,
				PlanOnly: true,
			},
			// Editing one connection must leave its sibling alone.
			{
				Config: testAccProviderConfig() + `
resource "openwebui_terminal_server" "first" {
  server_id = "tf-acc-first"
  url       = "http://first.invalid:8081"
}

resource "openwebui_terminal_server" "second" {
  server_id = "tf-acc-second"
  url       = "http://second.invalid:8080"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_terminal_server.first", "url", "http://first.invalid:8081"),
					resource.TestCheckResourceAttr("openwebui_terminal_server.second", "url", "http://second.invalid:8080"),
				),
			},
		},
	})
}
