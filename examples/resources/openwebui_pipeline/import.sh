# A pipeline is imported by its pipeline_id and the index of the pipeline server
# URL it is registered on, separated by a colon.
terraform import openwebui_pipeline.example rate_limit_filter:0

# The index defaults to 0, so a single-server install can leave it off.
terraform import openwebui_pipeline.example rate_limit_filter
