package aiproviderservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
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
	// probeMaxTokens has to cover a reasoning model's thinking as well as
	// its answer. A model that thinks before it speaks — Nemotron, the
	// DeepSeek-shaped servers, anything with reasoning turned on — spends
	// the budget on the chain of thought first, and a ceiling sized for a
	// two-field JSON object returned an empty reply that read as a broken
	// endpoint.
	probeMaxTokens = 1536
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

	// The probe gets its own deadline rather than inheriting the request's,
	// so a slow endpoint returns a verdict a person can read instead of the
	// handler timing out underneath it.
	ctx, cancel := context.WithTimeout(ctx, p.cfg.GetAIProbeTimeout())
	defer cancel()

	started := time.Now()
	resp, err := adapter.Complete(ctx, call)
	latency := time.Since(started).Milliseconds()

	if err != nil {
		return p.unreachable(err, latency)
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
			Detail:          emptyReplyAdvice(resp),
		}
	}

	return p.evaluate(provider, resp, latency)
}

// unreachable separates an endpoint that did not answer in time from one
// that could not be reached at all.
//
// They look identical in a transport error and mean opposite things to the
// person reading the result: the first is a busy or cold endpoint, usually a
// free or queued tier, and the provider is very likely configured correctly;
// the second is a wrong URL, a blocked network or a dead host. Reporting a
// timeout as "could not reach the endpoint" sent people to check a base URL
// that was right all along.
func (p *Prober) unreachable(err error, latency int64) *services.TestAIProviderResult {
	if isTimeout(err) {
		return &services.TestAIProviderResult{
			Success:   false,
			Message:   "The endpoint did not answer in time",
			LatencyMS: latency,
			Detail: fmt.Sprintf(
				"No response within %s. The test asks the model for a short reply, so a "+
					"queued free tier or a model still loading can exceed it while the "+
					"endpoint is perfectly healthy. Try again, or raise "+
					"documentIntelligence.aiProbeTimeout. Underlying error: %s",
				p.cfg.GetAIProbeTimeout(), err.Error(),
			),
		}
	}

	return &services.TestAIProviderResult{
		Success:   false,
		Message:   "Could not reach the endpoint",
		LatencyMS: latency,
		Detail:    err.Error(),
	}
}

// isTimeout reads a transport failure as a deadline rather than a refusal.
// net.Error covers the client's own timeouts; the context errors cover the
// deadline this probe sets; the string check catches http2's own wording,
// which arrives as a plain error rather than anything typed.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "timeout")
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

// emptyReplyAdvice says why a reply came back empty, which is nearly always
// a reasoning model that spent the whole budget thinking rather than an
// endpoint that is broken. Saying so is the difference between raising a
// limit and rewriting a working configuration.
func emptyReplyAdvice(resp *modeladapter.Response) string {
	switch {
	case resp.ReasoningTokens > 0:
		return fmt.Sprintf(
			"The model spent %d tokens thinking and had none left to answer with. "+
				"This is a reasoning model: lower its reasoning effort, or raise the "+
				"provider's max tokens.",
			resp.ReasoningTokens,
		)
	case resp.Truncated:
		return "The reply stopped at the output limit before it said anything. " +
			"Raise the provider's max tokens."
	default:
		return "The endpoint accepted the request and returned an empty message. " +
			"Check that the model name is one this endpoint serves."
	}
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

	// Both budgets are the probe's own. A response-header timeout sized for
	// a reachability check cuts off a queued endpoint before it has begun
	// answering, which is the one failure mode this test must not invent.
	timeout := p.cfg.GetAIProbeTimeout()
	client := httpsafe.NewClientWithPolicy(timeout, httpsafe.Policy{
		AllowPrivateNetworks:  allowPrivate,
		ResponseHeaderTimeout: timeout,
	})
	p.clients[allowPrivate] = client

	return client
}
