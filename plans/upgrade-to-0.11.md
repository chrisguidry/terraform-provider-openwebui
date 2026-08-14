# Upgrade the provider to Open WebUI v0.11

This provider was written against Open WebUI v0.9.0 to v0.9.6. We run
v0.11.0. This plan brings every resource and data source the provider
already ships up to v0.11.0, and adds Skills.

The required bar is two things:

1. Everything the provider already covers works on v0.11.0.
2. Skills work: a resource, a data source, and `skill_ids` on the model
   resource.

Everything in "Optional new surface" is below that bar. Do not start it
until the required sections pass.

## Evidence base

All backend claims in this plan name a file and a tag. The tags are
`v0.9.6` and `v0.11.0` in `github.com/open-webui/open-webui`. To check a
claim:

```bash
git clone --filter=blob:none https://github.com/open-webui/open-webui.git
cd open-webui
git diff v0.9.6 v0.11.0 -- backend/open_webui/routers/ backend/open_webui/models/
```

Two facts frame the whole upgrade:

- **No route the provider calls was removed.** A full route table
  extracted from `backend/open_webui/routers/*.py` at both tags shows one
  deletion across the entire backend, `POST /api/v1/tasks/active/chats`,
  which the provider never called. Every other difference is an addition.
  The upgrade is therefore about payload shapes and server-side
  behavior, not about moved endpoints.
- **Trailing slashes still matter.** `main.py` at v0.11.0 mounts the
  single-page app at `/` with `html=True` (line 2874). A path that
  matches no route falls through to that mount and returns the HTML
  shell with status 200, not a 404. `routers/models.py` at v0.11.0 keeps
  the comment `do NOT use "/" as path, conflicts with main.py` on the
  `/list` route. A wrong slash gives a JSON decode error, never a clean
  failure.

## How to work this plan

Sections 1 through 9 are the required work. Their dependencies:

- **Section 1 (test harness) blocks everything.** Do it first. No other
  section can be proven without it.
- **Section 2 (access grants) blocks sections 3 and 4**, because both
  touch the grant plumbing.
- **Sections 3, 4, 5, 6, 7, and 8 are independent of each other** once 1
  and 2 land. Six agents can take one each.
- **Section 9 (docs and README) depends on all of them.** Do it last.

Sections 10 onward are Part two, complete v0.11.0 coverage:

- **Section 1 still blocks everything.** Nothing in Part two can be
  proven without the throwaway container.
- **Every Part two section is independent of every other one**, with two
  exceptions: section 11.2 reuses the list-splice client helper and the
  mutex from section 10, and section 15.4 shares
  `permissions_helpers.go` with section 7.
- **Section 16.4 suggests an order** by value, not by dependency.
- Section 9 grows to cover the new resources. Still last.

Do not commit. Do not stash. Do not amend. Never use `--no-verify`.
Other agents share this tree, and four of them are working in
`internal/` right now. Part two touches files none of them have open, but
check `git status` before you start.

## Verdict index

Every resource and data source the provider ships today, with its
verdict against v0.11.0 and the section that covers it. Nothing was
removed from the API, so no item is REMOVED.

| Resource | Verdict | Section |
| --- | --- | --- |
| `openwebui_model` | CHANGED | 3 |
| `openwebui_knowledge` | UNCHANGED | 6.1 |
| `openwebui_knowledge_file` | CHANGED | 6.2 |
| `openwebui_file` | UNCHANGED | 6.3 |
| `openwebui_prompt` | UNCHANGED | 5.4 |
| `openwebui_group` | CHANGED | 7.1 |
| `openwebui_tool` | CHANGED | 5.1 |
| `openwebui_tool_valves` | UNCHANGED | 5.2 |
| `openwebui_function` | CHANGED | 5.5 |
| `openwebui_function_valves` | UNCHANGED | 5.2 |
| `openwebui_pipeline` | UNCHANGED | 8.3 |
| `openwebui_pipeline_valves` | UNCHANGED | 8.3 |
| `openwebui_config_import` | CHANGED | 8.1 |
| `openwebui_oauth_client` | CHANGED | 8.2 |
| `openwebui_connections_config` | UNCHANGED | 8.3 |
| `openwebui_tool_servers_config` | UNCHANGED | 8.3 |
| `openwebui_code_execution_config` | UNCHANGED | 8.3 |
| `openwebui_models_config` | UNCHANGED | 8.3 |
| `openwebui_suggestions_config` | UNCHANGED | 8.3 |
| `openwebui_banners_config` | UNCHANGED | 8.3 |
| `openwebui_openai_connections` | UNCHANGED | 8.3 |
| `openwebui_ollama_connections` | UNCHANGED | 8.3 |
| `openwebui_skill` | NEW | 4 |

| Data source | Verdict | Section |
| --- | --- | --- |
| `openwebui_model` | CHANGED | 3.5 |
| `openwebui_knowledge` | UNCHANGED | 6.1 |
| `openwebui_file` | UNCHANGED | 6.3 |
| `openwebui_files` | UNCHANGED | 6.3 |
| `openwebui_prompt` | UNCHANGED | 5.4 |
| `openwebui_group` | UNCHANGED | 7.2 |
| `openwebui_tool` | CHANGED | 5.3 |
| `openwebui_pipeline` | UNCHANGED | 8.3 |
| `openwebui_user` | UNCHANGED | 7.3 |
| `openwebui_config_export` | CHANGED | 8.1 |
| `openwebui_tool_server_verify` | UNCHANGED | 8.3 |
| `openwebui_openai_connection_verify` | UNCHANGED | 8.3 |
| `openwebui_ollama_connection_verify` | UNCHANGED | 8.3 |
| `openwebui_skill` | NEW | 4 |

Every UNCHANGED verdict means the endpoints and payloads match. Several
UNCHANGED items still carry pre-existing defects worth fixing while the
section is open; those are called out in sections 5.4, 6.4, 7.3, and 8.4.

### New resources and data sources in Part two

Part two extends the fork to every declarative surface v0.11.0 exposes.
Sixteen new resources, on top of `openwebui_skill` from section 4, plus
one optional seventeenth.

| New resource | Section |
| --- | --- |
| `openwebui_tool_server` | 10 |
| `openwebui_subagents_config` | 11.1 |
| `openwebui_terminal_server` | 11.2 |
| `openwebui_channel` | 12 |
| `openwebui_chat_config` | 13 |
| `openwebui_rag_embedding_config` | 14.2 |
| `openwebui_rag_config` | 14.3 |
| `openwebui_images_config` | 14.4 |
| `openwebui_audio_config` | 14.5 |
| `openwebui_admin_config` | 15.1 |
| `openwebui_ldap_config` | 15.2 |
| `openwebui_oauth_config` | 15.3 |
| `openwebui_default_user_permissions` | 15.4 |
| `openwebui_user` (promoted from data source) | 15.5 |
| `openwebui_task_config` | 15.6 |
| `openwebui_evaluation_config` | 15.7 |
| `openwebui_external_knowledge_connection` (optional) | 16.2 |

Optional new data sources: `openwebui_tool_server` (10.8),
`openwebui_channel` (12.4), `openwebui_config_namespace` (11.3),
`openwebui_images_verify`, `openwebui_images_models`,
`openwebui_audio_models`, `openwebui_audio_voices` (14.6), and
`openwebui_default_user_permissions_defaults` (15.4).

**Section 16 carries the router-level index**: all 31 backend routers
with a verdict of COVERED, NEW RESOURCE, MIXED, or EXCLUDED, and a
written reason for every exclusion. That table, not this one, is what
makes complete coverage a checkable claim.

---

## 1. Test harness: a throwaway v0.11.0 in docker

**Depends on: nothing. Blocks: every other section.**

The family's production instance is never a test target. The acceptance
suite creates, mutates, and deletes real objects. It must only ever point
at a container that gets thrown away.

### The bootstrap flow, from the backend code

The flow below is read off `backend/open_webui/routers/auths.py` at
v0.11.0. Each step names the line that proves it.

1. **Start the container empty.** `Users.has_users()` is false, so
   `signup` (line 885) skips the `ui.enable_signup` gate. The comment at
   line 898 states this directly: the first admin is not gated on
   `ENABLE_SIGNUP`.
2. **`POST /api/v1/auths/signup`** with `{name, email, password}`.
   `signup_handler` (line 827) promotes the user to admin at line 863,
   with the comment at line 861 confirming only the single user present
   at that point becomes admin. The response model is
   `SessionUserResponse` (line 235), which extends `Token`, so the body
   carries a JWT in its `token` field.
3. **`POST /api/v1/auths/api_key`** with `Authorization: Bearer <JWT>`.
   The handler is at line 1466 and returns `{"api_key": "sk-..."}`.

### The setting that blocks step 3

`_check_api_key_permission` (`routers/auths.py` at v0.11.0, line 1454)
raises 403 unless `auth.enable_api_keys` is true. That value is seeded
from `ENABLE_API_KEYS`, which **defaults to `False`**
(`backend/open_webui/config.py` at v0.11.0, line 2420). The default was
also `False` at v0.9.6, so this is not a regression, but the container
must set `ENABLE_API_KEYS=true` or step 3 returns 403.

Config is read from the database at v0.11.0 (`await
Config.get('auth.enable_api_keys')`), so the environment variable only
seeds a fresh database. Always start from an empty volume.

`validate_password` (`backend/open_webui/utils/auth.py` at v0.11.0, line
180) only enforces a pattern when `ENABLE_PASSWORD_VALIDATION` is set,
which is off by default. A simple test password is fine.

### Files to add

Add `scripts/testacc-up.sh` and `scripts/testacc-down.sh` at the repo
root, plus two Makefile targets.

`scripts/testacc-up.sh` does this:

1. `docker rm -f openwebui-testacc` and delete the volume, so every run
   starts from an empty database.
2. `docker run -d --name openwebui-testacc -p 8080:8080` with image
   `ghcr.io/open-webui/open-webui:v0.11.0` (the repo's own
   `docker-compose.yaml` uses `ghcr.io/open-webui/open-webui` on
   container port 8080) and environment:
   - `ENABLE_API_KEYS=true`. Required, see above.
   - `WEBUI_SECRET_KEY=terraform-acc-test`, for stable JWT signing.
   - `WEBUI_AUTH=True`, the default (`backend/open_webui/env.py` at
     v0.11.0, line 705); stated for clarity.
   - `OFFLINE_MODE=true` and `RAG_EMBEDDING_ENGINE=""` if the first boot
     is slow pulling embedding models.
3. Poll `GET /health` with a bounded timeout. Fail with a clear message
   if the container is not up within the timeout. Do not loop forever.
4. `POST /api/v1/auths/signup` with a fixed test identity, and read
   `.token` from the response.
5. `POST /api/v1/auths/api_key` with that JWT, and read `.api_key`.
6. Write `OPENWEBUI_ENDPOINT=http://localhost:8080` and
   `OPENWEBUI_TOKEN=<api_key>` to a `.env` file at the repo root. The
   Makefile already does `-include .env` and `export`, so `make testacc`
   picks them up with no further wiring.

Add a guard to the script: refuse to run if `.env` already names an
endpoint that is not `localhost` or `127.0.0.1`. This is the mechanical
protection against pointing the suite at production.

`scripts/testacc-down.sh` removes the container and its volume, and
deletes `.env`.

Makefile targets:

```make
testacc-up:
	./scripts/testacc-up.sh

testacc-down:
	./scripts/testacc-down.sh
```

`make testacc` already runs `go test ./internal/provider/... -v -count=1
-timeout 30m`. It needs `TF_ACC=1` too, which `testacc-up.sh` writes into
`.env` alongside the endpoint and token.

`testAccPreCheck` in `internal/provider/acceptance_test.go` already skips
when `TF_ACC`, `OPENWEBUI_TOKEN`, or `OPENWEBUI_ENDPOINT` is unset, so
the suite stays inert for anyone who has not run `testacc-up`.

### What to prove before moving on

Run `make testacc-up && make testacc` on the unmodified provider and
record which tests fail. That failure list is the real work list for
sections 3 through 8. Attach it to this plan as a comment in the PR.

---

## 2. Access grants: the shared wire plumbing

**Depends on: section 1. Blocks: sections 3 and 4.**

`internal/client/access_grants.go` converts between the provider's nested
`access_control` map and the API's flat `access_grants` list. The flat
list is already correct for v0.11.0. Two changes are needed.

### 2.1 The `anyone` principal type

`backend/open_webui/models/access_grants.py` at v0.11.0 adds a third
principal type. `PRINCIPAL_TYPE_ANYONE = 'anyone'` (line 14), and
`normalize_access_grants` accepts it only with `principal_id` of `*` and
`permission` of `read`. At v0.9.6 the same function accepted only `user`
and `group`.

The provider does not need to *write* `anyone` grants.
`filter_allowed_access_grants`
(`backend/open_webui/utils/access_control/__init__.py` at v0.11.0, line
220) strips every `anyone` grant unless the caller passes
`anyone_permission_key`, and no router in
`backend/open_webui/routers/` passes it at v0.11.0. So an `anyone` grant
sent to any endpoint this provider calls is discarded server-side.

The provider does need to *tolerate reading* them.
`grantsToAccessControl` in `internal/client/access_grants.go` hits its
`default: continue` branch for an unknown `principal_type`, so an
`anyone` grant is dropped silently. That is the correct outcome, but it
is currently accidental. Make it deliberate: add `case "anyone":
continue` with a comment naming why, so a future reader does not "fix"
the default branch into an error.

### 2.2 Wildcard grants are silently revoked

**Decided: model it. The client half is done.**
`internal/client/access_grants.go` now carries the wildcard grant as the
booleans `public_read` and `public_write` in the `access_control` map, and
`internal/client/access_grants_test.go` covers the round trip. A resource
that exposes `read_groups` / `write_groups` still needs its own
`public_read` / `public_write` attributes, and needs to set those two keys
on the map it hands the client. Until it does, the resource keeps dropping
wildcard grants on write.

This is a pre-existing correctness bug that v0.11 makes easier to hit.

`grantsToAccessControl` skips any grant whose `principal_id` is `*`
(the `g.PrincipalID == "*"` guard). The provider then computes the next
write from `access_control` alone, so a resource shared publicly through
the web UI loses that sharing on the next `terraform apply`, with no
plan diff to warn the operator.

Decide and record one of two answers:

- **Model it.** Add a `public_read` / `public_write` boolean pair to the
  resources that carry `access_control`, mapping to a `user` / `*`
  grant. This makes the revocation visible in the plan.
- **Refuse it.** Keep dropping wildcard grants on read, but have the
  write path preserve any wildcard grant already present on the server
  rather than dropping it.

Modelling it is the better answer for a Terraform provider, because
silent state loss is worse than an extra attribute. Implement it in
`internal/client/access_grants.go` and in each resource that exposes
`read_groups` / `write_groups`.

### 2.3 Tests

`internal/client/access_grants_test.go` gets cases for: an `anyone` grant
round-trips to nothing, a `user` / `*` grant maps to whichever answer
2.2 picks, and a mixed list keeps concrete group and user grants intact.

---

## 3. The model resource and data source

**Depends on: sections 1 and 2. Independent of sections 4 through 8.**

Verdict: **CHANGED**, in three ways that cause drift, plus the
`skill_ids` addition from section 4.

Files: `internal/provider/resource_model.go`,
`internal/provider/data_source_model.go`, `internal/client/models.go`.

`ModelForm` itself is unchanged. `backend/open_webui/models/models.py` at
v0.11.0 (line 172) still declares exactly `id`, `base_model_id`, `name`,
`meta`, `params`, `access_grants`, and `is_active`. The endpoints
`models/create`, `models/model`, `models/model/update`, and
`models/model/delete` all still exist with the same methods. The breakage
is entirely in how the server treats `meta`.

Background on why `meta` is fragile here: `resource_model.go` maps the
meta keys it maps (`profile_image_url`, `description`, `hidden`,
`capabilities`, `suggestion_prompts`, `tags`, `toolIds`,
`defaultFeatureIds`) onto typed attributes, deleting each from a working
copy as it goes, and serialises whatever is left into
`meta_additional_json` (see the `delete(additional, ...)` calls from line
1097 onward). Any meta key the server adds and the config never sets
lands in `meta_additional_json` and produces a permanent diff.

### 3.1 `meta.chat_variables_schema` is injected on read

`backend/open_webui/routers/models.py` at v0.11.0 adds
`add_chat_variables_schema` (line 52), which derives a schema from
`params.system` and writes it to `model_dict['meta']
['chat_variables_schema']`. It is called on `GET /models/model` (in
`get_model_by_id`) and on `GET /models/list`. Neither call existed at
v0.9.6.

