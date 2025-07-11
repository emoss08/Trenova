package claude

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

const (
	apiURL             = "https://api.anthropic.com/v1/messages"
	defaultMaxTokens   = 500
	defaultTemperature = 0.3
)

// Config holds Claude API configuration
type Config struct {
	APIKey      string
	MaxTokens   int
	Temperature float64
}

// Client is the Claude API client
type Client struct {
	config     Config
	httpClient *http.Client
	logger     *zerolog.Logger
}

type ClientParams struct {
	Logger *logger.Logger
	Config Config
}

// NewClient creates a new Claude API client
func NewClient(p ClientParams) *Client {
	config := p.Config

	log := p.Logger.With().Str("component", "claude-client").Logger()

	if config.MaxTokens == 0 {
		config.MaxTokens = defaultMaxTokens
	}
	if config.Temperature == 0 {
		config.Temperature = defaultTemperature
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: &log,
	}
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents the Claude API request
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"` //nolint:tagliatelle // this is for the API request
	Temperature float64   `json:"temperature"`
}

// Response represents the Claude API response
type Response struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`   //nolint:tagliatelle // this is for the API response
	StopSequence string `json:"stop_sequence"` //nolint:tagliatelle // this is for the API response
	Usage        struct {
		InputTokens  int `json:"input_tokens"`  //nolint:tagliatelle // this is for the API response
		OutputTokens int `json:"output_tokens"` //nolint:tagliatelle // this is for the API response
	} `json:"usage"`
}

// Complete sends a completion request to Claude API
func (c *Client) Complete(
	ctx context.Context,
	prompt, model string,
) (string, error) {
	req := Request{
		Model: model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	}

	body, err := sonic.Marshal(req)
	if err != nil {
		return "", eris.Wrap(err, "failed to marshal request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", eris.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", eris.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", eris.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("response", string(respBody)).
			Msg("Claude API error")
		return "", eris.Errorf("Claude API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var response Response
	if err = sonic.Unmarshal(respBody, &response); err != nil {
		return "", eris.Wrap(err, "failed to unmarshal response")
	}

	if len(response.Content) == 0 {
		return "", eris.New("empty response from Claude")
	}

	return response.Content[0].Text, nil
}
