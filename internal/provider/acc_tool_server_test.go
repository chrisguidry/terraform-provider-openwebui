package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// toolServerConnectionsRoute is the configs route these tests drive directly,
// so that a connection can exist before Terraform knows about it.
const toolServerConnectionsRoute = "/api/v1/configs/tool_servers"

func testAccToolServerRequest(t *testing.T, method string, payload any) map[string]any {
	t.Helper()

	endpoint := strings.TrimRight(os.Getenv("OPENWEBUI_ENDPOINT"), "/")

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode tool servers request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, endpoint+toolServerConnectionsRoute, body)
	if err != nil {
		t.Fatalf("build tool servers request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENWEBUI_TOKEN"))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform tool servers request: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read tool servers response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("tool servers request returned %d: %s", resp.StatusCode, raw)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode tool servers response: %v", err)
	}

	return decoded
}

func testAccToolServerConnections(t *testing.T) []any {
	t.Helper()

	decoded := testAccToolServerRequest(t, http.MethodGet, nil)
	connections, _ := decoded["TOOL_SERVER_CONNECTIONS"].([]any)

	return connections
}

func testAccToolServerFind(t *testing.T, serverID string) map[string]any {
	t.Helper()

	for _, item := range testAccToolServerConnections(t) {
		connection, ok := item.(map[string]any)
		if !ok {
			continue
		}

		info, _ := connection["info"].(map[string]any)
		if id, _ := info["id"].(string); id == serverID {
			return connection
		}
	}

	return nil
}

// testAccToolServerSeed registers a connection the way a hand-made one arrives,
// bypassing the provider, and removes it again when the test ends.
func testAccToolServerSeed(t *testing.T, connection map[string]any) {
	t.Helper()

	info, _ := connection["info"].(map[string]any)
	serverID, _ := info["id"].(string)

	connections := append(testAccToolServerConnections(t), connection)
	testAccToolServerRequest(t, http.MethodPost, map[string]any{"TOOL_SERVER_CONNECTIONS": connections})

	t.Cleanup(func() {
		remaining := []any{}
		for _, item := range testAccToolServerConnections(t) {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}

			entryInfo, _ := entry["info"].(map[string]any)
			if id, _ := entryInfo["id"].(string); id == serverID {
				continue
			}

			remaining = append(remaining, entry)
		}

		testAccToolServerRequest(t, http.MethodPost, map[string]any{"TOOL_SERVER_CONNECTIONS": remaining})
	})
}

// A connection registered outside Terraform must import and then plan clean.
// This is how the tool servers already running on an instance are adopted.
func TestAccToolServerImportsAnExistingConnection(t *testing.T) {
	const serverID = "tf-acc-import"
	const serverURL = "https://tools.example/imported"

	config := fmt.Sprintf(`%s
resource "openwebui_tool_server" "imported" {
  server_id = %q
  url       = %q
  path      = "openapi.json"
  type      = "openapi"
  auth_type = "none"
  enabled   = true
}
`, testAccProviderConfig(), serverID, serverURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccToolServerSeed(t, map[string]any{
				"url":       serverURL,
				"path":      "openapi.json",
				"type":      "openapi",
				"auth_type": "none",
				"key":       nil,
				"config":    map[string]any{"enable": true},
				"info":      map[string]any{"id": serverID},
			})
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       "openwebui_tool_server.imported",
				ImportState:        true,
				ImportStateId:      serverID,
				ImportStatePersist: true,
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// A connection with no info.id is imported by list position, and the identity
// is pinned on the write that follows.
func TestAccToolServerImportsAConnectionByPosition(t *testing.T) {
	const serverID = "tf-acc-position"
	const serverURL = "https://tools.example/position"

	config := fmt.Sprintf(`%s
resource "openwebui_tool_server" "adopted" {
  server_id = %q
  url       = %q
  path      = "openapi.json"
  type      = "openapi"
  auth_type = "none"
  enabled   = true
  name      = "Adopted"
}
`, testAccProviderConfig(), serverID, serverURL)

	var position int

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			position = len(testAccToolServerConnections(t))
			testAccToolServerSeed(t, map[string]any{
				"url":       serverURL,
				"path":      "openapi.json",
				"type":      "openapi",
				"auth_type": "none",
				"key":       nil,
				"config":    map[string]any{"enable": true},
			})
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:       config,
				ResourceName: "openwebui_tool_server.adopted",
				ImportState:  true,
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%d", serverID, position), nil
				},
				ImportStatePersist: true,
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool_server.adopted", "server_id", serverID),
					func(_ *terraform.State) error {
						if testAccToolServerFind(t, serverID) == nil {
							return fmt.Errorf("expected the adopted connection to carry info.id %q", serverID)
						}

						return nil
					},
				),
			},
		},
	})
}