The key is computed by the server and is never part of the request. With
the current code it lands in `meta_additional_json`, so **every model
with a system prompt shows a permanent diff**. This is the single most
disruptive change in the upgrade.

Fix: in the meta flatten function in `resource_model.go`, add
`delete(additional, "chat_variables_schema")` alongside the existing
unconditional deletes for `profile_image_url`, `description`, and
`hidden`. Do not expose it as an attribute; it is derived state, not
configuration. Apply the same delete in `data_source_model.go`.

### 3.2 `base_model_id` and `meta.profile_image_url` are now sticky

`backend/open_webui/routers/models.py` at v0.11.0, in
`update_model_by_id`:

```python
if 'base_model_id' not in form_data.model_fields_set:
    form_data.base_model_id = model.base_model_id

if 'profile_image_url' not in form_data.meta.model_fields_set:
    form_data.meta.profile_image_url = model.meta.profile_image_url
```

Neither block exists at v0.9.6. The server now carries the stored value
forward when the field is absent from the request, so **removing
`base_model_id` or `profile_image_url` from the Terraform config no
longer clears it**. The plan shows the removal, the apply reports
success, and the value stays.

Fix: the provider must send an explicit `null` rather than omitting the
field. In `internal/client/models.go`, `ModelForm.BaseModelID` is
`*string` with `json:"base_model_id,omitempty"`. The `omitempty` is what
makes the field absent. Drop `omitempty` so a nil pointer serialises as
`base_model_id: null`, which is present in `model_fields_set` and
therefore overwrites. Do the same for `profile_image_url` inside the meta
map: write the key with a nil value rather than not writing it.

Add an acceptance test that sets `base_model_id`, then removes it, and
asserts the read-back is empty.

### 3.3 `meta.knowledge` is rewritten server-side

`backend/open_webui/models/models.py` at v0.11.0 promotes `knowledge` to
a declared field on `ModelMeta` (line 73) with a `strip_knowledge_content`
validator (line 91) backed by
`strip_extracted_content_from_model_knowledge` (line 26). It removes
`data.content` and `file.data.content` from every entry. Worse,
`_to_model_model` (line 194) rewrites and commits the stored meta when
stripping changed anything, so the server mutates the row on read.

The provider has no `knowledge` attribute, so any value a user sets lands
in `meta_additional_json` and will not survive the round trip. Document
this in the `meta_additional_json` schema description: extracted content
under `knowledge` is stripped by the server and must not be set. If a
first-class `knowledge` attribute is wanted later, it belongs in the
optional section, not here.

### 3.4 `skill_ids`

`backend/open_webui/utils/middleware.py` at v0.11.0, line 2611, reads
`model.get('info', {}).get('meta', {}).get('skillIds', [])`. That is the
model-to-skill link, exactly parallel to `toolIds`.

`ModelMeta` is `ConfigDict(extra='allow')`
(`backend/open_webui/models/models.py` at v0.11.0, line 74), so
`skillIds` passes through as an extra with no backend change needed.

Add to `resource_model.go`:

- `SkillIDs types.List` on the model struct, tag `tfsdk:"skill_ids"`.
- A `skill_ids` `schema.ListAttribute` of `types.StringType`, `Optional`,
  described as the skills attached to this model, mirroring the existing
  `tool_ids` attribute at line 225.
- Expansion: `meta["skillIds"] = skills` when set, `[]any{}` when empty,
  mirroring the `toolIds` block at line 805.
- Flattening: a `data["skillIds"]` block with `toStringSlice` and
  `delete(additional, "skillIds")`, mirroring the `toolIds` block at line
  1151.

Mirror the read side into `data_source_model.go`.

### 3.5 Data source

`data_source_model.go` calls `GetModel`, which hits
`GET /api/v1/models/model?id=...`. That route is unchanged. It needs the
same `chat_variables_schema` delete and the same `skill_ids` flattening.

One thing to know but not to fix: `get_model_by_id` blanks `params` for
callers without write access. That behavior predates v0.11.0 and the
provider authenticates as an admin, so it does not apply.

---

## 4. Skills

**Depends on: sections 1 and 2. Independent of sections 3 and 5 through 8.**

Verdict: **NEW**. The provider has no skills support.

Skills are not new in v0.11.0. `backend/open_webui/routers/skills.py` and
`backend/open_webui/models/skills.py` both exist at v0.9.6, and the diff
between the tags changes only internals: event publishing, the config
lookup moving to `await Config.get(...)`, `order_by` and `direction`
query parameters on `/list`, and a split of the `workspace.skills`
permission check into `workspace.skills_import` and
`workspace.skills_export`. The request and response models are
unchanged. The provider simply never covered them. That stability is good
news: the target is not moving.

The router mounts at `/api/v1/skills` (`backend/open_webui/main.py` at
v0.11.0, line 812).

### 4.1 The wire contract

From `backend/open_webui/models/skills.py` at v0.11.0:

`SkillForm`, the create and update payload:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string, required | Lowercased and spaces replaced with `-` by the server, `routers/skills.py` line 168 |
| `name` | string, required | `UNIQUE` on the `skill` table, line 26 |
| `description` | string, optional | |
| `content` | string, required | |
| `meta` | `SkillMeta` | Only field is `tags: list[str]`, default `[]` |
| `is_active` | bool | Default `true` |
| `access_grants` | list of dict, optional | Flat grant list, same shape as models |

`SkillModel`, the full record, adds `user_id`, `created_at`, and
`updated_at`.

Routes, all from `backend/open_webui/routers/skills.py` at v0.11.0:

| Method and path | Response model | Line |
| --- | --- | --- |
| `GET /api/v1/skills/` | `list[SkillUserResponse]` | 37 |
| `GET /api/v1/skills/list` | `SkillAccessListResponse` | 70 |
| `GET /api/v1/skills/export` | `list[SkillModel]` | 128 |
| `POST /api/v1/skills/create` | `SkillResponse` | 155 |
| `GET /api/v1/skills/id/{id}` | `SkillAccessResponse` | 219 |
| `POST /api/v1/skills/id/{id}/update` | `SkillModel` | 260 |
| `POST /api/v1/skills/id/{id}/access/update` | `SkillModel` | 330 |
| `POST /api/v1/skills/id/{id}/toggle` | `SkillModel` | 375 |
| `DELETE /api/v1/skills/id/{id}/delete` | `bool` | 415 |

### 4.2 The `content` trap

`SkillResponse` (line 63) and `SkillUserResponse` (line 71) **do not
declare `content`**. Only `SkillModel` does. So `POST /create` and
`GET /id/{id}` are typed to return a record without the skill body.

`SkillUserResponse` carries `model_config = ConfigDict(extra='allow')`,
and the handlers build these objects with `**skill.model_dump()` from a
`SkillModel` that does have `content`, so in practice `content` should
survive as an extra field. **Verify this against the live container from
section 1 before building the read path on it.** If `content` does not
come back from `GET /id/{id}`, use `GET /api/v1/skills/export`, whose
response model is `list[SkillModel]` and therefore always carries
`content`, and select the matching id from it.

Note that `POST /create` returns `SkillResponse`, which does not extend
the `extra='allow'` class, so treat the create response as untrustworthy
for `content` regardless and re-read after create.

This is the one open question in the plan. Resolve it first. It determines
the shape of the client read method.

### 4.3 New files

`internal/client/skills.go`:

```go
type SkillForm struct {
    ID           string         `json:"id"`
    Name         string         `json:"name"`
    Description  *string        `json:"description,omitempty"`
    Content      string         `json:"content"`
    Meta         map[string]any `json:"meta"`
    IsActive     *bool          `json:"is_active,omitempty"`
    AccessControl map[string]any `json:"-"`
}
```

with `MarshalJSON` emitting `access_grants` via `accessControlToGrants`,
exactly as `ModelForm` does in `internal/client/models.go`.

`SkillResponse` mirrors it plus `UserID`, `CreatedAt`, `UpdatedAt`, and
an `UnmarshalJSON` that decodes `access_grants` through
`grantsToAccessControl`.

Methods: `CreateSkill`, `GetSkill`, `UpdateSkill`, `DeleteSkill`,
`ListSkills`. Paths are relative to the client's `/api/v1` base, so
`"skills/create"`, `"skills/id/<escaped id>"`,
`"skills/id/<escaped id>/update"`, `"skills/id/<escaped id>/delete"` with
`http.MethodDelete`, and `"skills/"` for the list. Keep the trailing
slash on the list path: the route is declared as `'/'`.

`internal/client/skills_test.go` holds httptest cases for each method, plus a
grant round-trip case, following `internal/client/models_access_test.go`.

`internal/provider/resource_skill.go` uses the type name
`req.ProviderTypeName + "_skill"`. Schema:

| Attribute | Type | Notes |
| --- | --- | --- |
| `id` | string, computed | Terraform identity |
| `skill_id` | string, required, requires replace | The API id; the server lowercases it and replaces spaces with `-`, so add a validator rejecting anything the server would rewrite, rather than accepting a silent mismatch |
| `name` | string, required | Unique across the instance |
| `description` | string, optional | |
| `content` | string, required | |
| `tags` | list of string, optional | Maps to `meta.tags` |
| `is_active` | bool, optional, default true | |
| `read_groups` | list of string, optional | Mirrors the model resource |
| `write_groups` | list of string, optional | Mirrors the model resource |
| `user_id` | string, computed | |
| `created_at` | int64, computed | |
| `updated_at` | int64, computed | |

The `skill_id` naming follows the existing convention: the model resource
uses `model_id` (line 136) and the tool resource uses `tool_id`.

`internal/provider/data_source_skill.go` looks up by `skill_id`,
exposing the same fields as computed, following
`internal/provider/data_source_tool.go`.

Register both in `internal/provider/provider.go`: `NewSkillResource` in
the `Resources` slice near line 166, `NewSkillDataSource` in
`DataSources` near line 193.

### 4.4 Tests

`internal/provider/acc_skill_test.go`, following the shape of
`acc_tool_test.go`:

- Create a skill with content and tags, check the read-back.
- Update the name, description, and content in place, and check that
  `skill_id` did not change.
- Change `read_groups` and check the grants round-trip.
- Import by id and check the state matches.
- A data source step reading a skill created in the same config.

`internal/provider/acc_model_test.go` gains a step that sets `skill_ids`
on a model to a skill created in the same config, and asserts the
read-back. This is the check that ties sections 3 and 4 together, so run
it once both have landed.

---

## 5. Tools, prompts, and functions

**Depends on: section 1. Independent of sections 3, 4, 6, 7, and 8.**

Router prefixes are unchanged: `/api/v1/tools`, `/api/v1/prompts`, and
`/api/v1/functions` (`backend/open_webui/main.py` at v0.11.0, lines 811,
810, and 818). Every route in this group survives with the same method
and the same trailing slash.

### 5.1 `openwebui_tool`: CHANGED, additive only

`ToolForm` is byte-identical at both tags: `id`, `name`, `content`,
`meta`, `access_grants` (`backend/open_webui/models/tools.py`). Nothing
required to keep it working. Three changes worth handling:

- **`ToolMeta` gained `has_user_valves: bool = False`**
  (`backend/open_webui/models/tools.py` at v0.11.0, line 39). The server
  computes it on both write paths from `hasattr(tool_module,
  'UserValves')` (`backend/open_webui/routers/tools.py` at v0.11.0), so
  omitting it on write loses nothing. Add
  `HasUserValves *bool` with `json:"has_user_valves,omitempty"` to
  `client.ToolMeta` in `internal/client/tools.go` and a computed
  `has_user_valves` bool to the tool resource and data source. This is
  useful signal, not a fix.
- **`ToolModel.content` and `user_id` are now nullable**
  (`backend/open_webui/models/tools.py` at v0.11.0, lines 44 and 47;
  both were required `str` at v0.9.6). Go decodes JSON `null` into a
  `string` as `""` without error, so nothing breaks. It does mean a null
  `user_id` reaches state as `""` rather than null. Leave it.
- **`GET /api/v1/tools/id/{id}` drops `content` for callers without
  write access** (`backend/open_webui/routers/tools.py` at v0.11.0,
  `get_tools_by_id` calls `data.pop('content', None)`).
  `client.ToolAccessResponse` has no `Content` field, so there is no
  effect today. Do not add one without handling the absent case.
  `GET /api/v1/tools/export` still returns content, because
  `export_tools` calls `Tools.get_tools` without `defer_content`, so
  `fetchToolContent` keeps working.

### 5.2 `openwebui_tool_valves` and `openwebui_function_valves`: UNCHANGED

Both keep `GET /valves`, `GET /valves/spec`, and `POST /valves/update`
with a `dict | None` body. **Valve encryption is at rest only.**
`backend/open_webui/utils/valves.py` is new at v0.11.0, but
`get_tool_valves_by_id` returns `decrypt_valves(...)` and the update
route returns `valves.model_dump(exclude_unset=True)`, so the wire stays
plaintext in both directions.

One operational caveat to record in the resource documentation, not the
code. Encryption is off by default:
`ENABLE_VALVE_ENCRYPTION = os.getenv('ENABLE_VALVE_ENCRYPTION',
'False')` (`backend/open_webui/env.py` at v0.11.0, line 721; the name
does not exist at v0.9.6). If an operator turns it on and later changes
`WEBUI_SECRET_KEY`, `decrypt_valves` catches `InvalidToken` and returns
`{}`. The GET then answers `{}` with status 200, and the provider records
empty valves. That is a silent permanent diff rather than an error.

### 5.3 `openwebui_tool` data source: CHANGED, no action

Reads the same two endpoints as the resource. Both still serve
everything it maps.

### 5.4 `openwebui_prompt` and its data source: UNCHANGED

`PromptForm` and `PromptModel` are byte-identical at both tags
(`backend/open_webui/models/prompts.py`, class at line 83 in each). The
router diff adds only `publish_event` calls and moves the permissions
lookup to `await Config.get('user.permissions')`. `GET /api/v1/prompts/`
still returns `list[PromptModel]` with `content`.

A pre-existing bug surfaced while checking, unrelated to the version
bump: **`PromptForm` has no `is_active` field at either tag.** The
`IsActive *bool` that `internal/client/prompts.go` sends is dropped by
pydantic, which ignores extra fields by default. Setting `is_active =
false` on `openwebui_prompt` therefore does nothing. The only way to flip
it is `POST /api/v1/prompts/id/{command}/toggle`, which the provider
never calls. Fix it by converging the toggle when plan and server
disagree, in the style of `convergeToggles` in
`internal/provider/resource_function.go`. Raise this with Chris before
building it: it changes the behavior of an attribute that has always been
inert.

### 5.5 `openwebui_function`: CHANGED, minor

`FunctionForm` is unchanged: `id`, `name`, `content`, `meta`
(`backend/open_webui/models/functions.py` at both tags). Functions carry
no `access_grants` and no `access_control` at either tag, and there is no
`/functions/id/{id}/access/update` route, so the sharing migration does
not touch this resource at all.

- `user_id` is now nullable on `FunctionModel`,
  `FunctionWithValvesModel`, and `FunctionResponse`
  (`backend/open_webui/models/functions.py` at v0.11.0). Same harmless
  outcome as tools.
- `FunctionResponse`, the create response, still omits `content` and has
  no `extra='allow'`, so content cannot leak through. `Create` already
  re-reads via `GetFunction`. No change.
- `POST /id/{id}/toggle` now dispatches event functions and schedules
  webhooks before flipping `is_active`
  (`backend/open_webui/routers/functions.py` at v0.11.0). The request and
  response contracts are identical; the call is just slower.

### 5.6 The `ENABLE_PLUGINS` gate

`ENABLE_PLUGINS` is new at v0.11.0 (`backend/open_webui/env.py` at
v0.11.0, line 1112, default `True`; the name does not exist at v0.9.6).
When false, `GET /tools/`, `GET /tools/list`, `GET /functions/`,
`GET /functions/list`, and `GET /functions/export` all return empty lists
(`backend/open_webui/routers/tools.py` lines 75 and 205,
`routers/functions.py` lines 49, 57, and 74, all at v0.11.0).

None of those are on the resource paths, so the resources are safe. But
`ListTools` and `ListToolAccess` in `internal/client/tools.go` would
return empty rather than failing. Add a note to the tool data source
documentation. Do not add a runtime check; the provider cannot read the
setting.

---

## 6. Knowledge and files

**Depends on: section 1. Independent of sections 3, 4, 5, 7, and 8.**

