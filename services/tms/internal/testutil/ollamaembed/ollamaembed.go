// Package ollamaembed embeds text through a local Ollama server's native
// /api/embed endpoint. It exists only to record the embedding fixtures the eval
// gate reads; the application has no Ollama provider.
package ollamaembed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
)

const (
	embedPath      = "/api/embed"
	documentPrefix = "search_document: "
	queryPrefix    = "search_query: "
	maxErrorBody   = 4 << 10
)

type Params struct {
	BaseURL    string
	Model      string
	Dimensions int
	Timeout    time.Duration
}

type Client struct {
	baseURL    string
	model      string
	dimensions int
	http       *http.Client
}

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Error      string      `json:"error"`
}

// BaseURLFromEnv is the server the recorders embed through:
// TRENOVA_EVAL_OLLAMA_URL, or Ollama's default address.
func BaseURLFromEnv() string {
	if baseURL := strings.TrimSpace(os.Getenv(agentevalgate.OllamaURLEnv)); baseURL != "" {
		return baseURL
	}

	return agentevalgate.DefaultOllamaURL
}

func New(params Params) *Client {
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(params.BaseURL), "/"),
		model:      params.Model,
		dimensions: params.Dimensions,
		http:       &http.Client{Timeout: params.Timeout},
	}
}

// Embed embeds inputs with nomic-embed-text's task prefixes: search_query for
// a query, search_document for everything else.
func (c *Client) Embed(
	ctx context.Context,
	purpose aiusageports.EmbeddingPurpose,
	inputs []string,
) ([][]float32, error) {
	prefix := documentPrefix
	if purpose == aiusageports.EmbeddingPurposeQuery {
		prefix = queryPrefix
	}

	prefixed := make([]string, len(inputs))
	for idx, input := range inputs {
		prefixed[idx] = prefix + input
	}

	var response embedResponse
	if err := c.post(ctx, embedRequest{
		Model:      c.model,
		Input:      prefixed,
		Dimensions: c.dimensions,
	}, &response); err != nil {
		return nil, err
	}

	if len(response.Embeddings) != len(inputs) {
		return nil, fmt.Errorf(
			"ollama returned %d vectors for %d texts",
			len(response.Embeddings),
			len(inputs),
		)
	}
	for idx, vector := range response.Embeddings {
		if len(vector) != c.dimensions {
			return nil, fmt.Errorf(
				"ollama returned %d dimensions for text %d, want %d",
				len(vector),
				idx,
				c.dimensions,
			)
		}
		for pos, value := range vector {
			if f := float64(value); math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, fmt.Errorf("vector %d component %d is not a finite number", idx, pos)
			}
		}
	}

	return response.Embeddings, nil
}

func (c *Client) post(ctx context.Context, body embedRequest, out *embedResponse) error {
	encoded, err := sonic.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+embedPath,
		bytes.NewReader(encoded),
	)
	if err != nil {
		return fmt.Errorf("build embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute embed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read embed response: %w", err)
	}

	if err = sonic.Unmarshal(payload, out); err != nil &&
		resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return fmt.Errorf("decode embed response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(out.Error)
		if message == "" {
			message = strings.TrimSpace(string(payload[:min(len(payload), maxErrorBody)]))
		}

		return fmt.Errorf("ollama returned %s: %s", resp.Status, message)
	}

	return nil
}