// The plainest case: an OpenAPI server with no authentication.
func TestAccToolServerOpenAPI(t *testing.T) {
	config := fmt.Sprintf(`%s
resource "openwebui_tool_server" "weather" {
  server_id = "tf-acc-weather"
  url       = "https://tools.example/weather"
  path      = "openapi.json"
  auth_type = "none"
  enabled   = true
}
`, testAccProviderConfig())

	updated := fmt.Sprintf(`%s
resource "openwebui_tool_server" "weather" {
  server_id = "tf-acc-weather"
  url       = "https://tools.example/weather-v2"
  path      = "openapi.json"
  auth_type = "none"
  enabled   = false
}
`, testAccProviderConfig())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "id", "tf-acc-weather"),
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "type", "openapi"),
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "enabled", "true"),
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "public_read", "false"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "url", "https://tools.example/weather-v2"),
					resource.TestCheckResourceAttr("openwebui_tool_server.weather", "enabled", "false"),
				),
			},
			{
				Config:            updated,
				ResourceName:      "openwebui_tool_server.weather",
				ImportState:       true,
				ImportStateId:     "tf-acc-weather",
				ImportStateVerify: true,
			},
		},
	})
}

// Two connections created in one apply exercise the client lock. Terraform
// runs them concurrently, and the write route replaces the whole list, so a
// missing entry here means the read-modify-write is not serialised.
func TestAccToolServerPairSurvivesOneApply(t *testing.T) {
	config := fmt.Sprintf(`%s
resource "openwebui_tool_server" "first" {
  server_id = "tf-acc-pair-first"
  url       = "https://tools.example/first"
  path      = "openapi.json"
  auth_type = "none"
}

resource "openwebui_tool_server" "second" {
  server_id = "tf-acc-pair-second"
  url       = "https://tools.example/second"
  path      = "openapi.json"
  auth_type = "none"
}
`, testAccProviderConfig())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					func(_ *terraform.State) error {
						for _, serverID := range []string{"tf-acc-pair-first", "tf-acc-pair-second"} {
							if testAccToolServerFind(t, serverID) == nil {
								return fmt.Errorf("expected %s to survive the apply", serverID)
							}
						}

						return nil
					},
				),
			},
		},
	})
}

// Editing one connection must leave every field of its neighbours alone,
// including the encrypted OAuth registration Terraform cannot re-create.
func TestAccToolServerEditLeavesSiblingInfoIntact(t *testing.T) {
	const siblingID = "tf-acc-sibling"
	const blob = "gAAAAABacceptancetestciphertext"

	config := fmt.Sprintf(`%s
resource "openwebui_tool_server" "edited" {
  server_id = "tf-acc-edited"
  url       = "https://tools.example/edited"
  path      = "openapi.json"
  auth_type = "none"
}
`, testAccProviderConfig())

	updated := strings.Replace(config, "https://tools.example/edited", "https://tools.example/edited-v2", 1)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccToolServerSeed(t, map[string]any{
				"url":       "https://tools.example/sibling",
				"path":      "",
				"type":      "mcp",
				"auth_type": "oauth_2.1",
				"key":       nil,
				"config":    map[string]any{"enable": true},
				"info": map[string]any{
					"id":                siblingID,
					"oauth_client_info": blob,
					"name":              "Sibling",
				},
			})
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: updated,
				Check: func(_ *terraform.State) error {
					sibling := testAccToolServerFind(t, siblingID)
					if sibling == nil {
						return fmt.Errorf("expected the sibling connection to survive the edit")
					}

					info, _ := sibling["info"].(map[string]any)
					if info["oauth_client_info"] != blob {
						return fmt.Errorf("expected the sibling to keep its oauth_client_info, got %v", info)
					}
					if info["name"] != "Sibling" {
						return fmt.Errorf("expected the sibling to keep its info.name, got %v", info)
					}

					return nil
				},
			},
		},
	})
}
