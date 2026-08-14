# Open WebUI makes the request, so the backend has to be reachable from the
# Open WebUI server, not from the machine running Terraform. The read fails if
# it is not.
data "openwebui_ollama_connection_verify" "local" {
  url = "http://ollama.internal:11434"
}
