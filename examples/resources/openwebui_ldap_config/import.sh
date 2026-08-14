# Open WebUI holds one LDAP configuration, so this resource is a singleton.
# Import it under its fixed id. The bind password comes back with it, unmasked,
# so treat the state file as a secret.
terraform import openwebui_ldap_config.example ldap
