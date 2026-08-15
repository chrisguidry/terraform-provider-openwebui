# Terraform Provider for Open WebUI

A Terraform provider that manages [Open WebUI](https://openwebui.com) resources
through its REST API. Declaratively manage knowledge bases, models, skills,
prompts, groups, users, channels, tools, tool servers, pipelines, functions,
files, model connections, and every admin-level configuration surface Open WebUI
exposes.

- **Provider address:** `registry.terraform.io/chrisguidry/openwebui`
- **Source:** `github.com/chrisguidry/terraform-provider-openwebui`

This is a fork of
[docktape/terraform-provider-openwebui](https://github.com/docktape/terraform-provider-openwebui),
which targets Open WebUI v0.9.x. The fork tracks Open WebUI v0.11,
adds the resources that version introduced (skills, tool servers,
terminal servers, channels, users, and the engine and identity
configuration surfaces), and covers every backend router or documents
why it is excluded — see API Coverage below. The original is MPL-2.0
and so is this fork.

## Compatibility

| Component      | Requirement                                             |
| -------------- | ------------------------------------------------------- |
| Terraform      | 1.6 or newer                                            |
| Open WebUI     | v0.11.x                                                 |
| Go             | 1.25+ (only required to build the provider)             |

This build targets one Open WebUI minor version, not a range. It reads
`meta.chat_variables_schema` on models and the flat configuration export, and
neither exists at v0.9.x.

Attributes backed by API endpoints that appeared in a later patch release are
marked optional and noted in their schema descriptions.

### The engine configuration resources need review at each release

`openwebui_rag_config`, `openwebui_rag_embedding_config`,
`openwebui_images_config`, and `openwebui_audio_config` track the `retrieval.py`,
`images.py`, and `audio.py` routers. Those three routers changed by 1,742 added
and 1,241 removed lines between v0.9.6 and v0.11.0, and `images.py` moved its
whole configuration surface from application state to the per-key config table.
Read the diff of these three routers at every Open WebUI release, and expect
these four resources to need work.

## Provider Configuration

```hcl
terraform {
  required_providers {
    openwebui = {
      source  = "chrisguidry/openwebui"
      version = "~> 1.4"
    }
  }
}

provider "openwebui" {
  endpoint = "https://openwebui.example.com"
  token    = var.openwebui_token
}

variable "openwebui_token" {
  type        = string
  sensitive   = true
  description = "Admin API token for the Open WebUI instance."
}
```

| Argument               | Env var              | Required | Description                                                           |
| ---------------------- | -------------------- | -------- | --------------------------------------------------------------------- |
| `endpoint`             | `OPENWEBUI_ENDPOINT` | Yes      | Base URL of the instance, e.g. `https://openwebui.example.com`.       |
| `token`                | `OPENWEBUI_TOKEN`    | Yes      | Bearer token used to authenticate API requests (sensitive).           |
| `insecure_skip_verify` | `OPENWEBUI_INSECURE` | No       | Disable TLS certificate verification. Not recommended for production. |

## Resources

| Resource                             | Description                                          |
| ------------------------------------ | ---------------------------------------------------- |
| `openwebui_knowledge`                | Knowledge base entries                               |
| `openwebui_knowledge_file`           | File attachments on a knowledge base                 |
| `openwebui_model`                    | Custom model definitions                              |
| `openwebui_skill`                    | Reusable skills a model can call                     |
| `openwebui_prompt`                   | Reusable prompt commands                             |
| `openwebui_group`                    | User groups with permissions                         |
| `openwebui_user`                     | User accounts                                        |
| `openwebui_channel`                  | Chat channels                                        |
| `openwebui_tool`                     | Python tools                                         |
| `openwebui_tool_valves`              | Settings (valves) for a tool                         |
| `openwebui_tool_server`              | One external tool server, OpenAPI or MCP             |
| `openwebui_tool_servers_config`      | The whole external tool server list                  |
| `openwebui_terminal_server`          | One terminal server the built-in terminal proxies to |
| `openwebui_pipeline`                 | Pipeline registrations                               |
| `openwebui_pipeline_valves`          | Settings (valves) for a pipeline                     |
| `openwebui_function`                 | Python functions                                     |
| `openwebui_function_valves`          | Settings (valves) for a function                     |
| `openwebui_file`                     | Uploaded files                                       |
| `openwebui_openai_connections`       | OpenAI-compatible model connections                  |
| `openwebui_ollama_connections`       | Ollama model connections                             |
| `openwebui_connections_config`       | Direct connection settings                           |
| `openwebui_code_execution_config`    | Code execution and interpreter settings              |
| `openwebui_models_config`            | Default model ordering and defaults                  |
| `openwebui_suggestions_config`       | Default prompt suggestions                           |
| `openwebui_banners_config`           | UI announcement banners                              |
| `openwebui_admin_config`             | Instance-wide administrative settings                |
| `openwebui_ldap_config`              | LDAP authentication settings                         |
| `openwebui_default_user_permissions` | Permissions every user starts with                   |
| `openwebui_chat_config`              | Chat context compaction and retention                |
| `openwebui_task_config`              | Task model and its prompt templates                  |
| `openwebui_evaluation_config`        | Arena evaluation models                              |
| `openwebui_subagents_config`         | Subagent settings                                    |
| `openwebui_rag_config`               | Retrieval settings                                   |
| `openwebui_rag_embedding_config`     | Embedding engine settings                            |
| `openwebui_images_config`            | Image generation and image editing engines           |
| `openwebui_audio_config`             | Speech to text and text to speech engines            |
| `openwebui_oauth_client`             | OAuth client registrations                           |
| `openwebui_config_import`            | Bulk configuration restore                           |

## Data Sources

| Data Source                          | Description                                        |
| ------------------------------------ | -------------------------------------------------- |
| `openwebui_model`                    | Look up a model by name or ID                      |
| `openwebui_knowledge`                | Look up a knowledge base by name or ID             |
| `openwebui_skill`                    | Look up a skill by name or ID                      |
| `openwebui_prompt`                   | Look up a prompt by command or ID                  |
| `openwebui_group`                    | Look up a group by name or ID                      |
| `openwebui_user`                     | Look up a user by email or ID                      |
| `openwebui_channel`                  | Look up a channel by name or ID                    |
| `openwebui_tool`                     | Look up a tool by name or ID                       |
| `openwebui_tool_server`              | Look up a tool server by its server ID             |
| `openwebui_pipeline`                 | Look up a pipeline by name or ID                   |
| `openwebui_file`                     | Look up a single uploaded file                     |
| `openwebui_files`                    | List uploaded files                                |
| `openwebui_config_export`            | Export the current Open WebUI configuration        |
| `openwebui_tool_server_verify`       | Verify connectivity to an external tool server     |
| `openwebui_openai_connection_verify` | Verify an OpenAI-compatible connection             |
| `openwebui_ollama_connection_verify` | Verify an Ollama connection                        |

Full per-resource and per-data-source documentation is in [`docs/`](docs), with
runnable configurations under [`examples/`](examples).

## API Coverage

Open WebUI v0.11.0 has 31 routers. The table names each one, what the provider
does with it, and why, so "covers the API" is a claim you can check.

Four reasons carry the exclusions:

- **(a)** No read route an admin token can reach, so state can never converge.
- **(b)** Runtime data rather than configuration.
- **(c)** The read masks or encrypts what the write sent.
- **(d)** Overlaps a surface the provider already manages.

| Router | Coverage |
| --- | --- |
| `analytics.py` | Excluded (b). Every route computes counts on the fly and stores nothing. |
| `audio.py` | `openwebui_audio_config`. |
| `auths.py` | `openwebui_admin_config` and `openwebui_ldap_config`. The OAuth settings are excluded, see below. |
| `automations.py` | Excluded (a), (b). The access check has no admin branch, so an admin token reading another user's automation gets a 404. |
| `calendar.py` | Excluded (b). Per-user calendars, created on a user permission with no admin branch. |
| `channels.py` | `openwebui_channel` and its data source. Messages and read receipts are excluded (b). |
| `chats.py` | `openwebui_chat_config`. Transcripts and their per-user lifecycle are excluded (b). |
| `configs.py` | `openwebui_connections_config`, `openwebui_tool_server`, `openwebui_tool_servers_config`, `openwebui_terminal_server`, `openwebui_code_execution_config`, `openwebui_models_config`, `openwebui_suggestions_config`, `openwebui_banners_config`, `openwebui_subagents_config`, `openwebui_oauth_client`, `openwebui_config_import`, `openwebui_config_export`. |
| `evaluations.py` | `openwebui_evaluation_config`. Feedback and leaderboards are excluded (b). |
| `files.py` | `openwebui_file`, `openwebui_file` and `openwebui_files` data sources. |
| `folders.py` | Excluded (b). Folders are per-owner, and deleting one deletes the owner's chats inside it. |
| `functions.py` | `openwebui_function` and `openwebui_function_valves`. |
| `groups.py` | `openwebui_group` and its data source. |
| `images.py` | `openwebui_images_config`. Generating and editing images is excluded (b). |
| `knowledge.py` | `openwebui_knowledge` and `openwebui_knowledge_file`. External knowledge connections are not built yet; they matter only to an instance that keeps its vectors in Qdrant, Milvus, or pgvector. |
| `memories.py` | Excluded (a), (b). Every read is scoped to the calling user, and the content is learned from conversations. |
| `models.py` | `openwebui_model` and its data source. |
| `notes.py` | Excluded (b). A per-user rich-text document, created on a user permission with no admin branch. |
| `notifications.py` | Excluded (b), (c). Targets live in user settings, and the read replaces the webhook URL with a masked one. |
| `ollama.py` | `openwebui_ollama_connections` and `openwebui_ollama_connection_verify`. |
| `openai.py` | `openwebui_openai_connections` and `openwebui_openai_connection_verify`. |
| `pipelines.py` | `openwebui_pipeline`, `openwebui_pipeline_valves`, and the pipeline data source. |
| `prompts.py` | `openwebui_prompt` and its data source. |
| `retrieval.py` | `openwebui_rag_config` and `openwebui_rag_embedding_config`. Ingesting, querying, and resetting are excluded (b). |
| `scim.py` | Excluded (a), (d). A SCIM facade over the same user and group tables, off by default, and its own settings are environment reads no API can write. |
| `skills.py` | `openwebui_skill` and its data source. |
| `tasks.py` | `openwebui_task_config`. |
| `terminals.py` | Excluded (d). A reverse proxy. The declarable object is the connection list, which `openwebui_terminal_server` manages through `configs.py`. |
| `tools.py` | `openwebui_tool`, `openwebui_tool_valves`, `openwebui_tool_server`, and the tool server data sources. |
| `users.py` | `openwebui_user`, its data source, and `openwebui_default_user_permissions`. |
| `utils.py` | Excluded (b), (d). Five one-shot actions. The only settings it reads belong to `openwebui_code_execution_config`. |

### Decided exclusions

- **OAuth settings (`auths.py`)**, excluded by decision. Open WebUI does not
  persist a key beginning with `oauth.` unless the instance sets
  `ENABLE_OAUTH_PERSISTENT_CONFIG`, which is off by default. With it off the
  apply succeeds, the read-back matches, and the value is gone at the next
  restart. Deployments normally set OAuth through the environment. Reach these
  keys with `openwebui_config_import` if you need them.
- **External knowledge connections (`knowledge.py`)**, not built. Full CRUD over
  one config key, useful only with an external vector store.
- **Channel webhooks (`channels.py`)**, not built. State could converge, since
  the read returns the token in cleartext, but there is no read by webhook id.
  Import would need a `channel_id/webhook_id` key resolved against the
  per-channel list. It is a follow-on to `openwebui_channel`, not part of
  covering the channel itself.

## Usage Examples

### Knowledge base

```hcl
resource "openwebui_knowledge" "example" {
  name        = "Support FAQ"
  description = "Knowledge base backing the support chatbot"

  read_groups  = ["Support"]
  write_groups = ["Support"]
}
```

### Model

```hcl
resource "openwebui_model" "example" {
  model_id      = "custom-rag"
  name          = "Custom RAG Model"
  base_model_id = "llama3.2"
  is_active     = true
  hidden        = false
  description   = "RAG-tuned model for the internal knowledge base"

  read_groups  = ["Support"]
  write_groups = ["Support"]

  # A grant can also name one account, by mail address or user ID.
  read_users = ["contractor@example.com"]

  params {
    temperature = 0.1
    num_ctx     = 4096
  }

  capabilities {
    vision     = false
    web_search = true
  }
}
```

### Group

```hcl
resource "openwebui_group" "example" {
  name        = "Support"
  description = "Support team access group"

  users = [
    "alice@example.com",
    "bob@example.com",
  ]

  permissions = {
    workspace = {
      models    = true
      knowledge = true
      prompts   = true
      tools     = false
    }
    chat = {
      file_upload = true
      delete      = true
      edit        = true
    }
    features = {
      web_search = true
    }
    access_grants = {
      allow_users = true
    }
    settings = {
      interface = true
    }
  }
}
```

### Prompt

```hcl
resource "openwebui_prompt" "example" {
  command   = "summarize"
  name      = "Summarise text"
  content   = "Summarise the following in three bullet points:\n\n{{text}}"
  is_active = true
  tags      = ["productivity", "writing"]
}
```

## License

This provider is distributed under the terms of the [Mozilla Public License 2.0](LICENSE) (MPL-2.0).

