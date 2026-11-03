package openaiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
)

func TestListModelsPreservesDisplayNameAndCreated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[` +
			`{"id":"model-b","object":"model","created":20,"owned_by":"vendor","display_name":"Model B"},` +
			`{"id":"model-a","object":"model","created":10,"owned_by":"vendor"}]}`))
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), providers.ModelConfig{
		BaseURL:    server.URL + "/v1",
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("models = %#v", models)
	}
	if models[0].ID != "model-a" || models[1].ID != "model-b" {
		t.Fatalf("model order = %#v", models)
	}
	if models[1].DisplayName != "Model B" {
		t.Fatalf("display name = %q, want %q", models[1].DisplayName, "Model B")
	}
	if models[1].Created != 20 {
		t.Fatalf("created = %d, want 20", models[1].Created)
	}
}
