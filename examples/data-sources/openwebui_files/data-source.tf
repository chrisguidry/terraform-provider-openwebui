data "openwebui_files" "all" {}

# The extracted text of every file arrives in data_json by default, which makes
# for a large state file when the documents are large. Set content = false to
# read the file records without it.
data "openwebui_files" "manuals" {
  filename = "manual"
  content  = false
  limit    = 20
}
