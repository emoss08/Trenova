package modeladapter

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	responsesEmbeddingsPath = "/v1/embeddings"
	chatEmbeddingsPath      = "/embeddings"
	dimensionsField         = "dimensions"
)

type openAIEmbeddingRequest struct {
	Model           string   `json:"model"`
	Input           []string `json:"input"`
	Dimensions      int      `json:"dimensions,omitempty"`
	InputType       string   `json:"input_type,omitempty"`
	OutputDimension int      `json:"output_dimension,omitempty"`
}

type openAIEmbeddingResponse struct {
	Model string                `json:"model"`
	Data  []openAIEmbeddingItem `json:"data"`
	Usage openAIEmbeddingUsage  `json:"usage"`
}

type openAIEmbeddingItem struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

type openAIEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (openAIResponsesAdapter) Embed(ctx context.Context, call *EmbedCall) (*EmbedResponse, error) {
	return embedOpenAIShape(ctx, call, responsesEmbeddingsPath)
}

func (openAIChatAdapter) Embed(ctx context.Context, call *EmbedCall) (*EmbedResponse, error) {
	return embedOpenAIShape(ctx, call, chatEmbeddingsPath)
}

func openAIEmbeddingRequestFor(call *EmbedCall) openAIEmbeddingRequest {
	body := openAIEmbeddingRequest{
		Model: call.Provider.Model,
		Input: call.wireInputs(),
	}

	if call.inputStyle() == aiprovider.EmbeddingInputStyleVoyageInputType {
		body.InputType = call.voyageInputType()
		body.OutputDimension = call.dimensions()

		return body
	}

	if !call.extraBodyNames(dimensionsField) {
		body.Dimensions = call.dimensions()
	}

	return body
}

func embedOpenAIShape(ctx context.Context, call *EmbedCall, path string) (*EmbedResponse, error) {
	payload, err := mergeExtraBody(openAIEmbeddingRequestFor(call), call.Provider)
	if err != nil {
		return nil, err
	}

	var envelope openAIEmbeddingResponse
	if err = postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+path,
		map[string]string{"Authorization": bearer(call.APIKey)},
		payload,
		&envelope,
	); err != nil {
		return nil, err
	}

	vectors, err := orderedEmbeddings(call, envelope.Data)
	if err != nil {
		return nil, err
	}
	if err = checkEmbeddings(call, vectors); err != nil {
		return nil, err
	}

	return &EmbedResponse{
		Vectors:         vectors,
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     max(envelope.Usage.TotalTokens, envelope.Usage.PromptTokens),
	}, nil
}

func orderedEmbeddings(call *EmbedCall, items []openAIEmbeddingItem) ([][]float32, error) {
	if err := checkVectorCount(call, len(items)); err != nil {
		return nil, err
	}

	vectors := make([][]float32, len(items))
	for _, item := range items {
		if item.Index < 0 || item.Index >= len(vectors) || vectors[item.Index] != nil {
			return inArrivalOrder(call, items)
		}
		vectors[item.Index] = item.Embedding
	}

	return vectors, nil
}

func inArrivalOrder(call *EmbedCall, items []openAIEmbeddingItem) ([][]float32, error) {
	vectors := make([][]float32, len(items))
	for idx, item := range items {
		if item.Index != 0 && item.Index != idx {
			return nil, errortypes.NewBusinessError(
				"Embedding provider {0} returned an unusable response",
				call.Provider.Name,
			).WithInternal(fmt.Errorf(
				"embedding indexes are neither ordered nor unique: %w",
				serviceports.ErrEmbeddingResponseInvalid,
			))
		}
		vectors[idx] = item.Embedding
	}

	return vectors, nil
}
