# A file is imported by the UUID Open WebUI assigned it on upload. Open WebUI
# never returns the local path, so source_path stays empty in state and the
# first plan after the import replaces the file by uploading it again.
terraform import openwebui_file.example 3b2d5c7a-1e4f-4b8c-9d0a-6f5e4d3c2b1a
