package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// wildcardPrincipalID is the principal ID Open WebUI reads as "every signed-in
// user". The public_read and public_write attributes carry it, so it never
// belongs in the per-user lists.
const wildcardPrincipalID = "*"

// accessPrincipals holds the resolved IDs of everyone a resource is shared
// with. Open WebUI records a principal type on every grant, so group IDs and
// user IDs stay in separate lists all the way to the wire.
type accessPrincipals struct {
	ReadGroups  []string
	WriteGroups []string
	ReadUsers   []string
	WriteUsers  []string
}

// accessPrincipalLists holds the same four sets as the Terraform list values a
// resource keeps in its plan and its state.
type accessPrincipalLists struct {
	ReadGroups  types.List
	WriteGroups types.List
	ReadUsers   types.List
	WriteUsers  types.List
}

// resolveAccessPrincipals maps the four sharing attributes of a plan onto the
// IDs Open WebUI stores. A group reference is a name or a group ID. A user
// reference is a mail address, a username, a display name, or a user ID.
func resolveAccessPrincipals(ctx context.Context, apiClient *client.Client, lists accessPrincipalLists, diags *diag.Diagnostics) accessPrincipals {
	readGroupsPath := path.Root("read_groups")
	writeGroupsPath := path.Root("write_groups")
	readUsersPath := path.Root("read_users")
	writeUsersPath := path.Root("write_users")

	readGroupNames := expandStringList(ctx, lists.ReadGroups, readGroupsPath, diags)
	writeGroupNames := expandStringList(ctx, lists.WriteGroups, writeGroupsPath, diags)
	readUserRefs := expandStringList(ctx, lists.ReadUsers, readUsersPath, diags)
	writeUserRefs := expandStringList(ctx, lists.WriteUsers, writeUsersPath, diags)

	return accessPrincipals{
		ReadGroups:  resolveGroupNamesToIDs(ctx, apiClient, readGroupNames, readGroupsPath, diags),
		WriteGroups: resolveGroupNamesToIDs(ctx, apiClient, writeGroupNames, writeGroupsPath, diags),
		ReadUsers:   resolveUserRefsToIDs(ctx, apiClient, readUserRefs, readUsersPath, diags),
		WriteUsers:  resolveUserRefsToIDs(ctx, apiClient, writeUserRefs, writeUsersPath, diags),
	}
}

// accessPrincipalNames holds the same four sets as the names a configuration
// writes: a group name for a group, a mail address for a user.
type accessPrincipalNames struct {
	ReadGroups  []string
	WriteGroups []string
	ReadUsers   []string
	WriteUsers  []string
}

// readAccessPrincipals reads the four sharing attributes back out of a live
// access_control map. Every stored ID becomes the group name or the mail
// address that a configuration names it by, so a config and the state it
// produces hold the same strings.
func readAccessPrincipals(ctx context.Context, apiClient *client.Client, access map[string]any) (accessPrincipalNames, diag.Diagnostics) {
	var diags diag.Diagnostics

	readGroups, readGroupDiags := fetchGroupNamesForIDs(ctx, apiClient, extractGroupIDsFromAccessControl(access, "read"))
	diags.Append(readGroupDiags...)
	writeGroups, writeGroupDiags := fetchGroupNamesForIDs(ctx, apiClient, extractGroupIDsFromAccessControl(access, "write"))
	diags.Append(writeGroupDiags...)

	readUsers, readUserDiags := fetchUserEmailsForIDs(ctx, apiClient, extractUserIDsFromAccessControl(access, "read"))
	diags.Append(readUserDiags...)
	writeUsers, writeUserDiags := fetchUserEmailsForIDs(ctx, apiClient, extractUserIDsFromAccessControl(access, "write"))
	diags.Append(writeUserDiags...)

	return accessPrincipalNames{
		ReadGroups:  readGroups,
		WriteGroups: writeGroups,
		ReadUsers:   readUsers,
		WriteUsers:  writeUsers,
	}, diags
}

