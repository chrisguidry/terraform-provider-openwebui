# The tool server list is one configuration block, so this resource is a
# singleton. Import it under its fixed id, and every registration comes with it.
# To manage one registration on its own, use openwebui_tool_server instead.
terraform import openwebui_tool_servers_config.example tool_servers
