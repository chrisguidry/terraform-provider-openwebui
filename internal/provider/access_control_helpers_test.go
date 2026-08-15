package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// accessControlIDs reads one list of principal IDs out of a built
// access_control map and fails the test when the shape is wrong.
func accessControlIDs(t *testing.T, control map[string]any, section, key string) []string {
	t.Helper()

	sectionMap, ok := control[section].(map[string]any)
	if !ok {
		t.Fatalf("expected a %s section, got %T", section, control[section])
	}

	ids, ok := sectionMap[key].([]string)
	if !ok {
		t.Fatalf("expected []string %s, got %T", key, sectionMap[key])
	}

	return ids
}

func TestBuildAccessControl_NoPrincipals(t *testing.T) {
	result := buildAccessControl(accessPrincipals{})
	if result != nil {
		t.Fatalf("expected nil when nothing is shared, got %+v", result)
	}
}

func TestBuildAccessControl_ReadOnly(t *testing.T) {
	result := buildAccessControl(accessPrincipals{ReadGroups: []string{"g1"}})
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	ids := accessControlIDs(t, result, "read", "group_ids")
	if len(ids) != 1 || ids[0] != "g1" {
		t.Fatalf("expected [g1], got %v", ids)
	}

	if users := accessControlIDs(t, result, "read", "user_ids"); len(users) != 0 {
		t.Fatalf("expected no read user_ids, got %v", users)
	}

	if _, hasWrite := result["write"]; hasWrite {
		t.Fatal("expected no write section for read-only access")
	}
}

func TestBuildAccessControl_ReadAndWrite(t *testing.T) {
	result := buildAccessControl(accessPrincipals{ReadGroups: []string{"g1"}, WriteGroups: []string{"g2"}})
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Read should contain both g1 (read-only) and g2 (write implies read)
	readIDs := accessControlIDs(t, result, "read", "group_ids")
	readSet := make(map[string]bool, len(readIDs))
	for _, id := range readIDs {
		readSet[id] = true
	}
	if !readSet["g1"] || !readSet["g2"] {
		t.Fatalf("expected read group_ids to include g1 and g2, got %v", readIDs)
	}

	// Write should contain only g2
	writeIDs := accessControlIDs(t, result, "write", "group_ids")
	if len(writeIDs) != 1 || writeIDs[0] != "g2" {
		t.Fatalf("expected write group_ids=[g2], got %v", writeIDs)
	}
}

func TestBuildAccessControl_WriteOnlyDeduplicatesRead(t *testing.T) {
	// g1 appears in both read and write. The merged read slice must hold it once.
	result := buildAccessControl(accessPrincipals{ReadGroups: []string{"g1"}, WriteGroups: []string{"g1"}})

	readIDs := accessControlIDs(t, result, "read", "group_ids")
	if len(readIDs) != 1 {
		t.Fatalf("expected exactly 1 read group (deduplicated), got %v", readIDs)
	}
}

// A resource shared with users but with no group still carries a read section,
// so that the user grants reach Open WebUI.
func TestBuildAccessControl_UsersOnly(t *testing.T) {
	result := buildAccessControl(accessPrincipals{ReadUsers: []string{"u1"}})
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if groups := accessControlIDs(t, result, "read", "group_ids"); len(groups) != 0 {
		t.Fatalf("expected no read group_ids, got %v", groups)
	}

	users := accessControlIDs(t, result, "read", "user_ids")
	if len(users) != 1 || users[0] != "u1" {
		t.Fatalf("expected read user_ids=[u1], got %v", users)
	}

	if _, hasWrite := result["write"]; hasWrite {
		t.Fatal("expected no write section for read-only access")
	}
}

// A write grant carries read with it for a user, exactly as it does for a group.
func TestBuildAccessControl_UserWriteImpliesRead(t *testing.T) {
	result := buildAccessControl(accessPrincipals{ReadUsers: []string{"u1"}, WriteUsers: []string{"u1", "u2"}})

	readUsers := accessControlIDs(t, result, "read", "user_ids")
	if len(readUsers) != 2 || readUsers[0] != "u1" || readUsers[1] != "u2" {
		t.Fatalf("expected read user_ids=[u1 u2], got %v", readUsers)
	}

	writeUsers := accessControlIDs(t, result, "write", "user_ids")
	if len(writeUsers) != 2 || writeUsers[0] != "u1" || writeUsers[1] != "u2" {
		t.Fatalf("expected write user_ids=[u1 u2], got %v", writeUsers)
	}
}

