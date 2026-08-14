# The resource records the last configuration import, so it is a singleton.
# Import it under its fixed id. State comes back holding the whole exported
# configuration, not the subset the configuration names, so trim config_json to
# the keys you manage before the next apply.
terraform import openwebui_config_import.restore config_import
