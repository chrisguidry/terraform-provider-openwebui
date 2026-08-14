package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

// fakeSkillAPI serves the skill routes against an in-memory store. It answers
// the create route without the skill body, the way Open WebUI does, so a test
// that sees content in state proves the resource read the skill back.
type fakeSkillAPI struct {
	mu     sync.Mutex
	skills map[string]map[string]any
}

func newFakeSkillAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := &fakeSkillAPI{skills: map[string]map[string]any{}}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return server
}

func (a *fakeSkillAPI) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	w.Header().Set("Content-Type", "application/json")

	switch {
	case path == "skills/create":
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, _ := form["id"].(string)
		a.skills[id] = a.store(id, form)
		_ = json.NewEncoder(w).Encode(a.summary(id))
	case strings.HasSuffix(path, "/update"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "skills/id/"), "/update")
		var form map[string]any
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, ok := a.skills[id]; !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		a.skills[id] = a.store(id, form)
		_ = json.NewEncoder(w).Encode(a.skills[id])
	case strings.HasSuffix(path, "/delete"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "skills/id/"), "/delete")
		delete(a.skills, id)
		_, _ = w.Write([]byte(`true`))
	case strings.HasPrefix(path, "skills/id/"):
		id := strings.TrimPrefix(path, "skills/id/")
		skill, ok := a.skills[id]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		record := map[string]any{"write_access": true}
		for key, value := range skill {
			record[key] = value
		}
		_ = json.NewEncoder(w).Encode(record)
	default:
		http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
	}
}

// store keeps the fields the API persists, defaulting the ones the form leaves out.
func (a *fakeSkillAPI) store(id string, form map[string]any) map[string]any {
	isActive, ok := form["is_active"].(bool)
	if !ok {
		isActive = true
	}

	grants, ok := form["access_grants"].([]any)
	if !ok {
		grants = []any{}
	}

	return map[string]any{
		"id":            id,
		"user_id":       "u1",
		"name":          form["name"],
		"description":   form["description"],
		"content":       form["content"],
		"meta":          form["meta"],
		"is_active":     isActive,
		"access_grants": grants,
		"created_at":    1,
		"updated_at":    2,
	}
}

// summary answers the way the create route does, with no content field.
func (a *fakeSkillAPI) summary(id string) map[string]any {
	record := map[string]any{}
	for key, value := range a.skills[id] {
		if key == "content" {
			continue
		}
		record[key] = value
	}
	return record
}

func testSkillProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "openwebui" {
  endpoint = %q
  token    = "test-token"
}
`, endpoint)
}

func testSkillResourceConfig(endpoint, name, content string) string {
	return fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id    = "code-review"
  name        = %q
  description = "Reviews a diff"
  content     = %q
  tags        = ["review", "code"]
  public_read = true
}
`, testSkillProviderConfig(endpoint), name, content)
}

func TestSkillResource_ReadsContentBackAndImports(t *testing.T) {
	server := newFakeSkillAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testSkillResourceConfig(server.URL, "Code Review", "# Code review\n"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "id", "code-review"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "skill_id", "code-review"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", "Code Review"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "content", "# Code review\n"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "tags.0", "review"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "is_active", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "public_read", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "public_write", "false"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "write_access", "true"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "user_id", "u1"),
				),
			},
			{
				// The imported state carries no configuration, so content here can
				// only have come from the read.
				ResourceName:      "openwebui_skill.test",
				ImportState:       true,
				ImportStateId:     "code-review",
				ImportStateVerify: true,
			},
			{
				Config: testSkillResourceConfig(server.URL, "Code Review v2", "# Code review, revised\n"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_skill.test", "skill_id", "code-review"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "name", "Code Review v2"),
					resource.TestCheckResourceAttr("openwebui_skill.test", "content", "# Code review, revised\n"),
				),
			},
		},
	})
}

func TestSkillResource_RejectsIDTheServerWouldRewrite(t *testing.T) {
	server := newFakeSkillAPI(t)

	config := fmt.Sprintf(`%s
resource "openwebui_skill" "test" {
  skill_id = "Code Review"
  name     = "Code Review"
  content  = "# Code review"
}
`, testSkillProviderConfig(server.URL))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`must be lowercase and must not contain spaces`),
			},
		},
	})
}