// Groups and users share one resource without either set disturbing the other.
func TestBuildAccessControl_GroupsAndUsers(t *testing.T) {
	result := buildAccessControl(accessPrincipals{
		ReadGroups:  []string{"g1"},
		WriteGroups: []string{"g2"},
		ReadUsers:   []string{"u1"},
		WriteUsers:  []string{"u2"},
	})

	readGroups := accessControlIDs(t, result, "read", "group_ids")
	if len(readGroups) != 2 || readGroups[0] != "g1" || readGroups[1] != "g2" {
		t.Fatalf("expected read group_ids=[g1 g2], got %v", readGroups)
	}

	readUsers := accessControlIDs(t, result, "read", "user_ids")
	if len(readUsers) != 2 || readUsers[0] != "u1" || readUsers[1] != "u2" {
		t.Fatalf("expected read user_ids=[u1 u2], got %v", readUsers)
	}

	writeGroups := accessControlIDs(t, result, "write", "group_ids")
	if len(writeGroups) != 1 || writeGroups[0] != "g2" {
		t.Fatalf("expected write group_ids=[g2], got %v", writeGroups)
	}

	writeUsers := accessControlIDs(t, result, "write", "user_ids")
	if len(writeUsers) != 1 || writeUsers[0] != "u2" {
		t.Fatalf("expected write user_ids=[u2], got %v", writeUsers)
	}
}

// A user named for writing opens the write section even when no group is.
func TestBuildAccessControl_WriteUsersOpenTheWriteSection(t *testing.T) {
	result := buildAccessControl(accessPrincipals{WriteUsers: []string{"u1"}})

	if groups := accessControlIDs(t, result, "write", "group_ids"); len(groups) != 0 {
		t.Fatalf("expected no write group_ids, got %v", groups)
	}

	writeUsers := accessControlIDs(t, result, "write", "user_ids")
	if len(writeUsers) != 1 || writeUsers[0] != "u1" {
		t.Fatalf("expected write user_ids=[u1], got %v", writeUsers)
	}
}

func TestWithPublicAccess_NilStaysNilWhenPrivate(t *testing.T) {
	if result := withPublicAccess(nil, false, false); result != nil {
		t.Fatalf("expected nil for a private resource with no groups, got %+v", result)
	}
}

func TestWithPublicAccess_NilGainsAMapWhenPublic(t *testing.T) {
	result := withPublicAccess(nil, true, false)
	if result["public_read"] != true {
		t.Fatalf("expected public_read=true, got %+v", result)
	}
	if result["public_write"] != false {
		t.Fatalf("expected public_write=false, got %+v", result)
	}
}

func TestWithPublicAccess_KeepsGroupSections(t *testing.T) {
	result := withPublicAccess(buildAccessControl(accessPrincipals{ReadGroups: []string{"g1"}}), false, true)
	if _, ok := result["read"].(map[string]any); !ok {
		t.Fatalf("expected the read section to survive, got %+v", result)
	}
	if result["public_read"] != false {
		t.Fatalf("expected public_read=false, got %+v", result)
	}
	if result["public_write"] != true {
		t.Fatalf("expected public_write=true, got %+v", result)
	}
}

func TestPublicAccessFromControl(t *testing.T) {
	access := map[string]any{"public_read": true, "public_write": false}
	if !publicAccessFromControl(access, "read") {
		t.Fatal("expected public_read=true")
	}
	if publicAccessFromControl(access, "write") {
		t.Fatal("expected public_write=false")
	}
	if publicAccessFromControl(nil, "read") {
		t.Fatal("expected false for a nil access_control map")
	}
}

