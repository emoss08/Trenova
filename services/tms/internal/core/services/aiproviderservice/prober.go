package aiproviderservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/httpsafe"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Prober issues a live call against a configured endpoint.
//
// The check that earns its keep is not reachability but schema adherence.
// Several OpenAI-compatible servers accept a json_schema request and quietly
// ignore it — Ollama's compatibility endpoint is the best-known case — and the
// only way to find out is to ask for a known shape and see what comes back.
// Discovering that here, while an administrator is looking at the form, is far
// better than discovering it from a malformed billing diagnosis weeks later.
type Prober struct {
	logger   *zap.Logger
	cfg      *config.DocumentIntelligenceConfig
	adapters *modeladapter.Registry

	clientsMu sync.Mutex
	clients   map[bool]*http.Client
}

type ProberParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

func NewProber(p ProberParams) *Prober {
	return &Prober{
		logger:   p.Logger.Named("service.aiprovider.prober"),
		cfg:      p.Config.GetDocumentIntelligenceConfig(),
		adapters: modeladapter.NewRegistry(),
		clients:  make(map[bool]*http.Client, 2),
	}
}

const (
	probeSystemPrompt = "You are a configuration test. Follow the output format exactly."
	probeUserPrompt   = "Reply with the word \"ready\" as the status, and the number 7 as the check."
	probeMaxTokens    = 256
)

type probePayload struct {
	Status string `json:"status"`
	Check  int    `json:"check"`
}

func probeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{"type": "string"},
			"check":  map[string]any{"type": "integer"},
		},
		"required":             []string{"status", "check"},
		"additionalProperties": false,
	}
}

// Probe calls the provider and reports what it found.
func (p *Prober) Probe(
	ctx context.Context,
	provider *aiprovider.Provider,
	apiKey string,
) *services.TestAIProviderResult {
	adapter, err := p.adapters.Get(provider.Kind)
	if err != nil {
		return &services.TestAIProviderResult{
			Success: false,
			Message: "This provider's protocol is not supported",
			Detail:  err.Error(),
		}
	}

	schema := probeSchema()
	call := &modeladapter.Call{
		Provider: provider,
		APIKey:   apiKey,
		Client:   p.clientFor(provider),
		Request: &modeladapter.Request{
			System: modeladapter.WithSchemaInstruction(
				probeSystemPrompt,
				schema,
				provider.StructuredOutputMode,
			),
			Messages:     modeladapter.UserMessage(probeUserPrompt),
			OutputSchema: schema,
			SchemaName:   "connection_probe",
			MaxTokens:    probeMaxTokens,
		},
	}

	started := time.Now()
	resp, err := adapter.Complete(ctx, call)
	latency := time.Since(started).Milliseconds()

	if err != nil {
		return &services.TestAIProviderResult{
			Success:   false,
			Message:   "Could not reach the endpoint",
			LatencyMS: latency,
			Detail:    err.Error(),
		}
	}

	if resp.Refused {
		return &services.TestAIProviderResult{
			Success:         false,
			Message:         "The model declined the test request",
			ModelIdentifier: resp.ModelIdentifier,
			LatencyMS:       latency,
		}
	}

	if strings.TrimSpace(resp.Text) == "" {
		return &services.TestAIProviderResult{
			Success:         false,
			Message:         "The endpoint replied with no content",
			ModelIdentifier: resp.ModelIdentifier,
			LatencyMS:       latency,
		}
	}

	return p.evaluate(provider, resp, latency)
}

func (p *Prober) evaluate(
	provider *aiprovider.Provider,
	resp *modeladapter.Response,
	latency int64,
) *services.TestAIProviderResult {
	result := &services.TestAIProviderResult{
		Success:         true,
		ModelIdentifier: resp.ModelIdentifier,
		LatencyMS:       latency,
	}

	var payload probePayload
	if err := modeladapter.ExtractJSON(resp.Text, &payload); err != nil {
		result.SchemaHonoured = false
		result.Message = "Connected, but the reply could not be read as JSON"
		result.Detail = p.schemaAdvice(provider)

		return result
	}

	if strings.TrimSpace(payload.Status) == "" {
		result.SchemaHonoured = false
		result.Message = "Connected, but the reply did not match the requested shape"
		result.Detail = p.schemaAdvice(provider)

		return result
	}

	result.SchemaHonoured = true
	result.Message = fmt.Sprintf("Connected to %s successfully", resp.ModelIdentifier)

	if resp.ModelIdentifier != "" && resp.ModelIdentifier != provider.Model {
		// A server that serves a different model than was asked for is a
		// misconfiguration worth surfacing, not a failure.
		result.Detail = fmt.Sprintf(
			"The endpoint reported serving %q, but this provider is configured for %q.",
			resp.ModelIdentifier, provider.Model,
		)
	}

	return result
}

// schemaAdvice names the next step rather than only reporting the symptom, since
// the fix depends on which mode the provider was set to.
func (p *Prober) schemaAdvice(provider *aiprovider.Provider) string {
	switch provider.StructuredOutputMode {
	case aiprovider.StructuredOutputJSONSchema:
		return "This endpoint is set to enforce JSON schemas but did not honour one. " +
			"Some OpenAI-compatible servers accept the request and ignore the schema. " +
			"Try setting structured output to Prompted."
	case aiprovider.StructuredOutputJSONMode:
		return "This endpoint is set to guarantee valid JSON but returned something else. " +
			"Try setting structured output to Prompted."
	case aiprovider.StructuredOutputPrompted:
		return "This model did not follow the requested format even when asked directly. " +
			"It may be too small for structured work; a larger or instruction-tuned " +
			"model is usually the fix."
	default:
		return ""
	}
}

func (p *Prober) clientFor(provider *aiprovider.Provider) *http.Client {
	p.clientsMu.Lock()
	defer p.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork
	if client, ok := p.clients[allowPrivate]; ok {
		return client
	}

	client := httpsafe.NewClientWithPolicy(p.cfg.GetAITimeout(), httpsafe.Policy{
		AllowPrivateNetworks:  allowPrivate,
		ResponseHeaderTimeout: p.cfg.GetAITimeout(),
	})
	p.clients[allowPrivate] = client

	return client
}
