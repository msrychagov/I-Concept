package gigachat

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"vucdiagnostic/internal/usecase"
)

const key = "SECRET-AUTH-KEY"

// service imitates both addresses of GigaChat: the token one and the chat one.
type service struct {
	mu        sync.Mutex
	tokens    int
	chats     int
	lastBody  map[string]any
	lastHead  http.Header
	oauthForm url.Values
	chat      func(w http.ResponseWriter, r *http.Request, n int)
	expires   time.Time
}

func (s *service) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.tokens++
		_ = r.ParseForm()
		s.oauthForm = r.PostForm
		s.lastHead = r.Header.Clone()
		if r.Header.Get("Authorization") != "Basic "+key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"message":"bad key `+r.Header.Get("Authorization")+`"}`)
			return
		}
		expires := s.expires
		if expires.IsZero() {
			expires = time.Now().Add(30 * time.Minute)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token-" + string(rune('0'+s.tokens)), "expires_at": expires.UnixMilli()})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.chats++
		n := s.chats
		s.lastBody = map[string]any{}
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &s.lastBody)
		head := r.Header.Clone()
		s.mu.Unlock()
		s.mu.Lock()
		s.lastHead = head
		s.mu.Unlock()
		if s.chat != nil {
			s.chat(w, r, n)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "GigaChat-3-Ultra:2026.10",
			"choices": []map[string]any{{"message": map[string]any{"content": `{"summary":"ok"}`}}},
			"usage":   map[string]any{"prompt_tokens": 120, "completion_tokens": 30},
		})
	})
	return mux
}

func newClient(t *testing.T, s *service) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(s.handler())
	t.Cleanup(server.Close)
	client, err := New(Config{AuthKey: key, Model: "GigaChat-3-Ultra", OAuthURL: server.URL + "/oauth", APIURL: server.URL + "/v1"})
	if err != nil {
		t.Fatal(err)
	}
	return client, server
}

var question = usecase.Completion{System: "Инструкция <b>", User: "{\"возраст\": 19}", Schema: json.RawMessage(`{"type":"object","required":["summary"]}`)}

func TestCompleteSendsTheRequestAsTheComparisonDid(t *testing.T) {
	s := &service{}
	client, _ := newClient(t, s)
	if client.Name() != "GigaChat-3-Ultra" {
		t.Fatalf("name %s", client.Name())
	}
	reply, err := client.Complete(context.Background(), question)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != `{"summary":"ok"}` || reply.Version != "GigaChat-3-Ultra:2026.10" || reply.In != 120 || reply.Out != 30 || reply.Seconds < 0 {
		t.Fatalf("reply %+v", reply)
	}
	if s.oauthForm.Get("scope") != "GIGACHAT_API_PERS" || !strings.HasPrefix(s.lastHead.Get("Authorization"), "Bearer token-") {
		t.Fatalf("form %v, header %v", s.oauthForm, s.lastHead)
	}
	body := s.lastBody
	format, _ := body["response_format"].(map[string]any)
	messages, _ := body["messages"].([]any)
	if body["model"] != "GigaChat-3-Ultra" || body["temperature"] != 0.01 || body["max_tokens"] != 2000.0 || len(messages) != 2 {
		t.Fatalf("body %v", body)
	}
	if format["type"] != "json_schema" || format["strict"] != true || format["schema"].(map[string]any)["type"] != "object" {
		t.Fatalf("response_format %v", format)
	}
	system, user := messages[0].(map[string]any), messages[1].(map[string]any)
	if system["role"] != "system" || system["content"] != question.System || user["role"] != "user" || user["content"] != question.User {
		t.Fatalf("messages %v", messages)
	}
}

func TestCompleteTakesTheLimitOfTheTask(t *testing.T) {
	s := &service{}
	client, _ := newClient(t, s)
	long := question
	long.MaxTokens = 4000
	if _, err := client.Complete(context.Background(), long); err != nil {
		t.Fatal(err)
	}
	if s.lastBody["max_tokens"] != 4000.0 {
		t.Fatalf("max_tokens %v", s.lastBody["max_tokens"])
	}
}

func TestOAuthRequestHasBasicKeyAndRqUID(t *testing.T) {
	s := &service{}
	client, _ := newClient(t, s)
	if _, err := client.access(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	id := s.lastHead.Get("RqUID")
	if s.lastHead.Get("Authorization") != "Basic "+key || len(id) != 36 || id[14] != '4' || s.lastHead.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("header %v", s.lastHead)
	}
}

func TestTokenIsKeptAndRenewedAMinuteBeforeTheEnd(t *testing.T) {
	s := &service{}
	client, _ := newClient(t, s)
	clock := time.Now()
	client.now = func() time.Time { return clock }
	for range 3 {
		if _, err := client.Complete(context.Background(), question); err != nil {
			t.Fatal(err)
		}
	}
	if s.tokens != 1 {
		t.Fatalf("%d tokens for 3 requests", s.tokens)
	}
	clock = clock.Add(29*time.Minute + 30*time.Second)
	if _, err := client.Complete(context.Background(), question); err != nil {
		t.Fatal(err)
	}
	if s.tokens != 2 || s.lastHead.Get("Authorization") != "Bearer token-2" {
		t.Fatalf("%d tokens, %s", s.tokens, s.lastHead.Get("Authorization"))
	}
}

func TestARejectedTokenIsReplacedOnce(t *testing.T) {
	s := &service{}
	s.chat = func(w http.ResponseWriter, r *http.Request, n int) {
		if r.Header.Get("Authorization") == "Bearer token-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]any{"content": "{}"}}}})
	}
	client, _ := newClient(t, s)
	if _, err := client.Complete(context.Background(), question); err != nil {
		t.Fatal(err)
	}
	if s.tokens != 2 || s.chats != 2 {
		t.Fatalf("%d tokens, %d chats", s.tokens, s.chats)
	}
	// Always rejected: gives up after one new token.
	s.chat = func(w http.ResponseWriter, r *http.Request, n int) { w.WriteHeader(http.StatusUnauthorized) }
	s.tokens, s.chats = 0, 0
	client.token = ""
	if _, err := client.Complete(context.Background(), question); err == nil || s.chats != 2 {
		t.Fatalf("%v after %d chats", err, s.chats)
	}
}

func TestErrorsDoNotContainTheKeyOrTheToken(t *testing.T) {
	s := &service{}
	s.chat = func(w http.ResponseWriter, r *http.Request, n int) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"message":"limit for `+r.Header.Get("Authorization")+` and `+key+`"}`)
	}
	client, _ := newClient(t, s)
	_, err := client.Complete(context.Background(), question)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("error %v", err)
	}
	for _, secret := range []string{key, "token-1"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("%q in %q", secret, err)
		}
	}
	// A wrong key: the token service echoes the header, the error does not.
	wrong, err := New(Config{AuthKey: "other-key", Model: "m", OAuthURL: client.config.OAuthURL, APIURL: client.config.APIURL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrong.Complete(context.Background(), question)
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "other-key") {
		t.Fatalf("error %v", err)
	}
}

