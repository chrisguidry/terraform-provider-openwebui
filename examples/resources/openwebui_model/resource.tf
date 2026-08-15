resource "openwebui_model" "example" {
  model_id      = "custom-rag"
  name          = "Custom RAG Model"
  base_model_id = "llama3.2"
  is_active     = true
  description   = "RAG-tuned model for the internal knowledge base"

  read_groups  = ["Support"]
  write_groups = ["Support"]

  # A grant can also name one account. Write the mail address the account signs
  # in with, or its user ID.
  read_users = ["contractor@example.com"]

  # Skills the model loads with every conversation, by skill_id.
  skill_ids = ["code-review"]

  params = {
    temperature = 0.1
    num_ctx     = 4096
  }

  capabilities = {
    vision     = false
    web_search = true
  }
}
