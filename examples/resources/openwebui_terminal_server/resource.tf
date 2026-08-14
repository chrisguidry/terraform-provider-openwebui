resource "openwebui_terminal_server" "example" {
  server_id = "workshop"
  name      = "Workshop terminals"
  url       = "http://terminals.internal:8080"
  auth_type = "bearer"
  key       = var.terminal_server_key
}

# An orchestrator hands out sessions under a policy it holds itself. Open WebUI
# stores only the id of that policy.
resource "openwebui_terminal_server" "orchestrated" {
  server_id   = "sandboxes"
  name        = "Sandbox orchestrator"
  url         = "http://sandboxes.internal:8080"
  server_type = "orchestrator"
  policy_id   = "default"
}

variable "terminal_server_key" {
  type      = string
  sensitive = true
}
