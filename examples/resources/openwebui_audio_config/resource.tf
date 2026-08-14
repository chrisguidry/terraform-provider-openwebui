# Open WebUI requires both blocks together. Setting stt.engine to the empty
# string loads faster-whisper into the Open WebUI process and downloads the
# model named by stt.whisper_model when it is not already cached.
resource "openwebui_audio_config" "example" {
  tts = {
    engine         = "openai"
    model          = "tts-1"
    voice          = "alloy"
    openai_api_key = var.openai_api_key
  }

  stt = {
    engine         = "openai"
    model          = "whisper-1"
    openai_api_key = var.openai_api_key
  }
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}
