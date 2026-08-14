package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDefaultUserPermissionsResource writes the full key set, checks the two
// keys whose Pydantic defaults disagree with the shipped configuration, and
// re-plans for a clean diff. Mutates the target Open WebUI instance: it replaces
// the default permissions of every user without a group.
func TestAccDefaultUserPermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccDefaultUserPermissionsConfig(nil, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("openwebui_default_user_permissions.test", "id"),
					// Both default to true on the write model and false in the
					// shipped configuration, so an explicit false has to stick.
					resource.TestCheckResourceAttr("openwebui_default_user_permissions.test", "permissions.sharing.public_tools", "false"),
					resource.TestCheckResourceAttr("openwebui_default_user_permissions.test", "permissions.sharing.public_notes", "false"),
					// The wire name of the chat import flag is `import`.
					resource.TestCheckResourceAttr("openwebui_default_user_permissions.test", "permissions.chat.import", "false"),
					// This route has no open_chats field at all.
					resource.TestCheckNoResourceAttr("openwebui_default_user_permissions.test", "permissions.sharing.open_chats"),
				),
			},
			{
				Config:   testAccProviderConfig() + testAccDefaultUserPermissionsConfig(nil, nil),
				PlanOnly: true,
			},
			{
				ResourceName:      "openwebui_default_user_permissions.test",
				ImportState:       true,
				ImportStateId:     "default_user_permissions",
				ImportStateVerify: true,
			},
			{
				Config: testAccProviderConfig() + testAccDefaultUserPermissionsConfig(map[string]bool{"features.web_search": true, "settings.interface": true}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_default_user_permissions.test", "permissions.features.web_search", "true"),
					resource.TestCheckResourceAttr("openwebui_default_user_permissions.test", "permissions.settings.interface", "true"),
				),
			},
		},
	})
}

// testAccDefaultUserPermissionsConfig renders every key of every category, which
// is what this resource requires. Keys named in overrides are set to the value
// given, keys named in omit are left out, and everything else is false.
func testAccDefaultUserPermissionsConfig(overrides map[string]bool, omit []string) string {
	omitted := make(map[string]struct{}, len(omit))
	for _, key := range omit {
		omitted[key] = struct{}{}
	}

	var config strings.Builder
	config.WriteString("\nresource \"openwebui_default_user_permissions\" \"test\" {\n  permissions = {\n")

	for _, category := range defaultUserPermissionCategories {
		config.WriteString("    " + category + " = {\n")
		for _, key := range defaultUserPermissionsKeys(category) {
			dotted := category + "." + key
			if _, skip := omitted[dotted]; skip {
				continue
			}
			fmt.Fprintf(&config, "      %q = %t\n", key, overrides[dotted])
		}
		for dotted, value := range overrides {
			key, found := strings.CutPrefix(dotted, category+".")
			if !found || containsPermissionKey(defaultUserPermissionsKeys(category), key) {
				continue
			}
			fmt.Fprintf(&config, "      %q = %t\n", key, value)
		}
		config.WriteString("    }\n")
	}

	config.WriteString("  }\n}\n")

	return config.String()
}