func TestExtractGroupIDsFromAccessControl_Nil(t *testing.T) {
	ids := extractGroupIDsFromAccessControl(nil, "read")
	if ids != nil {
		t.Fatalf("expected nil for nil access, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_MissingSection(t *testing.T) {
	ids := extractGroupIDsFromAccessControl(map[string]any{}, "read")
	if ids != nil {
		t.Fatalf("expected nil for missing section, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SliceAny(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{
			"group_ids": []any{"g1", "g2", ""},
			"user_ids":  []any{},
		},
	}
	ids := extractGroupIDsFromAccessControl(access, "read")
	// empty string is filtered out
	if len(ids) != 2 || ids[0] != "g1" || ids[1] != "g2" {
		t.Fatalf("expected [g1 g2], got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SliceString(t *testing.T) {
	access := map[string]any{
		"write": map[string]any{
			"group_ids": []string{"g3"},
			"user_ids":  []string{},
		},
	}
	ids := extractGroupIDsFromAccessControl(access, "write")
	if len(ids) != 1 || ids[0] != "g3" {
		t.Fatalf("expected [g3], got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_SectionNotMap(t *testing.T) {
	access := map[string]any{"read": "unexpected-string"}
	ids := extractGroupIDsFromAccessControl(access, "read")
	if ids != nil {
		t.Fatalf("expected nil for non-map section, got %v", ids)
	}
}

func TestExtractGroupIDsFromAccessControl_MissingGroupIDs(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{"user_ids": []any{"u1"}},
	}
	ids := extractGroupIDsFromAccessControl(access, "read")
	if ids != nil {
		t.Fatalf("expected nil for missing group_ids, got %v", ids)
	}
}

// One section holds both principal types, and each reader takes only its own.
func TestExtractPrincipalIDsFromAccessControl_SplitsUsersFromGroups(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{
			"group_ids": []any{"g1"},
			"user_ids":  []any{"u1", "u2"},
		},
	}

	groups := extractGroupIDsFromAccessControl(access, "read")
	if len(groups) != 1 || groups[0] != "g1" {
		t.Fatalf("expected group_ids=[g1], got %v", groups)
	}

	users := extractUserIDsFromAccessControl(access, "read")
	if len(users) != 2 || users[0] != "u1" || users[1] != "u2" {
		t.Fatalf("expected user_ids=[u1 u2], got %v", users)
	}
}

func TestExtractUserIDsFromAccessControl_MissingUserIDs(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{"group_ids": []any{"g1"}},
	}
	ids := extractUserIDsFromAccessControl(access, "read")
	if ids != nil {
		t.Fatalf("expected nil for missing user_ids, got %v", ids)
	}
}

// The wildcard ID names every signed-in user, which public_read carries. It
// must never reach read_users, where it would name an account that does not
// exist.
func TestExtractUserIDsFromAccessControl_DropsTheWildcard(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{
			"group_ids": []string{},
			"user_ids":  []any{"*", "u1"},
		},
	}
	ids := extractUserIDsFromAccessControl(access, "read")
	if len(ids) != 1 || ids[0] != "u1" {
		t.Fatalf("expected [u1], got %v", ids)
	}
}

// fakePrincipalsAPI serves the group and user routes that the sharing
// attributes read. The user search matches a name or a mail address, which is
// what Open WebUI does, so a user ID finds nothing there and has to be read
// from the account route.
func newFakePrincipalsAPI(t *testing.T) *client.Client {
	t.Helper()

	users := map[string]string{
		"u1": "parent@example.com",
		"u2": "kid@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
		w.Header().Set("Content-Type", "application/json")

		if path == "groups/" {
			_, _ = w.Write([]byte(`[{"id":"g1","name":"family","description":"","user_ids":[]}]`))
			return
		}

		if path == "users/" {
			query := strings.ToLower(r.URL.Query().Get("query"))
			matches := []string{}
			for id, email := range users {
				if query != "" && !strings.Contains(email, query) {
					continue
				}
				matches = append(matches, fmt.Sprintf(`{"id":%q,"name":"Account","email":%q,"role":"user","last_active_at":1,"updated_at":2,"created_at":3}`, id, email))
			}
			_, _ = fmt.Fprintf(w, `{"users":[%s],"total":%d}`, strings.Join(matches, ","), len(matches))
			return
		}

		email, ok := users[strings.TrimPrefix(path, "users/")]
		if !ok {
			http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":"Account","email":%q,"role":"user","last_active_at":1,"updated_at":2,"created_at":3}`, strings.TrimPrefix(path, "users/"), email)
	}))
	t.Cleanup(server.Close)

	apiClient, err := client.NewClient(server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return apiClient
}

func TestResolveUserRefsToIDs_ResolvesAMailAddress(t *testing.T) {
	var diags diag.Diagnostics
	ids := resolveUserRefsToIDs(context.Background(), newFakePrincipalsAPI(t), []string{"parent@example.com"}, path.Root("read_users"), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(ids) != 1 || ids[0] != "u1" {
		t.Fatalf("expected [u1], got %v", ids)
	}
}

// A user ID matches nothing in the search route, so the account route answers
// for it. This is what lets a configuration name either form.
func TestResolveUserRefsToIDs_ResolvesAUserID(t *testing.T) {
	var diags diag.Diagnostics
	ids := resolveUserRefsToIDs(context.Background(), newFakePrincipalsAPI(t), []string{"u2"}, path.Root("read_users"), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(ids) != 1 || ids[0] != "u2" {
		t.Fatalf("expected [u2], got %v", ids)
	}
}

func TestResolveUserRefsToIDs_ReportsAnUnknownReference(t *testing.T) {
	var diags diag.Diagnostics
	resolveUserRefsToIDs(context.Background(), newFakePrincipalsAPI(t), []string{"stranger@example.com"}, path.Root("read_users"), &diags)
	if !diags.HasError() {
		t.Fatal("expected an error for a reference that names no account")
	}
}

func TestResolveUserRefsToIDs_SkipsBlanksAndDuplicates(t *testing.T) {
	var diags diag.Diagnostics
	ids := resolveUserRefsToIDs(context.Background(), newFakePrincipalsAPI(t), []string{" parent@example.com ", "", "parent@example.com"}, path.Root("read_users"), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(ids) != 1 || ids[0] != "u1" {
		t.Fatalf("expected [u1], got %v", ids)
	}
}

// A stored ID becomes the mail address a configuration names it by, so a config
// holding an address and an API holding an ID agree.
func TestFetchUserEmailsForIDs_NamesEachAccountByItsAddress(t *testing.T) {
	emails, diags := fetchUserEmailsForIDs(context.Background(), newFakePrincipalsAPI(t), []string{"u1", "u2"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(emails) != 2 || emails[0] != "parent@example.com" || emails[1] != "kid@example.com" {
		t.Fatalf("expected [parent@example.com kid@example.com], got %v", emails)
	}
}

// An account deleted outside Terraform drops out of the list. The read still
// succeeds.
func TestFetchUserEmailsForIDs_DropsAMissingAccount(t *testing.T) {
	emails, diags := fetchUserEmailsForIDs(context.Background(), newFakePrincipalsAPI(t), []string{"u1", "gone"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if len(emails) != 1 || emails[0] != "parent@example.com" {
		t.Fatalf("expected [parent@example.com], got %v", emails)
	}
}

// One access_control map holds both principal types, and the read-back returns
// group names in one list and mail addresses in another.
func TestReadAccessPrincipals_SplitsUsersFromGroups(t *testing.T) {
	access := map[string]any{
		"read": map[string]any{
			"group_ids": []string{"g1"},
			"user_ids":  []string{"u1", "u2"},
		},
		"write": map[string]any{
			"group_ids": []string{},
			"user_ids":  []string{"u2"},
		},
	}

	names, diags := readAccessPrincipals(context.Background(), newFakePrincipalsAPI(t), access)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if len(names.ReadGroups) != 1 || names.ReadGroups[0] != "family" {
		t.Fatalf("expected read groups [family], got %v", names.ReadGroups)
	}
	if len(names.ReadUsers) != 2 || names.ReadUsers[0] != "parent@example.com" || names.ReadUsers[1] != "kid@example.com" {
		t.Fatalf("expected read users [parent@example.com kid@example.com], got %v", names.ReadUsers)
	}
	if len(names.WriteGroups) != 0 {
		t.Fatalf("expected no write groups, got %v", names.WriteGroups)
	}
	if len(names.WriteUsers) != 1 || names.WriteUsers[0] != "kid@example.com" {
		t.Fatalf("expected write users [kid@example.com], got %v", names.WriteUsers)
	}
}
