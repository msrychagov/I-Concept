package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vucdiagnostic/internal/usecase"
)

func TestCompleteSendsLocalJSONSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Errorf("unexpected path or method: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("bad request: %v", err)
		}
		if payload["model"] != "qwen3:8b" || payload["stream"] != false || payload["think"] != false {
			t.Errorf("unexpected payload %v", payload)
		}
		if _, ok := payload["format"].(map[string]any); !ok {
			t.Error("missing JSON schema")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"qwen3:8b","message":{"content":"{\"summary\":\"Готово\"}"},"eval_count":13}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "qwen3:8b")
	if err != nil {
		t.Fatal(err)
	}
	reply, err := client.Complete(context.Background(), usecase.Completion{System: "system", User: "user", Schema: json.RawMessage(`{"type":"object"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Out != 13 || reply.Version != "qwen3:8b" {
		t.Fatalf("unexpected reply %+v", reply)
	}
}

func TestPreparedSystemIsStoredOnceAndRequestsContainOnlyCurrentUser(t *testing.T) {
	var alias string
	creates, chats := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Model    string `json:"model"`
			From     string `json:"from"`
			System   string `json:"system"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/api/create":
			creates++
			alias = payload.Model
			if payload.From != "qwen3:1.7b" || payload.System != "common instruction" || !strings.HasPrefix(alias, "i-concept-report:") || len(payload.Messages) != 0 {
				t.Errorf("bad configuration: %+v", payload)
			}
			_, _ = w.Write([]byte(`{"status":"success"}`))
		case "/api/chat":
			chats++
			if chats <= 2 {
				if payload.Model != alias || len(payload.Messages) != 1 || payload.Messages[0].Role != "user" || payload.Messages[0].Content != []string{"student A", "student B"}[chats-1] {
					t.Errorf("history or instruction repeated: %+v", payload)
				}
			} else if payload.Model != "qwen3:1.7b" || len(payload.Messages) != 2 || payload.Messages[0].Content != "different instruction" {
				t.Errorf("unregistered system lost: %+v", payload)
			}
			_, _ = w.Write([]byte(`{"message":{"content":"{}"}}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "qwen3:1.7b")
	if err := client.Prepare(context.Background(), "common instruction"); err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"student A", "student B"} {
		if _, err := client.Complete(context.Background(), usecase.Completion{System: "common instruction", User: user}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := client.Complete(context.Background(), usecase.Completion{System: "different instruction", User: "student C"}); err != nil {
		t.Fatal(err)
	}
	if creates != 1 || chats != 3 {
		t.Fatalf("unexpected calls: %d / %d", creates, chats)
	}
}

func TestFailedPreparationKeepsInlineSystem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/create" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["model"] != "qwen3:1.7b" || len(payload["messages"].([]any)) != 2 {
			t.Error("fallback lost system instruction")
		}
		_, _ = w.Write([]byte(`{"message":{"content":"{}"}}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "qwen3:1.7b")
	if client.Prepare(context.Background(), "system") == nil {
		t.Fatal("expected preparation failure")
	}
	if _, err := client.Complete(context.Background(), usecase.Completion{System: "system", User: "data"}); err != nil {
		t.Fatal(err)
	}
}