Mount prefixes are unchanged: `/api/v1/knowledge` and `/api/v1/files`
(`backend/open_webui/main.py` at v0.11.0, lines 809 and 817). All fifteen
paths the Go client hits in this group exist at v0.11.0 with the same
method and the same trailing slash. Pagination shapes are unchanged:
`GET /knowledge/` still returns `{items, total}` with `PAGE_ITEM_COUNT =
30`, and `GET /files/` still returns `{items, total}`.

### 6.1 `openwebui_knowledge` and its data source: UNCHANGED

`KnowledgeForm` is byte-identical at both tags: `name`, `description`,
`access_grants` (`backend/open_webui/models/knowledge.py`). The handler
diffs are internal only: the permissions lookup moved to
`await Config.get('user.permissions')` and `publish_event` calls were
added. `KnowledgeResponse` and `KnowledgeFilesResponse` are identical.

One additive field: `KnowledgeUserModel` gained `file_count: int | None`
(`backend/open_webui/models/knowledge.py` at v0.11.0, line 154). Go
ignores unknown fields, so nothing breaks. Adding a computed
`file_count` to the data source is a small, obvious win. Take it.

### 6.2 `openwebui_knowledge_file`: CHANGED, response narrowed

The route, the form, and the `delete_file` query parameter are all
unchanged. `KnowledgeFileIdForm` is identical at both tags.

What changed: **`GET /knowledge/{id}/files` no longer returns each file's
`data`.** `KnowledgeTable.search_files_by_id` in
`backend/open_webui/models/knowledge.py` at v0.9.6 built each item from
`FileModel.model_validate(file).model_dump()`, which carried `data` (the
extracted content) and `path` through `FileModelResponse`'s
`extra='allow'`. At v0.11.0 the same method adds
`stmt = stmt.options(defer(File.data))` (line 445) and builds the
response from an explicit field list with no `data` and no `path`.

Effect on the provider: `FileUserResponse.Data` in
`internal/client/knowledge_files.go` (line 27) is now always nil.
`readKnowledgeFileState` in
`internal/provider/resource_knowledge_file.go` (line 234) serialises that
struct into the computed `file_json` attribute, so the first refresh
against v0.11.0 shows a `file_json` change on every existing
`openwebui_knowledge_file` where the `data` key disappears. It is state
churn, not an error, and `include_content=true` does not bring it back.

Fix: drop `Data` from `client.FileUserResponse` and describe `file_json`
in the schema as metadata only. Taking the one-time diff deliberately is
better than leaving a field that is structurally always nil.

### 6.3 `openwebui_file`, `openwebui_file` data source, and `openwebui_files` data source: UNCHANGED

`upload_file` has an identical signature at both tags: `file:
UploadFile`, `metadata: Optional[dict | str] = Form(None)`, `process:
bool = Query(True)`, `process_in_background: bool = Query(True)`
(`backend/open_webui/routers/files.py`). The query parameter names, the
`file` part, and the `metadata` form field all match
`internal/client/files.go`. `FileModel`, `FileModelResponse`,
`FileMeta`, and `FileMetadataResponse` in
`backend/open_webui/models/files.py` are unchanged between the tags.

`GET /files/search` still returns a bare list and still raises 404 on no
match, which `SearchFiles` maps to `ErrNotFound`.

Two behavior notes, neither requiring a code change:

- `POST /files/{id}/data/content/update` now raises 413 when the body
  exceeds `rag.file.max_size` MB (`backend/open_webui/routers/files.py`
  at v0.11.0, line 705). `UpdateFileContent` surfaces it as a generic
  `APIError`.
- `has_access_to_file` (`backend/open_webui/utils/access_control/files.py`
  at v0.11.0) now requires, for write and delete, that the knowledge base
  owner also own the file. The file routes short-circuit on
  `user.role == 'admin'`, so an admin token is unaffected. A non-admin
  token that could previously delete a file reachable only through a
  shared knowledge base now gets 404.

### 6.4 Pre-existing defects to fix while you are here

These are true at both tags. They are not caused by the upgrade, but this
section is the right place to fix them.

1. **`filesListPageSize` is wrong.** `internal/client/files.go` line 11
   sets it to 30 with a comment claiming it mirrors the backend. The
   backend `PAGE_SIZE` is 50 at both tags
   (`backend/open_webui/routers/files.py` at v0.11.0, line 473).
   `ListFiles` only survives because the `len(all) >= resp.Total` guard
   ends the loop first; the `< filesListPageSize` check never fires on a
   non-final page. Set it to 50 and fix the comment.
2. **`readKnowledgeFileState` reads only page 1.**
   `internal/provider/resource_knowledge_file.go` line 223 requests a
   single page of `GET /knowledge/{id}/files`, which caps at 30. A
   knowledge base with more than 30 files makes the provider treat every
   attachment past the first page as deleted and drop it from state.
   Page through the whole list, as `ListKnowledge` already does.
3. **`KnowledgeForm.Data` and `KnowledgeForm.Meta` are discarded.**
   `internal/client/knowledge.go` lines 19 and 20 send them, but
   `KnowledgeForm` in `backend/open_webui/models/knowledge.py` is a plain
   `BaseModel` with no `extra='allow'`, so pydantic drops them at both
   tags. The `data_json` and `meta_json` attributes on
   `openwebui_knowledge` therefore never round-trip. Either remove the
   attributes or mark them clearly as inert. Ask Chris which; removing an
   attribute is a breaking change for anyone using it.
4. **`KnowledgeResponse.Files` is typed wrong.**
   `internal/client/knowledge.go` types it `[]FileModel` with
   `Filename`, `UserID`, `Path`, and `Data`, but the backend serialises
   `list[FileMetadataResponse]`, carrying only `id`, `hash`, `meta`,
   `created_at`, and `updated_at`, at both tags. No provider code reads it,
   so this is dead weight rather than a live bug. Retype or delete.

---

## 7. Groups, users, and the permission key tree

**Depends on: section 1. Independent of sections 3, 4, 5, 6, and 8.**

### 7.1 `openwebui_group`: CHANGED, seven missing permission keys

The endpoints and forms are unchanged. `backend/open_webui/models/groups.py`
has an empty diff between the tags, and `routers/groups.py` gained only
`publish_event` calls and error-message text.

The whole change is the permission key tree.
`internal/provider/permissions_helpers.go` enumerates the keys a group's
`permissions` object may carry, and `filterPermissionKeys` (line 206)
raises a hard `AddAttributeError` for anything not on its lists. Seven
keys in `DEFAULT_USER_PERMISSIONS` (`backend/open_webui/config.py` at
v0.11.0, lines 1926 to 1999) are missing, so a practitioner cannot set
them at all:

| Dotted path | Default | Add to |
| --- | --- | --- |
| `workspace.skills_import` | `false` | `groupPermissionsWorkspaceKeys` |
| `workspace.skills_export` | `false` | `groupPermissionsWorkspaceKeys` |
| `sharing.folders` | `false` | `groupPermissionsSharingKeys` |
| `sharing.open_chats` | `false` | `groupPermissionsSharingKeys` |
| `access_grants.allow_groups` | `true` | `groupPermissionsAccessGrantsKeys` |
| `chat.import` | `true` | `groupPermissionsChatKeys` |
| `features.webhooks` | `false` | `groupPermissionsFeaturesKeys` |

**No key the provider already lists was removed, and nothing was renamed.** All
61 keys across the six existing slices still exist at v0.11.0.

Two facts to carry into the implementation:

- The wire name is `import`, not `import_`. The backend attribute is
  `ChatPermissions.import_` with `Field(default=True, alias='import')`
  (`backend/open_webui/routers/users.py` at v0.11.0). Use `"import"` in
  the Go slice.
- `sharing.open_chats` is in `DEFAULT_USER_PERMISSIONS` and is enforced
  at `backend/open_webui/routers/chats.py` at v0.11.0 line 2041, but it
  is absent from the `SharingPermissions` class. That is a backend
  inconsistency. It does not block the provider: `Group.permissions` is
  `Column(JSON)` and `GroupForm.permissions` is `Optional[dict]`
  (`backend/open_webui/models/groups.py` at v0.11.0), so group
  permissions are a free-form dict and every key round-trips.

`filterPermissionResponse` (line 241) currently drops unknown keys on
read, so a group carrying any of the seven shows nothing in state and no
perpetual diff. Adding the keys is therefore safe: it turns
unmanageable keys into managed ones without disturbing existing state.

Also fix the stale documentation strings that list a subset of the
allowed keys: `resource_group.go` lines 125, 135, and 145, and
`data_source_group.go` lines 88, 93, and 98. The `chat` description omits
`web_upload` and the `features` description names 5 of the 11 keys the
code already accepts. These strings feed `docs/resources/group.md` and
`docs/data-sources/group.md` through `tfplugindocs`.

Tests: `internal/provider/permissions_helpers_test.go` gains cases for
each of the seven keys, and `acc_group_test.go` gains a step setting
`workspace.skills_import` and `features.webhooks`.

### 7.2 `openwebui_group` data source: UNCHANGED

`GET /api/v1/groups/id/{id}/export` still returns `GroupExportResponse`,
which is `GroupResponse` plus `user_ids`
(`backend/open_webui/routers/groups.py` at v0.11.0, lines 136 to 160). It
still answers 401 with `ERROR_MESSAGES.NOT_FOUND` for a missing group, so
the `notFoundDetail` mapping in `internal/client/groups.go` lines 64 and
78 is still correct. It inherits the seven keys from 7.1 and needs no
other change.

### 7.3 `openwebui_user` data source: UNCHANGED

`GET /api/v1/users/` at v0.11.0 (`backend/open_webui/routers/users.py`,
lines 65 to 108) is byte-identical to v0.9.6. It still declares
`response_model=UserGroupIdsListResponse` and still returns
`{"users": [...], "total": N}`, so `listUsersResponse` in
`internal/client/users.go` line 26 is correct. The router's 410-line
growth is new endpoints (`/usage`, `/user/variables`, and
`/default/permissions/defaults`), not a shape change.
`GET /api/v1/users/{id}` still returns `UserActiveResponse` with
`extra='allow'`.

Pre-existing, present at both tags: `client.User.OAuthSubject`
(`internal/client/users.go` line 19, tagged `json:"oauth_sub"`) never
binds, because `UserModel` has `oauth: dict | None` and no `oauth_sub`
field. Remove the field or retype it to read `oauth`.

---

## 8. Config resources, connections, and pipelines

**Depends on: section 1. Independent of sections 3 through 7.**

Router mounts are unchanged: `/ollama` and `/openai` at the root,
`/api/v1/configs`, `/api/v1/pipelines` under the API prefix
(`backend/open_webui/main.py` at v0.11.0, lines 785, 786, 789, and 796).
The `doRaw(c.rootURL + "/openai/...")` split in
`internal/client/connections.go` still holds. Every path in this group
exists at v0.11.0 with the same method and the same trailing slash.

### 8.1 `openwebui_config_import` and `openwebui_config_export`: CHANGED

This is the largest change in this section, and the second largest in the
upgrade after the model `meta` drift.

`backend/open_webui/models/config.py` is **new at v0.11.0**; the file does
not exist at v0.9.6. It replaces the single-row JSON blob in
`backend/open_webui/internal/config.py` with a per-key table:
`Config(key=Text primary_key, value=JSON, updated_at=BigInteger)`, at
line 99. Three consequences on the wire:

- **The export shape went from nested to flat dotted keys.** At v0.9.6
  `export_config` returned `get_config()`, a nested dict, so banners came
  back as `{"ui": {"banners": [...]}}`. At v0.11.0 it returns
  `await Config.get_all()` (`backend/open_webui/routers/configs.py` line
  121), which is `{row.key: row.value}`, so the same banners come back as
  `{"ui.banners": [...]}`. **A `config_json` captured from a v0.9.x
  export must not be fed to v0.11.** It would write literal top-level
  rows named `ui` and `code_execution` that nothing reads. Say this
  plainly in the `openwebui_config_import` documentation.
- **Import went from replace to merge.** v0.9.6 called
  `async_save_config`, replacing the whole blob. v0.11.0 calls
  `await Config.upsert(form_data.config)`
  (`backend/open_webui/routers/configs.py` line 95), a per-key upsert
  that leaves untouched keys alone. The `MarkdownDescription` in
  `internal/provider/resource_config_import.go` line 44 says applying it
  "replaces the entire current configuration". That is now false. Reword
  it.
- **A likely apply-time failure.** `applyConfigImport`
  (`resource_config_import.go` lines 177 to 185) overwrites the state
  value of the Required `config_json` attribute with the server's
  response. At v0.9.6 that response equalled the input. At v0.11.0 it is
  `Config.get_all()`, a superset of what was sent, so Terraform should
  reject the apply with "Provider produced inconsistent result after
  apply" whenever `config_json` is not already the complete config. Fix
  it by keeping `plan.ConfigJSON` as the state value and dropping the
  response overwrite, or by exposing the round-tripped value as a
  separate Computed attribute.

  This one is inferred from reading both code paths, not observed. **Test
  it against the container from section 1 before writing the fix**, so
  the fix matches the actual failure.

Not used by the provider, but worth knowing: v0.11.0 adds
`GET /api/v1/configs/namespace/{namespace}`
(`backend/open_webui/routers/configs.py` line 124), which returns one
dotted namespace and is a cleaner read target than a full export.

### 8.2 `openwebui_oauth_client`: CHANGED, additive

`OAuthClientRegistrationForm` gains one optional field,
`oauth_scope: str | None = None`
(`backend/open_webui/routers/configs.py` at v0.11.0, line 162), threaded
into both the static and dynamic registration calls at lines 170 and 175.

Add `OAuthScope *string` with `json:"oauth_scope,omitempty"` to
`client.OAuthClientRegistrationForm` in `internal/client/configs.go` line
21, and an optional `oauth_scope` attribute to
`resource_oauth_client.go`. Nothing breaks without it.

The 400 detail text is now `f'Failed to register OAuth client: {e}'`
(line 184). If any test asserts the old string, update it.

Still missing from the Go struct at both tags, so pre-existing:
`client_secret` and `oauth_server_url`.

### 8.3 Unchanged in this section

Each of these was checked field-for-field against its Pydantic form at
both tags and needs no work:

| Resource or data source | Evidence |
| --- | --- |
| `openwebui_connections_config` | `ConnectionsConfigForm` still exactly `ENABLE_DIRECT_CONNECTIONS` and `ENABLE_BASE_MODELS_CACHE`, `routers/configs.py` at v0.11.0 line 134. Storage moved to `Config.upsert`; the wire is identical. |
| `openwebui_tool_servers_config` | `ToolServerConnection` and `ToolServersConfigForm` identical, `routers/configs.py` at v0.11.0 lines 217 to 231. |
| `openwebui_code_execution_config` | All 15 fields of `CodeInterpreterConfigForm` identical, `routers/configs.py` at v0.11.0 lines 680 to 695. |
| `openwebui_models_config` | `ModelsConfigForm` identical, `routers/configs.py` at v0.9.6 line 587 and v0.11.0 line 728. |
| `openwebui_suggestions_config` | `PromptSuggestion` and `SetDefaultSuggestionsForm` identical. There is still no `GET /configs/suggestions` at either tag, so the write-only read model stays correct. |
| `openwebui_banners_config` | `BannerModel` identical, `config.py` at v0.11.0 line 2127. |
| `openwebui_tool_server_verify` | `POST /configs/tool_servers/verify` present, same body, `routers/configs.py` at v0.11.0 line 545. |
| `openwebui_openai_connections` and `_verify` | `OpenAIConfigForm` character-identical at both tags, `routers/openai.py` at v0.11.0 lines 392 and 404. `ConnectionVerificationForm` unchanged, lines 788 to 795. |
| `openwebui_ollama_connections` and `_verify` | `OllamaConfigForm` identical, `routers/ollama.py` at v0.11.0 lines 288 and 305. Verify body unchanged, lines 247 to 252. |
| `openwebui_pipeline`, `_valves`, and the data source | All eight `routers/pipelines.py` routes present at both tags with the same methods. `AddPipelineForm` and `DeletePipelineForm` unchanged, v0.11.0 lines 305 and 355. The 116-line diff is internal: the OpenAI config lookup moved to `await get_openai_connection(urlIdx)` and the upload switched to `aiofiles` streaming. |

### 8.4 Pre-existing defects to fix while you are here

True at both tags, not caused by the upgrade.

