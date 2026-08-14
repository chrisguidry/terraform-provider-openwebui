package client

import (
	"context"
	"net/http"
)

// TaskConfigForm captures the task generation config. Every field of
// TaskConfigForm in backend/open_webui/routers/tasks.py is required, and the
// handler writes every field of the form it receives, so a request must carry
// all eighteen. The three nullable fields are pointers and are sent as null when
// unset, because omitting them is a 422.
//
// An empty template string means "use the built-in default": the generation
// handlers fall back to the DEFAULT_*_PROMPT_TEMPLATE constants when the stored
// value is empty.
type TaskConfigForm struct {
	TaskModel                            *string `json:"TASK_MODEL"`
	TaskModelExternal                    *string `json:"TASK_MODEL_EXTERNAL"`
	EnableTitleGeneration                bool    `json:"ENABLE_TITLE_GENERATION"`
	TitleGenerationPromptTemplate        string  `json:"TITLE_GENERATION_PROMPT_TEMPLATE"`
	ImagePromptGenerationPromptTemplate  string  `json:"IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE"`
	EnableAutocompleteGeneration         bool    `json:"ENABLE_AUTOCOMPLETE_GENERATION"`
	AutocompleteGenerationInputMaxLength int64   `json:"AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH"`
	AutocompleteGenerationPromptTemplate string  `json:"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE"`
	TagsGenerationPromptTemplate         string  `json:"TAGS_GENERATION_PROMPT_TEMPLATE"`
	FollowUpGenerationPromptTemplate     string  `json:"FOLLOW_UP_GENERATION_PROMPT_TEMPLATE"`
	EnableFollowUpGeneration             bool    `json:"ENABLE_FOLLOW_UP_GENERATION"`
	EnableTagsGeneration                 bool    `json:"ENABLE_TAGS_GENERATION"`
	EnableSearchQueryGeneration          bool    `json:"ENABLE_SEARCH_QUERY_GENERATION"`
	EnableRetrievalQueryGeneration       bool    `json:"ENABLE_RETRIEVAL_QUERY_GENERATION"`
	QueryGenerationPromptTemplate        string  `json:"QUERY_GENERATION_PROMPT_TEMPLATE"`
	ToolsFunctionCallingPromptTemplate   string  `json:"TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE"`
	EnableVoiceModePrompt                bool    `json:"ENABLE_VOICE_MODE_PROMPT"`
	VoiceModePromptTemplate              *string `json:"VOICE_MODE_PROMPT_TEMPLATE"`
}

// GetTaskConfig retrieves the task config.
func (c *Client) GetTaskConfig(ctx context.Context) (*TaskConfigForm, error) {
	var resp TaskConfigForm
	if err := c.do(ctx, http.MethodGet, "tasks/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetTaskConfig updates the task config and returns the stored values. The read
// path is tasks/config and the write path is tasks/config/update.
func (c *Client) SetTaskConfig(ctx context.Context, form TaskConfigForm) (*TaskConfigForm, error) {
	var resp TaskConfigForm
	if err := c.do(ctx, http.MethodPost, "tasks/config/update", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
