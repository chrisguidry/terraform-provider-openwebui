package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// toolServersFake serves GET and POST on the tool servers config route against
// an in-memory list, the way Open WebUI does: the GET returns the stored rows
// verbatim and the POST replaces the whole list.
type toolServersFake struct {
	mu        sync.Mutex
	entries   []map[string]any
	readDelay time.Duration
}

func (f *toolServersFake) handler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/configs/tool_servers" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			f.mu.Lock()
			entries := f.entries
			f.mu.Unlock()

			if f.readDelay > 0 {
				time.Sleep(f.readDelay)
			}

			writeToolServersResponse(t, w, entries)
		case http.MethodPost:
			var form struct {
				Connections []map[string]any `json:"TOOL_SERVER_CONNECTIONS"`
			}

			// Open WebUI stores the request as JSON, so the fake decodes
			// numbers as literals rather than as float64.
			decoder := json.NewDecoder(r.Body)
			decoder.UseNumber()
			if err := decoder.Decode(&form); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			f.mu.Lock()
			f.entries = form.Connections
			entries := f.entries
			f.mu.Unlock()

			writeToolServersResponse(t, w, entries)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (f *toolServersFake) stored() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.entries
}

func writeToolServersResponse(t *testing.T, w http.ResponseWriter, entries []map[string]any) {
	t.Helper()

	if entries == nil {
		entries = []map[string]any{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"TOOL_SERVER_CONNECTIONS": entries}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func newToolServersTestClient(t *testing.T, fake *toolServersFake) *Client {
	t.Helper()

	server := httptest.NewServer(fake.handler(t))
	t.Cleanup(server.Close)

	c, err := NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return c
}

func toolServerEntryFixture(id, url string) ToolServerEntry {
	return ToolServerEntry{
		"url":       url,
		"path":      "openapi.json",
		"type":      "openapi",
		"auth_type": "none",
		"key":       nil,
		"config":    map[string]any{"enable": true},
		"info":      map[string]any{"id": id},
	}
}

func TestCreateToolServerConnectionAppendsToEmptyList(t *testing.T) {
	fake := &toolServersFake{}
	c := newToolServersTestClient(t, fake)

	stored, err := c.CreateToolServerConnection(context.Background(), toolServerEntryFixture("weather", "https://weather.example"))
	if err != nil {
		t.Fatalf("CreateToolServerConnection: %v", err)
	}

	if toolServerEntryID(stored) != "weather" {
		t.Fatalf("expected the stored entry to carry info.id weather, got %v", stored)
	}

	if len(fake.stored()) != 1 {
		t.Fatalf("expected one stored connection, got %v", fake.stored())
	}
}

func TestCreateToolServerConnectionKeepsSiblingsIntact(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":       "https://paperless.example",
		"path":      "",
		"type":      "mcp",
		"auth_type": "oauth_2.1",
		"key":       nil,
		"config":    map[string]any{"enable": true},
		"info": map[string]any{
			"id":                "paperless",
			"oauth_client_info": "gAAAAABlciphertext",
		},
		"spec_type": "url",
	}}}
	c := newToolServersTestClient(t, fake)

	if _, err := c.CreateToolServerConnection(context.Background(), toolServerEntryFixture("weather", "https://weather.example")); err != nil {
		t.Fatalf("CreateToolServerConnection: %v", err)
	}

	entries := fake.stored()
	if len(entries) != 2 {
		t.Fatalf("expected two stored connections, got %v", entries)
	}

	sibling := entries[0]
	info, ok := sibling["info"].(map[string]any)
	if !ok {
		t.Fatalf("expected the sibling to keep its info object, got %v", sibling)
	}
	if info["oauth_client_info"] != "gAAAAABlciphertext" {
		t.Fatalf("expected the sibling to keep its oauth_client_info, got %v", info)
	}
	if sibling["spec_type"] != "url" {
		t.Fatalf("expected the sibling to keep its unmodelled spec_type field, got %v", sibling)
	}
}

func TestCreateToolServerConnectionRejectsADuplicateID(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":  "https://weather.example",
		"info": map[string]any{"id": "weather"},
	}}}
	c := newToolServersTestClient(t, fake)

	_, err := c.CreateToolServerConnection(context.Background(), toolServerEntryFixture("weather", "https://weather.example"))
	if !errors.Is(err, ErrToolServerExists) {
		t.Fatalf("expected ErrToolServerExists, got %v", err)
	}
}

