# Open WebUI makes the request, so the endpoint has to be reachable from the
# Open WebUI server, not from the machine running Terraform. The read fails if
# the endpoint refuses the credentials.
data "openwebui_openai_connection_verify" "openai" {
  url = "https://api.openai.com/v1"
  key = var.openai_api_key
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}