1. **`openwebui_models_config` nulls two fields on every write.**
   `client.ModelsConfigForm` (`internal/client/configs.go` line 63)
   declares 3 of the 5 fields in `ModelsConfigForm`. Every
   `POST /api/v1/configs/models` therefore omits `DEFAULT_MODEL_METADATA`
   and `DEFAULT_MODEL_PARAMS`, pydantic defaults them to `None`, and
   v0.11.0 writes `models.default_metadata = null` and
   `models.default_params = null`. This is silent data loss. Add both
   fields.
2. **`client.ToolServerConnection` omits `info`.** The backend has had it
   since v0.9.6, and `set_tool_servers_config` reads it for the OAuth
   client key via `(connection.get('info') or {}).get('id')`.
3. **`client.GroupUpdateForm.Meta`** (`internal/client/groups.go` line
   23) is sent, but `GroupUpdateForm` has no `meta` field at either tag,
   so it is dropped. **`client.GroupResponse.AdminIDs`** (line 36) is
   never serialised at either tag.
4. **`client.AddPipelineForm`** (`internal/client/pipelines.go` line 15)
   sends a `key` field the backend has never declared. Harmless; remove
   it.

---

## 9. Documentation and the compatibility claim

**Depends on: sections 1 through 8. Do this last.**

1. Add `examples/resources/openwebui_skill/resource.tf` and
   `import.sh`, and `examples/data-sources/openwebui_skill/data-source.tf`.
   `tfplugindocs` reads these directories by resource name.
2. Run `make docs`. It runs `tfplugindocs` through `go generate`
   (`tools/tools.go`). Do not hand-edit anything under `docs/`.
3. Update the Compatibility table in `README.md` line 16 from
   `v0.9.0 – v0.9.6` to `v0.11.0`. State one version, not a range: the
   `meta.chat_variables_schema` handling in section 3.1 and the flat
   config export in section 8.1 both make this build wrong for v0.9.x.
4. Add `openwebui_skill` to the resource table (README line 66 onward)
   and to the data source table (line 92 onward).
5. Add a `CHANGELOG.md` entry naming the breaking changes: the model
   `base_model_id` clearing fix, the `file_json` narrowing on
   `openwebui_knowledge_file`, and the config export format change.

---

## Optional new surface

**Superseded by Part two.** This table was written when the goal was the
existing resources plus skills. The goal is now complete v0.11.0
coverage, so most of these "exclude" verdicts have been revisited with
evidence in sections 10 through 16. Where the two disagree, **section 16
wins**. The table is kept because its reasoning is still the record of
why each area was first set aside.

Only one router file is new at v0.11.0: `routers/notifications.py`. Every
other area listed below already existed at v0.9.6 and is simply surface
the provider never covered.

| Area | What it is | Recommendation |
| --- | --- | --- |
| Subagents config (`GET`/`POST /api/v1/configs/subagents`) | Admin settings for the new subagent runner: enable, concurrency, iteration and output caps, system prompt. `routers/configs.py` at v0.11.0 lines 777 and 782 | **Include.** It is an admin config area, the same shape as the six config resources the provider already ships, and it is genuinely new at v0.11.0. |
| `oauth_scope` on the OAuth client | Covered in section 8.2 | **Include.** Already required-adjacent and additive. |
| `file_count` on the knowledge data source | Covered in section 6.1 | **Include.** One computed attribute. |
| `has_user_valves` on the tool resource | Covered in section 5.1 | **Include.** One computed attribute. |
| Terminal servers config (`POST /api/v1/configs/terminal_servers/lifecycle` and `/refresh`) | Lifecycle actions for terminal servers, not declarative config | **Exclude.** Actions, not state. Terraform has no good shape for them. |
| External knowledge connections (`/api/v1/knowledge/external/...`) | CRUD for connections to third-party knowledge providers, creating knowledge bases marked `meta.source == 'external'`. `routers/knowledge.py` at v0.11.0 lines 632 to 1032 | **Consider later.** It is real declarative config and a genuine v0.11 addition, but it is a large surface, and external knowledge bases are read-only to the file routes (lines 119 to 127). Only worth it if we actually use external knowledge. |
| Notes (`/api/v1/notes`) | User-owned notes with sharing, 12 routes | **Exclude.** User content, not infrastructure. |
| Folders (`/api/v1/folders`) | Chat folders with sharing, 11 routes | **Exclude.** User content. |
| Channels (`/api/v1/channels`) | Chat channels, 28 routes | **Exclude.** User content. |
| Calendar and automations | Per-user scheduling features | **Exclude.** User content. |
| Notifications (`/api/v1/notifications`) | The only new router file at v0.11.0 | **Exclude.** Runtime delivery, no durable config to declare. |
| SCIM (`/api/v1/scim/v2`) | User and group provisioning from an identity provider | **Exclude.** It provisions the same users and groups the provider already manages. Two writers over one dataset is a fight, not a feature. |
| User variables (`GET`/`POST /api/v1/users/user/variables`) | Per-user chat variables, new at v0.11.0 | **Exclude.** Per-user state. |
| `GET /api/v1/models/base/tags` and `GET /api/v1/files/count` | New read-only helpers | **Exclude.** No resource needs them. |

## Open questions

Two things this plan could not settle from the backend source alone.
Both needed the container from section 1. **Both are now answered**; see
the integration pass at the end of this document.

1. **Does `GET /api/v1/skills/id/{id}` return `content`?** *Answered:
   yes.* The response model does not declare it, but `extra='allow'` on
   the parent class lets it through, exactly as section 4.2 predicted.
   The fallback through `GET /api/v1/skills/export` is not needed.
2. **Does `openwebui_config_import` actually fail its apply on v0.11.0?**
   Section 8.1 inferred "Provider produced inconsistent result after
   apply" from the two code paths. This was exercised under
   `OPENWEBUI_TEST_CONFIG_IMPORT=1` during the integration pass.

---

# Part two: complete v0.11.0 coverage

Sections 1 through 9 bring the existing resources up to v0.11.0 and add
skills. Sections 10 onward extend the fork to cover every declarative
surface v0.11.0 exposes.

"Complete" here means every stable admin-level and workspace-level
configuration surface an operator would declare: the things that define
an instance, not the things that record its use. Every router in
`backend/open_webui/routers/` gets a verdict in section 15, and every
exclusion carries one of four written reasons:

- **(a) no read route**, so Terraform state can never converge;
- **(b) runtime data** rather than configuration;
- **(c) secrets returned encrypted or masked**, so state cannot converge;
- **(d) overlaps** a surface the provider already manages.

## 10. Tool servers as first-class resources

**Depends on: section 1. Independent of sections 11 through 14. On the
critical path.**

Verdict: **NEW RESOURCE**, `openwebui_tool_server`.

Registering a tool server is infrastructure. The downstream instance runs
five MCP servers wired into its models, and those connections should be
declared, not hand-made. Today the provider models the entire
`TOOL_SERVER_CONNECTIONS` list as one `openwebui_tool_servers_config`
object, and that resource is unsafe: section 8.4 records that
`client.ToolServerConnection` omits `info`, so writing through it
destroys the OAuth registration of every MCP server on the instance.

This section replaces that with one resource per connection.

### 10.1 The wire shape

`ToolServerConnection` (`backend/open_webui/routers/configs.py` at
v0.11.0, lines 217 to 227):

| Field | Type | Default | Owner |
| --- | --- | --- | --- |
| `url` | `str` | required | Terraform |
| `path` | `str` | required | Terraform |
| `type` | `str \| None` | `'openapi'` | Terraform |
| `auth_type` | `str \| None` | required, nullable | Terraform |
| `headers` | `dict \| str \| None` | `None` | Terraform |
| `key` | `str \| None` | required, nullable | Terraform, sensitive |
| `config` | `dict \| None` | required, nullable | Split, see 10.2 |
| `info` | `dict \| None` | `None` | Split, see 10.3 |

The class carries `model_config = ConfigDict(extra='allow')` (line 227),
so any field the provider does not model survives the round trip. That is
the property this whole design rests on.

`type` is `openapi` or `mcp` (line 220). For `openapi` servers a second
discriminator applies: `spec_type` is `url` or `json`
(`backend/open_webui/utils/tools.py` at v0.11.0, line 1477). With
`spec_type = 'json'` the spec is read from a `spec` field holding inline
JSON (line 1490) instead of being fetched from `path`. Both ride through
as extras, so model them as optional attributes.

`auth_type` values seen in the source: `bearer` and `none`
(`backend/open_webui/utils/tools.py` at v0.11.0, lines 1466 and 1468),
plus `oauth_2.1` and `oauth_2.1_static`
(`backend/open_webui/routers/configs.py` at v0.11.0, line 250). Only the
OAuth pair triggers the client-manager path; `bearer` reads `key`
directly and `none` sends nothing. So a non-OAuth tool server, OpenAPI or
MCP, is a plain create with no extra machinery.

### 10.2 The `config` sub-object

Four keys are read from `config` at v0.11.0:

| Key | Read at | Meaning |
| --- | --- | --- |
| `enable` | `routers/tools.py:127`, `utils/tools.py:1460` | Whether the server is active. **This is the enable flag; there is no top-level one.** |
| `access_grants` | `routers/tools.py:108` and `:144` | Per-server sharing, the same flat grant list as models |
| `oauth_scope` | `utils/oauth.py:744` | Overridden by `info.oauth_scope` when both are set |
| `oauth_resource_parameter` | `utils/oauth.py:737` | Overridden by `info.oauth_resource_parameter` |

Terraform owns all four. Model `enable` as a top-level `enabled` bool
that maps into `config.enable`, and `access_grants` through the existing
`read_groups` / `write_groups` pattern. Preserve any other key in
`config` untouched.

### 10.3 The `info` sub-object, and what Terraform must not touch

`info` is where the danger lives.

| Key | Read at | Owner |
| --- | --- | --- |
| `id` | `routers/tools.py:143`, `utils/tools.py:1472` | **Terraform**, see 10.4 |
| `oauth_client_info` | `utils/oauth.py:717` | **Server**, opaque ciphertext |
| `oauth_client_id` | `utils/oauth.py:720` | Terraform, only for `oauth_2.1_static` |
| `oauth_client_secret` | `utils/oauth.py:720` | Terraform, sensitive, only for `oauth_2.1_static` |
| `oauth_scope` | `utils/oauth.py:744` | Terraform |
| `oauth_resource_parameter` | `utils/oauth.py:737` | Terraform |

`info.oauth_client_info` is a Fernet ciphertext string, produced by
`encrypt_data` (`backend/open_webui/utils/oauth.py` at v0.11.0, lines 265
to 270) and consumed by `resolve_oauth_client_info` (line 709), which
calls `decrypt_data(info.get('oauth_client_info', ''))` at line 717.

Two facts make it safe to carry:

- **`GET /api/v1/configs/tool_servers` returns it verbatim.** The handler
  is `return {'TOOL_SERVER_CONNECTIONS': await
  Config.get('tool_server.connections')}`
  (`backend/open_webui/routers/configs.py` at v0.11.0, line 236). It
  returns the stored rows with no masking, no redaction, and no
  re-encryption. So the value round-trips byte for byte and Terraform
  state converges.
- **Nothing re-encrypts it on write.** `set_tool_servers_config` stores
  `connection.model_dump()` as given (line 261).

So the rule for the resource is: **`info` is a preserve-unknown-fields
block.** Read the whole `info` object from the server, keep it in state
as an opaque map, and on write merge the Terraform-owned keys over the
stored object rather than replacing it. Expose `oauth_client_info` as a
Computed and Sensitive attribute so it never appears in a plan diff, and
never accept it as configuration.

The Fernet key is `OAUTH_CLIENT_INFO_ENCRYPTION_KEY`, which defaults to
`WEBUI_SECRET_KEY` (`backend/open_webui/env.py` at v0.11.0, line 817).
Rotating either invalidates every stored blob. Say so in the resource
documentation: the blob is opaque to Terraform and Terraform cannot
re-create it after a key rotation.

### 10.4 The `id` question

**`info.id` is client-supplied, not server-generated.** There is no
`uuid4` or equivalent anywhere on this path in the backend. The evidence:

- For MCP, `tool_id = f'server:mcp:{info.get("id")}'`
  (`backend/open_webui/routers/tools.py` at v0.11.0, line 143). There is
  no fallback, so an absent `info.id` produces the literal string
  `server:mcp:None`.
- For OpenAPI, `id = info.get('id')` and `if not id: id = str(idx)`
  (`backend/open_webui/utils/tools.py` at v0.11.0, lines 1472 to 1474),
  giving `server:{id}` at `routers/tools.py:107`. The positional fallback
  is the reason an OpenAPI server without an explicit id changes identity
  when the list is reordered.

This is good news. Terraform sets `info.id` itself, so it is stable by
construction and cannot drift when a sibling field is edited. Make
`server_id` a Required attribute with `RequiresReplace`, and say in its
description that models reference this server as
`server:mcp:<server_id>` in their `tool_ids`, so changing it breaks every
model that points at it.

**ImportState works**, keyed on `info.id`: read
`GET /api/v1/configs/tool_servers`, find the entry whose `info.id`
matches, and populate state from it. This is how the five existing
hand-made connections get adopted. Because a hand-made OpenAPI connection
may have no `info.id` at all, the import path must also accept a list
index as a fallback key, and the resource should then write an explicit
`info.id` on the next apply to pin the identity.

### 10.5 The DCR lifecycle

This is the question that settles whether a Terraform-created OAuth
connection works on its own. The answer is **no, and it does not
self-heal.**

Dynamic client registration runs in exactly one place:
`POST /api/v1/configs/oauth/clients/register`
(`backend/open_webui/routers/configs.py` at v0.11.0, line 172). It calls
`get_oauth_client_info_with_dynamic_client_registration` when no
`client_secret` is supplied, or
`get_oauth_client_info_with_static_credentials` when one is, and returns
`{'status': True, 'oauth_client_info': encrypt_data(...)}` at lines 201
and 202. It **returns** the blob rather than storing it. The caller is
expected to write it into the connection's `info`.

At tool-server write time nothing registers. `set_tool_servers_config`
(line 239) only *loads* an existing registration: for each MCP connection
with an OAuth `auth_type` and a `server_id`, it calls
`resolve_oauth_client_info(connection)`, which decrypts
`info.oauth_client_info` (`utils/oauth.py:717`). When that key is absent,
`decrypt_data('')` raises, and the surrounding `except Exception ...
continue` (line 281 of configs.py) swallows it. The connection is stored,
but no OAuth client is registered, so the server never authenticates.

There is no lazy path. `set_tool_servers` only fetches spec data
(`backend/open_webui/utils/tools.py` at v0.11.0, lines 1134 to 1149).

**What the resource must do about it.** The provider already has an
`openwebui_oauth_client` resource that calls the registration endpoint,
so the two compose:

```hcl
resource "openwebui_oauth_client" "paperless" {
  url       = "https://paperless.mcp.theguidrys.us"
  client_id = "paperless"
  type      = "mcp"
}

resource "openwebui_tool_server" "paperless" {
  server_id = "paperless"
  type      = "mcp"
  url       = "https://paperless.mcp.theguidrys.us"
  auth_type = "oauth_2.1"
  enabled   = true

  oauth_client_info = openwebui_oauth_client.paperless.oauth_client_info
}
```

The `type = "mcp"` argument becomes the `type` query parameter, which the
handler uses to build `oauth_client_id = f'{type}:{form_data.client_id}'`
(line 182). That matches the `client_key = f'{server_type}:{server_id}'`
the tool-server writer uses at line 253, and the `f'mcp:{server_id}'`
token lookup at `routers/tools.py:139`. So `client_id` on the OAuth
client resource must equal `server_id` on the tool server resource.

**This composition needs one change to make it work.**
`resource_oauth_client.go` currently declares only `id`, `url`,
`client_id`, `client_name`, `oauth_scope`, and `type`. It throws the
response away: `RegisterOAuthClient` in `internal/client/configs.go` line
237 returns `map[string]any` holding the blob, and nothing reads it. Add
an `oauth_client_info` attribute, Computed and Sensitive, carrying
`resp["oauth_client_info"]`. Do this as part of section 8.2, which is
already editing that file.

Note the consequence for `terraform destroy` and re-apply: the
registration endpoint performs a real DCR against the upstream identity
provider every time it runs, so re-creating the OAuth client resource
mints a new registration. That is correct but not free. Give
`openwebui_oauth_client` a `RequiresReplace` on `url`, `client_id`, and
`type`, and nothing else, so ordinary edits to the tool server do not
re-register.

For `auth_type = "oauth_2.1_static"` the flow is simpler: set
`info.oauth_client_id` and `info.oauth_client_secret` directly and skip
the `openwebui_oauth_client` resource. `resolve_oauth_client_info`
overlays them onto the decrypted blob (`utils/oauth.py` lines 719 to
722), but it still calls `decrypt_data` first, so a static connection
also needs a registration blob. Register it with `client_secret` set,
which takes the static branch at `configs.py:186`.

