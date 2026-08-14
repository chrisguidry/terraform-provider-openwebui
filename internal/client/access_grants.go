package client

// wildcardPrincipalID is the principal ID Open WebUI reads as "every signed-in
// user". A grant of principal type "user" on this ID is what the web UI calls
// public sharing.
const wildcardPrincipalID = "*"

// accessGrant is the Open WebUI wire representation of a single access grant.
type accessGrant struct {
	ID            string `json:"id,omitempty"`
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	Permission    string `json:"permission"`
}

// accessControlToGrants converts the provider's nested access_control map into
// the flat access_grants list the Open WebUI API expects. The map holds a
// "read" and a "write" section, each {"group_ids": [...], "user_ids": [...]},
// plus the booleans "public_read" and "public_write" for the wildcard grants.
// A nil map yields an empty list (owner-only / private).
func accessControlToGrants(ac map[string]any) []accessGrant {
	grants := []accessGrant{}
	if ac == nil {
		return grants
	}
	for _, permission := range []string{"read", "write"} {
		if public, ok := ac["public_"+permission].(bool); ok && public {
			grants = append(grants, accessGrant{PrincipalType: "user", PrincipalID: wildcardPrincipalID, Permission: permission})
		}
		section, ok := ac[permission].(map[string]any)
		if !ok {
			continue
		}
		for _, id := range anyToStrings(section["group_ids"]) {
			grants = append(grants, accessGrant{PrincipalType: "group", PrincipalID: id, Permission: permission})
		}
		for _, id := range anyToStrings(section["user_ids"]) {
			grants = append(grants, accessGrant{PrincipalType: "user", PrincipalID: id, Permission: permission})
		}
	}
	return grants
}

// grantsToAccessControl converts an access_grants list back into the provider's
// nested access_control map. A wildcard user grant becomes the "public_read" or
// "public_write" boolean, so that public sharing done in the web UI is visible
// to the provider instead of being revoked on the next write. Returns nil when
// the list holds nothing the provider models.
func grantsToAccessControl(grants []accessGrant) map[string]any {
	read := map[string]any{"group_ids": []string{}, "user_ids": []string{}}
	write := map[string]any{"group_ids": []string{}, "user_ids": []string{}}
	sections := map[string]map[string]any{"read": read, "write": write}
	public := map[string]bool{"read": false, "write": false}

	found := false
	for _, g := range grants {
		if g.PrincipalID == "" {
			continue
		}
		section, ok := sections[g.Permission]
		if !ok {
			continue
		}
		var key string
		switch g.PrincipalType {
		case "group":
			if g.PrincipalID == wildcardPrincipalID {
				// A wildcard group ID names no group. Open WebUI reads public
				// sharing off the user principal only.
				continue
			}
			key = "group_ids"
		case "user":
			if g.PrincipalID == wildcardPrincipalID {
				public[g.Permission] = true
				found = true
				continue
			}
			key = "user_ids"
		case "anyone":
			// An "anyone" grant shares with unauthenticated visitors. Every
			// route this client calls strips it server-side, so there is
			// nothing to carry into access_control. Dropping it is deliberate.
			continue
		default:
			continue
		}
		cur, _ := section[key].([]string)
		section[key] = append(cur, g.PrincipalID)
		found = true
	}
	if !found {
		return nil
	}
	return map[string]any{
		"read":         read,
		"write":        write,
		"public_read":  public["read"],
		"public_write": public["write"],
	}
}

func anyToStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
