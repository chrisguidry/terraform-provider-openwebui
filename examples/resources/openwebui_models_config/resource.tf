# Open WebUI writes all five settings on every update, so name every one you
# care about. An attribute left out is written as null and its stored value is
# lost.
resource "openwebui_models_config" "example" {
  default_models        = "llama3.2"
  default_pinned_models = "llama3.2,gpt-4o"
  model_order_list      = ["llama3.2", "gpt-4o", "custom-rag"]
}
