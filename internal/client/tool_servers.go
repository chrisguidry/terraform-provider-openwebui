package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

// toolServerConnectionsPath is the configs route that holds the whole list.
const toolServerConnectionsPath = "configs/tool_servers"

// ToolServerEntry is one entry of TOOL_SERVER_CONNECTIONS, held as the raw
// decoded object instead of a struct. Open WebUI declares ToolServerConnection
// with extra='allow' and stores every field it receives, so a fixed struct
// would drop each field the provider does not model. One of those fields is
// info.oauth_client_info, the encrypted registration an MCP server
// authenticates with, and losing it takes the server offline.
type ToolServerEntry map[string]any

// UnmarshalJSON decodes an entry with numbers kept as json.Number, so that a
// value the provider only carries through is written back with the digits it
// arrived with.
func (e *ToolServerEntry) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	*e = raw

	return nil
}

// toolServerConnectionsMu serialises the read-modify-write on the connection
// list. POST /api/v1/configs/tool_servers replaces the list as a whole, so two
// connections written at the same time lose one of the two writes. Terraform
// runs instances of one resource type concurrently, ten at a time by default,
// which makes that collision ordinary rather than rare. The lock covers a
// single process: two terraform runs against one Open WebUI instance can still
// lose a connection.
var toolServerConnectionsMu sync.Mutex

// toolServerEntriesEnvelope is the request and response shape of the tool
// servers config route.
type toolServerEntriesEnvelope struct {
	Connections []ToolServerEntry `json:"TOOL_SERVER_CONNECTIONS"`
}

// ToolServerLocator identifies one connection within the list.
//
// ID matches info.id, which the client supplies and Open WebUI never
// generates. URL and Path match a connection that carries no info.id at all,
// which is how a connection made by hand in the web UI arrives. The first
// write through this client gives such a connection an info.id and the
// fallback stops being used.
type ToolServerLocator struct {
	ID   string
	URL  string
	Path string
}

// ErrToolServerExists reports that a connection already holds the requested
// info.id.
var ErrToolServerExists = errors.New("openwebui: tool server connection already exists")

// ListToolServerConnections returns every stored tool server connection.
func (c *Client) ListToolServerConnections(ctx context.Context) ([]ToolServerEntry, error) {
	var resp toolServerEntriesEnvelope
	if err := c.do(ctx, http.MethodGet, toolServerConnectionsPath, nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Connections, nil
}

// GetToolServerConnection returns the connection the locator names.
func (c *Client) GetToolServerConnection(ctx context.Context, locator ToolServerLocator) (ToolServerEntry, error) {
	entries, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	index := findToolServerConnection(entries, locator)
	if index < 0 {
		return nil, ErrNotFound
	}

	return entries[index], nil
}

// GetToolServerConnectionAt returns the connection at a list position. Import
// uses it to adopt a connection that carries no info.id.
func (c *Client) GetToolServerConnectionAt(ctx context.Context, index int) (ToolServerEntry, error) {
	entries, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(entries) {
		return nil, ErrNotFound
	}

	return entries[index], nil
}

// CreateToolServerConnection appends a connection to the list and returns the
// stored entry. It fails with ErrToolServerExists when the list already holds
// the info.id in desired, so that a create never adopts a connection somebody
// else made.
func (c *Client) CreateToolServerConnection(ctx context.Context, desired ToolServerEntry) (ToolServerEntry, error) {
	toolServerConnectionsMu.Lock()
	defer toolServerConnectionsMu.Unlock()

	entries, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	id := toolServerEntryID(desired)
	if id != "" && findToolServerConnection(entries, ToolServerLocator{ID: id}) >= 0 {
		return nil, ErrToolServerExists
	}

	merged := mergeToolServerEntry(nil, desired)
	entries = append(entries, merged)

	return c.writeToolServerConnections(ctx, entries, ToolServerLocator{
		ID:   id,
		URL:  toolServerEntryString(merged, "url"),
		Path: toolServerEntryString(merged, "path"),
	}, merged)
}

// UpsertToolServerConnection merges desired over the stored connection the
// locator names and writes the whole list back. A connection the locator does
// not find is appended, so an entry deleted outside Terraform comes back.
func (c *Client) UpsertToolServerConnection(ctx context.Context, locator ToolServerLocator, desired ToolServerEntry) (ToolServerEntry, error) {
	toolServerConnectionsMu.Lock()
	defer toolServerConnectionsMu.Unlock()

	entries, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	var merged ToolServerEntry

	index := findToolServerConnection(entries, locator)
	if index >= 0 {
		merged = mergeToolServerEntry(entries[index], desired)
		entries[index] = merged
	} else {
		merged = mergeToolServerEntry(nil, desired)
		entries = append(entries, merged)
	}

	return c.writeToolServerConnections(ctx, entries, ToolServerLocator{
		ID:   toolServerEntryID(merged),
		URL:  toolServerEntryString(merged, "url"),
		Path: toolServerEntryString(merged, "path"),
	}, merged)
}

// DeleteToolServerConnection splices one connection out of the list. A
// connection the locator does not find is already gone, which is not an error.
func (c *Client) DeleteToolServerConnection(ctx context.Context, locator ToolServerLocator) error {
	toolServerConnectionsMu.Lock()
	defer toolServerConnectionsMu.Unlock()

	entries, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return err
	}

	index := findToolServerConnection(entries, locator)
	if index < 0 {
		return nil
	}

	remaining := make([]ToolServerEntry, 0, len(entries)-1)
	remaining = append(remaining, entries[:index]...)
	remaining = append(remaining, entries[index+1:]...)

	_, err = c.writeToolServerConnections(ctx, remaining, ToolServerLocator{}, nil)

	return err
}

