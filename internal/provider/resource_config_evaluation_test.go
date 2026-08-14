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

// fakeEvaluationConfigAPI serves the arena evaluation routes, applying each
// field only when the request carries it.
type fakeEvaluationConfigAPI struct {
	mu      sync.Mutex
	enabled bool
	models  []any
}

func newFakeEvaluationConfigAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeEvaluationConfigAPI{models: []any{}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeEvaluationConfigAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if enabled, ok := form["ENABLE_EVALUATION_ARENA_MODELS"].(bool); ok {
			a.enabled = enabled
		}
		if models, ok := form["EVALUATION_ARENA_MODELS"].([]any); ok {
			a.models = models
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ENABLE_EVALUATION_ARENA_MODELS": a.enabled,
		"EVALUATION_ARENA_MODELS":        a.models,
	})
}

func testEvaluationConfigProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

func TestEvaluationConfigResource_WritesArenaModels(t *testing.T) {
	server := newFakeEvaluationConfigAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_evaluation_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models_json = jsonencode([
    {
      id   = "arena"
      name = "Arena"
      meta = { profile_image_url = "/favicon.png" }
    }
  ])
}
`, testEvaluationConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "id", "evaluation"),
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "enable_evaluation_arena_models", "true"),
					resource.TestMatchResourceAttr("openwebui_evaluation_config.test", "evaluation_arena_models_json", regexp.MustCompile(`"id":\s*"arena"`)),
				),
			},
			{
				ResourceName:      "openwebui_evaluation_config.test",
				ImportState:       true,
				ImportStateId:     "evaluation",
				ImportStateVerify: true,
			},
		},
	})
}

// The route applies a field only when the request carries it, so a
// configuration that names one setting leaves the other one alone.
func TestEvaluationConfigResource_PartialWriteKeepsTheOtherSetting(t *testing.T) {
	server := newFakeEvaluationConfigAPI(t)

	enabled := fmt.Sprintf(`%s
resource "openwebui_evaluation_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models_json   = jsonencode([{ id = "arena", name = "Arena", meta = {} }])
}
`, testEvaluationConfigProviderConfig(server.URL))

	listOnly := fmt.Sprintf(`%s
resource "openwebui_evaluation_config" "test" {
  enable_evaluation_arena_models = true
  evaluation_arena_models_json   = jsonencode([])
}
`, testEvaluationConfigProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: enabled},
			{
				Config: listOnly,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "evaluation_arena_models_json", "[]"),
					resource.TestCheckResourceAttr("openwebui_evaluation_config.test", "enable_evaluation_arena_models", "true"),
				),
			},
		},
	})
}
