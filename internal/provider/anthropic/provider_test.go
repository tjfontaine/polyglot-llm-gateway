package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestListModels(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{
  "data": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "type": "model",
      "display_name": "Claude 3.5 Sonnet",
      "created_at": "2024-10-22T00:00:00Z"
    },
    {
      "id": "claude-3-haiku-20240307",
      "type": "model",
      "display_name": "Claude 3 Haiku",
      "created_at": "2024-03-07T00:00:00Z"
    }
  ],
  "first_id": "claude-3-5-sonnet-20241022",
  "last_id": "claude-3-haiku-20240307",
  "has_more": false
}`)
	}))
	defer ts.Close()

	p := New("test-key", option.WithBaseURL(ts.URL))

	list, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels returned error: %v", err)
	}

	if list.Object != "list" {
		t.Fatalf("expected object 'list', got %q", list.Object)
	}

	if len(list.Data) != 2 {
		t.Fatalf("expected 2 models, got %d", len(list.Data))
	}

	if list.Data[0].ID != "claude-3-5-sonnet-20241022" || list.Data[0].Object != "model" {
		t.Fatalf("unexpected first model: %+v", list.Data[0])
	}
}
