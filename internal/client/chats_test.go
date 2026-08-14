package client

import (
	"context"
	"net/http"
	"testing"
)

// The write takes all six settings at once, so each one goes on the wire.
func TestSetChatConfigSendsEveryField(t *testing.T) {
	var body map[string]any
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = decodeJSONBody(t, r)
		_, _ = w.Write([]byte(`{"CONTEXT_COMPACTION_MODEL":"","ENABLE_CONTEXT_COMPACTION":true,"CONTEXT_COMPACTION_TOKEN_THRESHOLD":80000,"CONTEXT_COMPACTION_TOKEN_CAP":80000,"CONTEXT_COMPACTION_RETENTION_PERCENTAGE":40,"CONTEXT_COMPACTION_PROMPT_TEMPLATE":"Summarise."}`))
	})

	updated, err := c.SetChatConfig(context.Background(), ChatConfigForm{
		EnableContextCompaction:              true,
		ContextCompactionTokenThreshold:      80_000,
		ContextCompactionRetentionPercentage: 40,
		ContextCompactionPromptTemplate:      "Summarise.",
	})
	if err != nil {
		t.Fatalf("SetChatConfig: %v", err)
	}

	fields := []string{
		"CONTEXT_COMPACTION_MODEL",
		"ENABLE_CONTEXT_COMPACTION",
		"CONTEXT_COMPACTION_TOKEN_THRESHOLD",
		"CONTEXT_COMPACTION_TOKEN_CAP",
		"CONTEXT_COMPACTION_RETENTION_PERCENTAGE",
		"CONTEXT_COMPACTION_PROMPT_TEMPLATE",
	}
	for _, field := range fields {
		if _, ok := body[field]; !ok {
			t.Fatalf("expected %s in the request body, got %v", field, body)
		}
	}

	// An unset cap reads back as the threshold, never as null.
	if updated.ContextCompactionTokenCap == nil || *updated.ContextCompactionTokenCap != 80_000 {
		t.Fatalf("expected the cap to read back as the threshold, got %v", updated.ContextCompactionTokenCap)
	}
}

// The server clamps the threshold, the cap, and the retention percentage, and
// answers with what it stored.
func TestSetChatConfigReadsBackClampedValues(t *testing.T) {
	c := newChannelsTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"CONTEXT_COMPACTION_MODEL":"","ENABLE_CONTEXT_COMPACTION":true,"CONTEXT_COMPACTION_TOKEN_THRESHOLD":1,"CONTEXT_COMPACTION_TOKEN_CAP":1,"CONTEXT_COMPACTION_RETENTION_PERCENTAGE":50,"CONTEXT_COMPACTION_PROMPT_TEMPLATE":"Summarise."}`))
	})

	updated, err := c.SetChatConfig(context.Background(), ChatConfigForm{
		EnableContextCompaction:              true,
		ContextCompactionTokenThreshold:      0,
		ContextCompactionRetentionPercentage: 90,
		ContextCompactionPromptTemplate:      "Summarise.",
	})
	if err != nil {
		t.Fatalf("SetChatConfig: %v", err)
	}

	if updated.ContextCompactionTokenThreshold != 1 || updated.ContextCompactionRetentionPercentage != 50 {
		t.Fatalf("expected the clamped values, got %+v", updated)
	}
}

func TestGetChatConfigUsesTheChatsRouter(t *testing.T) {
	var path string
	c := newChannelsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"CONTEXT_COMPACTION_MODEL":"","ENABLE_CONTEXT_COMPACTION":false,"CONTEXT_COMPACTION_TOKEN_THRESHOLD":80000,"CONTEXT_COMPACTION_TOKEN_CAP":80000,"CONTEXT_COMPACTION_RETENTION_PERCENTAGE":40,"CONTEXT_COMPACTION_PROMPT_TEMPLATE":""}`))
	})

	if _, err := c.GetChatConfig(context.Background()); err != nil {
		t.Fatalf("GetChatConfig: %v", err)
	}

	if path != "/api/v1/chats/config" {
		t.Fatalf("expected the chats router, got %q", path)
	}
}
