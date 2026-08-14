package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// knowledgeFilesPageHandler serves a knowledge base holding one more file than a
// single page carries. The extra file is only reachable on page 2.
func knowledgeFilesPageHandler(t *testing.T, pages *[]string) http.HandlerFunc {
	t.Helper()
	total := knowledgeFilesPageSize + 1

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/knowledge/k1/files" {
			t.Fatalf("expected /api/v1/knowledge/k1/files, got %q", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		*pages = append(*pages, page)
		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "1":
			items := make([]string, 0, knowledgeFilesPageSize)
			for i := 0; i < knowledgeFilesPageSize; i++ {
				items = append(items, fmt.Sprintf(`{"id":"f%d","user_id":"u","filename":"f%d.txt","meta":{},"created_at":1,"updated_at":2}`, i, i))
			}
			_, _ = fmt.Fprintf(w, `{"items":[%s],"total":%d}`, strings.Join(items, ","), total)
		case "2":
			_, _ = fmt.Fprintf(w, `{"items":[{"id":"flast","user_id":"u","filename":"last.txt","meta":{},"created_at":1,"updated_at":2}],"total":%d}`, total)
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}
}

func TestListKnowledgeFilesReadsEveryPage(t *testing.T) {
	var pages []string
	c := newKnowledgeTestClient(t, knowledgeFilesPageHandler(t, &pages))

	files, err := c.ListKnowledgeFiles(context.Background(), "k1")
	if err != nil {
		t.Fatalf("ListKnowledgeFiles: %v", err)
	}
	if len(files) != knowledgeFilesPageSize+1 {
		t.Fatalf("expected %d files, got %d", knowledgeFilesPageSize+1, len(files))
	}
	if files[len(files)-1].ID != "flast" {
		t.Fatalf("the file on page 2 is missing: last id is %q", files[len(files)-1].ID)
	}
	if len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
		t.Fatalf("expected pages 1 and 2, got %v", pages)
	}
}

func TestListKnowledgeFilesStopsOnAShortPage(t *testing.T) {
	var pages []string
	c := newKnowledgeTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"f1","user_id":"u","filename":"f1.txt","meta":{},"created_at":1,"updated_at":2}],"total":1}`))
	})

	files, err := c.ListKnowledgeFiles(context.Background(), "k1")
	if err != nil {
		t.Fatalf("ListKnowledgeFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if len(pages) != 1 {
		t.Fatalf("expected a single request, got pages %v", pages)
	}
}

func TestCountKnowledgeFilesReadsTheTotal(t *testing.T) {
	var pages []string
	c := newKnowledgeTestClient(t, knowledgeFilesPageHandler(t, &pages))

	count, err := c.CountKnowledgeFiles(context.Background(), "k1")
	if err != nil {
		t.Fatalf("CountKnowledgeFiles: %v", err)
	}
	if count != knowledgeFilesPageSize+1 {
		t.Fatalf("expected %d, got %d", knowledgeFilesPageSize+1, count)
	}
	if len(pages) != 1 {
		t.Fatalf("expected the count to cost one request, got pages %v", pages)
	}
}

func TestFileUserResponseCarriesMetadataOnly(t *testing.T) {
	var item FileUserResponse
	body := `{"id":"f1","user_id":"u","hash":"h","filename":"f1.txt","meta":{"name":"f1.txt","content_type":"text/plain","size":12},"created_at":1,"updated_at":2}`
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(encoded, &out); err != nil {
		t.Fatalf("unmarshal round trip: %v", err)
	}
	if _, ok := out["data"]; ok {
		t.Fatalf("the knowledge file listing has no data key: %+v", out)
	}
	if out["filename"] != "f1.txt" {
		t.Fatalf("expected filename=f1.txt, got %+v", out["filename"])
	}
}
