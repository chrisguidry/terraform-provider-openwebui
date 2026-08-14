# Settings this resource does not name keep the values the instance already
# holds, so a configuration can manage as few of them as it likes.
resource "openwebui_admin_config" "example" {
  webui_url         = "https://chat.example.com"
  enable_signup     = false
  default_user_role = "pending"
  jwt_expires_in    = "7d"

  enable_channels             = true
  channel_model_response_mode = "thread"
  enable_notes                = true
  enable_folders              = true

  # Open WebUI stores 0 as empty for these three, so an empty string is the way
  # to say "no limit".
  folder_max_file_count   = "50"
  automation_max_count    = ""
  automation_min_interval = "300"

  pending_user_overlay_title   = "Waiting for approval"
  pending_user_overlay_content = "An administrator reviews every new account."
}
