package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

// terminalServersState is a stand-in for the single config key Open WebUI keeps
// the list under. The handler answers a GET with the list and replaces it on a
// POST, which is what makes a read-modify-write cycle observable in a test.
type terminalServersState struct {
	connections []any
	writes      int
}

func newTerminalServersHandler(t *testing.T, initial string) (*terminalServersState, http.HandlerFunc) {
	t.Helper()

	state := &terminalServersState{}
	if initial != "" {
		if err := json.Unmarshal([]byte(initial), &state.connections); err != nil {
			t.Fatalf("decode the initial connections: %v", err)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			var payload struct {
				Connections []any `json:"TERMINAL_SERVER_CONNECTIONS"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode the write payload: %v", err)
			}
			state.connections = payload.Connections
			state.writes++
		}

		if err := json.NewEncoder(w).Encode(map[string]any{"TERMINAL_SERVER_CONNECTIONS": state.connections}); err != nil {
			t.Fatalf("encode the response: %v", err)
		}
	}

	return state, handler
}

func TestAddTerminalServerConnectionSplicesIntoAnEmptyList(t *testing.T) {
	state, handler := newTerminalServersHandler(t, "")
	c := newRecordingClient(t, handler)

	path := "/openapi.json"
	stored, err := c.AddTerminalServerConnection(context.Background(), TerminalServerConnection{
		ID:   "shell",
		URL:  "http://terminals.invalid",
		Path: &path,
	})
	if err != nil {
		t.Fatalf("AddTerminalServerConnection: %v", err)
	}

	if stored.ID != "shell" || stored.URL != "http://terminals.invalid" {
		t.Fatalf("expected the stored connection back, got %+v", stored)
	}
	if len(state.connections) != 1 {
		t.Fatalf("expected one connection stored, got %v", state.connections)
	}
}

// Open WebUI replaces the whole list on every write, so a connection this
// provider does not manage has to pass through untouched, unknown keys included.
func TestAddTerminalServerConnectionKeepsSiblings(t *testing.T) {
	state, handler := newTerminalServersHandler(t, `[{"id":"legacy","url":"http://legacy.invalid","surprise":{"nested":true}}]`)
	c := newRecordingClient(t, handler)

	if _, err := c.AddTerminalServerConnection(context.Background(), TerminalServerConnection{
		ID:  "shell",
		URL: "http://terminals.invalid",
	}); err != nil {
		t.Fatalf("AddTerminalServerConnection: %v", err)
	}

	if len(state.connections) != 2 {
		t.Fatalf("expected two connections stored, got %v", state.connections)
	}

	legacy, ok := state.connections[0].(map[string]any)
	if !ok {
		t.Fatalf("expected the first connection to be an object, got %T", state.connections[0])
	}
	if _, ok := legacy["surprise"]; !ok {
		t.Fatalf("expected the sibling's unknown key to survive, got %v", legacy)
	}
}

func TestAddTerminalServerConnectionRefusesADuplicateID(t *testing.T) {
	state, handler := newTerminalServersHandler(t, `[{"id":"shell","url":"http://terminals.invalid"}]`)
	c := newRecordingClient(t, handler)

	_, err := c.AddTerminalServerConnection(context.Background(), TerminalServerConnection{ID: "shell", URL: "http://other.invalid"})
	if !errors.Is(err, ErrTerminalServerExists) {
		t.Fatalf("expected ErrTerminalServerExists, got %v", err)
	}
	if state.writes != 0 {
		t.Fatalf("expected no write on a duplicate id, got %d", state.writes)
	}
}

func TestUpdateTerminalServerConnectionReplacesOneEntry(t *testing.T) {
	state, handler := newTerminalServersHandler(t, `[{"id":"first","url":"http://first.invalid"},{"id":"shell","url":"http://old.invalid"}]`)
	c := newRecordingClient(t, handler)

	stored, err := c.UpdateTerminalServerConnection(context.Background(), TerminalServerConnection{ID: "shell", URL: "http://new.invalid"})
	if err != nil {
		t.Fatalf("UpdateTerminalServerConnection: %v", err)
	}
	if stored.URL != "http://new.invalid" {
		t.Fatalf("expected the new URL back, got %+v", stored)
	}

	first, ok := state.connections[0].(map[string]any)
	if !ok || first["url"] != "http://first.invalid" {
		t.Fatalf("expected the sibling to keep its URL, got %v", state.connections[0])
	}
}

func TestUpdateTerminalServerConnectionReportsAMissingEntry(t *testing.T) {
	_, handler := newTerminalServersHandler(t, `[{"id":"other","url":"http://other.invalid"}]`)
	c := newRecordingClient(t, handler)

	_, err := c.UpdateTerminalServerConnection(context.Background(), TerminalServerConnection{ID: "shell", URL: "http://new.invalid"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteTerminalServerConnectionRemovesOnlyItsEntry(t *testing.T) {
	state, handler := newTerminalServersHandler(t, `[{"id":"first","url":"http://first.invalid"},{"id":"shell","url":"http://terminals.invalid"}]`)
	c := newRecordingClient(t, handler)

	if err := c.DeleteTerminalServerConnection(context.Background(), "shell"); err != nil {
		t.Fatalf("DeleteTerminalServerConnection: %v", err)
	}

	if len(state.connections) != 1 {
		t.Fatalf("expected one connection left, got %v", state.connections)
	}
	first, ok := state.connections[0].(map[string]any)
	if !ok || first["id"] != "first" {
		t.Fatalf("expected the sibling to survive, got %v", state.connections[0])
	}
}

// A connection that is already gone is not an error, so a destroy after an
// out-of-band delete converges.
func TestDeleteTerminalServerConnectionIsQuietWhenAlreadyGone(t *testing.T) {
	state, handler := newTerminalServersHandler(t, `[{"id":"first","url":"http://first.invalid"}]`)
	c := newRecordingClient(t, handler)

	if err := c.DeleteTerminalServerConnection(context.Background(), "shell"); err != nil {
		t.Fatalf("DeleteTerminalServerConnection: %v", err)
	}
	if state.writes != 0 {
		t.Fatalf("expected no write when there is nothing to remove, got %d", state.writes)
	}
}

func TestGetTerminalServerConnectionReportsAMissingEntry(t *testing.T) {
	_, handler := newTerminalServersHandler(t, `[{"id":"other","url":"http://other.invalid"}]`)
	c := newRecordingClient(t, handler)

	_, err := c.GetTerminalServerConnection(context.Background(), "shell")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// An instance that has never stored a list answers with a null, which reads as
// an empty list rather than an error.
func TestListTerminalServerConnectionsAcceptsANullList(t *testing.T) {
	c := newRecordingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS":null}`))
	})

	connections, err := c.ListTerminalServerConnections(context.Background())
	if err != nil {
		t.Fatalf("ListTerminalServerConnections: %v", err)
	}
	if len(connections) != 0 {
		t.Fatalf("expected an empty list, got %v", connections)
	}
}

