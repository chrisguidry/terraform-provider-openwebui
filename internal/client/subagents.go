package client

import (
	"context"
	"net/http"
)

// SubagentsConfigForm captures subagent configuration. Every field of
// SubagentsConfigForm in backend/open_webui/routers/configs.py is required, and
// the handler writes every field it receives, so a request must carry all seven.
type SubagentsConfigForm struct {
	EnableSubagents            bool   `json:"ENABLE_SUBAGENTS"`
	SubagentsBackgroundEnabled bool   `json:"SUBAGENTS_BACKGROUND_ENABLED"`
	SubagentsMaxConcurrent     int64  `json:"SUBAGENTS_MAX_CONCURRENT"`
	SubagentsMaxAsync          int64  `json:"SUBAGENTS_MAX_ASYNC"`
	SubagentsMaxIterations     int64  `json:"SUBAGENTS_MAX_ITERATIONS"`
	SubagentsMaxOutput         int64  `json:"SUBAGENTS_MAX_OUTPUT"`
	SubagentsSystemPrompt      string `json:"SUBAGENTS_SYSTEM_PROMPT"`
}

// GetSubagentsConfig retrieves the subagents config.
func (c *Client) GetSubagentsConfig(ctx context.Context) (*SubagentsConfigForm, error) {
	var resp SubagentsConfigForm
	if err := c.do(ctx, http.MethodGet, "configs/subagents", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetSubagentsConfig updates the subagents config and returns the stored values.
func (c *Client) SetSubagentsConfig(ctx context.Context, form SubagentsConfigForm) (*SubagentsConfigForm, error) {
	var resp SubagentsConfigForm
	if err := c.do(ctx, http.MethodPost, "configs/subagents", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
