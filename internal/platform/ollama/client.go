// Package ollama provides a local-only (self-hosted) Ollama adapter.
package ollama

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"vucdiagnostic/internal/usecase"
)

type Client struct {
	URL      string
	Model    string
	http     *http.Client
	mu       sync.RWMutex
	prepared map[string]string
}

func New(url, model string) (*Client, error) {
	url, model = strings.TrimRight(strings.TrimSpace(url), "/"), strings.TrimSpace(model)
	if model == "" || !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		return nil, errors.New("ollama: set a model and an HTTP URL")
	}
	return &Client{URL: url, Model: model, http: &http.Client{Timeout: 10 * time.Minute}, prepared: make(map[string]string)}, nil
}

func (c *Client) Name() string { return "ollama/" + c.Model }

// Prepare stores a system instruction in a derived Ollama model. It reuses the
// base weights and stores no participant messages or conversation history.
func (c *Client) Prepare(ctx context.Context, system string) error {
	digest := sha256.Sum256([]byte(c.Model + "\x00" + system))
	name := fmt.Sprintf("i-concept-report:%x", digest[:12])
	raw, err := json.Marshal(map[string]any{"model": name, "from": c.Model, "system": system, "stream": false})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/create", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ollama: prepare system prompt: %w", err)
	}
	defer res.Body.Close()
	var result struct {
		Status string `json:"status"`
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama: prepare returned HTTP %d", res.StatusCode)
	}
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result) != nil || result.Status != "success" {
		return errors.New("ollama: system prompt preparation failed")
	}
	c.mu.Lock()
	c.prepared[system] = name
	c.mu.Unlock()
	return nil
}

func (c *Client) Complete(ctx context.Context, prompt usecase.Completion) (usecase.Reply, error) {
	c.mu.RLock()
	prepared := c.prepared[prompt.System]
	c.mu.RUnlock()
	model := c.Model
	messages := []map[string]string{{"role": "system", "content": prompt.System}, {"role": "user", "content": prompt.User}}
	if prepared != "" {
		model = prepared
		messages = []map[string]string{{"role": "user", "content": prompt.User}}
	}
	reqBody := map[string]any{
		"model":    model,
		"stream":   false,
		"think":    false,
		"messages": messages,
		"options":  map[string]any{"temperature": 0.15, "num_ctx": 8192, "num_predict": 5000},
	}
	if len(prompt.Schema) > 0 {
		reqBody["format"] = json.RawMessage(prompt.Schema)
	} else {
		reqBody["format"] = "json"
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return usecase.Reply{}, errors.New("ollama: cannot encode request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return usecase.Reply{}, errors.New("ollama: invalid server address")
	}
	req.Header.Set("Content-Type", "application/json")
	started := time.Now()
	res, err := c.http.Do(req)
	if err != nil {
		return usecase.Reply{}, errors.New("ollama: local model unavailable or timed out")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return usecase.Reply{}, errors.New("ollama: cannot read model response")
	}
	if res.StatusCode != http.StatusOK {
		return usecase.Reply{}, fmt.Errorf("ollama: model returned HTTP %d", res.StatusCode)
	}
	var answer struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if json.Unmarshal(body, &answer) != nil || strings.TrimSpace(answer.Message.Content) == "" {
		return usecase.Reply{}, errors.New("ollama: empty or invalid model response")
	}
	return usecase.Reply{Text: answer.Message.Content, Version: answer.Model, In: answer.PromptEvalCount, Out: answer.EvalCount, Seconds: time.Since(started).Seconds()}, nil
}

var _ usecase.TextModel = (*Client)(nil)
