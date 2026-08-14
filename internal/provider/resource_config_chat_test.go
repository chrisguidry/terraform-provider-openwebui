package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// fakeChatConfigAPI serves the context compaction routes and clamps on write
// the way Open WebUI does.
type fakeChatConfigAPI struct {
	mu     sync.Mutex
	values map[string]any
}

func newFakeChatConfigAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeChatConfigAPI{values: map[string]any{
		"CONTEXT_COMPACTION_MODEL":                "",
		"ENABLE_CONTEXT_COMPACTION":               false,
		"CONTEXT_COMPACTION_TOKEN_THRESHOLD":      80000,
		"CONTEXT_COMPACTION_TOKEN_CAP":            80000,
		"CONTEXT_COMPACTION_RETENTION_PERCENTAGE": 25,
		"CONTEXT_COMPACTION_PROMPT_TEMPLATE":      "",
	}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeChatConfigAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		threshold := chatConfigFloor(chatConfigNumber(form["CONTEXT_COMPACTION_TOKEN_THRESHOLD"]), 1)
		tokenCap := chatConfigNumber(form["CONTEXT_COMPACTION_TOKEN_CAP"])
		if tokenCap == 0 {
			tokenCap = threshold
		}
		retention := chatConfigNumber(form["CONTEXT_COMPACTION_RETENTION_PERCENTAGE"])
		if retention > 50 {
			retention = 50
		}
		if retention < 10 {
			retention = 10
		}

		a.values["CONTEXT_COMPACTION_MODEL"] = chatConfigString(form["CONTEXT_COMPACTION_MODEL"])
		a.values["ENABLE_CONTEXT_COMPACTION"] = form["ENABLE_CONTEXT_COMPACTION"]
		a.values["CONTEXT_COMPACTION_TOKEN_THRESHOLD"] = threshold
		a.values["CONTEXT_COMPACTION_TOKEN_CAP"] = chatConfigFloor(tokenCap, 1)
		a.values["CONTEXT_COMPACTION_RETENTION_PERCENTAGE"] = retention
		a.values["CONTEXT_COMPACTION_PROMPT_TEMPLATE"] = chatConfigString(form["CONTEXT_COMPACTION_PROMPT_TEMPLATE"])
	}

	_ = json.NewEncoder(w).Encode(a.values)
}

func chatConfigNumber(value any) int {
	number, _ := value.(float64)
	return int(number)
}

func chatConfigString(value any) string {
	text, _ := value.(string)
	return text
}

func chatConfigFloor(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func testChatConfigProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

// An attribute the configuration does not name keeps the value the instance
// holds, because the route writes all six settings at once.
func TestChatConfigResource_KeepsUnnamedSettings(t *testing.T) {
	server := newFakeChatConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_chat_config" "test" {
  enable_context_compaction          = true
  context_compaction_token_threshold = 40000
  context_compaction_prompt_template = "Summarise the conversation."
}
`, testChatConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "id", "chat"),
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "enable_context_compaction", "true"),
					// The cap follows the threshold when nothing sets it.
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_token_cap", "40000"),
					// The stored retention survives, rather than falling back to 40.
					resource.TestCheckResourceAttr("openwebui_chat_config.test", "context_compaction_retention_percentage", "25"),
				),
			},
			{
				ResourceName:      "openwebui_chat_config.test",
				ImportState:       true,
				ImportStateId:     "chat",
				ImportStateVerify: true,
			},
		},
	})
}

func TestChatConfigResource_RejectsRetentionOutsideTheServersBounds(t *testing.T) {
	server := newFakeChatConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_chat_config" "test" {
  enable_context_compaction               = true
  context_compaction_token_threshold      = 40000
  context_compaction_retention_percentage = 90
  context_compaction_prompt_template      = "Summarise."
}
`, testChatConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`must be between 10`),
			},
		},
	})
}

func TestChatConfigResource_RejectsAThresholdBelowOne(t *testing.T) {
	server := newFakeChatConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_chat_config" "test" {
  enable_context_compaction          = true
  context_compaction_token_threshold = 0
  context_compaction_prompt_template = "Summarise."
}
`, testChatConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`at least 1`),
			},
		},
	})
}
