package modeladapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	maxModelListBytes    = 16 << 20
	anthropicModelsLimit = 1000
	maxAnthropicPages    = 10
)

var (
	embeddingFamilyID = regexp.MustCompile(`(^|[-_/.:])(bge|e5|gte)([-_/.:]|$)`)
	perMillion        = decimal.NewFromInt(1_000_000)
)

type ModelsCall struct {
	Provider *aiprovider.Provider
	APIKey   string
	Client   *http.Client
	// ServerCredentialsAllowed lets a provider sign in to its platform with
	// the server's own cloud identity.
	ServerCredentialsAllowed bool
}

type ModelInfo struct {
	ID                   string
	DisplayName          string
	ContextWindow        int
	Embedding            bool
	InputCostPerMillion  *decimal.Decimal
	OutputCostPerMillion *decimal.Decimal
	CreatedAt            int64
}

func ListModels(ctx context.Context, call *ModelsCall) ([]ModelInfo, error) {
	switch call.Provider.Kind {
	case aiprovider.KindAnthropicMessages:
		return listAnthropicModels(ctx, call)
	case aiprovider.KindOpenAIResponses:
		return listOpenAIModels(ctx, call, call.Provider.ResolvedBaseURL()+"/v1/models")
	case aiprovider.KindOpenAIChat:
		return listOpenAIModels(ctx, call, call.Provider.ResolvedBaseURL()+"/models")
	default:
		return nil, fmt.Errorf("unsupported provider kind %q", call.Provider.Kind)
	}
}

func IsEmbeddingModelID(id string) bool {
	lowered := strings.ToLower(id)
	if strings.Contains(lowered, "rerank") {
		return false
	}

	return strings.Contains(lowered, "embed") ||
		strings.HasPrefix(lowered, "voyage-") ||
		embeddingFamilyID.MatchString(lowered)
}

func getModelList(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	headers map[string]string,
	out any,
) error {
	payload, err := getBody(ctx, client, endpoint, headers)
	if err != nil {
		return err
	}
	if err = sonic.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode provider response: %w", err)
	}

	return nil
}

func getBody(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	headers map[string]string,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute provider request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxModelListBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read provider response: %w", err)
	}
	if len(payload) > maxModelListBytes {
		return nil, fmt.Errorf(
			"the provider's model list is larger than %d bytes",
			maxModelListBytes,
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, transportError(resp, payload)
	}

	return payload, nil
}

// ErrModelsNotListed is a platform that serves no Models API: Bedrock,
// Vertex AI and Foundry name their models their own way, so the model ID is
// typed instead.
var ErrModelsNotListed = errors.New("this platform does not list its models")

// listAnthropicModels reads the Models API through the provider's platform,
// signed in the same way its calls are. The window each model reports is the
// one it has; a model that reports none takes the one its id is known by.
func listAnthropicModels(ctx context.Context, call *ModelsCall) ([]ModelInfo, error) {
	if !call.Provider.ResolvedPlatform().ListsModels() {
		return nil, ErrModelsNotListed
	}

	adapterCall := &Call{
		Provider:                 call.Provider,
		APIKey:                   call.APIKey,
		Client:                   call.Client,
		ServerCredentialsAllowed: call.ServerCredentialsAllowed,
	}
	baseURL := call.Provider.ResolvedBaseURL()
	if baseURL == "" {
		return nil, errors.New("this provider has no address to list models at")
	}
	platformOpts, err := newPlatformCredentials().platformOptions(ctx, adapterCall)
	if err != nil {
		return nil, err
	}

	opts := make([]option.RequestOption, 0, len(platformOpts)+4)
	opts = append(opts,
		option.WithBaseURL(baseURL),
		option.WithHTTPClient(call.Client),
		option.WithMaxRetries(0),
		option.WithMiddleware(
			anthropicTransport(adapterCall, false),
			capResponseBody(maxModelListBytes),
		),
	)
	opts = append(opts, platformOpts...)
	service := anthropic.NewModelService(opts...)

	pager := service.ListAutoPaging(ctx, anthropic.ModelListParams{
		Limit: param.NewOpt[int64](anthropicModelsLimit),
	})
	models := make([]ModelInfo, 0, 16)
	for pager.Next() && len(models) < anthropicModelsLimit*maxAnthropicPages {
		item := pager.Current()
		info := ModelInfo{
			ID:          item.ID,
			DisplayName: item.DisplayName,
			CreatedAt:   positiveUnix(item.CreatedAt.Unix()),
		}
		if item.MaxInputTokens > 0 {
			info.ContextWindow = int(item.MaxInputTokens)
		}
		models = append(models, withKnownWindow(info))
	}
	if err = pager.Err(); err != nil {
		return nil, err
	}

	return dedupeModels(models), nil
}

// capResponseBody stops reading a reply past limit bytes: an endpoint that
// sends without end is broken, and must not use up the worker's memory.
func capResponseBody(limit int64) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		resp, err := next(req)
		if err != nil || resp == nil {
			return resp, err
		}
		resp.Body = &cappedBody{body: resp.Body, remaining: limit + 1, limit: limit}

		return resp, nil
	}
}

type cappedBody struct {
	body      io.ReadCloser
	remaining int64
	limit     int64
}

func (c *cappedBody) Read(p []byte) (int, error) {
	if c.remaining <= 0 {
		return 0, fmt.Errorf("the provider's model list is larger than %d bytes", c.limit)
	}
	if int64(len(p)) > c.remaining {
		p = p[:c.remaining]
	}
	n, err := c.body.Read(p)
	c.remaining -= int64(n)
	if c.remaining <= 0 && err == nil {
		return n, fmt.Errorf("the provider's model list is larger than %d bytes", c.limit)
	}

	return n, err
}

