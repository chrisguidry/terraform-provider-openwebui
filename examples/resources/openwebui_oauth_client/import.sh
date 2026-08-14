# An OAuth client is imported by its client_id. The resource never reads the
# registration back from Open WebUI, so the import adopts the identifier alone
# and the next apply writes the registration from the configuration.
terraform import openwebui_oauth_client.example my-terraform-client
