package client

import (
	"context"
	"net/http"
)

// DefaultUserPermissions is the whole user.permissions blob, the permissions
// every user without a group carries. All six sections are required: the
// UserPermissions model in backend/open_webui/routers/users.py gives none of
// them a default, so a partial write is a 422.
//
// The values are booleans. The maps are map[string]any so that a key Open WebUI
// later gives a different type still decodes, and the provider can report it.
type DefaultUserPermissions struct {
	Workspace    map[string]any `json:"workspace"`
	Sharing      map[string]any `json:"sharing"`
	AccessGrants map[string]any `json:"access_grants"`
	Chat         map[string]any `json:"chat"`
	Features     map[string]any `json:"features"`
	Settings     map[string]any `json:"settings"`
}

// GetDefaultUserPermissions retrieves the default user permissions. Open WebUI
// fills every absent sub-key from its Pydantic model, so the response always
// carries the full key set.
func (c *Client) GetDefaultUserPermissions(ctx context.Context) (*DefaultUserPermissions, error) {
	var resp DefaultUserPermissions
	if err := c.do(ctx, http.MethodGet, "users/default/permissions", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetDefaultUserPermissions replaces the whole permissions blob and returns what
// Open WebUI stored.
func (c *Client) SetDefaultUserPermissions(ctx context.Context, perms DefaultUserPermissions) (*DefaultUserPermissions, error) {
	var resp DefaultUserPermissions
	if err := c.do(ctx, http.MethodPost, "users/default/permissions", nil, perms, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
