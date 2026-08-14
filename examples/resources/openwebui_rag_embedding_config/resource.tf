resource "openwebui_rag_embedding_config" "example" {
  rag_embedding_engine     = "openai"
  rag_embedding_model      = "text-embedding-3-small"
  rag_embedding_batch_size = 100

  openai_config = {
    url = "https://api.openai.com/v1"
    key = var.openai_api_key
  }
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}