// An unset optional field is left out of the request, because Open WebUI applies
// its own default only when the key is absent.
func TestAddTerminalServerConnectionOmitsUnsetFields(t *testing.T) {
	var body map[string]any
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell","url":"http://terminals.invalid","path":"/openapi.json","auth_type":"bearer","enabled":true}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"TERMINAL_SERVER_CONNECTIONS":[]}`))
	})

	stored, err := c.AddTerminalServerConnection(context.Background(), TerminalServerConnection{ID: "shell", URL: "http://terminals.invalid"})
	if err != nil {
		t.Fatalf("AddTerminalServerConnection: %v", err)
	}

	sent, ok := body["TERMINAL_SERVER_CONNECTIONS"].([]any)
	if !ok || len(sent) != 1 {
		t.Fatalf("expected one connection in the request, got %v", body)
	}
	entry, ok := sent[0].(map[string]any)
	if !ok {
		t.Fatalf("expected the connection to be an object, got %T", sent[0])
	}
	for _, field := range []string{"path", "auth_type", "enabled", "name", "key", "config", "server_type", "policy_id"} {
		if _, present := entry[field]; present {
			t.Fatalf("expected %s to be left out of the request, got %v", field, entry)
		}
	}

	if stored.Path == nil || *stored.Path != "/openapi.json" {
		t.Fatalf("expected the server default path back, got %+v", stored)
	}
	if stored.Enabled == nil || !*stored.Enabled {
		t.Fatalf("expected the server default enabled back, got %+v", stored)
	}
}
