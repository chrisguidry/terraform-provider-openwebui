package client

import (
	"context"
	"net/http"
	"testing"
)

// The read path and the write path differ: tasks/config reads and
// tasks/config/update writes.
func TestTaskConfigUsesTheAsymmetricPaths(t *testing.T) {
	var readPath, writePath string
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			readPath = r.URL.Path
		} else {
			writePath = r.URL.Path
		}
		_, _ = w.Write([]byte(`{"TASK_MODEL":null,"TASK_MODEL_EXTERNAL":null,"ENABLE_TITLE_GENERATION":true,"TITLE_GENERATION_PROMPT_TEMPLATE":"","IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE":"","ENABLE_AUTOCOMPLETE_GENERATION":true,"AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH":-1,"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE":"","TAGS_GENERATION_PROMPT_TEMPLATE":"","FOLLOW_UP_GENERATION_PROMPT_TEMPLATE":"","ENABLE_FOLLOW_UP_GENERATION":true,"ENABLE_TAGS_GENERATION":true,"ENABLE_SEARCH_QUERY_GENERATION":true,"ENABLE_RETRIEVAL_QUERY_GENERATION":true,"QUERY_GENERATION_PROMPT_TEMPLATE":"","TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE":"","ENABLE_VOICE_MODE_PROMPT":false,"VOICE_MODE_PROMPT_TEMPLATE":null}`))
	})

	if _, err := c.GetTaskConfig(context.Background()); err != nil {
		t.Fatalf("GetTaskConfig: %v", err)
	}
	if _, err := c.SetTaskConfig(context.Background(), TaskConfigForm{}); err != nil {
		t.Fatalf("SetTaskConfig: %v", err)
	}

	if readPath != "/api/v1/tasks/config" {
		t.Fatalf("expected the read path tasks/config, got %s", readPath)
	}
	if writePath != "/api/v1/tasks/config/update" {
		t.Fatalf("expected the write path tasks/config/update, got %s", writePath)
	}
}

// Every field is required, and the three nullable ones must be sent as null
// rather than left out. An empty template is a real value that asks Open WebUI
// for its built-in prompt, so it has to survive the round trip.
func TestSetTaskConfigSendsNullsAndEmptyTemplates(t *testing.T) {
	var body map[string]any
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSONBody(t, r)
		_, _ = w.Write([]byte(`{"TASK_MODEL":null,"TITLE_GENERATION_PROMPT_TEMPLATE":""}`))
	})

	updated, err := c.SetTaskConfig(context.Background(), TaskConfigForm{})
	if err != nil {
		t.Fatalf("SetTaskConfig: %v", err)
	}

	for _, field := range []string{"TASK_MODEL", "TASK_MODEL_EXTERNAL", "VOICE_MODE_PROMPT_TEMPLATE"} {
		value, ok := body[field]
		if !ok {
			t.Fatalf("expected %s in the request body, got %v", field, body)
		}
		if value != nil {
			t.Fatalf("expected %s to be sent as null, got %v", field, value)
		}
	}

	if body["TITLE_GENERATION_PROMPT_TEMPLATE"] != "" {
		t.Fatalf("expected an empty title template to be sent, got %v", body["TITLE_GENERATION_PROMPT_TEMPLATE"])
	}
	if updated.TaskModel != nil {
		t.Fatalf("expected a null task model to read back as nil, got %v", *updated.TaskModel)
	}
	if updated.TitleGenerationPromptTemplate != "" {
		t.Fatalf("expected the empty template to read back empty, got %q", updated.TitleGenerationPromptTemplate)
	}
}

func TestSetTaskConfigSendsAllEighteenFields(t *testing.T) {
	var body map[string]any
	c := newRecordingClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSONBody(t, r)
		_, _ = w.Write([]byte(`{}`))
	})

	if _, err := c.SetTaskConfig(context.Background(), TaskConfigForm{}); err != nil {
		t.Fatalf("SetTaskConfig: %v", err)
	}

	if len(body) != 18 {
		t.Fatalf("expected all 18 task config fields in the request body, got %d: %v", len(body), body)
	}
}
