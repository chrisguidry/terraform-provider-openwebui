package client

import (
	"context"
	"net/http"
	"testing"
)

// The handler applies a field only when the request carries it, so an omitted
// field must stay out of the body.
func TestSetEvaluationConfigOmitsAbsentFields(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"ENABLE_EVALUATION_ARENA_MODELS":true,"EVALUATION_ARENA_MODELS":[]}`))
	})

	enabled := true
	if _, err := c.SetEvaluationConfig(context.Background(), EvaluationConfigForm{EnableArenaModels: &enabled}); err != nil {
		t.Fatalf("SetEvaluationConfig: %v", err)
	}

	if _, ok := body["EVALUATION_ARENA_MODELS"]; ok {
		t.Fatalf("expected no arena models key, got %v", body)
	}
	if body["ENABLE_EVALUATION_ARENA_MODELS"] != true {
		t.Fatalf("expected the enable flag on the wire, got %v", body)
	}
}

// An empty list is a value, not an omission: it clears the arena models.
func TestSetEvaluationConfigSendsAnEmptyList(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"ENABLE_EVALUATION_ARENA_MODELS":false,"EVALUATION_ARENA_MODELS":[]}`))
	})

	models := []any{}
	if _, err := c.SetEvaluationConfig(context.Background(), EvaluationConfigForm{ArenaModels: &models}); err != nil {
		t.Fatalf("SetEvaluationConfig: %v", err)
	}

	list, ok := body["EVALUATION_ARENA_MODELS"].([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("expected an empty list on the wire, got %v", body["EVALUATION_ARENA_MODELS"])
	}
}

// The elements are untyped, so a key the provider does not model has to survive.
func TestGetEvaluationConfigKeepsUnknownElementKeys(t *testing.T) {
	c := newChannelsTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ENABLE_EVALUATION_ARENA_MODELS":true,"EVALUATION_ARENA_MODELS":[{"id":"arena","name":"Arena","meta":{"profile_image_url":"/x.png"},"future_key":7}]}`))
	})

	config, err := c.GetEvaluationConfig(context.Background())
	if err != nil {
		t.Fatalf("GetEvaluationConfig: %v", err)
	}

	if len(config.ArenaModels) != 1 {
		t.Fatalf("expected one arena model, got %v", config.ArenaModels)
	}
	model, ok := config.ArenaModels[0].(map[string]any)
	if !ok || model["future_key"] != float64(7) {
		t.Fatalf("expected the unmodelled key to survive, got %v", config.ArenaModels[0])
	}
}
