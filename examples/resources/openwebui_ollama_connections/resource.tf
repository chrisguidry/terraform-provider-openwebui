# The resource owns the whole list. Every apply replaces it, so a connection
# added by hand in the web UI disappears on the next apply.
resource "openwebui_ollama_connections" "example" {
  enabled = true

  connections = [
    {
      url             = "http://ollama.internal:11434"
      connection_type = "local"
      tags            = ["local"]
    },
    {
      url       = "https://ollama.example.com"
      key       = var.remote_ollama_key
      prefix_id = "remote"
      model_ids = ["llama3.2", "qwen2.5-coder"]
    },
  ]
}

variable "remote_ollama_key" {
  type      = string
  sensitive = true
}
