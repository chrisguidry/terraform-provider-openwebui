# The password is write-only: Terraform sends it to the provider and never
# records it. No route reads a password back, so a password changed elsewhere is
# invisible here. To rotate one, change the password and the marker together.
resource "openwebui_user" "kid" {
  name  = "Alex Doe"
  email = "alex@example.com"
  role  = "user"

  password         = var.sam_password
  password_version = "2026-08-14"
}

# Group membership belongs to the group, which owns the list of users.
resource "openwebui_group" "family" {
  name        = "Family"
  description = "Everyone in the house"
  users       = [openwebui_user.kid.email]
}

variable "sam_password" {
  type      = string
  sensitive = true
}
