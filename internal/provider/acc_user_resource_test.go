package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserResource creates an account, renames it, promotes it, and deletes
// it. Mutates the target Open WebUI instance.
func TestAccUserResource(t *testing.T) {
	email := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig(email, "Terraform Test", "user", "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_user.test", "email", email),
					resource.TestCheckResourceAttr("openwebui_user.test", "name", "Terraform Test"),
					resource.TestCheckResourceAttr("openwebui_user.test", "role", "user"),
					resource.TestCheckResourceAttrSet("openwebui_user.test", "id"),
					// The password is write-only and never lands in state.
					resource.TestCheckNoResourceAttr("openwebui_user.test", "password"),
				),
			},
			{
				ResourceName:      "openwebui_user.test",
				ImportState:       true,
				ImportStateVerify: true,
				// An imported account carries no password and no marker for one.
				ImportStateVerifyIgnore: []string{"password_version"},
			},
			{
				Config: testAccUserResourceConfig(email, "Terraform Test, renamed", "admin", "1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_user.test", "name", "Terraform Test, renamed"),
					resource.TestCheckResourceAttr("openwebui_user.test", "role", "admin"),
				),
			},
			{
				// A new marker is what makes the provider write the password.
				Config: testAccUserResourceConfig(email, "Terraform Test, renamed", "admin", "2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_user.test", "password_version", "2"),
				),
			},
		},
	})
}

// Open WebUI lowercases the address it stores, so an uppercase one never
// converges.
func TestAccUserResource_RejectsAnUppercaseEmail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_user" "invalid" {
  name     = "Terraform Test"
  email    = "Terraform.Test@example.com"
  password = "correct-horse-battery"
}
`,
				ExpectError: regexp.MustCompile(`must be lowercase`),
			},
		},
	})
}

// The primary admin is the account this instance created first. Importing it
// and applying a demotion has to fail before anything reaches the API.
//
// The import has to persist, or the demotion step would create an account
// instead of updating one. The last step then drops the account from state with
// a `removed` block: Open WebUI refuses to delete its primary admin, so a state
// that still held it would fail the closing destroy.
func TestAccUserResource_RefusesToDemoteThePrimaryAdmin(t *testing.T) {
	userID := testAccRequireEnv(t, "OPENWEBUI_TEST_PRIMARY_ADMIN_ID")
	email := testAccRequireEnv(t, "OPENWEBUI_TEST_PRIMARY_ADMIN_EMAIL")

	config := fmt.Sprintf(`%s
resource "openwebui_user" "primary" {
  name  = "Primary Admin"
  email = %q
  role  = "user"
}
`, testAccProviderConfig(), email)

	forget := testAccProviderConfig() + `
removed {
  from = openwebui_user.primary

  lifecycle {
    destroy = false
  }
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       "openwebui_user.primary",
				ImportState:        true,
				ImportStateId:      userID,
				ImportStatePersist: true,
			},
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Refusing to demote the primary admin`),
			},
			{
				Config: forget,
			},
		},
	})
}

func testAccUserResourceConfig(email, name, role, passwordVersion string) string {
	return fmt.Sprintf(`%s
resource "openwebui_user" "test" {
  name             = %q
  email            = %q
  role             = %q
  password         = "correct-horse-battery"
  password_version = %q
}
`, testAccProviderConfig(), name, email, role, passwordVersion)
}