### 10.6 The read-modify-write race

`POST /api/v1/configs/tool_servers` replaces the entire list. A
per-server resource therefore has to read the list, splice its own entry,
and write the list back. Two `openwebui_tool_server` resources applying
concurrently will lose one of the two writes.

This is a real problem, not a theoretical one: Terraform's default
`-parallelism` is 10, and it does run instances of the same resource type
concurrently.

**The fix is a mutex in the client.** The provider builds one
`*client.Client` and hands the same pointer to every resource through
`req.ProviderData`, so a `sync.Mutex` on the client serialises every
read-modify-write within a single Terraform run. Add it to
`internal/client/configs.go` as a field on `Client`, taken by a new
`UpsertToolServerConnection` and `DeleteToolServerConnection` pair that
do the read, splice, and write under the lock. Do not put the locking in
the resource layer; it belongs where the non-atomic operation lives.

The mutex does not help across processes. **State the single-operator
assumption in the resource documentation:** two `terraform apply` runs
against the same instance at the same time can lose a tool server. That
is acceptable here and matches how the rest of this repository is
operated.

### 10.7 What happens to `openwebui_tool_servers_config`

Keep it, but make it safe and document the overlap. The two resources
manage the same list, so using both is a fight.

- Fix its `info` handling as described in 10.3, so it stops destroying
  OAuth registrations. This is required regardless: section 8.4 lists it
  as a live data-loss bug.
- Add a note to both resources' documentation saying they are mutually
  exclusive, and that `openwebui_tool_server` is the recommended one.
- Do not deprecate it in this release. Deprecation is a separate decision
  to put to Chris.

### 10.8 Files

| File | Change |
| --- | --- |
| `internal/client/tool_servers.go` | New. `ToolServerConnection` with `Info` and `Config` as `map[string]any`, plus `ListToolServerConnections`, `UpsertToolServerConnection`, `DeleteToolServerConnection` under the mutex |
| `internal/client/configs.go` | Add `mu sync.Mutex` to `Client`; add `Info` to the existing `ToolServerConnection` (section 8.4 item 2) |
| `internal/client/tool_servers_test.go` | New. httptest cases: splice into an empty list, splice into a populated list, preserve an unknown `info` key, delete by id |
| `internal/provider/resource_tool_server.go` | New. Modeled on `resource_config_tool_servers.go` for the wire mapping and on `resource_model.go` for the `read_groups` / `write_groups` and preserve-unknown handling |
| `internal/provider/data_source_tool_server.go` | New. Read one connection by `server_id` |
| `internal/provider/resource_oauth_client.go` | Add Computed, Sensitive `oauth_client_info` |
| `internal/provider/provider.go` | Register both |
| `internal/provider/acc_tool_server_test.go` | New, see below |

Acceptance tests: create a `none`-auth OpenAPI server and check the
read-back; create two servers in one apply and check both survive, which
is the mutex regression test; edit one server's `url` and check a sibling
server's `info` is untouched; import an existing connection by
`server_id`. Do not write an acceptance test that performs a real DCR
against an external identity provider; cover the OAuth path with a client
unit test that asserts an unknown `info.oauth_client_info` survives a
sibling-field edit.

## 11. The rest of the configs router

**Depends on: section 1. Independent of sections 10 and 12 through 14.**

`backend/open_webui/routers/configs.py` has 25 routes at v0.11.0. The
provider covers 10 of them. Section 10 takes the tool-server pair. This
section takes what is left.

### 11.1 `openwebui_subagents_config`: NEW RESOURCE

A clean GET and POST pair over a flat form, new at v0.11.0.

Endpoints: `GET /api/v1/configs/subagents`
(`backend/open_webui/routers/configs.py` at v0.11.0, line 777) and
`POST /api/v1/configs/subagents` (line 782). Both are `get_admin_user`.

`SubagentsConfigForm` (line 767), every field required:

| Field | Type |
| --- | --- |
| `ENABLE_SUBAGENTS` | bool |
| `SUBAGENTS_BACKGROUND_ENABLED` | bool |
| `SUBAGENTS_MAX_CONCURRENT` | int |
| `SUBAGENTS_MAX_ASYNC` | int |
| `SUBAGENTS_MAX_ITERATIONS` | int |
| `SUBAGENTS_MAX_OUTPUT` | int |
| `SUBAGENTS_SYSTEM_PROMPT` | str |

Storage keys are in `SUBAGENTS_CONFIG_KEYS` (lines 71 to 77), mapping
each to `subagents.*`. No secrets. ImportState is trivial: a singleton
with a fixed id.

Model it on `resource_config_code_execution.go`. This is the simplest new
resource in the plan and a good warm-up section.

### 11.2 `openwebui_terminal_server`: NEW RESOURCE

Terminal servers are the same shape of problem as tool servers: a list
under one config key, with a GET and a POST.

Endpoints: `GET /api/v1/configs/terminal_servers`
(`backend/open_webui/routers/configs.py` at v0.11.0, line 320) and
`POST /api/v1/configs/terminal_servers` (line 325). The GET returns
`{'TERMINAL_SERVER_CONNECTIONS': await
Config.get('terminal_server.connections')}` at line 322, raw and
unmasked, so state converges.

`TerminalServerConnection` (line 296), with `extra='allow'` at line 313:

| Field | Type | Default |
| --- | --- | --- |
| `id` | `str \| None` | `''` |
| `name` | `str \| None` | `''` |
| `enabled` | `bool \| None` | `True` |
| `url` | `str` | required |
| `path` | `str \| None` | `'/openapi.json'` |
| `key` | `str \| None` | `''`, sensitive |
| `auth_type` | `str \| None` | `'bearer'` |
| `config` | `dict \| None` | `None` |
| `server_type` | `str \| None` | `None` |
| `policy_id` | `str \| None` | `None` |

Unlike tool servers this one has a top-level `id` and a top-level
`enabled`, so it is the easier of the two. Note that the POST dumps each
connection with `exclude={'policy', 'lifecycle'}` (line 332), so those
two keys are dropped on every write regardless of what is sent. Do not
model them.

Build it after section 10 and reuse the same list-splice-and-write client
helper and the same mutex. The read-modify-write race and the
single-operator assumption from 10.6 apply identically.

### 11.3 Excluded routes in this router

| Route | Line | Verdict |
| --- | --- | --- |
| `POST /terminal_servers/verify` | 349 | **EXCLUDED (b)**. Runtime probe of a remote server. It matches the existing `openwebui_tool_server_verify` data source, so add it as a data source only if section 11.2 needs a preflight check. |
| `POST /terminal_servers/policy` | 424 | **EXCLUDED (b)**. Proxies a policy read or write to a remote orchestrator (line 427); the state lives on that server, not in Open WebUI. |
| `POST /terminal_servers/lifecycle` | 461 | **EXCLUDED (b)**. Lifecycle action, not state. |
| `POST /terminal_servers/refresh` | 498 | **EXCLUDED (b)**. Cache refresh action. |
| `GET /models/defaults` | 736 | **EXCLUDED (d)**. Returns only `DEFAULT_MODEL_METADATA`, which `openwebui_models_config` already writes. A read-only echo of a field the provider owns. |
| `GET /namespace/{namespace}` | 124 | **DATA SOURCE, optional.** Returns one dotted config namespace (line 126). A cleaner read than a full export, and a useful escape hatch for config the provider does not model. Low cost, low priority. |

## 12. Channels

**Depends on: section 1. Independent of sections 10, 11, 13, 14, and 15.**

Verdict: **NEW RESOURCE**, `openwebui_channel`, scoped to standard
channels only.

`backend/open_webui/routers/channels.py` holds three different things
behind one prefix. Only one of them is declarative.

A `Channel` row whose `type` is unset is a *standard* channel: every user
who passes its access grants sees it, and **only an admin can create
one**. `backend/open_webui/routers/channels.py` at v0.11.0, line 288:

```python
if form_data.type not in ['group', 'dm'] and user.role != 'admin':
    # Only admins can create standard channels (joined by default)
```

Rows with `type` of `group` or `dm` are user-created rooms backed by a
`channel_member` table (`backend/open_webui/models/channels.py` at
v0.11.0, line 97). A standard channel has no membership rows at all:
`insert_new_channel` builds `ChannelMember` objects only for the group
and dm types (`models/channels.py` lines 361 to 373). Visibility for a
standard channel comes entirely from `access_grants`.

That admin gate is what separates this from notes, folders, and calendars
in section 15. An admin curating a set of channels is defining the
instance.

### 12.1 Endpoints

| Operation | Method and path | Line |
| --- | --- | --- |
| Create | `POST /api/v1/channels/create` | 279 |
| Read | `GET /api/v1/channels/{id}` | 367 |
| Update | `POST /api/v1/channels/{id}/update` | 684 |
| Delete | `DELETE /api/v1/channels/{id}/delete` | 729 |

The read admits an admin regardless of grants (line 407), so
**ImportState by channel id works** with the admin token the provider
already uses.

### 12.2 Schema

`CreateChannelForm` (`backend/open_webui/models/channels.py` at v0.11.0,
line 243) is `ChannelForm` (line 232) plus `type`:

| Field | Type | Default | Expose |
| --- | --- | --- | --- |
| `name` | `str` | `''` | Yes |
| `description` | `str \| None` | `None` | Yes |
| `is_private` | `bool \| None` | `None` | Yes |
| `data` | `dict \| None` | `None` | As `data_json` |
| `meta` | `dict \| None` | `None` | As `meta_json` |
| `access_grants` | `list[dict] \| None` | `None` | As `read_groups` / `write_groups` |
| `group_ids` | `list[str] \| None` | `None` | **No**, see below |
| `user_ids` | `list[str] \| None` | `None` | **No**, see below |
| `type` | `str \| None` | `None` | No, always unset for this resource |

`ChannelModel` carries a flat `access_grants` list (`models/channels.py`
line 82), filled on every read by `_to_channel_model` (lines 263 to 272)
and written through `AccessGrants.set_access_grants('channel', ...)` on
create (line 376) and on update when not `None` (lines 846 to 847). So
the section 2 grant plumbing applies unchanged.

### 12.3 Three convergence traps

Encode all three or the resource will not converge.

1. **`name` is lowercased on create but not on update.**
   `models/channels.py` line 351 stores `form_data.name.lower()`;
   the update path at line 839 writes the value verbatim. A create with
   `name = "General"` reads back `general`, and the next update writes
   `General` again. Normalize the value in the provider, or validate that
   `name` is already lowercase and reject anything else. Validation is
   the better answer: it makes the constraint visible in the plan instead
   of silently rewriting what the operator asked for.
2. **`type` cannot be changed.** It is absent from `ChannelForm`
   (`models/channels.py` lines 232 to 241), so update cannot alter it.
   The resource never sets it, but if a future variant does, it needs
   `RequiresReplace`.
3. **`group_ids` and `user_ids` are silently ignored** for standard
   channels (`models/channels.py` line 361). Do not put them in the
   schema. A field that accepts a value and discards it is worse than no
   field.

### 12.4 Files

`internal/client/channels.go`, `internal/client/channels_test.go`,
`internal/provider/resource_channel.go`,
`internal/provider/data_source_channel.go`,
`internal/provider/acc_channel_test.go`, plus registration in
`internal/provider/provider.go`.

Model the CRUD and import shape on `internal/provider/resource_prompt.go`.
Take the sharing handling from `internal/provider/resource_group.go`,
which is the existing resource closest to the flat grant list.

### 12.5 Channel webhooks: a follow-on, not part of this section

Full CRUD exists (`routers/channels.py` lines 1859, 1890, and 1926) and
`ChannelWebhookModel` returns `token` in cleartext
(`models/channels.py` line 214), so state can converge. But there is **no
read-by-webhook-id**; the only read is the per-channel list at line 1840,
so import needs a composite `channel_id/webhook_id` key resolved by
filtering that list. Build it after `openwebui_channel` works, or not at
all. It is not required for complete coverage of the channel object
itself.

## 13. Chat context compaction config

**Depends on: section 1. Independent of every other section.**

Verdict: **NEW RESOURCE**, `openwebui_chat_config`.

This one hides in `routers/chats.py`, which is otherwise entirely
transcript traffic. `GET /api/v1/chats/config`
(`backend/open_webui/routers/chats.py` at v0.11.0, line 817) and
`POST /api/v1/chats/config` (line 822) are both `get_admin_user` and both
read and write six flat `Config` keys under `chat.context_compaction.*`
(lines 55 to 62). Nothing in the provider covers them.

`ChatConfigForm` (line 137):

| Field | Type | Default |
| --- | --- | --- |
| `CONTEXT_COMPACTION_MODEL` | `str \| None` | `''` |
| `ENABLE_CONTEXT_COMPACTION` | `bool` | required |
| `CONTEXT_COMPACTION_TOKEN_THRESHOLD` | `int` | required |
| `CONTEXT_COMPACTION_TOKEN_CAP` | `int \| None` | `None` |
| `CONTEXT_COMPACTION_RETENTION_PERCENTAGE` | `int` | `40` |
| `CONTEXT_COMPACTION_PROMPT_TEMPLATE` | `str` | required |

**The server clamps three of these on write** (lines 823 to 825):

```python
threshold = max(1, int(form_data.CONTEXT_COMPACTION_TOKEN_THRESHOLD))
token_cap = max(1, int(form_data.CONTEXT_COMPACTION_TOKEN_CAP or threshold))
retention_percentage = min(50, max(10, int(form_data.CONTEXT_COMPACTION_RETENTION_PERCENTAGE)))
```

So a configuration outside those bounds is silently corrected, and
Terraform sees a permanent diff. Two things follow. First, add matching
validators to the schema (`int64validator.Between(10, 50)` on retention,
`AtLeast(1)` on threshold and cap) so the plan rejects the value instead
of the apply silently changing it. Second, note that
`CONTEXT_COMPACTION_TOKEN_CAP` defaults to the threshold when null or
zero, so it can never read back as null once written; make it Optional
and Computed.

The GET fills defaults for a null model, cap, and retention (lines 190 to
195), which is why those three must be Computed rather than plain
Optional.

**Do not route these keys through `openwebui_config_import` instead.**
That path writes `Config` directly and bypasses the clamping at lines 823
to 825, so the two would fight. The dedicated resource is the correct
shape.

Model it on `internal/provider/resource_config_code_execution.go`. It is
a singleton, so ImportState uses a fixed id.

Files: add `GetChatConfig` and `SetChatConfig` to
`internal/client/configs.go` (paths `chats/config`, not under
`configs/`), `internal/provider/resource_config_chat.go`, and
`internal/provider/acc_chat_config_test.go`.

## 14. Engine configuration: RAG, images, and audio

**Depends on: section 1. The four resources below are independent of each
other and of every other section.**

These three routers hold Open WebUI's engine-level admin settings. They
are the largest block of unmanaged configuration in the product, and all
of it is declarable.

One fact governs the whole section. Every value here is stored in the
`config` table as one row per dotted key
(`backend/open_webui/models/config.py` at v0.11.0, lines 100 to 106).
`Config.get_many` (lines 148 to 165) reads the raw JSON with no masking,
and `Config.upsert` (lines 196 to 218) writes it back the same way.
**No secret in this section is masked, redacted, or omitted on read**, so
every one of these resources can converge its state. Mark the credentials
`Sensitive: true`, the way `resource_config_code_execution.go` already
does for the Jupyter token.

Mount prefixes are `/api/v1/retrieval`, `/api/v1/images`, and
`/api/v1/audio` (`backend/open_webui/main.py` at v0.11.0, lines 794, 792,
and 793).

### 14.1 Write semantics differ per surface, and getting this wrong nulls data

This is the single most important thing to carry into the
implementation. Three different behaviors, all in this section:

| Surface | Semantics | Evidence |
| --- | --- | --- |
| `retrieval` top-level fields | **Omit to keep.** Each field is written as `form_data.X if form_data.X is not None else config.X` | `routers/retrieval.py` line 1195 and throughout |
| `retrieval.web` block | **Whole form.** Every field is assigned unconditionally once `form_data.web is not None`. Only `BRAVE_SEARCH_CONTEXT_TOKENS` is guarded | `routers/retrieval.py` lines 1247 to 1326, guard at line 1271 |
| `images` and `audio` | **Whole form.** `config_updates(form_data.model_dump(), ...)` writes every key every time | `routers/images.py` line 301, `routers/audio.py` lines 289 to 294 |