// flattenAccessPrincipals reads the four sharing attributes out of a live
// access_control map as Terraform lists. An attribute that names nobody becomes
// an empty list, so a config that sets it to [] matches the state it produces.
func flattenAccessPrincipals(ctx context.Context, apiClient *client.Client, access map[string]any) (accessPrincipalLists, diag.Diagnostics) {
	names, diags := readAccessPrincipals(ctx, apiClient, access)

	readGroups, readGroupDiags := flattenStringSlice(ctx, names.ReadGroups)
	diags.Append(readGroupDiags...)
	writeGroups, writeGroupDiags := flattenStringSlice(ctx, names.WriteGroups)
	diags.Append(writeGroupDiags...)
	readUsers, readUserDiags := flattenStringSlice(ctx, names.ReadUsers)
	diags.Append(readUserDiags...)
	writeUsers, writeUserDiags := flattenStringSlice(ctx, names.WriteUsers)
	diags.Append(writeUserDiags...)

	return accessPrincipalLists{
		ReadGroups:  readGroups,
		WriteGroups: writeGroups,
		ReadUsers:   readUsers,
		WriteUsers:  writeUsers,
	}, diags
}

// nullableAccessList converts one sharing attribute into a state value that is
// null when it names nobody. The knowledge and prompt resources leave an unset
// sharing attribute null rather than storing an empty list.
func nullableAccessList(ctx context.Context, names []string, diags *diag.Diagnostics) types.List {
	if len(names) == 0 {
		return types.ListNull(types.StringType)
	}

	list, listDiags := types.ListValueFrom(ctx, types.StringType, names)
	diags.Append(listDiags...)
	if listDiags.HasError() {
		return types.ListNull(types.StringType)
	}

	return list
}

func resolveGroupNamesToIDs(ctx context.Context, apiClient *client.Client, names []string, attribute path.Path, diags *diag.Diagnostics) []string {
	if len(names) == 0 {
		return nil
	}

	groups, err := apiClient.ListGroups(ctx)
	if err != nil {
		diags.AddAttributeError(
			attribute,
			"Unable to list groups",
			fmt.Sprintf("Failed to retrieve groups from Open WebUI: %v", err),
		)
		return nil
	}

	byName := make(map[string]string, len(groups))
	byID := make(map[string]string, len(groups))
	for _, g := range groups {
		lowerName := strings.ToLower(g.Name)
		byName[lowerName] = g.ID
		byID[strings.ToLower(g.ID)] = g.ID
	}

	var ids []string
	for _, raw := range names {
		identifier := strings.TrimSpace(raw)
		if identifier == "" {
			continue
		}

		key := strings.ToLower(identifier)
		if id, ok := byName[key]; ok {
			ids = append(ids, id)
			continue
		}
		if id, ok := byID[key]; ok {
			ids = append(ids, id)
			continue
		}

		group, err := apiClient.GetGroup(ctx, identifier)
		if err == nil {
			ids = append(ids, group.ID)
			continue
		}

		diags.AddAttributeError(
			attribute,
			"Unknown group reference",
			fmt.Sprintf("No Open WebUI group was found for %q.", identifier),
		)
	}

	return uniqueStrings(ids)
}

func fetchGroupNamesForIDs(ctx context.Context, apiClient *client.Client, ids []string) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(ids) == 0 {
		return nil, diags
	}

	groups, err := apiClient.ListGroups(ctx)
	if err != nil {
		diags.AddError(
			"Unable to list groups",
			fmt.Sprintf("Failed to retrieve groups from Open WebUI: %v", err),
		)
		return nil, diags
	}

	byID := make(map[string]string, len(groups))
	for _, g := range groups {
		byID[g.ID] = g.Name
	}

	var names []string
	for _, id := range ids {
		if name, ok := byID[id]; ok {
			names = append(names, name)
			continue
		}

		group, err := apiClient.GetGroup(ctx, id)
		if err != nil {
			if err == client.ErrNotFound {
				continue
			}
			diags.AddError(
				"Fetch group failed",
				fmt.Sprintf("Failed to retrieve group %s: %v", id, err),
			)
			continue
		}

		names = append(names, group.Name)
	}

	return uniqueStrings(names), diags
}

// resolveUserRefsToIDs maps user references onto Open WebUI user IDs. It takes
// the same references the group membership attribute takes, so one address
// names the same account everywhere in a configuration.
func resolveUserRefsToIDs(ctx context.Context, apiClient *client.Client, references []string, attribute path.Path, diags *diag.Diagnostics) []string {
	if len(references) == 0 {
		return nil
	}

	identifiers := make([]string, 0, len(references))
	for _, raw := range references {
		identifier := strings.TrimSpace(raw)
		if identifier == "" {
			continue
		}
		identifiers = append(identifiers, identifier)
	}

	return uniqueStrings(resolveUsernamesToIDs(ctx, apiClient, identifiers, attribute, diags))
}