// An edit of one field must not disturb info.oauth_client_info, the encrypted
// OAuth registration the server owns and Terraform cannot re-create.
func TestUpsertToolServerConnectionPreservesUnknownInfoKeys(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":       "https://paperless.example",
		"path":      "",
		"type":      "mcp",
		"auth_type": "oauth_2.1",
		"key":       nil,
		"config":    map[string]any{"enable": true, "retries": json.Number("3")},
		"info": map[string]any{
			"id":                "paperless",
			"oauth_client_info": "gAAAAABlciphertext",
			"name":              "Paperless",
		},
	}}}
	c := newToolServersTestClient(t, fake)

	desired := ToolServerEntry{
		"url":       "https://paperless.mcp.example",
		"path":      "",
		"type":      "mcp",
		"auth_type": "oauth_2.1",
		"key":       nil,
		"config":    map[string]any{"enable": true},
		"info":      map[string]any{"id": "paperless"},
	}

	stored, err := c.UpsertToolServerConnection(context.Background(), ToolServerLocator{ID: "paperless"}, desired)
	if err != nil {
		t.Fatalf("UpsertToolServerConnection: %v", err)
	}

	if stored["url"] != "https://paperless.mcp.example" {
		t.Fatalf("expected the edited url, got %v", stored["url"])
	}

	info, ok := stored["info"].(map[string]any)
	if !ok {
		t.Fatalf("expected an info object, got %v", stored["info"])
	}
	if info["oauth_client_info"] != "gAAAAABlciphertext" {
		t.Fatalf("expected oauth_client_info to survive the edit, got %v", info)
	}
	if info["name"] != "Paperless" {
		t.Fatalf("expected the unmodelled info.name to survive the edit, got %v", info)
	}

	config, ok := stored["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected a config object, got %v", stored["config"])
	}
	if fmt.Sprint(config["retries"]) != "3" {
		t.Fatalf("expected the unmodelled config.retries to survive the edit, got %v", config)
	}
}

func TestUpsertToolServerConnectionClearsInfoKeysSetToNil(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":  "https://weather.example",
		"info": map[string]any{"id": "weather", "oauth_scope": "read"},
	}}}
	c := newToolServersTestClient(t, fake)

	desired := toolServerEntryFixture("weather", "https://weather.example")
	desired["info"] = map[string]any{"id": "weather", "oauth_scope": nil}

	stored, err := c.UpsertToolServerConnection(context.Background(), ToolServerLocator{ID: "weather"}, desired)
	if err != nil {
		t.Fatalf("UpsertToolServerConnection: %v", err)
	}

	info, _ := stored["info"].(map[string]any)
	if _, present := info["oauth_scope"]; present {
		t.Fatalf("expected oauth_scope to be removed, got %v", info)
	}
}

// A connection made by hand in the web UI can carry no info.id at all. The
// locator falls back to url and path, and the write pins the identity.
func TestUpsertToolServerConnectionAdoptsAnEntryWithoutAnID(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":       "https://weather.example",
		"path":      "openapi.json",
		"type":      "openapi",
		"auth_type": "none",
		"key":       nil,
		"config":    map[string]any{"enable": true},
	}}}
	c := newToolServersTestClient(t, fake)

	locator := ToolServerLocator{ID: "weather", URL: "https://weather.example", Path: "openapi.json"}
	if _, err := c.UpsertToolServerConnection(context.Background(), locator, toolServerEntryFixture("weather", "https://weather.example")); err != nil {
		t.Fatalf("UpsertToolServerConnection: %v", err)
	}

	entries := fake.stored()
	if len(entries) != 1 {
		t.Fatalf("expected the hand-made connection to be adopted rather than duplicated, got %v", entries)
	}

	info, _ := entries[0]["info"].(map[string]any)
	if info["id"] != "weather" {
		t.Fatalf("expected the write to pin info.id, got %v", entries[0])
	}
}

func TestDeleteToolServerConnectionSplicesByID(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{
		{"url": "https://a.example", "info": map[string]any{"id": "a"}},
		{"url": "https://b.example", "info": map[string]any{"id": "b"}},
		{"url": "https://c.example", "info": map[string]any{"id": "c"}},
	}}
	c := newToolServersTestClient(t, fake)

	if err := c.DeleteToolServerConnection(context.Background(), ToolServerLocator{ID: "b"}); err != nil {
		t.Fatalf("DeleteToolServerConnection: %v", err)
	}

	entries := fake.stored()
	if len(entries) != 2 {
		t.Fatalf("expected two remaining connections, got %v", entries)
	}

	for _, entry := range entries {
		info, _ := entry["info"].(map[string]any)
		if info["id"] == "b" {
			t.Fatalf("expected connection b to be gone, got %v", entries)
		}
	}
}