For every whole-form surface the provider must send the complete object
on every write. A field left out is written as its Pydantic default, or
as null. This is the same failure already recorded against
`ModelsConfigForm` in section 8.4 item 1, and it will be worse here
because these forms are much larger.

### 14.2 `openwebui_rag_embedding_config`: NEW RESOURCE

Endpoints: `GET /api/v1/retrieval/embedding`
(`backend/open_webui/routers/retrieval.py` at v0.11.0, line 451) and
`POST /api/v1/retrieval/embedding/update` (line 520).

`EmbeddingModelUpdateForm` (lines 493 to 501), eight fields:

| Field | Type | Default | Storage key |
| --- | --- | --- | --- |
| `RAG_EMBEDDING_ENGINE` | `str` | required | `rag.embedding_engine` |
| `RAG_EMBEDDING_MODEL` | `str` | required | `rag.embedding_model` |
| `RAG_EMBEDDING_BATCH_SIZE` | `int \| None` | `1` | `rag.embedding_batch_size` |
| `ENABLE_ASYNC_EMBEDDING` | `bool \| None` | `True` | `rag.enable_async_embedding` |
| `RAG_EMBEDDING_CONCURRENT_REQUESTS` | `int \| None` | `0` | `rag.embedding_concurrent_requests` |
| `openai_config` | `OpenAIConfigForm \| None` | `None` | nested, line 477 |
| `ollama_config` | `OllamaConfigForm \| None` | `None` | nested, line 482 |
| `azure_openai_config` | `AzureOpenAIConfigForm \| None` | `None` | nested, line 487 |

The three nested forms carry `url` and `key`, and the Azure one adds
`version`. They are written only when `RAG_EMBEDDING_ENGINE` matches
`ollama`, `openai`, or `azure_openai` (lines 532 to 548); a null block
leaves the stored values alone.

Secrets, all plaintext on read: `openai_config.key` (line 463),
`ollama_config.key` (line 467), `azure_openai_config.key` (line 471).

Two operational notes for the documentation. `POST /embedding/update`
loads the embedding model into the process and unloads the previous one
(lines 504 to 583), so an apply that switches engines does real work and
can be slow. And `GET /config` calls `config.save()` (line 618), so a
read persists any key that was still serving a default. Both are
idempotent.

### 14.3 `openwebui_rag_config`: NEW RESOURCE

Endpoints: `GET /api/v1/retrieval/config`
(`backend/open_webui/routers/retrieval.py` at v0.11.0, line 615) and
`POST /api/v1/retrieval/config/update` (line 938).

The specification is `ConfigForm` (lines 850 to 935), about 70 fields,
plus the nested `WebConfig` (lines 773 to 847), about 60 more. Mirror
both classes field for field; the storage keys are in
`RETRIEVAL_CONFIG_KEYS` (lines 252 to 408). Reproducing 130 rows here
would go stale faster than the source, so treat those line ranges as the
authoritative field list and read them when you build the schema.

What the plan must record, because it is not obvious from the field list:

- **The `web` block is whole-form.** See 14.1. Always send every field.
- **`YOUTUBE_LOADER_TRANSLATION` cannot converge.** It is in `WebConfig`
  at line 841 but **absent from `RETRIEVAL_CONFIG_KEYS`**, verified by
  searching lines 252 to 408. The POST writes it to
  `request.app.state.YOUTUBE_LOADER_TRANSLATION` (line 1320) and the GET
  reads it back from process memory (line 762). It never reaches the
  database and is lost on restart. **Leave it out of the schema.** A
  Terraform attribute that silently resets on every container restart is
  worse than no attribute.
- **Four `FILE_*` fields use an empty string as the "clear to null"
  sentinel** (lines 1216 to 1227): `FILE_MAX_SIZE`, `FILE_MAX_COUNT`,
  `FILE_IMAGE_COMPRESSION_WIDTH`, and `FILE_IMAGE_COMPRESSION_HEIGHT`.
  Their Terraform type must be a string, not an int, or the resource
  cannot express "unlimited".
- **The POST response is not a full echo.** It omits
  `DATALAB_MARKER_FORMAT_LINES`, `DDGS_BACKEND`,
  `ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS`, `MINERU_FILE_EXTENSIONS`,
  and `RAG_RERANKING_BATCH_SIZE`, all of which the GET returns. **Refresh
  from `GET /config` after every write** rather than trusting the POST
  body. This is the same failure mode as the config-import bug in
  section 8.1.
- **Four keys are loaded but unreachable through this API:**
  `AZURE_AI_SEARCH_API_KEY`, `AZURE_AI_SEARCH_ENDPOINT`,
  `AZURE_AI_SEARCH_INDEX_NAME` (lines 254 to 256), and
  `TIKTOKEN_ENCODING_NAME` (line 385). Neither route reads or writes
  them. Note them in the documentation as reachable only through
  `openwebui_config_import`.

Secrets, all plaintext on read. Top level, from the GET response builder
at lines 619 to 770: `DATALAB_MARKER_API_KEY` (637),
`EXTERNAL_DOCUMENT_LOADER_API_KEY` (649),
`EXTERNAL_DOCUMENT_LOADER_HEADERS` (650, may carry an `Authorization`
header), `DOCLING_API_KEY` (653), `DOCUMENT_INTELLIGENCE_KEY` (656),
`MISTRAL_OCR_API_KEY` (659), `PADDLEOCR_VL_TOKEN` (662),
`MINERU_API_KEY` (666), `RAG_EXTERNAL_RERANKER_API_KEY` (675).

Under `web`: `OLLAMA_CLOUD_WEB_SEARCH_API_KEY` (707), `YACY_PASSWORD`
(713), `GOOGLE_PSE_API_KEY` (714), `BRAVE_SEARCH_API_KEY` (716),
`KAGI_SEARCH_API_KEY` (718), `MOJEEK_SEARCH_API_KEY` (719),
`BOCHA_SEARCH_API_KEY` (720), `SERPSTACK_API_KEY` (721),
`SERPER_API_KEY` (723), `SERPHOUSE_API_KEY` (724), `SERPLY_API_KEY`
(726), `TAVILY_API_KEY` (728), `SEARCHAPI_API_KEY` (729),
`SERPAPI_API_KEY` (731), `JINA_API_KEY` (733),
`BING_SEARCH_V7_SUBSCRIPTION_KEY` (736), `EXA_API_KEY` (737),
`PERPLEXITY_API_KEY` (738), `MICROSOFT_WEB_IQ_API_KEY` (743),
`SOUGOU_API_SK` (746), `FIRECRAWL_API_KEY` (752),
`EXTERNAL_WEB_SEARCH_API_KEY` (757), `EXTERNAL_WEB_LOADER_API_KEY`
(759), `YANDEX_WEB_SEARCH_API_KEY` (764), `YOUCOM_API_KEY` (766),
`LINKUP_API_KEY` (767).

Model on `resource_config_code_execution.go` for the singleton shape.
The `web` block is a `SingleNestedAttribute`;
`resource_config_tool_servers.go` is the nearest existing precedent for
nested objects, though it holds a list. The four free-form dict fields
(`EXTERNAL_DOCUMENT_LOADER_HEADERS`, `DOCLING_PARAMS`, `MINERU_PARAMS`,
`LINKUP_SEARCH_PARAMS`) go through `json_helpers.go`.

Given the size, consider splitting `web` into its own
`openwebui_web_search_config` resource. Both blocks write through the
same POST, so two resources would contend on one endpoint. Do not split
it. One resource, one endpoint.

### 14.4 `openwebui_images_config`: NEW RESOURCE

Endpoints: `GET /api/v1/images/config`
(`backend/open_webui/routers/images.py` at v0.11.0, line 271) and
`POST /api/v1/images/config/update` (line 276).

`ImagesConfig` (lines 228 to 268) has 33 fields, all required with no
Pydantic defaults, covering generation and editing across OpenAI,
Automatic1111, ComfyUI, and Gemini. Storage keys are in
`IMAGE_CONFIG_KEYS` (lines 67 to 102). Mirror the class.

Four things the field list does not tell you:

- **`IMAGE_CONFIG_KEYS` also maps `USER_PERMISSIONS`** (line 101), but
  `ImagesConfig` has no such field, so the `response_model` drops it from
  the GET and `config_updates` filters it from the write. This router
  never touches user permissions. Do not model it.
- **The server validates on write.** `IMAGE_SIZE` must match
  `^\d+x\d+$`, or be `auto`, or be empty (lines 288 to 293), and `auto`
  only when the model matches `IMAGE_AUTO_SIZE_MODELS_REGEX_PATTERN`
  (lines 278 to 286). `IMAGE_STEPS` must be non-negative (lines 295 to
  299). Mirror all three as schema validators so the plan rejects a bad
  value instead of the apply returning 400.
- **Both ComfyUI base URLs are stripped of a trailing slash before
  storage** (lines 302 to 303). Normalize in the provider or the state
  drifts whenever someone writes a trailing slash.
- **The POST is not side-effect free.** It calls `set_image_model` (line
  305), which for the `automatic1111` engine issues a live GET and POST
  to `/sdapi/v1/options` on the configured host (lines 176 to 197). That
  call swallows its own exception (lines 196 to 197), so an apply against
  an unreachable Automatic1111 still succeeds. Say so in the resource
  documentation: a successful apply does not prove the engine is
  reachable.

Secrets, all plaintext on read (the GET is
`return await get_config_values(IMAGE_CONFIG_KEYS)` at line 272):
`IMAGES_OPENAI_API_KEY`, `COMFYUI_API_KEY`, `IMAGES_GEMINI_API_KEY`,
`IMAGES_EDIT_OPENAI_API_KEY`, `IMAGES_EDIT_GEMINI_API_KEY`,
`IMAGES_EDIT_COMFYUI_API_KEY`, and `AUTOMATIC1111_API_AUTH`, which is a
`user:password` string base64-encoded only at call time (lines 322 to
329).

`COMFYUI_WORKFLOW_NODES` and `IMAGES_EDIT_COMFYUI_WORKFLOW_NODES` are
`list[dict]` with no fixed schema, and `IMAGES_OPENAI_API_PARAMS`,
`AUTOMATIC1111_API_AUTH`, and `AUTOMATIC1111_PARAMS` are
`dict | str | None`. All five go through `json_helpers.go` as JSON string
attributes.

### 14.5 `openwebui_audio_config`: NEW RESOURCE

Endpoints: `GET /api/v1/audio/config`
(`backend/open_webui/routers/audio.py` at v0.11.0, line 279) and
`POST /api/v1/audio/config/update` (line 287).

One resource with two mandatory nested blocks, because
`AudioConfigUpdateForm` (lines 274 to 276) requires both `tts` and `stt`.

- `TTSConfigForm`, lines 238 to 251, 13 fields, keys in `TTS_CONFIG_KEYS`
  (lines 79 to 93). Covers OpenAI, Azure Speech, and Mistral.
- `STTConfigForm`, lines 254 to 271, 17 fields, keys in `STT_CONFIG_KEYS`
  (lines 95 to 113). Covers local faster-whisper, OpenAI, Deepgram,
  Azure, and Mistral.

Both blocks are whole-form on write (lines 289 to 294). Four fields carry
Pydantic defaults and will be silently reset if the provider omits them:
`tts.OPENAI_PARAMS`, `stt.OPENAI_API_REQUEST_FORMAT`,
`stt.SUPPORTED_CONTENT_TYPES`, and `stt.ALLOWED_EXTENSIONS`.

Secrets, all plaintext on read (the GET returns
`get_config_values` for both key maps at lines 280 to 283):
`tts.OPENAI_API_KEY`, `tts.API_KEY` (the generic engine key used by
ElevenLabs and others), `tts.MISTRAL_API_KEY`, `stt.OPENAI_API_KEY`,
`stt.DEEPGRAM_API_KEY`, `stt.AZURE_API_KEY`, and `stt.MISTRAL_API_KEY`.

Operational note: when `stt.ENGINE` is the empty string the POST loads
faster-whisper into the process, downloading the model if it is not
cached (lines 296 to 301). An apply that switches to the local engine can
take minutes on its first run. Set an explicit timeout expectation in the
acceptance test.

Model on `resource_config_code_execution.go`, with two
`SingleNestedAttribute` blocks. The two `stt` list fields use
`list_helpers.go` and `tts.OPENAI_PARAMS` uses `json_helpers.go`.

### 14.6 Data sources and exclusions

| Route | Line | Verdict |
| --- | --- | --- |
| `GET /api/v1/images/config/url/verify` | images.py:332 | **DATA SOURCE, optional.** Takes no arguments; reports whether the currently configured engine answers. Model on `data_source_tool_server_verify.go`. |
| `GET /api/v1/images/models` | images.py:366 | **DATA SOURCE, optional.** Static for OpenAI and Gemini, live for ComfyUI. |
| `GET /api/v1/audio/models` | audio.py:1332 | **DATA SOURCE, optional.** |
| `GET /api/v1/audio/voices` | audio.py:1437 | **DATA SOURCE, optional.** Useful for validating `tts.VOICE`. |
| `POST /retrieval/process/*` (6 routes) | retrieval.py:1836, 2087, 2131, 2132, 2552, 3010 | **EXCLUDED (b).** Ingests and embeds documents. |
| `POST /retrieval/query/doc`, `/query/collection` | retrieval.py:2743, 2809 | **EXCLUDED (b).** Runs retrieval. |
| `POST /retrieval/delete`, `/reset/db`, `/reset/uploads` | retrieval.py:2878, 2938, 2954 | **EXCLUDED (b).** Destructive actions. |
| `GET /retrieval/ef/{text}` | retrieval.py:2989 | **EXCLUDED (b).** Computes an embedding, and only registered when `ENV == 'dev'` (line 2987). |
| `POST /images/generations`, `POST /images/edit` | images.py:558, 852 | **EXCLUDED (b).** Produce images. |
| `POST /audio/speech`, `POST /audio/transcriptions` | audio.py:556, 1179 | **EXCLUDED (b).** Synthesize and transcribe. |

### 14.7 A warning about version drift

Between v0.9.6 and v0.11.0 these three routers changed by 1,742
insertions and 1,241 deletions. `images.py` moved its entire config
surface from `request.app.state.config` to the per-key `config` table.
`TTSConfigForm` happens to be byte-identical across the two tags, but
that is luck rather than stability. These four resources will need
review at every Open WebUI minor release. State that in the README
alongside the compatibility table from section 9.

## 15. Identity, permissions, tasks, and evaluations

**Depends on: section 1. The resources below are independent of each
other and of every other section, except 15.4, which shares
`permissions_helpers.go` with section 7 and should land after it.**

No secret in this section is masked. `Config.get` and `Config.get_many`
(`backend/open_webui/models/config.py` at v0.11.0, lines 135 to 162)
return the raw stored JSON with no redaction. Both the LDAP bind password
and the OAuth client secret come back in plaintext, so state converges
for both. Mark them `Sensitive: true`.

### 15.1 `openwebui_admin_config`: NEW RESOURCE

Endpoints: `GET /api/v1/auths/admin/config`
(`backend/open_webui/routers/auths.py` at v0.11.0, line 1184) and
`POST /api/v1/auths/admin/config` (line 1220).

`AdminConfig` (lines 1189 to 1217) has 28 fields covering signup, the
default user role, API key settings, JWT expiry, and the enable flags for
folders, automations, channels, calendar, memories, and notes. Storage
keys are in `ADMIN_CONFIG_KEYS` (lines 104 to 133). Mirror the class; the
GET has no `response_model` and returns all 28 keys as a bare dict.

**The server drops invalid values instead of rejecting them.** Lines 1229
to 1240:

```python
if form_data.DEFAULT_USER_ROLE not in ['pending', 'user', 'admin']:
    updates.pop('ui.default_user_role', None)
if form_data.CHANNEL_MODEL_RESPONSE_MODE not in ['thread', 'channel']:
    updates.pop('channels.model_response_mode', None)
if not re.match(r'^(-1|0|(-?\d+(\.\d+)?)(ms|s|m|h|d|w))$', form_data.JWT_EXPIRES_IN):
    updates.pop('auth.jwt_expiry', None)
```

The request still returns 200. Terraform would show permanent drift with
no error to explain it. **Add all three as schema validators** so the plan
fails loudly instead.

Three fields typed `int | str | None` (`FOLDER_MAX_FILE_COUNT`,
`AUTOMATION_MAX_COUNT`, `AUTOMATION_MIN_INTERVAL`) are cast to `int` when
truthy and written as `''` otherwise (lines 1222 to 1226). So null, zero,
and empty string all store `''`. **Model them as strings**, or an
Int64 attribute cannot round-trip and zero becomes inexpressible.

