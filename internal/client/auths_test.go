package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// The three count limits arrive as a JSON number, a string, or null depending
// on what was last written.
func TestFlexStringAcceptsEveryStoredShape(t *testing.T) {
	cases := map[string]string{
		`50`:   "50",
		`"50"`: "50",
		`""`:   "",
		`null`: "",
	}

	for input, expected := range cases {
		var value FlexString
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		if string(value) != expected {
			t.Fatalf("expected %s to decode as %q, got %q", input, expected, value)
		}
	}
}

func TestFlexStringWritesAString(t *testing.T) {
	encoded, err := json.Marshal(FlexString("50"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != `"50"` {
		t.Fatalf("expected a JSON string, got %s", encoded)
	}
}

func TestFlexStringInt(t *testing.T) {
	if FlexString("50").Int() != 50 {
		t.Fatalf("expected 50, got %d", FlexString("50").Int())
	}
	if FlexString("").Int() != 0 {
		t.Fatalf("expected an empty value to count as 0, got %d", FlexString("").Int())
	}
}

func TestAdminConfigRoundTrip(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"SHOW_ADMIN_DETAILS":true,"ADMIN_EMAIL":null,"WEBUI_URL":"https://chat.example.com","ENABLE_SIGNUP":false,"DEFAULT_USER_ROLE":"pending","JWT_EXPIRES_IN":"4h","FOLDER_MAX_FILE_COUNT":50,"AUTOMATION_MAX_COUNT":"","CHANNEL_MODEL_RESPONSE_MODE":"thread"}`))
	})

	updated, err := c.SetAdminConfig(context.Background(), AdminConfig{
		WebUIURL:           "https://chat.example.com",
		DefaultUserRole:    "pending",
		JWTExpiresIn:       "4h",
		FolderMaxFileCount: FlexString("50"),
	})
	if err != nil {
		t.Fatalf("SetAdminConfig: %v", err)
	}

	if body["FOLDER_MAX_FILE_COUNT"] != "50" {
		t.Fatalf("expected the limit to go out as a string, got %v", body["FOLDER_MAX_FILE_COUNT"])
	}
	if updated.FolderMaxFileCount != "50" {
		t.Fatalf("expected the stored number to read back as text, got %q", updated.FolderMaxFileCount)
	}
	if updated.AutomationMaxCount != "" {
		t.Fatalf("expected an empty limit to stay empty, got %q", updated.AutomationMaxCount)
	}
	if updated.AdminEmail != nil {
		t.Fatalf("expected a null admin email to stay null, got %v", *updated.AdminEmail)
	}
}

// The LDAP switch is spelled enable_ldap on the way in and ENABLE_LDAP on the
// way out.
func TestSetLDAPEnabledTranslatesTheFieldName(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"ENABLE_LDAP":true}`))
	})

	enabled, err := c.SetLDAPEnabled(context.Background(), true)
	if err != nil {
		t.Fatalf("SetLDAPEnabled: %v", err)
	}

	if body["enable_ldap"] != true {
		t.Fatalf("expected enable_ldap in the request body, got %v", body)
	}
	if !enabled {
		t.Fatalf("expected ENABLE_LDAP to decode as true")
	}
}

// The bind password is stored and returned in plain text, so state converges.
func TestGetLDAPServerConfigReadsTheBindPassword(t *testing.T) {
	c := newChannelsTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"label":"Directory","host":"ldap.example.com","port":636,"attribute_for_mail":"mail","attribute_for_username":"uid","app_dn":"cn=svc,dc=example,dc=com","app_dn_password":"bind-secret","search_base":"dc=example,dc=com","search_filters":"","use_tls":true,"certificate_path":null,"validate_cert":true,"ciphers":"ALL","enable_group_management":false,"enable_group_creation":false,"attribute_for_groups":"memberOf"}`))
	})

	config, err := c.GetLDAPServerConfig(context.Background())
	if err != nil {
		t.Fatalf("GetLDAPServerConfig: %v", err)
	}

	if config.AppDNPassword != "bind-secret" {
		t.Fatalf("expected the bind password, got %q", config.AppDNPassword)
	}
	if config.Port == nil || *config.Port != 636 {
		t.Fatalf("expected port 636, got %v", config.Port)
	}
	if config.CertificatePath != nil {
		t.Fatalf("expected a null certificate path to stay null, got %q", *config.CertificatePath)
	}
}

func TestGetSessionUserReadsTheCallersIdentity(t *testing.T) {
	var path string
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"u1","email":"root@example.com","name":"Root","role":"admin"}`))
	})

	session, err := c.GetSessionUser(context.Background())
	if err != nil {
		t.Fatalf("GetSessionUser: %v", err)
	}

	if path != "/api/v1/auths/" {
		t.Fatalf("expected the session route, got %q", path)
	}
	if session.ID != "u1" {
		t.Fatalf("expected the caller's id, got %+v", session)
	}
}
