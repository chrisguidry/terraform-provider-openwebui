# A standard channel is visible to everyone its access grants allow, and only an
# admin can create one. Open WebUI lowercases the name it stores, so write the
# lowercase form here.
resource "openwebui_channel" "announcements" {
  name        = "announcements"
  description = "House news, one post at a time"
  is_private  = false

  read_groups  = ["Family"]
  write_groups = ["Parents"]
}

# A channel shared with two accounts by name. An account that posts also reads,
# so read_users names both.
resource "openwebui_channel" "planning" {
  name        = "planning"
  description = "Trip planning"

  read_users  = ["grandma@example.com", "parent@example.com"]
  write_users = ["parent@example.com"]
}

# A channel every signed-in user can read and post in.
resource "openwebui_channel" "watercooler" {
  name         = "watercooler"
  description  = "Anything goes"
  public_read  = true
  public_write = true
}