Model on `resource_config_code_execution.go`.

### 15.2 `openwebui_ldap_config`: NEW RESOURCE

Merge two routes into one resource: `GET`/`POST
/api/v1/auths/admin/config/ldap/server` (lines 1264 and 1269) carry the
16-field `LdapServerConfig` (lines 1245 to 1261), and `GET`/`POST
/api/v1/auths/admin/config/ldap` (lines 1295 and 1304) carry the single
`enable_ldap` flag. One object matches how the admin panel presents it.

Three fields are new at v0.11.0 and absent from the v0.9.6 class:
`enable_group_management`, `enable_group_creation`, and
`attribute_for_groups`.

`app_dn_password` maps to `ldap.server.app_password` (line 143) and is
declared `str` on the response model (line 1252), so **the GET returns
the bind password in plaintext**. State converges. Mark it Sensitive.

Watch the asymmetric field naming on the enable flag: the POST takes
`{"enable_ldap": bool}` (line 1305) but the GET returns
`{"ENABLE_LDAP": ...}` (line 1297). The client must translate.

Unlike the admin config, this route **does** raise on bad input: `label`,
`host`, `attribute_for_mail`, `attribute_for_username`, and `search_base`
must be non-empty (lines 1271 to 1281), and `attribute_for_groups` must
be non-empty when `enable_group_management` is true (lines 1285 to 1286).

### 15.3 `openwebui_oauth_config`: EXCLUDED, decided 2026-08-14

Chris decided to skip this resource. The reasons below stand as the
documented exclusion: a write is not persisted unless the instance sets
`ENABLE_OAUTH_PERSISTENT_CONFIG`, and deployments normally own OAuth
settings through the environment. Record it as excluded in the section
16 coverage table with this rationale. The analysis stays here in case
a persistent-config instance ever wants the resource.

Endpoints: `GET`/`POST /api/v1/auths/admin/config/oauth` (lines 1443 and
1448), carrying the 34-field `OAuthConfigForm` (lines 1315 to 1364).

**Read this before building it. OAuth config is not persisted by
default.** `Config.persistent_enabled_for`
(`backend/open_webui/models/config.py` at v0.11.0, lines 131 to 137)
returns `False` for any key starting with `oauth.` unless
`OAUTH_PERSISTENT_ENABLED` is set, and that comes from
`ENABLE_OAUTH_PERSISTENT_CONFIG`, which **defaults to `False`**
(`backend/open_webui/config.py` at v0.11.0, line 3187). With it off, a
write lands in the in-process `Config.DEFAULTS` dict rather than the
database. The apply succeeds, the read-back matches, and the whole thing
evaporates on restart and never reaches another replica.

So the resource must document `ENABLE_OAUTH_PERSISTENT_CONFIG=true` as a
prerequisite. Consider failing the apply with a clear diagnostic when it
is off, rather than writing state that will silently vanish. That check
needs a way to read the flag, which no API exposes, so the honest option
is a loud note in the schema description plus a required
`acknowledge_non_persistent` style opt-in. Put this decision to Chris
before building.

This is the one **partial-write** surface here: the POST uses
`model_dump(exclude_none=True)` (line 1450), so omitted fields are left
alone and a field can never be reset to null. Every attribute must be
`Optional` and `Computed`.

Three fields translate between a comma string and a list:
`OAUTH_ALLOWED_DOMAINS`, `OAUTH_ADMIN_ROLES`, and `OAUTH_ALLOWED_ROLES`
(`OAUTH_COMMA_LIST_FIELDS`, lines 1367 to 1371). The round trip is stable.
`OAUTH_BLOCKED_GROUPS` is **not** in that set even though it defaults to
the JSON string `'[]'`; treat it as an opaque string.

`OAUTH_GROUP_DEFAULT_SHARE` is typed `bool | str | None` (line 1336). Go
cannot hold that in one typed attribute. Model it as a string and
document that `"true"` and `"false"` are accepted.

Scope limit worth stating in the documentation: this router reads only
the generic OIDC keys. The `oauth.google.*`, `oauth.microsoft.*`,
`oauth.github.*`, and `oauth.feishu.*` families
(`backend/open_webui/config.py` at v0.11.0, lines 3119 to 3151) are not
exposed here and remain reachable only through
`openwebui_config_import`.

### 15.4 `openwebui_default_user_permissions`: NEW RESOURCE

**Land this after section 7**, which is already editing
`permissions_helpers.go`.

Endpoints: `GET /api/v1/users/default/permissions`
(`backend/open_webui/routers/users.py` at v0.11.0, line 392) and
`POST /api/v1/users/default/permissions` (line 405). It writes the whole
`user.permissions` blob as one object, and all six sections are required
(`UserPermissions`, lines 260 to 266, gives none of them a default), so a
partial POST is a 422.

The six section classes are `WorkspacePermissions` (line 175),
`SharingPermissions` (line 191), `AccessGrantsPermissions` (line 209),
`ChatPermissions` (line 214), `FeaturesPermissions` (line 240), and
`SettingsPermissions` (line 256). They are the same shape as group
permissions, so reuse `permissionsAttrTypes()` from section 7.

**Two traps, both verified in the source, both specific to this route.**

1. **`sharing.open_chats` does not exist on this form.**
   `DEFAULT_USER_PERMISSIONS` carries it
   (`backend/open_webui/config.py` at v0.11.0, line 1950) and section 7
   correctly adds it to the *group* key set, but `SharingPermissions`
   (`routers/users.py` lines 191 to 206) has no such field. The POST
   writes `model_dump(by_alias=True)` over the whole blob (lines 405 to
   417), so **every write through this route deletes `open_chats` from
   stored config.** Drop it from this resource's key set, and say in the
   schema description that applying this resource clears
   `sharing.open_chats`. Do not silently reuse the group key list.
2. **Two Pydantic defaults disagree with the environment defaults.**
   `SharingPermissions.public_tools` and `.public_notes` default to
   `True` on the form, while `DEFAULT_USER_PERMISSIONS` sets both to
   `False` (`config.py` lines 1794 and 1809). `GET` fills missing
   sub-keys from the class, not from the environment defaults, so a blob
   missing either key reads back as `true`. Never let a provider default
   lean on either value; require them explicitly.

Use `"import"`, not `"import_"`, on the wire: `ChatPermissions.import_`
carries `alias='import'` (line 231) and the POST dumps `by_alias=True`.
This matches the same note in section 7.1.

`GET /api/v1/users/default/permissions/defaults` (line 419), new at
v0.11.0, reads the module-level `DEFAULT_USER_PERMISSIONS` and has no
matching POST. **DATA SOURCE, optional and low value.** It exposes
factory settings for diffing. Skip it unless someone asks.

### 15.5 `openwebui_user`: PROMOTE FROM DATA SOURCE TO RESOURCE

Full CRUD exists across two routers.

| Operation | Method and path | Line |
| --- | --- | --- |
| Create | `POST /api/v1/auths/add` | auths.py:1081 |
| Read | `GET /api/v1/users/{user_id}` | users.py:745 |
| Update | `POST /api/v1/users/{user_id}/update` | users.py:866 |
| Delete | `DELETE /api/v1/users/{user_id}` | users.py:990 |

`AddUserForm` (`backend/open_webui/models/auths.py` at v0.11.0, lines 93
to 95, extending `SignupForm` at line 80) is `name`, `email`, `password`,
`profile_image_url` (default `/user.png`), and `role` (default
`pending`). Create is admin-only (`auths.py:1085`).

`UserUpdateForm` (`backend/open_webui/models/users.py` at v0.11.0, lines
262 to 267) is `role`, `name`, `email`, `profile_image_url`, and
`password`, all optional, applied only when not null (`users.py` lines
922 to 930). Genuinely partial.

Six limits to encode and document:

1. **Password is write-only.** No read path returns it, so the provider
   cannot detect an out-of-band change. Make it Optional and Sensitive,
   never Computed, and accept that there is no drift detection. A
   write-only attribute or a `password_version` trigger is the cleaner
   shape; pick one and say which.
2. **ImportState works** on the user UUID against
   `GET /api/v1/users/{user_id}`, for every attribute except `password`,
   which lands null. `UserActiveResponse` carries `extra='allow'`, so the
   full user dump comes through.
3. **The primary admin is protected.** A non-primary admin cannot update
   the first user, and the first user cannot demote itself (`users.py`
   lines 875 to 895). Deleting the first user is refused (lines 993 to
   1004), as is self-deletion (lines 1029 to 1033). Terraform gets a 403,
   not a no-op. Surface it as a clear diagnostic.
4. **`role` has no server-side enum** on `UserUpdateForm`. Validate
   `pending`, `user`, `admin` client-side, matching what `auths.py:1229`
   enforces for `DEFAULT_USER_ROLE`.
5. **Do not put `groups` on this resource.** There is no route in
   `users.py` that sets a user's groups, and `openwebui_group` already
   owns membership. Two resources writing one edge is a fight.
6. **Create is not idempotent.** `POST /auths/add` returns 400 on a taken
   email (`auths.py:1092`). There is no upsert.

The create response is `SigninResponse` and **includes a live JWT for the
new user** (`auths.py` lines 1130 to 1139). The client must read `id` and
discard the token. Never persist it to state.

Extend `internal/client/users.go`, which today has only `SearchUsers` and
`GetUser`. Model the resource on `resource_group.go`.

### 15.6 `openwebui_task_config`: NEW RESOURCE

The cleanest surface in this whole plan. Endpoints:
`GET /api/v1/tasks/config` (`backend/open_webui/routers/tasks.py` at
v0.11.0, line 78) and `POST /api/v1/tasks/config/update` (line 104). Note
the asymmetric paths.

`TaskConfigForm` (lines 83 to 101) has 18 fields covering the task model,
title generation, tag generation, autocomplete, query generation, follow
ups, the tools function-calling template, and voice mode. `TASK_CONFIG_KEYS`
(lines 40 to 59) maps them one to one, with no drops on write and no
extras on read, and both routes return `get_config_values(TASK_CONFIG_KEYS)`
(lines 80 and 107). Read-after-write converges exactly.

Every field is required. An empty-string template means "use the built-in
default", and the handlers fall back to the `DEFAULT_*_PROMPT_TEMPLATE`
constants imported at lines 7 to 17. **Treat `""` as a real value, not as
null.**

Authorization is asymmetric: the GET is `get_verified_user` (line 79) and
the POST is `get_admin_user` (line 105). No effect on the provider, which
uses an admin token.

Model on `resource_config_code_execution.go`.

### 15.7 `openwebui_evaluation_config`: NEW RESOURCE

Endpoints: `GET /api/v1/evaluations/config`
(`backend/open_webui/routers/evaluations.py` at v0.11.0, line 268) and
`POST /api/v1/evaluations/config` (line 283). Both return
`get_config_values(EVALUATION_CONFIG_KEYS)` (lines 269 and 294), so
read-after-write converges.

`UpdateConfigForm` (lines 278 to 280) has two fields:
`ENABLE_EVALUATION_ARENA_MODELS` (`bool | None`, key
`evaluation.arena.enable`) and `EVALUATION_ARENA_MODELS`
(`list[dict] | None`, key `evaluation.arena.models`). Each is applied
only when not null (lines 289 to 292), so this is a partial write.

`EVALUATION_ARENA_MODELS` has **no Pydantic model**; the API never
validates the elements. The consumer at
`backend/open_webui/utils/models.py` lines 99 to 112 requires `id`,
`name`, and `meta` on each and raises `KeyError` on a malformed entry.
Because the elements are untyped, carry the list as a JSON string
attribute through `json_helpers.go`. A typed nested block would read
better but would reject keys a future release adds.

One behavior to document: an empty list is not the same as disabled. With
`evaluation.arena.enable` true and the list empty,
`utils/models.py` lines 113 to 127 injects `DEFAULT_ARENA_MODEL`
(`config.py` lines 2046 to 2054) at request time. That injection is never
written back to config, so Terraform sees no drift, but users still see
an arena model.

Model on `resource_config_models.go`, which already carries a
list-of-objects config payload.

### 15.8 Exclusions in this slice

| Surface | Verdict |
| --- | --- |
| `POST`/`GET`/`DELETE /api/v1/auths/api_key` | **EXCLUDED (b).** Mints and revokes the API key of the *calling* identity; there is no user-id parameter, so it can only ever manage the credential Terraform is authenticating with. |
| `/signin`, `/signup`, `/signout`, `/ldap`, `/oauth/{provider}/token/exchange`, `/oauth/sessions/{provider}` | **EXCLUDED (b).** Login and session flows. |
| `/update/profile`, `/update/timezone`, `/update/password` | **EXCLUDED (b).** Act on the calling user's own session. |
| `GET /api/v1/auths/admin/details` | **EXCLUDED (d).** A derived read of `auth.admin.email` plus a user lookup; adds nothing over 15.1. |
| `/api/v1/users/user/*` (settings, status, info, variables) | **EXCLUDED (b).** Caller-scoped self-service. `user/variables` is new at v0.11.0 and is per-user chat substitution, not instance config. |
| `GET /api/v1/users/usage`, `/{user_id}/oauth/sessions`, `/{user_id}/active`, `/{user_id}/profile/image` | **EXCLUDED (b).** Runtime and analytics. |
| `GET /api/v1/users/groups`, `/{user_id}/groups` | **EXCLUDED (d).** Overlaps `openwebui_group.users`. |
| `GET /api/v1/users/permissions` | **EXCLUDED (b).** Effective permissions for the caller. |
| All `/api/v1/tasks/*/completions` (8 routes) | **EXCLUDED (b).** Live LLM inference. |
| All `/api/v1/evaluations/feedback*` and `/leaderboard*` | **EXCLUDED (b).** User-submitted ratings and derived rankings. |

### 15.9 A security check that already passes

`GET /api/v1/configs/export` returns `Config.get_all()`, which is **every
key in the instance in plaintext**, including
`ldap.server.app_password` and `oauth.client_secret`. Both
`data_source_config_export.go` (line 45) and `resource_config_import.go`
(line 62) already mark their `config_json` attribute `Sensitive: true`,
so nothing needs fixing. Keep it that way: the new resources in this
section add more secrets to that same blob.

## 16. Router coverage index

Every router in `backend/open_webui/routers/` at v0.11.0, with a verdict.
This table is what makes "complete coverage" a checkable claim rather
than an assertion. There are 31 routers.

Exclusion reasons: **(a)** no read route, so state can never converge;
**(b)** runtime data rather than configuration; **(c)** secrets returned
encrypted or masked; **(d)** overlaps a surface the provider already
manages.

| Router | Routes | Verdict | Section |
| --- | --- | --- | --- |
| `analytics.py` | 8 | EXCLUDED (b) | 16.1 |
| `audio.py` | 6 | NEW RESOURCE | 14.5 |
| `auths.py` | 23 | NEW RESOURCE (3) | 15.1, 15.2, 15.3 |
| `automations.py` | 8 | EXCLUDED (a), (b) | 16.1 |
| `calendar.py` | 13 | EXCLUDED (b) | 16.1 |
| `channels.py` | 28 | NEW RESOURCE | 12 |
| `chats.py` | 50 | NEW RESOURCE (config only), rest EXCLUDED (b) | 13, 16.1 |
| `configs.py` | 25 | MIXED: 10 covered, 3 new | 8, 10, 11 |
| `evaluations.py` | 15 | NEW RESOURCE | 15.7 |
| `files.py` | 14 | COVERED | 6.3 |
| `folders.py` | 11 | EXCLUDED (b) | 16.1 |
| `functions.py` | 17 | COVERED | 5.5 |
| `groups.py` | 11 | COVERED | 7.1 |
| `images.py` | 6 | NEW RESOURCE | 14.4 |
| `knowledge.py` | 35 | COVERED, plus one optional new resource | 6.1, 6.2, 16.2 |
| `memories.py` | 11 | EXCLUDED (a), (b) | 16.1 |
| `models.py` | 15 | COVERED | 3 |
| `notes.py` | 12 | EXCLUDED (b) | 16.1 |
| `notifications.py` | 7 | EXCLUDED (b), (c) | 16.1 |
| `ollama.py` | 46 | COVERED | 8.3 |
| `openai.py` | 9 | COVERED | 8.3 |
| `pipelines.py` | 8 | COVERED | 8.3 |
| `prompts.py` | 15 | COVERED | 5.4 |
| `retrieval.py` | 16 | NEW RESOURCE (2) | 14.2, 14.3 |
| `scim.py` | 15 | EXCLUDED (a), (d) | 16.1 |
| `skills.py` | 9 | COVERED | 4 |
| `tasks.py` | 10 | NEW RESOURCE | 15.6 |
| `terminals.py` | 3 | EXCLUDED (d) | 16.1 |
| `tools.py` | 15 | MIXED: covered, plus tool servers | 5.1, 10 |
| `users.py` | 26 | MIXED: data source covered, 2 new | 7.3, 15.4, 15.5 |
| `utils.py` | 5 | EXCLUDED (b), (d) | 16.1 |

