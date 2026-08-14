package client

import (
	"encoding/json"
	"testing"
)

func TestKnowledgeFormSendsOnlyTheFieldsTheAPIAccepts(t *testing.T) {
	form := KnowledgeForm{
		Name:        "K",
		Description: "d",
		AccessControl: map[string]any{
			"read":        map[string]any{"group_ids": []string{"g1"}, "user_ids": []string{}},
			"public_read": true,
		},
	}
	data, err := json.Marshal(form)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"data", "meta"} {
		if _, ok := out[key]; ok {
			t.Fatalf("%q must not be sent, the API form has no such field: %+v", key, out)
		}
	}
	if out["name"] != "K" || out["description"] != "d" {
		t.Fatalf("unexpected form: %+v", out)
	}
	if _, ok := out["access_grants"].([]any); !ok {
		t.Fatalf("expected access_grants list, got %+v", out["access_grants"])
	}
}

func TestKnowledgeFilesResponseDecodesFileMetadata(t *testing.T) {
	body := `{"id":"k1","user_id":"u1","name":"K","description":"d","created_at":1,"updated_at":2,
	"files":[{"id":"f1","hash":"h1","meta":{"name":"f1.txt"},"created_at":3,"updated_at":4}],"access_grants":[]}`

	var resp KnowledgeFilesResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	file := resp.Files[0]
	if file.ID != "f1" || file.CreatedAt != 3 || file.UpdatedAt != 4 {
		t.Fatalf("unexpected file metadata: %+v", file)
	}
	if file.Hash == nil || *file.Hash != "h1" {
		t.Fatalf("expected hash h1, got %+v", file.Hash)
	}
}
