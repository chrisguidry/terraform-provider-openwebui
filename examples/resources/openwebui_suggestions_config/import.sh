# The prompt suggestions are one configuration block, so this resource is a
# singleton. Import it under its fixed id. The suggestions API serves no read
# route, so the import adopts the id alone and the next apply writes the list
# from the configuration.
terraform import openwebui_suggestions_config.example suggestions
