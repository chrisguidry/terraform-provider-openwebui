resource "openwebui_subagents_config" "example" {
  enable_subagents             = true
  subagents_background_enabled = false
  subagents_max_concurrent     = 2
  subagents_max_async          = 1
  subagents_max_iterations     = 8
  subagents_max_output         = 4000
  subagents_system_prompt      = ""
}
