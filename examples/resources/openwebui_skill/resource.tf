# A skill is a Markdown document. Keep a long one in a file next to the
# configuration and load it, so the Markdown stays readable.
resource "openwebui_skill" "code_review" {
  skill_id    = "code-review"
  name        = "Code review"
  description = "House rules the assistant follows when it reviews a change"
  content     = file("${path.module}/skills/code-review.md")

  tags = ["engineering"]

  read_groups  = ["Support"]
  write_groups = ["Support"]
}

# A short skill reads well inline.
resource "openwebui_skill" "release_notes" {
  skill_id    = "release-notes"
  name        = "Release notes"
  description = "Format for the notes that ship with a release"

  content = <<-EOT
    # Release notes

    Write one line for each user-visible change. Name the setting or the
    command that changed, and say what it does now. Leave out refactors and
    dependency bumps.
  EOT

  public_read = true
}
