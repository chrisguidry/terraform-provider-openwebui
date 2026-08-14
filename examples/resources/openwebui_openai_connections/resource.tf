# The resource owns the whole list. Every apply replaces it, so a connection
# added by hand in the web UI disappears on the next apply. List order sets
# request priority.
resource "openwebui_openai_connections" "example" {
  enabled = true

  connections = [
    {
      url       = "https://api.openai.com/v1"
      key       = var.openai_api_key
      prefix_id = "openai"
      model_ids = ["gpt-4o-mini"]
    },
    {
      url          = "https://example-resource.openai.azure.com/openai"
      key          = var.azure_openai_api_key
      api_provider = "azure"
      api_version  = "2024-02-01"
      prefix_id    = "azure"
    },
  ]
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}

variable "azure_openai_api_key" {
  type      = string
  sensitive = true
}
