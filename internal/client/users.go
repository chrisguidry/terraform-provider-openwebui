package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// User represents an Open WebUI user account.
type User struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Username     *string `json:"username"`
	Role         string  `json:"role"`
	ProfileImage string  `json:"profile_image_url"`
	Bio          *string `json:"bio"`
	LastActiveAt int64   `json:"last_active_at"`
	UpdatedAt    int64   `json:"updated_at"`
	CreatedAt    int64   `json:"created_at"`
}

// listUsersResponse models the API contract for GET /users/.
type listUsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// SearchUsers finds users whose username, email, or name matches the provided query.
// The API paginates server-side (fixed page size); the limit argument is retained
// for call-site compatibility but is no longer a supported query parameter.
func (c *Client) SearchUsers(ctx context.Context, query string, limit int) ([]User, error) {
	values := url.Values{}
	if query != "" {
		values.Set("query", query)
	}

	var resp listUsersResponse
	if err := c.do(ctx, http.MethodGet, "users/", values, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Users, nil
}

// GetUser retrieves a user by identifier.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var resp User
	path := fmt.Sprintf("users/%s", url.PathEscape(id))
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// AddUserForm creates a user account. The route is admin-only and refuses an
// email that is already taken, so there is no upsert.
type AddUserForm struct {
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	Password        string  `json:"password"`
	ProfileImageURL *string `json:"profile_image_url,omitempty"`
	Role            *string `json:"role,omitempty"`
}

// addUserResponse reads the identity out of the create response. The response
// also carries a live JWT for the new account. Naming only these four fields
// keeps the token out of everything downstream of this call.
type addUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// UserUpdateForm updates a user account. Every field is optional and the server
// applies only the ones that are present, so an omitted field keeps its value.
type UserUpdateForm struct {
	Role            *string `json:"role,omitempty"`
	Name            *string `json:"name,omitempty"`
	Email           *string `json:"email,omitempty"`
	ProfileImageURL *string `json:"profile_image_url,omitempty"`
	Password        *string `json:"password,omitempty"`
}

// AddUser creates a user account and returns its identifier. The route lives in
// the auths router; every other user operation lives in the users router.
func (c *Client) AddUser(ctx context.Context, form AddUserForm) (*User, error) {
	var created addUserResponse
	if err := c.do(ctx, http.MethodPost, "auths/add", nil, form, &created); err != nil {
		return nil, err
	}

	// The create response reports the profile image as a route rather than the
	// stored value, so the account is read back for its real attributes.
	return c.GetUser(ctx, created.ID)
}

// UpdateUser changes a user account. Open WebUI protects the primary admin: an
// admin who is not the primary admin cannot update it, and the primary admin
// cannot change its own role away from admin. Both answer 403.
func (c *Client) UpdateUser(ctx context.Context, id string, form UserUpdateForm) (*User, error) {
	var resp User
	path := fmt.Sprintf("users/%s/update", url.PathEscape(id))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteUser removes a user account. Open WebUI refuses to delete the primary
// admin, and refuses to let a caller delete itself. Both answer 403.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	path := fmt.Sprintf("users/%s", url.PathEscape(id))
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

// GetPrimaryAdmin returns the earliest-created account, which is the primary
// admin Open WebUI protects. It asks for the same ordering the server's own
// check uses, so the two agree on which account that is. A nil result means the
// instance holds no users.
func (c *Client) GetPrimaryAdmin(ctx context.Context) (*User, error) {
	values := url.Values{}
	values.Set("order_by", "created_at")
	values.Set("direction", "asc")
	values.Set("page", "1")

	var resp listUsersResponse
	if err := c.do(ctx, http.MethodGet, "users/", values, nil, &resp); err != nil {
		return nil, err
	}

	if len(resp.Users) == 0 {
		return nil, nil
	}

	first := resp.Users[0]
	return &first, nil
}
