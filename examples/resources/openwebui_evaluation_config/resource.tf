# The arena models are untyped on the wire, so they travel as JSON text. Each
# element needs an id, a name, and a meta object.
resource "openwebui_evaluation_config" "example" {
  enable_evaluation_arena_models = true

  evaluation_arena_models_json = jsonencode([
    {
      id   = "arena-model"
      name = "Arena Model"
      meta = {
        profile_image_url = "/favicon.png"
        description       = "Two models answer, you pick the better one."
        model_ids         = null
      }
    }
  ])
}
