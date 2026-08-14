package client

import (
	"reflect"
	"testing"
)

func TestAccessControlToGrants(t *testing.T) {
	ac := map[string]any{
		"read":  map[string]any{"group_ids": []string{"g1", "g2"}, "user_ids": []string{}},
		"write": map[string]any{"group_ids": []string{"g2"}, "user_ids": []string{}},
	}
	grants := accessControlToGrants(ac)
	if len(grants) != 3 {
		t.Fatalf("expected 3 grants, got %d: %+v", len(grants), grants)
	}

	want := map[string]bool{"group/g1/read": true, "group/g2/read": true, "group/g2/write": true}
	for _, g := range grants {
		key := g.PrincipalType + "/" + g.PrincipalID + "/" + g.Permission
		if !want[key] {
			t.Fatalf("unexpected grant %q in %+v", key, grants)
		}
	}
}

func TestAccessControlToGrantsNil(t *testing.T) {
	grants := accessControlToGrants(nil)
	if grants == nil || len(grants) != 0 {
		t.Fatalf("expected empty non-nil slice, got %+v", grants)
	}
}

func TestGrantsToAccessControl(t *testing.T) {
	grants := []accessGrant{
		{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
		{PrincipalType: "group", PrincipalID: "g2", Permission: "write"},
		{PrincipalType: "user", PrincipalID: "u1", Permission: "read"},
		{PrincipalType: "anyone", PrincipalID: "*", Permission: "read"},
	}
	ac := grantsToAccessControl(grants)
	read, ok := ac["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read to be map[string]any, got %T", ac["read"])
	}
	write, ok := ac["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write to be map[string]any, got %T", ac["write"])
	}
	if !reflect.DeepEqual(read["group_ids"], []string{"g1"}) {
		t.Fatalf("read.group_ids = %+v", read["group_ids"])
	}
	if !reflect.DeepEqual(read["user_ids"], []string{"u1"}) {
		t.Fatalf("read.user_ids = %+v", read["user_ids"])
	}
	if !reflect.DeepEqual(write["group_ids"], []string{"g2"}) {
		t.Fatalf("write.group_ids = %+v", write["group_ids"])
	}
}

func TestGrantsToAccessControlEmpty(t *testing.T) {
	if ac := grantsToAccessControl(nil); ac != nil {
		t.Fatalf("expected nil, got %+v", ac)
	}
	onlyAnyone := []accessGrant{{PrincipalType: "anyone", PrincipalID: "*", Permission: "read"}}
	if ac := grantsToAccessControl(onlyAnyone); ac != nil {
		t.Fatalf("expected nil for an anyone-only list, got %+v", ac)
	}
	onlyWildcardGroup := []accessGrant{{PrincipalType: "group", PrincipalID: "*", Permission: "read"}}
	if ac := grantsToAccessControl(onlyWildcardGroup); ac != nil {
		t.Fatalf("expected nil for a wildcard group, got %+v", ac)
	}
}

func TestGrantsToAccessControlPublic(t *testing.T) {
	grants := []accessGrant{
		{PrincipalType: "user", PrincipalID: "*", Permission: "read"},
		{PrincipalType: "user", PrincipalID: "*", Permission: "write"},
	}
	ac := grantsToAccessControl(grants)
	if ac["public_read"] != true {
		t.Fatalf("public_read = %+v, want true", ac["public_read"])
	}
	if ac["public_write"] != true {
		t.Fatalf("public_write = %+v, want true", ac["public_write"])
	}
}

func TestGrantsToAccessControlKeepsConcreteGrantsAlongsidePublic(t *testing.T) {
	grants := []accessGrant{
		{PrincipalType: "user", PrincipalID: "*", Permission: "read"},
		{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
		{PrincipalType: "user", PrincipalID: "u1", Permission: "write"},
		{PrincipalType: "anyone", PrincipalID: "*", Permission: "read"},
	}
	ac := grantsToAccessControl(grants)

	read, ok := ac["read"].(map[string]any)
	if !ok {
		t.Fatalf("expected read to be map[string]any, got %T", ac["read"])
	}
	write, ok := ac["write"].(map[string]any)
	if !ok {
		t.Fatalf("expected write to be map[string]any, got %T", ac["write"])
	}
	if !reflect.DeepEqual(read["group_ids"], []string{"g1"}) {
		t.Fatalf("read.group_ids = %+v", read["group_ids"])
	}
	if !reflect.DeepEqual(write["user_ids"], []string{"u1"}) {
		t.Fatalf("write.user_ids = %+v", write["user_ids"])
	}
	if ac["public_read"] != true {
		t.Fatalf("public_read = %+v, want true", ac["public_read"])
	}
	if ac["public_write"] != false {
		t.Fatalf("public_write = %+v, want false", ac["public_write"])
	}
}

func TestAccessControlToGrantsPublic(t *testing.T) {
	ac := map[string]any{
		"read":         map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
		"public_read":  true,
		"public_write": true,
	}
	grants := accessControlToGrants(ac)

	got := map[string]bool{}
	for _, g := range grants {
		got[g.PrincipalType+"/"+g.PrincipalID+"/"+g.Permission] = true
	}
	for _, want := range []string{"user/*/read", "user/*/write", "group/g1/read"} {
		if !got[want] {
			t.Fatalf("missing grant %q in %+v", want, grants)
		}
	}
	if len(grants) != 3 {
		t.Fatalf("expected 3 grants, got %d: %+v", len(grants), grants)
	}
}

func TestAccessControlToGrantsPublicFalse(t *testing.T) {
	ac := map[string]any{
		"read":         map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
		"public_read":  false,
		"public_write": false,
	}
	grants := accessControlToGrants(ac)
	if len(grants) != 1 {
		t.Fatalf("expected only the group grant, got %+v", grants)
	}
}

func TestAccessGrantsPublicRoundTrip(t *testing.T) {
	ac := map[string]any{
		"read":         map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
		"write":        map[string]any{"group_ids": []string{}, "user_ids": []string{"u1"}},
		"public_read":  true,
		"public_write": false,
	}
	back := grantsToAccessControl(accessControlToGrants(ac))
	if !reflect.DeepEqual(back, ac) {
		t.Fatalf("round trip changed access_control:\n got %+v\nwant %+v", back, ac)
	}
}
