package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultEndpoint = "https://openrouter.ai/api/alpha/decisions"
	defaultModel    = "typesafe/jev-1.13"
	defaultTimeout  = 3 * time.Second
	maxRequestSize  = 512 << 10
	maxResponseSize = 1 << 20
	retryDelay      = 15 * time.Second
)

type Config struct {
	APIKey   string
	Endpoint string
	Model    string
	Timeout  time.Duration
	SiteURL  string
	AppName  string
}

type Client struct {
	apiKey     string
	endpoint   string
	model      string
	siteURL    string
	appName    string
	httpClient *http.Client
	mu         sync.Mutex
	retryAfter time.Time
}

type Decision struct {
	Choice        string
	Confidence    float64
	Probabilities map[string]float64
	Model         string
}

type Choice struct {
	Instructions string
	Criteria     map[string]string
}

type choiceQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type decisionRequest struct {
	Model     string                    `json:"model"`
	State     any                       `json:"state"`
	Questions map[string]choiceQuestion `json:"questions"`
}

type decisionResponse struct {
	Model   string `json:"model"`
	Answers map[string]struct {
		Type          string             `json:"type"`
		Choice        string             `json:"choice"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"answers"`
}

func FromEnv() *Client {
	if disabled(os.Getenv("JEV_ENABLED")) {
		return nil
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		return nil
	}
	timeout := defaultTimeout
	if raw := strings.TrimSpace(os.Getenv("JEV_TIMEOUT")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	client, err := New(Config{
		APIKey: apiKey, Endpoint: os.Getenv("JEV_API_URL"), Model: os.Getenv("JEV_MODEL"), Timeout: timeout,
		SiteURL: os.Getenv("OPENROUTER_SITE_URL"), AppName: os.Getenv("OPENROUTER_APP_NAME"),
	})
	if err != nil {
		return nil
	}
	return client
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("OpenRouter API key is required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		apiKey: strings.TrimSpace(cfg.APIKey), endpoint: endpoint, model: model,
		siteURL: strings.TrimSpace(cfg.SiteURL), appName: strings.TrimSpace(cfg.AppName),
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) Choose(ctx context.Context, state any, instruction string, criteria map[string]string) (Decision, error) {
	decisions, err := c.ChooseMany(ctx, state, map[string]Choice{"match": {Instructions: instruction, Criteria: criteria}})
	if err != nil {
		return Decision{}, err
	}
	return decisions["match"], nil
}

func (c *Client) ChooseMany(ctx context.Context, state any, choices map[string]Choice) (decisions map[string]Decision, err error) {
	if c == nil {
		return nil, errors.New("Jev is disabled")
	}
	if c.isCoolingDown() {
		return nil, errors.New("Jev is temporarily unavailable after a previous failure")
	}
	defer func() {
		if err != nil {
			c.markUnavailable()
		}
	}()
	if len(choices) == 0 {
		return nil, errors.New("at least one Jev choice is required")
	}
	questions := make(map[string]choiceQuestion, len(choices))
	for id, choice := range choices {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(choice.Instructions) == "" {
			return nil, errors.New("Jev question id and instruction are required")
		}
		if len(choice.Criteria) < 2 {
			return nil, fmt.Errorf("Jev choice %q requires at least two criteria", id)
		}
		questions[id] = choiceQuestion{Type: "choice", Instructions: choice.Instructions, Criteria: choice.Criteria}
	}
	payload, err := json.Marshal(decisionRequest{Model: c.model, State: state, Questions: questions})
	if err != nil {
		return nil, fmt.Errorf("encode Jev request: %w", err)
	}
	if len(payload) > maxRequestSize {
		return nil, errors.New("Jev request exceeds size limit")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build Jev request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.siteURL != "" {
		req.Header.Set("HTTP-Referer", c.siteURL)
	}
	if c.appName != "" {
		req.Header.Set("X-OpenRouter-Title", c.appName)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Jev: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read Jev response: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, errors.New("Jev response exceeds size limit")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Jev returned HTTP %d", resp.StatusCode)
	}
	var decoded decisionResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode Jev response: %w", err)
	}
	decisions = make(map[string]Decision, len(choices))
	for id, choice := range choices {
		answer, ok := decoded.Answers[id]
		if !ok || answer.Type != "choice" {
			return nil, fmt.Errorf("Jev response is missing choice %q", id)
		}
		if _, ok := choice.Criteria[answer.Choice]; !ok {
			return nil, fmt.Errorf("Jev returned unknown choice %q for %q", answer.Choice, id)
		}
		if math.IsNaN(answer.Confidence) || math.IsInf(answer.Confidence, 0) || answer.Confidence < 0 || answer.Confidence > 1 {
			return nil, fmt.Errorf("Jev returned invalid confidence for %q", id)
		}
		for key, probability := range answer.Probabilities {
			if _, ok := choice.Criteria[key]; !ok || math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
				return nil, fmt.Errorf("Jev returned invalid probabilities for %q", id)
			}
		}
		decisions[id] = Decision{Choice: answer.Choice, Confidence: answer.Confidence, Probabilities: answer.Probabilities, Model: decoded.Model}
	}
	return decisions, nil
}

func (c *Client) isCoolingDown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.retryAfter)
}

func (c *Client) markUnavailable() {
	c.mu.Lock()
	c.retryAfter = time.Now().Add(retryDelay)
	c.mu.Unlock()
}

func disabled(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && !enabled
}