func TestDeleteToolServerConnectionIgnoresAMissingEntry(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{
		{"url": "https://a.example", "info": map[string]any{"id": "a"}},
	}}
	c := newToolServersTestClient(t, fake)

	if err := c.DeleteToolServerConnection(context.Background(), ToolServerLocator{ID: "gone"}); err != nil {
		t.Fatalf("DeleteToolServerConnection: %v", err)
	}

	if len(fake.stored()) != 1 {
		t.Fatalf("expected the list to be untouched, got %v", fake.stored())
	}
}

// The write is a read-modify-write against a route that replaces the whole
// list. Without the client mutex, connections written at the same time lose
// each other.
func TestToolServerWritesSerialise(t *testing.T) {
	fake := &toolServersFake{readDelay: 5 * time.Millisecond}
	c := newToolServersTestClient(t, fake)

	const count = 8

	var wg sync.WaitGroup
	errs := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("server-%d", i)
			_, err := c.CreateToolServerConnection(context.Background(), toolServerEntryFixture(id, "https://"+id+".example"))
			errs <- err
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("CreateToolServerConnection: %v", err)
		}
	}

	if len(fake.stored()) != count {
		t.Fatalf("expected %d stored connections, got %d", count, len(fake.stored()))
	}
}

func TestToolServerNumbersRoundTripUnchanged(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":    "https://weather.example",
		"info":   map[string]any{"id": "weather"},
		"config": map[string]any{"enable": true, "timeout": json.Number("9007199254740993")},
	}}}
	c := newToolServersTestClient(t, fake)

	desired := toolServerEntryFixture("weather", "https://weather.example")
	if _, err := c.UpsertToolServerConnection(context.Background(), ToolServerLocator{ID: "weather"}, desired); err != nil {
		t.Fatalf("UpsertToolServerConnection: %v", err)
	}

	config, _ := fake.stored()[0]["config"].(map[string]any)
	if fmt.Sprint(config["timeout"]) != "9007199254740993" {
		t.Fatalf("expected the stored integer to survive the round trip, got %v", config["timeout"])
	}
}

// The whole-list resource models no info, so the write has to read the stored
// one back or every OAuth-authenticated MCP server loses its registration.
func TestSetToolServerConnectionsPreservingInfoCarriesTheStoredInfo(t *testing.T) {
	fake := &toolServersFake{entries: []map[string]any{{
		"url":       "https://paperless.example",
		"path":      "",
		"type":      "mcp",
		"auth_type": "oauth_2.1",
		"key":       nil,
		"config":    map[string]any{"enable": true},
		"info": map[string]any{
			"id":                "paperless",
			"oauth_client_info": "gAAAAABlciphertext",
		},
	}}}
	c := newToolServersTestClient(t, fake)

	authType := "oauth_2.1"
	connections := []ToolServerConnection{{
		URL:      "https://paperless.example",
		Path:     "",
		AuthType: &authType,
		Config:   map[string]any{"enable": true},
	}}

	if _, err := c.SetToolServerConnectionsPreservingInfo(context.Background(), connections); err != nil {
		t.Fatalf("SetToolServerConnectionsPreservingInfo: %v", err)
	}

	info, _ := fake.stored()[0]["info"].(map[string]any)
	if info["oauth_client_info"] != "gAAAAABlciphertext" {
		t.Fatalf("expected the stored oauth_client_info to survive the write, got %v", fake.stored()[0])
	}
}

func TestToolServerAccessGrantsRoundTrip(t *testing.T) {
	accessControl := map[string]any{
		"read":         map[string]any{"group_ids": []string{"group-a"}, "user_ids": []string{}},
		"write":        map[string]any{"group_ids": []string{"group-b"}, "user_ids": []string{}},
		"public_read":  true,
		"public_write": false,
	}

	grants := ToolServerAccessGrants(accessControl)
	if len(grants) != 3 {
		t.Fatalf("expected three grants, got %v", grants)
	}

	restored := ToolServerAccessControl(grants)
	if !publicFlag(restored, "public_read") {
		t.Fatalf("expected public_read to survive, got %v", restored)
	}
	if publicFlag(restored, "public_write") {
		t.Fatalf("expected public_write to stay false, got %v", restored)
	}

	read, _ := restored["read"].(map[string]any)
	readGroups, _ := read["group_ids"].([]string)
	if len(readGroups) != 1 || readGroups[0] != "group-a" {
		t.Fatalf("expected the read group to survive, got %v", restored)
	}
}

func publicFlag(accessControl map[string]any, key string) bool {
	value, _ := accessControl[key].(bool)

	return value
}