func (c *cappedBody) Close() error { return c.body.Close() }

type openAIModelEntry struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	DisplayName      string         `json:"display_name"`
	Type             string         `json:"type"`
	ContextLength    int            `json:"context_length"`
	ContextWindow    int            `json:"context_window"`
	MaxModelLen      int            `json:"max_model_len"`
	MaxContextLength int            `json:"max_context_length"`
	Pricing          map[string]any `json:"pricing"`
	Created          any            `json:"created"`
	Architecture     *struct {
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
}

type openAIModelList struct {
	Data   []openAIModelEntry `json:"data"`
	Models []openAIModelEntry `json:"models"`
}

func listOpenAIModels(
	ctx context.Context,
	call *ModelsCall,
	endpoint string,
) ([]ModelInfo, error) {
	payload, err := getBody(ctx, call.Client, endpoint, bearerHeaders(call.APIKey))
	if err != nil {
		return nil, err
	}

	return decodeOpenAIModels(payload)
}

// ListReferencePrices reads a public, OpenAI-shaped model catalog that lists
// prices, such as OpenRouter's, without credentials. Models it gives no price
// are left out, so what comes back is only what can fill a gap.
func ListReferencePrices(
	ctx context.Context,
	client *http.Client,
	endpoint string,
) ([]ModelInfo, error) {
	payload, err := getBody(ctx, client, endpoint, nil)
	if err != nil {
		return nil, err
	}

	models, err := decodeOpenAIModels(payload)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(models, func(model ModelInfo) bool {
		return model.InputCostPerMillion == nil || !model.InputCostPerMillion.IsPositive()
	}), nil
}

func decodeOpenAIModels(payload []byte) ([]ModelInfo, error) {
	entries, err := openAIEntries(payload)
	if err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(entries))
	for idx := range entries {
		entry := &entries[idx]
		if strings.TrimSpace(entry.ID) == "" {
			continue
		}
		info := ModelInfo{
			ID:          entry.ID,
			DisplayName: stringutils.FirstNonEmptyTrimmed(entry.DisplayName, entry.Name),
			ContextWindow: intutils.FirstPositive(
				entry.ContextLength,
				entry.ContextWindow,
				entry.MaxModelLen,
				entry.MaxContextLength,
			),
			Embedding: entry.Type == "embedding" || entry.embeddingOutput() ||
				IsEmbeddingModelID(entry.ID),
			CreatedAt: unixFromAny(entry.Created),
		}
		info.InputCostPerMillion, info.OutputCostPerMillion = entry.prices()
		models = append(models, withKnownWindow(info))
	}

	return dedupeModels(models), nil
}

func openAIEntries(payload []byte) ([]openAIModelEntry, error) {
	if trimmed := bytes.TrimSpace(payload); len(trimmed) > 0 && trimmed[0] == '[' {
		var entries []openAIModelEntry
		if err := sonic.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("decode provider response: %w", err)
		}

		return entries, nil
	}

	var list openAIModelList
	if err := sonic.Unmarshal(payload, &list); err != nil {
		return nil, fmt.Errorf("decode provider response: %w", err)
	}
	if len(list.Data) > 0 {
		return list.Data, nil
	}

	return list.Models, nil
}

func (e *openAIModelEntry) embeddingOutput() bool {
	if e.Architecture == nil || len(e.Architecture.OutputModalities) == 0 {
		return false
	}

	return slices.Contains(e.Architecture.OutputModalities, "embeddings")
}

func (e *openAIModelEntry) prices() (*decimal.Decimal, *decimal.Decimal) {
	if len(e.Pricing) == 0 {
		return nil, nil
	}
	if input, ok := priceOf(e.Pricing["input"]); ok {
		output, _ := priceOf(e.Pricing["output"])
		return input, output
	}

	input, _ := priceOf(e.Pricing["prompt"])
	output, _ := priceOf(e.Pricing["completion"])

	return scaled(input), scaled(output)
}

func priceOf(value any) (*decimal.Decimal, bool) {
	var parsed decimal.Decimal
	switch typed := value.(type) {
	case string:
		d, err := decimal.NewFromString(strings.TrimSpace(typed))
		if err != nil {
			return nil, false
		}
		parsed = d
	case float64:
		parsed = decimal.NewFromFloat(typed)
	default:
		return nil, false
	}
	if parsed.IsNegative() {
		return nil, false
	}

	return &parsed, true
}

func scaled(perToken *decimal.Decimal) *decimal.Decimal {
	if perToken == nil {
		return nil
	}
	value := perToken.Mul(perMillion).Round(6)

	return &value
}

func unixFromAny(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return positiveUnix(int64(typed))
	case int64:
		return positiveUnix(typed)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0
		}

		return positiveUnix(parsed)
	default:
		return 0
	}
}

func positiveUnix(value int64) int64 {
	if value <= 0 {
		return 0
	}

	return value
}

func withKnownWindow(info ModelInfo) ModelInfo {
	if info.ContextWindow > 0 || info.Embedding {
		return info
	}
	if tokens, ok := aiprovider.KnownContextWindow(info.ID); ok {
		info.ContextWindow = tokens
	}

	return info
}

func dedupeModels(models []ModelInfo) []ModelInfo {
	seen := make(map[string]struct{}, len(models))

	return slices.DeleteFunc(models, func(model ModelInfo) bool {
		if _, dup := seen[model.ID]; dup {
			return true
		}
		seen[model.ID] = struct{}{}

		return false
	})
}
