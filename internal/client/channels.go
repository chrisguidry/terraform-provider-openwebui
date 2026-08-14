package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ChannelForm is the payload for creating and updating a standard channel. The
// update handler assigns every field it receives, so an omitted field clears
// the stored value. Each one is sent on every write, null included.
//
// The form carries no type, group_ids, or user_ids. Those three belong to the
// group and dm channels the web UI creates, and Open WebUI ignores them for a
// standard channel.
type ChannelForm struct {
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	IsPrivate     *bool          `json:"is_private"`
	Data          map[string]any `json:"data"`
	Meta          map[string]any `json:"meta"`
	AccessControl map[string]any `json:"-"`
}

// MarshalJSON serialises the form using the API's access_grants list, derived
// from the provider's access_control representation.
func (f ChannelForm) MarshalJSON() ([]byte, error) {
	type alias ChannelForm
	return json.Marshal(struct {
		alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{
		alias:        alias(f),
		AccessGrants: accessControlToGrants(f.AccessControl),
	})
}

// Channel is a channel record. The create, update, and list routes answer with
// the base model; the read route adds write_access on top of it.
type Channel struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Type          *string        `json:"type"`
	Name          string         `json:"name"`
	Description   *string        `json:"description"`
	IsPrivate     *bool          `json:"is_private"`
	Data          map[string]any `json:"data"`
	Meta          map[string]any `json:"meta"`
	AccessControl map[string]any `json:"-"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	WriteAccess   *bool          `json:"write_access,omitempty"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (c *Channel) UnmarshalJSON(data []byte) error {
	type alias Channel
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// CreateChannel creates a standard channel. Open WebUI lowercases the name on
// create, so the stored name matches the request only when the request already
// holds a lowercase one. Only an admin may create a channel of this kind.
func (c *Client) CreateChannel(ctx context.Context, form ChannelForm) (*Channel, error) {
	var resp Channel
	if err := c.do(ctx, http.MethodPost, "channels/create", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetChannel retrieves a channel by ID. An admin reads any channel regardless
// of its access grants.
func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	var resp Channel
	if err := c.do(ctx, http.MethodGet, "channels/"+url.PathEscape(id), nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateChannel updates a channel by ID. The update path stores the name
// verbatim, without the lowercasing the create path applies.
func (c *Client) UpdateChannel(ctx context.Context, id string, form ChannelForm) (*Channel, error) {
	var resp Channel
	if err := c.do(ctx, http.MethodPost, "channels/"+url.PathEscape(id)+"/update", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteChannel removes a channel by ID.
func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "channels/"+url.PathEscape(id)+"/delete", nil, nil, nil)
}

// ListChannels returns every channel in the instance for an admin token.
func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	var resp []Channel
	if err := c.do(ctx, http.MethodGet, "channels/list", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
