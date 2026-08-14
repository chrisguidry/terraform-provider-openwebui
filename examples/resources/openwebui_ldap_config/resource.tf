# Open WebUI stores the bind password in plain text and returns it on read, so
# it lands in Terraform state. Keep the state file as secret as the password.
resource "openwebui_ldap_config" "example" {
  enable_ldap = true

  label                  = "Household directory"
  host                   = "ldap.example.com"
  port                   = 636
  use_tls                = true
  validate_cert          = true
  attribute_for_mail     = "mail"
  attribute_for_username = "uid"

  app_dn          = "cn=openwebui,ou=services,dc=example,dc=com"
  app_dn_password = var.ldap_bind_password
  search_base     = "ou=people,dc=example,dc=com"
  search_filters  = "(objectClass=inetOrgPerson)"

  enable_group_management = true
  enable_group_creation   = false
  attribute_for_groups    = "memberOf"
}