func TestSkillResponseToModel_UsesResponseContent(t *testing.T) {
	ctx := context.Background()
	description := "Reviews a diff"
	writeAccess := true
	access := &client.SkillAccessResponse{
		ID:          "code-review",
		UserID:      "u1",
		Name:        "Code Review",
		Description: &description,
		Content:     "# From the server",
		Meta:        client.SkillMeta{Tags: []string{"review"}},
		IsActive:    true,
		AccessControl: map[string]any{
			"read":         map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"write":        map[string]any{"group_ids": []string{}, "user_ids": []string{}},
			"public_read":  true,
			"public_write": false,
		},
		CreatedAt:   1,
		UpdatedAt:   2,
		WriteAccess: &writeAccess,
	}

	state, diags := skillResponseToModel(ctx, nil, access, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.Content.ValueString() != "# From the server" {
		t.Fatalf("expected the response content, got %q", state.Content.ValueString())
	}
	if state.SkillID.ValueString() != "code-review" || state.ID.ValueString() != "code-review" {
		t.Fatalf("expected both identifiers to hold the API id, got %+v", state)
	}
	if !state.PublicRead.ValueBool() || state.PublicWrite.ValueBool() {
		t.Fatalf("expected public_read=true and public_write=false, got %v/%v", state.PublicRead, state.PublicWrite)
	}
	if !state.WriteAccess.ValueBool() {
		t.Fatalf("expected write_access=true, got %v", state.WriteAccess)
	}

	var tags []string
	if err := state.Tags.ElementsAs(ctx, &tags, false); err != nil {
		t.Fatalf("ElementsAs tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "review" {
		t.Fatalf("unexpected tags: %v", tags)
	}
}

func TestSkillResponseToModel_FallsBackToPlanContent(t *testing.T) {
	ctx := context.Background()
	access := &client.SkillAccessResponse{ID: "s1", UserID: "u1", Name: "S", IsActive: true}

	state, diags := skillResponseToModel(ctx, nil, access, types.StringValue("# From the plan"))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if state.Content.ValueString() != "# From the plan" {
		t.Fatalf("expected the plan content, got %q", state.Content.ValueString())
	}
	if state.PublicRead.ValueBool() || state.PublicWrite.ValueBool() {
		t.Fatalf("expected both public flags false without grants, got %v/%v", state.PublicRead, state.PublicWrite)
	}
	if !state.WriteAccess.IsNull() {
		t.Fatalf("expected write_access null when the API omits it, got %v", state.WriteAccess)
	}
}

func TestSkillFormFromPlan_CarriesPublicGrants(t *testing.T) {
	ctx := context.Background()
	plan := skillResourceModel{
		SkillID:     types.StringValue("code-review"),
		Name:        types.StringValue("Code Review"),
		Description: types.StringValue("Reviews a diff"),
		Content:     types.StringValue("# Code review"),
		Tags:        types.ListValueMust(types.StringType, []attr.Value{types.StringValue("review")}),
		IsActive:    types.BoolValue(false),
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
		PublicRead:  types.BoolValue(true),
		PublicWrite: types.BoolValue(false),
	}

	form, diags := skillFormFromPlan(ctx, nil, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if form.ID != "code-review" || form.Content != "# Code review" {
		t.Fatalf("unexpected form: %+v", form)
	}
	if form.IsActive == nil || *form.IsActive {
		t.Fatalf("expected is_active=false on the form, got %v", form.IsActive)
	}
	if len(form.Meta.Tags) != 1 || form.Meta.Tags[0] != "review" {
		t.Fatalf("unexpected tags on the form: %+v", form.Meta.Tags)
	}
	if form.AccessControl["public_read"] != true || form.AccessControl["public_write"] != false {
		t.Fatalf("expected the public flags on access_control, got %+v", form.AccessControl)
	}
}

func TestSkillFormFromPlan_NoAccessControlWithoutSharing(t *testing.T) {
	ctx := context.Background()
	plan := skillResourceModel{
		SkillID:     types.StringValue("s1"),
		Name:        types.StringValue("S"),
		Content:     types.StringValue("x"),
		Description: types.StringNull(),
		Tags:        types.ListNull(types.StringType),
		IsActive:    types.BoolNull(),
		ReadGroups:  types.ListNull(types.StringType),
		WriteGroups: types.ListNull(types.StringType),
		PublicRead:  types.BoolValue(false),
		PublicWrite: types.BoolValue(false),
	}

	form, diags := skillFormFromPlan(ctx, nil, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if form.AccessControl != nil {
		t.Fatalf("expected no access_control for a private skill, got %+v", form.AccessControl)
	}
	if form.Description != nil {
		t.Fatalf("expected no description on the form, got %v", *form.Description)
	}
	if form.IsActive != nil {
		t.Fatalf("expected is_active omitted when unset, got %v", *form.IsActive)
	}
}
