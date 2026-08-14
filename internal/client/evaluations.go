package client

import (
	"context"
	"net/http"
)

// EvaluationConfig holds the arena evaluation settings.
//
// ArenaModels is untyped on the wire: the API validates nothing about the
// elements. The consumer in the chat path reads id, name, and meta off each
// one, so an element missing any of the three breaks model listing at request
// time rather than at write time.
type EvaluationConfig struct {
	EnableArenaModels bool  `json:"ENABLE_EVALUATION_ARENA_MODELS"`
	ArenaModels       []any `json:"EVALUATION_ARENA_MODELS"`
}

// EvaluationConfigForm is the write shape. The handler applies each field only
// when it is present, so an omitted field leaves the stored value alone.
type EvaluationConfigForm struct {
	EnableArenaModels *bool  `json:"ENABLE_EVALUATION_ARENA_MODELS,omitempty"`
	ArenaModels       *[]any `json:"EVALUATION_ARENA_MODELS,omitempty"`
}

// GetEvaluationConfig retrieves the arena evaluation settings.
func (c *Client) GetEvaluationConfig(ctx context.Context) (*EvaluationConfig, error) {
	var resp EvaluationConfig
	if err := c.do(ctx, http.MethodGet, "evaluations/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetEvaluationConfig writes the arena evaluation settings and returns the
// stored values.
func (c *Client) SetEvaluationConfig(ctx context.Context, form EvaluationConfigForm) (*EvaluationConfig, error) {
	var resp EvaluationConfig
	if err := c.do(ctx, http.MethodPost, "evaluations/config", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