Counts: 10 COVERED, 8 NEW RESOURCE, 3 MIXED, 10 EXCLUDED. One of
the COVERED routers, `knowledge.py`, also carries one optional new
resource; see section 16.2.

Two routers are mounted conditionally. `analytics.py` mounts only when
`ENABLE_ADMIN_ANALYTICS` is true, which is the default
(`backend/open_webui/config.py` at v0.11.0, line 2072). `scim.py` mounts
only when `ENABLE_SCIM` is true, which is **not** the default
(`backend/open_webui/env.py` at v0.11.0, line 851). Both are excluded
anyway.

### 16.1 The exclusions, with evidence

Each of these is excluded on a fact, not on taste.

**`analytics.py`, excluded (b).** Every route is a GET taking date and group
filters and returning counts computed on the fly, for example
`ChatMessages.get_message_count_by_model(...)` at line 66. There is no
create, update, or delete in the file and no persisted object to own.

**`automations.py`, excluded (a) and (b).** `check_automation_access` is
`if not automation or user.id != automation.user_id: raise 404`
(lines 59 to 63) with **no admin bypass**, unlike `folders.py:290` or
`calendar.py:68`. An admin token reading another user's automation gets
404, so state cannot converge. The rrule schedule is also bound to
`user.timezone` (lines 178 and 188). The instance-level knobs
(`automations.enable`, `automations.max_count`,
`automations.min_interval`) live in `Config` and are covered by
`openwebui_admin_config` in section 15.1.

**`calendar.py`, excluded (b).** The closest call among the exclusions. Calendars
do carry `access_grants` (`models/calendar.py:116`) and `GET
/{calendar_id}` does admit an admin (`calendar.py:68`), so it is
technically importable. But nothing here is admin-curated: creation is
gated only on the `features.calendar` user permission (lines 44 to 48)
with no admin branch, `is_default` is per-user state, and the only
system-owned calendar is synthesized in memory per request and never
persisted (`SCHEDULED_TASKS_CALENDAR_ID`, line 34, appended at lines 94
to 105). `calendar.enable` is covered by section 15.1.

**`chats.py`, excluded (b), except `/config`.** Transcripts and their per-user
lifecycle: pin, archive, fork, clone, tag, share, mark read. `POST
/shared/{id}/access/update` (line 2017) does write `access_grants` for
`resource_type='shared_chat'`, but that grant hangs off a conversation a
user chose to share. The admin-only reads are gated behind
`ENABLE_ADMIN_CHAT_ACCESS` (lines 717 to 720) and `ENABLE_ADMIN_EXPORT`
(lines 1030 to 1032). The context-compaction config pair is section 13.

**`folders.py`, excluded (b).** `FolderModel` does **not** carry `access_grants`
(`models/folders.py` lines 36 to 48); only `SharedFolderResponse` does
(line 73), and `GET /{id}` declares `response_model=None` and
hand-attaches grants onto a dict (lines 287 to 288). Uniqueness is scoped
per owner (lines 78 to 80) and deleting a folder cascades into
`Chats.delete_chats_by_user_id_and_folder_id` for the owner's chats (line
689). `POST /{id}/update/expanded` (line 435) and `POST /{id}/read` (line
599) are UI view state and read receipts. `folders.enable` and
`folders.max_file_count` are covered by section 15.1.

**`memories.py`, excluded (a) and (b).** There is no `GET /{memory_id}`; the only
reads are `GET /` (line 56, hard-scoped to
`Memories.get_memories_by_user_id(user.id)`) and the search POSTs, all
caller-scoped. An admin token can never read another user's memory, so
state cannot converge. The content is conversation-derived: the column
comment reads `content = Column(Text)  # free-form text learned from
conversation` (`models/memories.py:25`). `memories.enable` is covered by
section 15.1.

**`notes.py`, excluded (b).** A per-user rich-text document: `NoteForm` is
`title` plus a free-form `data` dict (`models/notes.py` lines 65 to 69),
whose body is editor-authored markdown at `data.content.md`. Creation is
gated only on the `features.notes` user permission (lines 212 to 219)
with no admin branch, in contrast to the explicit `user.role != 'admin'`
gate on standard channels at `channels.py:288`. A read-by-id does exist
(line 255), so this is excluded on content grounds, not reachability.
`notes.enable` is covered by section 15.1.

**`notifications.py`, excluded (b) and (c).** The only new router file at
v0.11.0. There is no table: targets live inside
`user.settings['notifications']`, written through
`Users.update_user_settings_by_id` (`utils/notifications.py` lines 129,
176, and 209). Every route is scoped to `user.id` from the token with no
admin path. And the read masks the secret: `_public_target` pops
`config['url']` and substitutes `config['url_masked']`
(`utils/notifications.py` lines 81 to 93), so `GET /targets` never
returns the webhook URL that was written.

**`scim.py`, excluded (a) and (d).** A SCIM 2.0 façade over the same `user` and
`group` tables; the handlers import `Groups`, `GroupModel`, `Users`, and
`UserModel` directly (lines 22 to 23), and the file's own docstring calls
it experimental. It is not a separate config surface: the three metadata
routes (lines 401, 423, 453) return hardcoded literals with no writer
anywhere in the file, and the real settings (`ENABLE_SCIM`, `SCIM_TOKEN`,
`SCIM_AUTH_PROVIDER`) are plain `os.getenv` reads at
`backend/open_webui/env.py` lines 851 to 853, never `Config` rows, so no
API can write them. The provisioning routes duplicate `openwebui_user`
and `openwebui_group`.

**`terminals.py`, excluded (d).** A reverse proxy, as its module docstring states
(lines 1 to 6). `GET /` reads the admin-set list out of `Config` and
returns only `id`, `url`, and `name` (lines 71 to 82); the catch-all
route and the websocket forward bytes to the upstream server. The
declarable object is `terminal_server.connections`, whose entire API
lives in `routers/configs.py` and is section 11.2.

**`utils.py`, excluded (b) and (d).** Five one-shot actions: derive a gravatar
URL (line 25), run `black.format_str` over a string (line 34), execute
code in Jupyter (lines 52 to 65), render a chat to PDF (line 83), and
stream the SQLite file (lines 96 to 111). No route creates, reads, or
deletes a stored object. The only configuration it touches is read-only
and already owned by `openwebui_code_execution_config` (lines 46 to 70).

### 16.2 External knowledge connections: NEW RESOURCE, optional

The one declarative surface in `knowledge.py` the provider does not
cover. It is genuinely new at v0.11.0 and genuinely admin-level, so it
belongs in a complete-coverage plan, but it is low priority: it only
matters to an instance that puts its vectors in an external store.

Full CRUD over a list stored under the single `Config` key
`external_knowledge.connections`
(`backend/open_webui/routers/knowledge.py` at v0.11.0, line 502):

| Operation | Method and path | Line |
| --- | --- | --- |
| List | `GET /api/v1/knowledge/external/connections` | 632 |
| Create | `POST /api/v1/knowledge/external/connections` | 641 |
| Read | `GET /api/v1/knowledge/external/connections/{id}` | 662 |
| Update | `PATCH /api/v1/knowledge/external/connections/{id}` | 674 |
| Delete | `DELETE /api/v1/knowledge/external/connections/{id}` | 700 |

`ExternalKnowledgeConnectionForm` (lines 487 to 494) is `name`,
`provider`, `endpoint`, `auth_config`, `config`, `capabilities`, and
`enabled`. `provider` must be one of `qdrant`, `milvus`, or `pgvector`
(`EXTERNAL_KNOWLEDGE_PROVIDERS`, line 503, enforced at lines 508 to 511).
Mirror that as a schema validator.

**`auth_config` is never returned.** `_sanitize_external_connection`
(line 565) does `sanitized.pop('auth_config', None)` and substitutes
`auth_configured: bool`. Every read route runs it, including the create
and update responses (lines 637, 651, 671, and 689). So this surface
converges on everything except its credentials.

That does not make it excludable, because there is a precedent in this
plan: treat `auth_config` exactly as section 15.5 treats the user
password. Write-only and Sensitive, never Computed, with no drift
detection, and surface the server's `auth_configured` boolean as a
Computed attribute so at least the presence or absence of credentials is
visible in state.

Unlike tool servers in section 10, **the id is server-generated**:
`_external_connection_dict` assigns `str(uuid.uuid4())` (line 583).
Terraform cannot choose it. ImportState works by that id.

The list is stored under one `Config` key, so this resource has the same
read-modify-write race as section 10.6 and must reuse the same mutex.

Excluded from this resource, all reason (b), runtime actions: `POST
/external/connections/{id}/test` (729), `POST /external/source/test`
(801), and `POST /external/connections/{id}/retrieve-test` (825). The
three creation helpers `POST /external/knowledge/create` (841), `POST
/external/source/create` (891), and `PATCH /external/source/{id}` (956)
produce ordinary knowledge bases marked `meta.source == 'external'`;
they are an alternate constructor for the object `openwebui_knowledge`
already manages, so **excluded, reason (d)**.

### 16.3 What "complete" does not include

Three honest gaps, so the claim is not overstated.

1. **Provider-specific OAuth keys.** `oauth.google.*`,
   `oauth.microsoft.*`, `oauth.github.*`, and `oauth.feishu.*`
   (`backend/open_webui/config.py` at v0.11.0, lines 3119 to 3151) have
   no typed API surface. Reachable only through
   `openwebui_config_import`.
2. **Four RAG keys with no route.** `AZURE_AI_SEARCH_API_KEY`,
   `AZURE_AI_SEARCH_ENDPOINT`, `AZURE_AI_SEARCH_INDEX_NAME`, and
   `TIKTOKEN_ENCODING_NAME`, per section 14.3.
3. **`YOUTUBE_LOADER_TRANSLATION`**, which is deliberately excluded
   because it never reaches the database, per section 14.3.

All three are reachable through `openwebui_config_import`, which is the
untyped escape hatch. Say so in that resource's documentation, along with
the warning that managing a key through both `config_import` and a typed
resource makes the two fight.

### 16.4 Suggested order for Part two

The sections are independent, so this is about value, not dependency.

1. **Section 10, tool servers.** On the critical path. Five hand-made MCP
   connections are waiting to be imported.
2. **Sections 11.1 and 15.6**, subagents and task config. Small, clean,
   fully converging surfaces. Good first pickups for new agents.
3. **Section 15.4**, default user permissions. Reuses section 7's work.
4. **Sections 14.4 and 14.5**, images and audio. Medium, self-contained.
5. **Section 15.5**, the user resource. Higher risk, because password
   handling and the primary-admin guards need care.
6. **Sections 14.2 and 14.3**, the RAG pair. The largest schemas in the
   plan.
7. **Sections 12, 13, 15.1, 15.2, 15.7**, and **11.2**. Independent, take
   them as capacity allows.
8. **Section 15.3**, OAuth config: excluded, decided 2026-08-14. See the
   section for the rationale; nothing to build.

---

## Integration pass, 2026-08-14

The whole suite ran against the throwaway `ghcr.io/open-webui/open-webui:v0.11.0`
container from `scripts/testacc-up.sh`, with Terraform 1.15.8 and
`OPENWEBUI_TEST_CONFIG_IMPORT=1`.

**Result: 64 acceptance tests, 58 pass, 6 skip, 0 fail.** The six skips all
need a second service to point at: `OPENWEBUI_OAUTH_CLIENT_URL`,
`OPENWEBUI_TEST_OPENAI_URL`, `OPENWEBUI_PIPELINE_URL` (three tests), and
`OPENWEBUI_TOOL_SERVER_URL`.

Two tests failed on the first run. Both were faults in the tests, not in the
provider:

- `TestAccModelResourceProfileImageURL` ended on a config the new
  `profile_image_url` validator rejects. The post-test destroy plans the last
  config, so the destroy failed too. The `ExpectError` step now runs first.
- `TestAccSkillResource_ReadGroups` dropped both the group resource and the
  `read_groups` attribute in one step. `read_groups` is Optional and Computed,
  so the plan kept the group name from state while Terraform deleted the group,
  and the update failed to resolve it. The test now revokes with
  `read_groups = []` while the group still exists, then removes the group in a
  third step.

Section 4.4's coupling step is now
`TestAccModelResourceSkillIDsFromSkillResource`: a model whose `skill_ids`
points at an `openwebui_skill` resource in the same config.

Hand-verified through a real `terraform apply` with `dev_overrides`: a skill, a
knowledge base, and a model carrying `skill_ids`, `knowledge_ids`, and
`public_read = true`. On the wire the model held `meta.skillIds` with the skill
id, `meta.knowledge` with the resolved knowledge object, and a single
`user`/`*`/`read` grant. The re-plan was empty and the destroy left nothing
behind. `GET /api/v1/skills/id/{id}` returned `content`, which settles open
question 1 the way section 4.2 hoped.

Two limits the provider's `profile_image_url` validator does not model, both
operator-tunable on the server: `PROFILE_IMAGE_MAX_DATA_URI_SIZE` (unset by
default, so no limit) and `PROFILE_IMAGE_ALLOWED_MIME_TYPES`. An instance that
narrows either one can still drop a value the provider accepted.

## Integration pass for part two, 2026-08-14

The whole suite ran against the same throwaway `v0.11.0` container, with
Terraform 1.15.8, `OPENWEBUI_TEST_CONFIG_IMPORT=1`, and the primary-admin
variables `scripts/testacc-up.sh` now writes into `.env` from the signup
response.

**Result: 100 acceptance tests, 93 pass, 7 skip, 0 fail**, plus 216 provider
unit tests and the client package. Every skip needs a second service the harness
does not run: `OPENWEBUI_OAUTH_CLIENT_URL`, `OPENWEBUI_TEST_OPENAI_URL`,
`OPENWEBUI_PIPELINE_URL` (three tests), `OPENWEBUI_TOOL_SERVER_URL`, and
`OPENWEBUI_ACC_WHISPER`, which downloads a faster-whisper model.

Four tests failed on the first run. Where the v0.11.0 source contradicts this
plan, the source won.

- **The static OAuth branch was unreachable.** `OAuthClientRegistrationForm` now
  carries `client_secret` and `oauth_server_url` (section 8.2 listed both as
  pre-existing gaps), and `openwebui_oauth_client` exposes them, the secret as
  Sensitive. Without them `configs.py:186` could never take the static branch,
  so an `oauth_2.1_static` tool server had no registration to overlay.
- **A ComfyUI base URL cannot be normalised in state.** Open WebUI stores it
  through `str.strip('/')`, and the provider trims the request to match. It must
  *not* trim the value it records: Terraform rejects both a plan value that
  differs from the configuration and an applied value that differs from the
  plan. `engineTrimmedURLValue` keeps the recorded form while it still names the
  server the API returned, which is what makes a URL written with a slash
  converge.
- **A config resource has no remote delete, so its last step is what the
  instance keeps.** `TestAccRAGEmbeddingConfigResource_OpenAIBlock` left the
  embedding engine on `openai` with a fake key, and every later test that
  attached a document then failed against `api.openai.com`. It now restores the
  built-in engine in a third step.
- **The primary admin cannot be left in Terraform state.** The demotion test
  needs `ImportStatePersist`, or its second step creates an account instead of
  updating one. It then has to drop the account with a `removed` block, because
  the provider refuses to delete the primary admin and the closing destroy would
  fail.
- `WEBUI_URL` is empty on a fresh instance, so `TestAccAdminConfigResource`
  proves the unnamed-settings read with `channel_model_response_mode` instead.

Hand-verified through `terraform import` with `dev_overrides`: an
`oauth_2.1` MCP connection carrying a registration blob and an unmodelled
`info` key, seeded by raw API call, plus an OpenAPI connection. Both imported,
both planned empty, and an apply that changed the OpenAPI server's `path` and
the MCP server's `description` left the blob and the unmodelled key identical
byte for byte. The destroy removed both and left
`TOOL_SERVER_CONNECTIONS` empty.

`golangci-lint` v2.12.2, which CI runs, reported one `ineffassign` in
`flattenAudioConfig`: `priorSTT` was built and never read, because no STT
attribute carries recorded JSON text.
