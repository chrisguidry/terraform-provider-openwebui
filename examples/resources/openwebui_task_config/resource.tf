# An empty template asks Open WebUI for the prompt built into it. A model id
# left null makes each task run on the model of the chat it belongs to.
resource "openwebui_task_config" "example" {
  task_model          = null
  task_model_external = null

  enable_title_generation          = true
  title_generation_prompt_template = ""

  enable_tags_generation          = true
  tags_generation_prompt_template = ""

  enable_autocomplete_generation           = true
  autocomplete_generation_input_max_length = -1
  autocomplete_generation_prompt_template  = ""

  enable_search_query_generation    = true
  enable_retrieval_query_generation = true
  query_generation_prompt_template  = ""

  enable_follow_up_generation          = true
  follow_up_generation_prompt_template = ""

  image_prompt_generation_prompt_template = ""
  tools_function_calling_prompt_template  = ""

  enable_voice_mode_prompt   = false
  voice_mode_prompt_template = null
}
