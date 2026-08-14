# An OpenAPI tool server that needs no authentication.
resource "openwebui_tool_server" "weather" {
  server_id = "weather"
  url       = "https://tools.example.com/weather"
  path      = "openapi.json"
  auth_type = "none"
  enabled   = true
}

# An OpenAPI tool server behind a bearer token, shared with one group.
resource "openwebui_tool_server" "billing" {
  server_id = "billing"
  url       = "https://tools.example.com/billing"
  path      = "openapi.json"
  auth_type = "bearer"
  key       = var.billing_tool_token

  read_groups = ["finance"]
}

# An MCP server that authenticates with OAuth 2.1. Registering the OAuth client
# is a separate call, so the two resources compose. The client_id of the OAuth
# client must equal the server_id of the tool server.
resource "openwebui_oauth_client" "paperless" {
  url       = "https://paperless.mcp.example.com"
  client_id = "paperless"
  type      = "mcp"
}

resource "openwebui_tool_server" "paperless" {
  server_id   = "paperless"
  type        = "mcp"
  url         = "https://paperless.mcp.example.com"
  auth_type   = "oauth_2.1"
  enabled     = true
  name        = "Paperless"
  description = "Document search"

  oauth_client_info = openwebui_oauth_client.paperless.oauth_client_info
}

# An MCP server whose identity provider issued the credentials up front. The
# OAuth client registers with the secret, and the tool server overlays the same
# client id and secret onto the registration.
resource "openwebui_oauth_client" "invoices" {
  url           = "https://invoices.mcp.example.com"
  client_id     = "invoices"
  client_secret = var.invoices_oauth_client_secret
  type          = "mcp"
}

resource "openwebui_tool_server" "invoices" {
  server_id = "invoices"
  type      = "mcp"
  url       = "https://invoices.mcp.example.com"
  auth_type = "oauth_2.1_static"
  enabled   = true
  name      = "Invoices"

  oauth_client_id     = "invoices"
  oauth_client_secret = var.invoices_oauth_client_secret
  oauth_client_info   = openwebui_oauth_client.invoices.oauth_client_info
}

variable "billing_tool_token" {
  type      = string
  sensitive = true
}

variable "invoices_oauth_client_secret" {
  type      = string
  sensitive = true
}