// writeToolServerConnections posts the list and returns the stored form of one
// entry. Open WebUI echoes what it stored, so the response is the value the
// caller must put in state. The fallback covers a response body the server
// leaves empty.
func (c *Client) writeToolServerConnections(ctx context.Context, entries []ToolServerEntry, locator ToolServerLocator, fallback ToolServerEntry) (ToolServerEntry, error) {
	if entries == nil {
		entries = []ToolServerEntry{}
	}

	var resp toolServerEntriesEnvelope
	form := toolServerEntriesEnvelope{Connections: entries}
	if err := c.do(ctx, http.MethodPost, toolServerConnectionsPath, nil, form, &resp); err != nil {
		return nil, err
	}

	if index := findToolServerConnection(resp.Connections, locator); index >= 0 {
		return resp.Connections[index], nil
	}

	return fallback, nil
}

// SetToolServerConnectionsPreservingInfo replaces the whole connection list
// and carries each stored info object onto the connection that replaces it,
// matched on url and path.
//
// The list resource models no info, so a plain write would drop it and take
// every OAuth-authenticated MCP server on the instance offline. It runs under
// the same lock as the per-connection writes, so the two resources cannot lose
// each other's work inside one Terraform run.
func (c *Client) SetToolServerConnectionsPreservingInfo(ctx context.Context, connections []ToolServerConnection) (*ToolServersConfigForm, error) {
	toolServerConnectionsMu.Lock()
	defer toolServerConnectionsMu.Unlock()

	stored, err := c.ListToolServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	for i := range connections {
		if connections[i].Info != nil {
			continue
		}

		locator := ToolServerLocator{URL: connections[i].URL, Path: connections[i].Path}
		for _, entry := range stored {
			if toolServerEntryString(entry, "url") != locator.URL || toolServerEntryString(entry, "path") != locator.Path {
				continue
			}

			if info, ok := entry["info"].(map[string]any); ok {
				connections[i].Info = info
			}

			break
		}
	}

	var resp ToolServersConfigForm
	form := ToolServersConfigForm{Connections: connections}
	if err := c.do(ctx, http.MethodPost, toolServerConnectionsPath, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// findToolServerConnection returns the position of the connection the locator
// names, or -1.
func findToolServerConnection(entries []ToolServerEntry, locator ToolServerLocator) int {
	if locator.ID != "" {
		for i, entry := range entries {
			if toolServerEntryID(entry) == locator.ID {
				return i
			}
		}
	}

	if locator.URL == "" {
		return -1
	}

	for i, entry := range entries {
		if toolServerEntryID(entry) != "" {
			continue
		}
		if toolServerEntryString(entry, "url") == locator.URL && toolServerEntryString(entry, "path") == locator.Path {
			return i
		}
	}

	return -1
}

// mergeToolServerEntry writes desired over existing.
//
// A top-level field of desired replaces the stored value, including a nil,
// because Open WebUI declares auth_type, key and config as required fields
// that accept null. A field desired leaves out keeps its stored value.
//
// Inside info and config the merge runs per key: a key desired leaves out
// keeps its stored value, and a key desired sets to nil is removed. That is
// what carries info.oauth_client_info, the ciphertext only the server can
// read, through an edit of a sibling field.
func mergeToolServerEntry(existing, desired ToolServerEntry) ToolServerEntry {
	merged := ToolServerEntry{}
	for key, value := range existing {
		merged[key] = value
	}

	for key, value := range desired {
		switch key {
		case "info", "config":
			merged[key] = mergeToolServerSubObject(merged[key], value)
		default:
			merged[key] = value
		}
	}

	return merged
}

func mergeToolServerSubObject(existing, desired any) map[string]any {
	merged := map[string]any{}
	if stored, ok := existing.(map[string]any); ok {
		for key, value := range stored {
			merged[key] = value
		}
	}

	updates, ok := desired.(map[string]any)
	if !ok {
		return merged
	}

	for key, value := range updates {
		if value == nil {
			delete(merged, key)
			continue
		}
		merged[key] = value
	}

	return merged
}

// toolServerEntryID reads info.id, the identity Open WebUI builds tool IDs
// from as server:mcp:<id> and server:<id>.
func toolServerEntryID(entry ToolServerEntry) string {
	info, ok := entry["info"].(map[string]any)
	if !ok {
		return ""
	}

	id, _ := info["id"].(string)

	return id
}

func toolServerEntryString(entry ToolServerEntry, key string) string {
	value, _ := entry[key].(string)

	return value
}

// ToolServerAccessGrants converts the provider's nested access_control map
// into the flat grant list stored at config.access_grants.
func ToolServerAccessGrants(accessControl map[string]any) []any {
	grants := accessControlToGrants(accessControl)

	list := make([]any, 0, len(grants))
	for _, grant := range grants {
		list = append(list, map[string]any{
			"principal_type": grant.PrincipalType,
			"principal_id":   grant.PrincipalID,
			"permission":     grant.Permission,
		})
	}

	return list
}

// ToolServerAccessControl converts a stored config.access_grants list back
// into the provider's nested access_control map.
func ToolServerAccessControl(stored any) map[string]any {
	items, ok := stored.([]any)
	if !ok {
		return nil
	}

	grants := make([]accessGrant, 0, len(items))
	for _, item := range items {
		fields, ok := item.(map[string]any)
		if !ok {
			continue
		}

		principalType, _ := fields["principal_type"].(string)
		principalID, _ := fields["principal_id"].(string)
		permission, _ := fields["permission"].(string)

		grants = append(grants, accessGrant{
			PrincipalType: principalType,
			PrincipalID:   principalID,
			Permission:    permission,
		})
	}

	return grantsToAccessControl(grants)
}
