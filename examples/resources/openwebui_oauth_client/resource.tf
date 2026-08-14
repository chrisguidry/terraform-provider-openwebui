# Dynamic client registration: Open WebUI registers with the provider and keeps
# the credentials it is issued.
resource "openwebui_oauth_client" "example" {
  url         = "https://auth.example.com"
  client_id   = "my-terraform-client"
  client_name = "Terraform Managed Client"
  type        = "mcp"
}

# Static credentials: a client_secret makes Open WebUI build the registration
# from credentials the identity provider already issued, instead of registering
# a new one. This is the registration an oauth_2.1_static tool server reads.
resource "openwebui_oauth_client" "static" {
  url              = "https://paperless.mcp.example.com"
  client_id        = "paperless"
  client_secret    = var.paperless_oauth_client_secret
  oauth_server_url = "https://auth.example.com"
  oauth_scope      = "openid profile"
  type             = "mcp"
}
