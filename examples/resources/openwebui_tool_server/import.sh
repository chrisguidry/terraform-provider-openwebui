# A tool server is imported by its server_id, which Open WebUI stores as info.id.
terraform import openwebui_tool_server.paperless paperless

# A connection registered by hand in the web UI may carry no info.id. Import it
# by its position in the list, and the next apply pins the identity.
terraform import openwebui_tool_server.weather weather@2
