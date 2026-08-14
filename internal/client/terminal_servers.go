package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// ErrTerminalServerExists reports that a terminal server connection with the
// requested id is already registered.
var ErrTerminalServerExists = errors.New("openwebui: terminal server connection already exists")

// terminalServerWriteMutex serialises the read-modify-write cycle that every
// single-connection change needs, because POST /configs/terminal_servers
// replaces the whole list. Terraform runs instances of the same resource type
// concurrently, so two connections applied in one run would otherwise lose one
// of the two writes. The lock is package level rather than per client so that a
// process holding more than one client cannot race with itself. It does not
// reach across processes: two concurrent terraform apply runs against the same
// Open WebUI can still lose a connection.
var terminalServerWriteMutex sync.Mutex

// TerminalServerConnection is one entry of the TERMINAL_SERVER_CONNECTIONS
// list. The nullable fields are pointers and carry omitempty, because Open WebUI
// applies its own default only when the key is absent: an explicit null stays
// null. `policy` and `lifecycle` are absent by design, as the write handler
// drops both from every connection it stores.
type TerminalServerConnection struct {
	ID         string         `json:"id"`
	Name       *string        `json:"name,omitempty"`
	Enabled    *bool          `json:"enabled,omitempty"`
	URL        string         `json:"url"`
	Path       *string        `json:"path,omitempty"`
	Key        *string        `json:"key,omitempty"`
	AuthType   *string        `json:"auth_type,omitempty"`
	Config     map[string]any `json:"config,omitempty"`
	ServerType *string        `json:"server_type,omitempty"`
	PolicyID   *string        `json:"policy_id,omitempty"`
}

// terminalServersEnvelope carries the list as raw JSON so that connections this
// provider does not manage survive a write untouched.
type terminalServersEnvelope struct {
	Connections []json.RawMessage `json:"TERMINAL_SERVER_CONNECTIONS"`
}

// ListTerminalServerConnections returns every registered terminal server.
func (c *Client) ListTerminalServerConnections(ctx context.Context) ([]TerminalServerConnection, error) {
	raw, err := c.listTerminalServerEntries(ctx)
	if err != nil {
		return nil, err
	}

	connections := make([]TerminalServerConnection, 0, len(raw))
	for i, entry := range raw {
		var conn TerminalServerConnection
		if err := json.Unmarshal(entry, &conn); err != nil {
			return nil, fmt.Errorf("decode terminal server connection %d: %w", i, err)
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// GetTerminalServerConnection returns one terminal server by id. It returns
// ErrNotFound when no connection carries that id.
func (c *Client) GetTerminalServerConnection(ctx context.Context, id string) (*TerminalServerConnection, error) {
	connections, err := c.ListTerminalServerConnections(ctx)
	if err != nil {
		return nil, err
	}

	for _, conn := range connections {
		if conn.ID == id {
			found := conn
			return &found, nil
		}
	}

	return nil, ErrNotFound
}

// AddTerminalServerConnection appends a connection to the list. It returns
// ErrTerminalServerExists when the id is taken, so that Terraform reports a
// collision instead of adopting a connection it did not create.
func (c *Client) AddTerminalServerConnection(ctx context.Context, conn TerminalServerConnection) (*TerminalServerConnection, error) {
	terminalServerWriteMutex.Lock()
	defer terminalServerWriteMutex.Unlock()

	entries, err := c.listTerminalServerEntries(ctx)
	if err != nil {
		return nil, err
	}

	if index := indexOfTerminalServer(entries, conn.ID); index >= 0 {
		return nil, ErrTerminalServerExists
	}

	encoded, err := json.Marshal(conn)
	if err != nil {
		return nil, fmt.Errorf("encode terminal server connection: %w", err)
	}

	return c.writeTerminalServerEntries(ctx, append(entries, encoded), conn.ID)
}

// UpdateTerminalServerConnection replaces one connection in the list. It returns
// ErrNotFound when the id is gone.
func (c *Client) UpdateTerminalServerConnection(ctx context.Context, conn TerminalServerConnection) (*TerminalServerConnection, error) {
	terminalServerWriteMutex.Lock()
	defer terminalServerWriteMutex.Unlock()

	entries, err := c.listTerminalServerEntries(ctx)
	if err != nil {
		return nil, err
	}

	index := indexOfTerminalServer(entries, conn.ID)
	if index < 0 {
		return nil, ErrNotFound
	}

	encoded, err := json.Marshal(conn)
	if err != nil {
		return nil, fmt.Errorf("encode terminal server connection: %w", err)
	}

	entries[index] = encoded

	return c.writeTerminalServerEntries(ctx, entries, conn.ID)
}

// DeleteTerminalServerConnection removes one connection from the list. A
// connection that is already gone is not an error.
func (c *Client) DeleteTerminalServerConnection(ctx context.Context, id string) error {
	terminalServerWriteMutex.Lock()
	defer terminalServerWriteMutex.Unlock()

	entries, err := c.listTerminalServerEntries(ctx)
	if err != nil {
		return err
	}

	index := indexOfTerminalServer(entries, id)
	if index < 0 {
		return nil
	}

	remaining := make([]json.RawMessage, 0, len(entries)-1)
	remaining = append(remaining, entries[:index]...)
	remaining = append(remaining, entries[index+1:]...)

	_, err = c.writeTerminalServerEntries(ctx, remaining, "")

	return err
}

// listTerminalServerEntries reads the stored list. Open WebUI returns the raw
// config value, which is null on an instance that has never stored a list.
func (c *Client) listTerminalServerEntries(ctx context.Context) ([]json.RawMessage, error) {
	var resp terminalServersEnvelope
	if err := c.do(ctx, http.MethodGet, "configs/terminal_servers", nil, nil, &resp); err != nil {
		return nil, err
	}

	if resp.Connections == nil {
		return []json.RawMessage{}, nil
	}

	return resp.Connections, nil
}

// writeTerminalServerEntries writes the whole list back and returns the entry
// carrying id from the response. An empty id asks for no entry back.
func (c *Client) writeTerminalServerEntries(ctx context.Context, entries []json.RawMessage, id string) (*TerminalServerConnection, error) {
	payload := terminalServersEnvelope{Connections: entries}
	if payload.Connections == nil {
		payload.Connections = []json.RawMessage{}
	}

	var resp terminalServersEnvelope
	if err := c.do(ctx, http.MethodPost, "configs/terminal_servers", nil, payload, &resp); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, nil
	}

	index := indexOfTerminalServer(resp.Connections, id)
	if index < 0 {
		return nil, ErrNotFound
	}

	var stored TerminalServerConnection
	if err := json.Unmarshal(resp.Connections[index], &stored); err != nil {
		return nil, fmt.Errorf("decode terminal server connection: %w", err)
	}

	return &stored, nil
}

// indexOfTerminalServer finds a connection by id without decoding fields the
// caller does not manage.
func indexOfTerminalServer(entries []json.RawMessage, id string) int {
	for i, entry := range entries {
		var probe struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(entry, &probe); err != nil {
			continue
		}
		if probe.ID == id {
			return i
		}
	}

	return -1
}
