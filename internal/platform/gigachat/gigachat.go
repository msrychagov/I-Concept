// Package gigachat is the client of the GigaChat API behind usecase.TextModel.
// The key and the access token never appear in an error or in a log line.
package gigachat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"vucdiagnostic/internal/usecase"
)

const (
	DefaultOAuthURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	DefaultAPIURL   = "https://api.giga.chat/v1"
	DefaultScope    = "GIGACHAT_API_PERS"

	temperature = 0.01 // GigaChat wants it above zero
	maxTokens   = 2000
	timeout     = 120 * time.Second
	// The token lives 30 minutes; a new one is taken a minute before its end.
	renewBefore = time.Minute
)

// Config is the settings from the environment. Model and AuthKey are required:
// without them the model is off. CAFile is the root certificate of the Ministry
// of Digital Development that signs the servers; it is added to the system roots.
type Config struct {
	AuthKey, Model, Scope, CAFile, OAuthURL, APIURL string
}

// Enabled says whether the settings are enough to call the model.
func (c Config) Enabled() bool { return c.AuthKey != "" && c.Model != "" }

type Client struct {
	config Config
	http   *http.Client
	now    func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

func New(config Config) (*Client, error) {
	if !config.Enabled() {
		return nil, errors.New("gigachat: the key and the model are required")
	}
	if config.Scope == "" {
		config.Scope = DefaultScope
	}
	if config.OAuthURL == "" {
		config.OAuthURL = DefaultOAuthURL
	}
	if config.APIURL == "" {
		config.APIURL = DefaultAPIURL
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if config.CAFile != "" {
		pem, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("gigachat: certificate file: %w", err)
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("gigachat: certificate file has no certificate")
		}
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, Proxy: http.ProxyFromEnvironment}
	return &Client{config: config, http: &http.Client{Timeout: timeout, Transport: transport}, now: time.Now}, nil
}

// Name is the model from the settings.
func (c *Client) Name() string { return c.config.Model }

// clean removes the key and the token from a text.
func (c *Client) clean(text string) string {
	c.mu.Lock()
	token := c.token
	c.mu.Unlock()
	for _, secret := range []string{c.config.AuthKey, token} {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "***")
		}
	}
	return text
}

func (c *Client) fail(format string, args ...any) error {
	return errors.New(c.clean("gigachat: " + fmt.Sprintf(format, args...)))
}

// apiMessage reads the message of an error answer; the rest of the body is dropped.
func apiMessage(body []byte) string {
	var answer struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &answer)
	return truncate(answer.Message, 200)
}

func truncate(text string, n int) string {
	if runes := []rune(text); len(runes) > n {
		return string(runes[:n]) + "…"
	}
	return text
}

func requestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// access returns a valid token, asking for a new one when it is about to expire.
func (c *Client) access(ctx context.Context, renew bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !renew && c.token != "" && c.now().Before(c.expires.Add(-renewBefore)) {
		return c.token, nil
	}
	body := strings.NewReader(url.Values{"scope": {c.config.Scope}}.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.OAuthURL, body)
	if err != nil {
		return "", errors.New("gigachat: bad address of the token service")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("RqUID", requestID())
	request.Header.Set("Authorization", "Basic "+c.config.AuthKey)
	data, status, err := c.send(request)
	if err != nil {
		return "", errors.New(c.cleanLocked("gigachat: no token: " + err.Error()))
	}
	if status != http.StatusOK {
		return "", errors.New(c.cleanLocked(fmt.Sprintf("gigachat: no token: HTTP %d %s", status, apiMessage(data))))
	}
	var answer struct {
		Token   string `json:"access_token"`
		Expires int64  `json:"expires_at"`
	}
	if json.Unmarshal(data, &answer) != nil || answer.Token == "" {
		return "", errors.New("gigachat: unexpected answer of the token service")
	}
	c.token, c.expires = answer.Token, time.UnixMilli(answer.Expires)
	return c.token, nil
}

// cleanLocked is clean for a caller that holds the lock.
func (c *Client) cleanLocked(text string) string {
	for _, secret := range []string{c.config.AuthKey, c.token} {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "***")
		}
	}
	return text
}

func (c *Client) send(request *http.Request) ([]byte, int, error) {
	response, err := c.http.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	return data, response.StatusCode, err
}

// Complete asks the model for one answer that follows the schema. A rejected
// token is replaced once.
func (c *Client) Complete(ctx context.Context, completion usecase.Completion) (usecase.Reply, error) {
	limit := maxTokens
	if completion.MaxTokens > 0 {
		limit = completion.MaxTokens
	}
	body := map[string]any{
		"model":       c.config.Model,
		"temperature": temperature,
		"max_tokens":  limit,
		"messages": []map[string]string{
			{"role": "system", "content": completion.System},
			{"role": "user", "content": completion.User},
		},
	}
	if len(completion.Schema) > 0 {
		body["response_format"] = map[string]any{"type": "json_schema", "schema": completion.Schema, "strict": true}
	}
	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return usecase.Reply{}, errors.New("gigachat: cannot encode the request")
	}
	for attempt := 0; ; attempt++ {
		token, err := c.access(ctx, attempt > 0)
		if err != nil {
			return usecase.Reply{}, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(c.config.APIURL, "/")+"/chat/completions", bytes.NewReader(payload.Bytes()))
		if err != nil {
			return usecase.Reply{}, errors.New("gigachat: bad address of the service")
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		started := c.now()
		data, status, err := c.send(request)
		seconds := c.now().Sub(started).Seconds()
		switch {
		case err != nil:
			return usecase.Reply{}, c.fail("request failed: %v", err)
		case status == http.StatusUnauthorized && attempt == 0:
			continue
		case status != http.StatusOK:
			return usecase.Reply{}, c.fail("HTTP %d %s", status, apiMessage(data))
		}
		var answer struct {
			Model   string `json:"model"`
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				In  int `json:"prompt_tokens"`
				Out int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(data, &answer) != nil || len(answer.Choices) == 0 {
			return usecase.Reply{}, errors.New("gigachat: unexpected answer of the service")
		}
		return usecase.Reply{Text: answer.Choices[0].Message.Content, Version: answer.Model, In: answer.Usage.In, Out: answer.Usage.Out, Seconds: seconds}, nil
	}
}
