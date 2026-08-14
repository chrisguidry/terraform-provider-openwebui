resource "openwebui_pipeline" "example" {
  url = "http://pipelines.internal:9099"
}

# A pipeline is identified by its id and the index of the server URL it is
# registered on, so the valves name both.
resource "openwebui_pipeline_valves" "example" {
  pipeline_id = openwebui_pipeline.example.pipeline_id
  url_idx     = openwebui_pipeline.example.url_idx

  valves_json = jsonencode({
    temperature = 0.7
  })
}
