package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SkillMeta captures descriptive metadata for a skill.
type SkillMeta struct {
	Tags []string `json:"tags"`
}

// SkillForm represents the payload for creating or updating skills.
type SkillForm struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Content       string         `json:"content"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      *bool          `json:"is_active,omitempty"`
	AccessControl map[string]any `json:"-"`
}

// MarshalJSON serialises the form using the API's access_grants list, derived
// from the provider's access_control representation.
func (f SkillForm) MarshalJSON() ([]byte, error) {
	type alias SkillForm
	return json.Marshal(struct {
		alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{
		alias:        alias(f),
		AccessGrants: accessControlToGrants(f.AccessControl),
	})
}

// SkillResponse captures the skill record returned by the create endpoint. The
// API declares no content field on this response, so a caller that needs the
// skill body must read the skill back by ID.
type SkillResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillResponse) UnmarshalJSON(data []byte) error {
	type alias SkillResponse
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// SkillModel is the full skill record, including the skill body. The update and
// export endpoints return it.
type SkillModel struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Content       string         `json:"content"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillModel) UnmarshalJSON(data []byte) error {
	type alias SkillModel
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// SkillAccessResponse captures skill details with access metadata. Content is
// not declared on the API's response model, but the handler builds the response
// from the full record and allows extra fields, so the body arrives with it.
type SkillAccessResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Content       string         `json:"content"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	User          *User          `json:"user,omitempty"`
	WriteAccess   *bool          `json:"write_access,omitempty"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillAccessResponse) UnmarshalJSON(data []byte) error {
	type alias SkillAccessResponse
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// SkillUserResponse captures skill details including user metadata.
type SkillUserResponse struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   *string        `json:"description,omitempty"`
	Meta          SkillMeta      `json:"meta"`
	IsActive      bool           `json:"is_active"`
	AccessControl map[string]any `json:"-"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
	User          *User          `json:"user,omitempty"`
}

// UnmarshalJSON decodes the API's access_grants list into the provider's
// access_control representation.
func (r *SkillUserResponse) UnmarshalJSON(data []byte) error {
	type alias SkillUserResponse
	aux := struct {
		*alias
		AccessGrants []accessGrant `json:"access_grants"`
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.AccessControl = grantsToAccessControl(aux.AccessGrants)
	return nil
}

// CreateSkill provisions a new skill. The server lowercases the requested ID and
// replaces spaces with hyphens, so read the ID off the response.
func (c *Client) CreateSkill(ctx context.Context, form SkillForm) (*SkillResponse, error) {
	var resp SkillResponse
	if err := c.do(ctx, http.MethodPost, "skills/create", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetSkill retrieves a skill by ID, including its content.
func (c *Client) GetSkill(ctx context.Context, id string) (*SkillAccessResponse, error) {
	var resp SkillAccessResponse
	path := "skills/id/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateSkill updates an existing skill by ID.
func (c *Client) UpdateSkill(ctx context.Context, id string, form SkillForm) (*SkillModel, error) {
	var resp SkillModel
	path := "skills/id/" + url.PathEscape(id) + "/update"
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteSkill removes a skill by ID.
func (c *Client) DeleteSkill(ctx context.Context, id string) error {
	path := "skills/id/" + url.PathEscape(id) + "/delete"
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListSkills returns skill summaries. The route is declared as "/", and a path
// without the trailing slash falls through to the web application mount and
// answers with HTML.
func (c *Client) ListSkills(ctx context.Context) ([]SkillUserResponse, error) {
	var resp []SkillUserResponse
	if err := c.do(ctx, http.MethodGet, "skills/", nil, nil, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
