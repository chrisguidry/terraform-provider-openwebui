# A user is imported by the UUID Open WebUI assigned the account, not by the
# email address. No route reads a password back, so set password and
# password_version in the configuration after the import.
terraform import openwebui_user.kid 2f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a
