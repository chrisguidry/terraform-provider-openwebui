# Every attribute this resource does not name keeps the value Open WebUI already
# holds. The web block is the exception: Open WebUI writes all of its keys on
# every write, so the resource reads them back and sends them whole.
resource "openwebui_rag_config" "example" {
  chunk_size    = 1000
  chunk_overlap = 100
  top_k         = 4

  # The empty string removes the limit.
  file_max_size  = "50"
  file_max_count = ""

  web = {
    enable_web_search       = true
    web_search_engine       = "tavily"
    web_search_result_count = 5
    tavily_api_key          = var.tavily_api_key
  }
}

variable "tavily_api_key" {
  type      = string
  sensitive = true
}
