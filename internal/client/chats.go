package client

import (
	"context"
	"net/http"
)

// ChatConfigForm carries the context compaction settings. The routes live under
// the chats router, not the configs router, and both of them answer with the
// full set of six values.
//
// The write clamps three of the six: the token threshold and the token cap to
// at least 1, and the retention percentage to between 10 and 50. A value
// outside those bounds comes back corrected instead of refused.
type ChatConfigForm struct {
	ContextCompactionModel               *string `json:"CONTEXT_COMPACTION_MODEL"`
	EnableContextCompaction              bool    `json:"ENABLE_CONTEXT_COMPACTION"`
	ContextCompactionTokenThreshold      int64   `json:"CONTEXT_COMPACTION_TOKEN_THRESHOLD"`
	ContextCompactionTokenCap            *int64  `json:"CONTEXT_COMPACTION_TOKEN_CAP"`
	ContextCompactionRetentionPercentage int64   `json:"CONTEXT_COMPACTION_RETENTION_PERCENTAGE"`
	ContextCompactionPromptTemplate      string  `json:"CONTEXT_COMPACTION_PROMPT_TEMPLATE"`
}

// GetChatConfig retrieves the context compaction settings.
func (c *Client) GetChatConfig(ctx context.Context) (*ChatConfigForm, error) {
	var resp ChatConfigForm
	if err := c.do(ctx, http.MethodGet, "chats/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetChatConfig writes the context compaction settings and returns the stored
// values, which hold the server's clamping.
func (c *Client) SetChatConfig(ctx context.Context, form ChatConfigForm) (*ChatConfigForm, error) {
	var resp ChatConfigForm
	if err := c.do(ctx, http.MethodPost, "chats/config", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