// fetchUserEmailsForIDs maps user IDs onto the address that names each account,
// in the order the IDs arrive. An account Open WebUI no longer holds drops out
// of the list, which is how a deletion outside Terraform leaves state.
func fetchUserEmailsForIDs(ctx context.Context, apiClient *client.Client, ids []string) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(ids) == 0 {
		return nil, diags
	}

	var emails []string
	for _, id := range ids {
		user, err := apiClient.GetUser(ctx, id)
		if err != nil {
			if err == client.ErrNotFound {
				continue
			}
			diags.AddError(
				"Fetch user failed",
				fmt.Sprintf("Failed to retrieve user %s: %v", id, err),
			)
			continue
		}

		emails = append(emails, userLabel(user, id))
	}

	return uniqueStrings(emails), diags
}

// userLabel is the name an account carries in Terraform state. Open WebUI
// identifies an account by its mail address, so that comes first. The other
// fields cover an account that holds no address.
func userLabel(user *client.User, id string) string {
	if user.Email != "" {
		return user.Email
	}
	if user.Username != nil && *user.Username != "" {
		return *user.Username
	}
	if user.Name != "" {
		return user.Name
	}

	return id
}

// buildAccessControl turns resolved principal IDs into the nested access_control
// map that the client converts into access grants. A write grant carries read
// with it, so every principal named for writing is named for reading as well. A
// resource shared with nobody yields a nil map, which the client sends as
// owner-only.
func buildAccessControl(principals accessPrincipals) map[string]any {
	mergedReadGroups := uniqueStrings(append(principals.ReadGroups, principals.WriteGroups...))
	mergedReadUsers := uniqueStrings(append(principals.ReadUsers, principals.WriteUsers...))

	if len(mergedReadGroups) == 0 && len(mergedReadUsers) == 0 {
		return nil
	}

	control := map[string]any{
		"read": accessControlSection(mergedReadGroups, mergedReadUsers),
	}

	if len(principals.WriteGroups) > 0 || len(principals.WriteUsers) > 0 {
		control["write"] = accessControlSection(principals.WriteGroups, principals.WriteUsers)
	}

	return control
}

// accessControlSection builds one half of an access_control map. Both lists are
// copied, so an empty one is an empty list rather than a null and the caller's
// slices stay as they were.
func accessControlSection(groupIDs, userIDs []string) map[string]any {
	return map[string]any{
		"group_ids": append([]string{}, groupIDs...),
		"user_ids":  append([]string{}, userIDs...),
	}
}

// withPublicAccess records the public-sharing flags on an access_control map.
// The client turns each true flag into a wildcard user grant, which is what
// Open WebUI reads as "every signed-in user". A resource that shares with
// nobody keeps a nil map, which the client sends as owner-only.
func withPublicAccess(control map[string]any, publicRead, publicWrite bool) map[string]any {
	if control == nil {
		if !publicRead && !publicWrite {
			return nil
		}
		control = map[string]any{}
	}

	control["public_read"] = publicRead
	control["public_write"] = publicWrite

	return control
}

// publicAccessFromControl reports whether an access_control map carries the
// wildcard grant for the given permission.
func publicAccessFromControl(access map[string]any, permission string) bool {
	public, _ := access["public_"+permission].(bool)
	return public
}

func extractGroupIDsFromAccessControl(access map[string]any, section string) []string {
	return extractPrincipalIDsFromAccessControl(access, section, "group_ids")
}

// extractUserIDsFromAccessControl reads the user principals out of one section
// of an access_control map. The wildcard ID names every signed-in user rather
// than an account, and public_read and public_write already carry it, so it
// stays out of the per-user lists.
func extractUserIDsFromAccessControl(access map[string]any, section string) []string {
	var ids []string
	for _, id := range extractPrincipalIDsFromAccessControl(access, section, "user_ids") {
		if id == wildcardPrincipalID {
			continue
		}
		ids = append(ids, id)
	}

	return ids
}

func extractPrincipalIDsFromAccessControl(access map[string]any, section, key string) []string {
	if access == nil {
		return nil
	}

	raw, ok := access[section]
	if !ok || raw == nil {
		return nil
	}

	sectionMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	idsRaw, ok := sectionMap[key]
	if !ok || idsRaw == nil {
		return nil
	}

	switch v := idsRaw.(type) {
	case []any:
		var ids []string
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				ids = append(ids, str)
			}
		}
		return ids
	case []string:
		return v
	default:
		return nil
	}
}