func TestBrokenAnswersAndCancellation(t *testing.T) {
	s := &service{}
	s.chat = func(w http.ResponseWriter, r *http.Request, n int) { _, _ = io.WriteString(w, `{"choices":[]}`) }
	client, _ := newClient(t, s)
	if _, err := client.Complete(context.Background(), question); err == nil {
		t.Fatal("empty choices accepted")
	}
	s.chat = func(w http.ResponseWriter, r *http.Request, n int) { <-r.Context().Done() }
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := client.Complete(ctx, question); err == nil || time.Since(started) > 5*time.Second {
		t.Fatalf("cancelled request: %v", err)
	}
}

func TestSettings(t *testing.T) {
	if _, err := New(Config{Model: "m"}); err == nil {
		t.Fatal("no key accepted")
	}
	if _, err := New(Config{AuthKey: key}); err == nil {
		t.Fatal("no model accepted")
	}
	if (Config{AuthKey: key}).Enabled() || !(Config{AuthKey: key, Model: "m"}).Enabled() {
		t.Fatal("Enabled")
	}
	missing := filepath.Join(t.TempDir(), "none.crt")
	if _, err := New(Config{AuthKey: key, Model: "m", CAFile: missing}); err == nil {
		t.Fatal("missing certificate file accepted")
	}
	junk := filepath.Join(t.TempDir(), "junk.crt")
	_ = os.WriteFile(junk, []byte("not a certificate"), 0o600)
	if _, err := New(Config{AuthKey: key, Model: "m", CAFile: junk}); err == nil {
		t.Fatal("junk certificate file accepted")
	}
	client, err := New(Config{AuthKey: key, Model: "m"})
	if err != nil || client.config.Scope != "GIGACHAT_API_PERS" || client.config.OAuthURL != DefaultOAuthURL || client.config.APIURL != DefaultAPIURL {
		t.Fatalf("defaults: %+v %v", client, err)
	}
}

// The servers are signed by a root that the system does not know: the client
// trusts the one from the file, and only that one.
func TestCertificateFileMakesAnUnknownRootTrusted(t *testing.T) {
	server := httptest.NewTLSServer((&service{}).handler())
	defer server.Close()
	cert := filepath.Join(t.TempDir(), "root.crt")
	block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(cert, block, 0o600); err != nil {
		t.Fatal(err)
	}
	config := Config{AuthKey: key, Model: "m", OAuthURL: server.URL + "/oauth", APIURL: server.URL + "/v1"}
	without, _ := New(config)
	if _, err := without.Complete(context.Background(), question); err == nil {
		t.Fatal("unknown root accepted")
	}
	config.CAFile = cert
	with, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := with.Complete(context.Background(), question); err != nil {
		t.Fatal(err)
	}
}

var _ usecase.TextModel = (*Client)(nil)
