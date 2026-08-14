# A successful apply does not prove the engine answers. The update route asks
# Automatic1111 to load the model and swallows the error when the host is down.
resource "openwebui_images_config" "example" {
  enable_image_generation = true
  image_generation_engine = "openai"
  image_generation_model  = "dall-e-3"
  image_size              = "1024x1024"
  image_steps             = 50
  images_openai_api_key   = var.openai_api_key

  enable_image_edit = false
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}
