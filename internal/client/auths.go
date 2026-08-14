package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// FlexString carries a config value the API declares as `int | str | None`.
// Open WebUI casts such a value to an int when it is truthy and stores an empty
// string otherwise, so the same key reads back as a JSON number, a string, or
// null depending on what was last written. Holding it as a string keeps zero
// and "unset" expressible, which an integer cannot do.
type FlexString string

// UnmarshalJSON accepts a JSON string, number, or null.
func (f *FlexString) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*f = ""
		return nil
	}

	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		*f = FlexString(value)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(trimmed, &number); err != nil {
		return fmt.Errorf("decode flexible string value %s: %w", trimmed, err)
	}
	*f = FlexString(number.String())
	return nil
}

// MarshalJSON always writes a JSON string. The route accepts a string for these
// keys and casts it, so the string form round-trips without losing the empty
// value.
func (f FlexString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(f))
}

// Int returns the numeric value, or 0 when the value is empty or not a number.
func (f FlexString) Int() int64 {
	value, err := strconv.ParseInt(string(f), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// AdminConfig holds the 28 instance-wide settings of the admin panel: signup,
// the default user role, API keys, JWT expiry, and the feature switches for
// folders, automations, channels, calendar, memories, and notes.
//
// The write drops a value it considers invalid and still answers 200, so a bad
// default user role, channel response mode, or JWT expiry leaves the stored
// value untouched with no error.
type AdminConfig struct {
	ShowAdminDetails                  bool       `json:"SHOW_ADMIN_DETAILS"`
	AdminEmail                        *string    `json:"ADMIN_EMAIL"`
	WebUIURL                          string     `json:"WEBUI_URL"`
	EnableSignup                      bool       `json:"ENABLE_SIGNUP"`
	EnableAPIKeys                     bool       `json:"ENABLE_API_KEYS"`
	EnableAPIKeysEndpointRestrictions bool       `json:"ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS"`
	APIKeysAllowedEndpoints           string     `json:"API_KEYS_ALLOWED_ENDPOINTS"`
	DefaultUserRole                   string     `json:"DEFAULT_USER_ROLE"`
	DefaultGroupID                    string     `json:"DEFAULT_GROUP_ID"`
	JWTExpiresIn                      string     `json:"JWT_EXPIRES_IN"`
	EnableCommunitySharing            bool       `json:"ENABLE_COMMUNITY_SHARING"`
	EnableMessageRating               bool       `json:"ENABLE_MESSAGE_RATING"`
	EnableFolders                     bool       `json:"ENABLE_FOLDERS"`
	FolderMaxFileCount                FlexString `json:"FOLDER_MAX_FILE_COUNT"`
	AutomationMaxCount                FlexString `json:"AUTOMATION_MAX_COUNT"`
	AutomationMinInterval             FlexString `json:"AUTOMATION_MIN_INTERVAL"`
	EnableAutomations                 bool       `json:"ENABLE_AUTOMATIONS"`
	EnableChannels                    bool       `json:"ENABLE_CHANNELS"`
	ChannelModelResponseMode          string     `json:"CHANNEL_MODEL_RESPONSE_MODE"`
	EnableCalendar                    bool       `json:"ENABLE_CALENDAR"`
	EnableMemories                    bool       `json:"ENABLE_MEMORIES"`
	EnableMemorySystemContext         bool       `json:"ENABLE_MEMORY_SYSTEM_CONTEXT"`
	EnableNotes                       bool       `json:"ENABLE_NOTES"`
	EnableUserWebhooks                bool       `json:"ENABLE_USER_WEBHOOKS"`
	EnableUserStatus                  bool       `json:"ENABLE_USER_STATUS"`
	PendingUserOverlayTitle           *string    `json:"PENDING_USER_OVERLAY_TITLE"`
	PendingUserOverlayContent         *string    `json:"PENDING_USER_OVERLAY_CONTENT"`
	ResponseWatermark                 *string    `json:"RESPONSE_WATERMARK"`
}

// GetAdminConfig retrieves the admin configuration.
func (c *Client) GetAdminConfig(ctx context.Context) (*AdminConfig, error) {
	var resp AdminConfig
	if err := c.do(ctx, http.MethodGet, "auths/admin/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetAdminConfig writes the admin configuration and returns the stored values.
func (c *Client) SetAdminConfig(ctx context.Context, config AdminConfig) (*AdminConfig, error) {
	var resp AdminConfig
	if err := c.do(ctx, http.MethodPost, "auths/admin/config", nil, config, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// LDAPServerConfig holds the directory connection the LDAP sign-in flow uses.
// AppDNPassword is the bind password. The read route declares it as a plain
// string and returns it unmasked, so a caller sees the stored secret.
type LDAPServerConfig struct {
	Label                 string  `json:"label"`
	Host                  string  `json:"host"`
	Port                  *int64  `json:"port"`
	AttributeForMail      string  `json:"attribute_for_mail"`
	AttributeForUsername  string  `json:"attribute_for_username"`
	AppDN                 string  `json:"app_dn"`
	AppDNPassword         string  `json:"app_dn_password"`
	SearchBase            string  `json:"search_base"`
	SearchFilters         string  `json:"search_filters"`
	UseTLS                bool    `json:"use_tls"`
	CertificatePath       *string `json:"certificate_path"`
	ValidateCert          bool    `json:"validate_cert"`
	Ciphers               *string `json:"ciphers"`
	EnableGroupManagement bool    `json:"enable_group_management"`
	EnableGroupCreation   bool    `json:"enable_group_creation"`
	AttributeForGroups    string  `json:"attribute_for_groups"`
}

// ldapEnableResponse is the read shape of the LDAP switch. The write takes
// enable_ldap and the read answers with ENABLE_LDAP, so the two spellings are
// not interchangeable.
type ldapEnableResponse struct {
	EnableLDAP bool `json:"ENABLE_LDAP"`
}

// ldapEnableForm is the write shape of the LDAP switch.
type ldapEnableForm struct {
	EnableLDAP bool `json:"enable_ldap"`
}

// GetLDAPServerConfig retrieves the LDAP directory connection.
func (c *Client) GetLDAPServerConfig(ctx context.Context) (*LDAPServerConfig, error) {
	var resp LDAPServerConfig
	if err := c.do(ctx, http.MethodGet, "auths/admin/config/ldap/server", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetLDAPServerConfig writes the LDAP directory connection. The route rejects
// an empty label, host, mail attribute, username attribute, or search base, and
// rejects an empty group attribute while group management is on.
func (c *Client) SetLDAPServerConfig(ctx context.Context, config LDAPServerConfig) (*LDAPServerConfig, error) {
	var resp LDAPServerConfig
	if err := c.do(ctx, http.MethodPost, "auths/admin/config/ldap/server", nil, config, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetLDAPEnabled reports whether LDAP sign-in is enabled.
func (c *Client) GetLDAPEnabled(ctx context.Context) (bool, error) {
	var resp ldapEnableResponse
	if err := c.do(ctx, http.MethodGet, "auths/admin/config/ldap", nil, nil, &resp); err != nil {
		return false, err
	}

	return resp.EnableLDAP, nil
}

// SetLDAPEnabled turns LDAP sign-in on or off.
func (c *Client) SetLDAPEnabled(ctx context.Context, enabled bool) (bool, error) {
	var resp ldapEnableResponse
	if err := c.do(ctx, http.MethodPost, "auths/admin/config/ldap", nil, ldapEnableForm{EnableLDAP: enabled}, &resp); err != nil {
		return false, err
	}

	return resp.EnableLDAP, nil
}

// SessionUser identifies the account whose token the client carries.
type SessionUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// GetSessionUser returns the account the client authenticates as.
func (c *Client) GetSessionUser(ctx context.Context) (*SessionUser, error) {
	var resp SessionUser
	if err := c.do(ctx, http.MethodGet, "auths/", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
