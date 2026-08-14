# Acceptance baseline against Open WebUI v0.11.0

The work list for sections 3 through 8 of `upgrade-to-0.11.md`. Recorded on
2026-08-14 against the unmodified provider at commit `ee6d34c`, plus the
section 2 client change.

How it was run:

```bash
make testacc-up
TF_ACC_TERRAFORM_PATH=/usr/bin/terraform make testacc
```

The target was the throwaway container from `scripts/testacc-up.sh`:
`ghcr.io/open-webui/open-webui:v0.11.0` on `http://localhost:8080`, empty
database, admin API key. Terraform 1.15.8.

## Result: 43 tests, 36 pass, 7 skip, 0 fail

The suite is green on v0.11.0. That is a smaller result than it sounds,
because the suite does not exercise the payload shapes that v0.11.0
changed. The hand-verified failures below are the real work list.

### The 7 skips, all for a missing environment variable

| Test | Missing variable |
| --- | --- |
| `TestAccConfigImportResource` | `OPENWEBUI_TEST_CONFIG_IMPORT` |
| `TestAccOAuthClientResource` | `OPENWEBUI_OAUTH_CLIENT_URL` |
| `TestAccOpenAIConnectionsResource` | `OPENWEBUI_TEST_OPENAI_URL` |
| `TestAccPipelineResource` | `OPENWEBUI_PIPELINE_URL` |
| `TestAccPipelineDataSource` | `OPENWEBUI_PIPELINE_URL` |
| `TestAccPipelineValvesResource` | `OPENWEBUI_PIPELINE_URL` |
| `TestAccToolServerVerifyDataSource` | `OPENWEBUI_TOOL_SERVER_URL` |

Six of the seven need a second service to point at. `testacc-up.sh` does
not start one, and the acceptance run must not reach any host but the
container.

`TestAccConfigImportResource` needs nothing but the gate. It was run
separately with `OPENWEBUI_TEST_CONFIG_IMPORT=1` and **passed**, because it
feeds a full export straight back in. See the config import entry below for
the case it misses.

## Hand-verified failures

Each of these was reproduced with a scratch acceptance test against the
same container, then the scratch test was deleted. Write the permanent
test in the section that owns the fix.

| Config that fails | Error | Section |
| --- | --- | --- |
| `meta_additional_json` set on `openwebui_model` | `.meta_additional_json: was {"info":"custom"}, but now {"info":"custom","knowledge":null}` | 3.3 |
| `profile_image_url = "/img.png"` on `openwebui_model` | `.profile_image_url: was cty.StringVal("/img.png"), but now null` | 3.2 |
| `base_model_id` removed from an existing `openwebui_model` | `.base_model_id: was null, but now cty.StringVal("llama3.2")` | 3.2 |
| `config_json` holding less than the whole config on `openwebui_config_import` | `.config_json: inconsistent values for sensitive attribute` | 8.1 |

All four are `Provider produced inconsistent result after apply`. All four
break the apply. None of them is a silent diff.

## What the API says

Read off the live container, with the plan section each fact belongs to.

- **`meta.knowledge` is always present and null (3.3).** `ModelMeta`
  declares `knowledge`, so every model read carries `"knowledge": null`,
  whether or not anything set it. The provider's meta flatten has no
  `knowledge` case, so the key reaches `meta_additional_json`. This is the
  live model breakage, and it hits every model. Fix it with
  `delete(additional, "knowledge")` next to the `profile_image_url`,
  `description`, and `hidden` deletes.
- **`meta.chat_variables_schema` needs the chat-variable syntax (3.1).**
  `get_chat_variables_schema` (`backend/open_webui/utils/chat_variables.py`
  line 127) only returns a schema when the system prompt matches
  `{{ chat.variables.<key> }}`. A plain system prompt produces nothing. The
  key appears on `GET /models/model`, not on the create or update response.
  A model with `system = "Hello {{ chat.variables.name }}"` and no
  `meta_additional_json` in its config applies clean and re-plans clean,
  because `meta_additional_json` is Computed. Set `meta_additional_json` in
  the same config and the apply fails. The delete in section 3.1 is still
  the right fix.
- **`meta.profile_image_url` is validated and cleared (new, 3.2).**
  `validate_profile_image_url` (`backend/open_webui/utils/validate.py` line
  30) accepts only an empty string, a known static asset path, the user
  profile-image route, an `http(s)` URL with a host, and a
  `data:image/{png,jpeg,gif,webp}` URI. `ModelMeta.check_profile_image_url`
  catches the `ValueError` and stores `None`. Any other value, such as
  `/img.png`, is dropped without an error. The provider needs either a
  validator that rejects what the server would drop, or documentation
  naming the accepted forms.
- **`base_model_id` carries forward on update (3.2, confirmed).** An update
  that omits the field keeps the stored value. An update that sends
  `"base_model_id": null` clears it. Dropping `omitempty` is the right fix.
- **A wildcard user grant survives, an `anyone` grant does not (2.1, 2.2,
  confirmed).** A write of `user`/`*`/`read` plus `anyone`/`*`/`read` reads
  back as the `user`/`*` grant alone.
- **The config export is flat dotted keys (8.1, confirmed).**
  `GET /api/v1/configs/export` answers 394 top-level keys of the form
  `audio.stt.engine`, not a nested tree.
- **The config import answers with the whole config (8.1, confirmed).**
  `POST /api/v1/configs/import` with `{"config": {"ui.banners": []}}`
  answers all 394 keys. `applyConfigImport` writes that answer into the
  Required `config_json`, which is what breaks the apply.
- **`GET /api/v1/skills/id/{id}` does return `content` (open question 1,
  answered).** The read path can use it directly. `POST /skills/create` and
  `GET /skills/` both omit `content`, so re-read after create, as section
  4.2 says.
- **`GET /knowledge/{id}/files` carries no `data` and no `path` (6.2,
  confirmed).** `search_files_by_id`
  (`backend/open_webui/models/knowledge.py` line 560) defers `File.data` at
  line 636 and builds each item from a field list of `id`, `user_id`, `hash`,
  `filename`, `meta`, `created_at`, `updated_at`, and `user`. The response
  is a `KnowledgeFileListResponse` with `items`, `directories`,
  `breadcrumbs`, and `total`.
- **The seven missing group permission keys round-trip through the API
  (7.1, confirmed).** A group written with `workspace.skills_import`,
  `workspace.skills_export`, `sharing.folders`, `sharing.open_chats`,
  `access_grants.allow_groups`, `chat.import`, and `features.webhooks` reads
  every one of them back. The provider's `filterPermissionKeys` is the only
  thing blocking them.

## Corrections to the plan

- **3.1 overstates the trigger.** "Every model with a system prompt shows a
  permanent diff" is wrong. It takes the `{{ chat.variables.* }}` syntax,
  and it only breaks a config that sets `meta_additional_json`.
- **3.3 needs a fix, not a note.** `meta.knowledge` is not a
  set-it-yourself problem. The server returns it as null on every read, so
  the provider must delete it from the additional map.
- **3.2 predicts a silent no-op.** "The plan shows the removal, the apply
  reports success, and the value stays" is wrong. Terraform catches the
  mismatch and fails the apply.
- **8.1's inference is right and the failure is real.** The message names a
  sensitive attribute rather than the value, because `config_json` is
  Sensitive.
