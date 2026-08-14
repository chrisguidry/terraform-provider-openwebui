# Compaction summarises the older part of a conversation once it passes the
# token threshold. Open WebUI clamps the retention percentage to between 10 and
# 50, and treats an unset cap as the threshold.
resource "openwebui_chat_config" "example" {
  enable_context_compaction               = true
  context_compaction_model                = ""
  context_compaction_token_threshold      = 80000
  context_compaction_token_cap            = 120000
  context_compaction_retention_percentage = 40
  context_compaction_prompt_template      = ""
}
