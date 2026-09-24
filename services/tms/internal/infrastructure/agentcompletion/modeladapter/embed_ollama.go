package modeladapter

import (
	"context"

	"github.com/emoss08/trenova/shared/stringutils"
)

const ollamaEmbedPath = "/api/embed"

type ollamaEmbedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type ollamaEmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	PromptEvalCount int         `json:"prompt_eval_count"`
}

func (ollamaAdapter) Embed(ctx context.Context, call *EmbedCall) (*EmbedResponse, error) {
	body := ollamaEmbedRequest{
		Model:      call.Provider.Model,
		Input:      call.wireInputs(),
		Dimensions: call.dimensions(),
	}

	var envelope ollamaEmbedResponse
	if err := postJSON(
		ctx,
		call.Client,
		call.Provider.ResolvedBaseURL()+ollamaEmbedPath,
		bearerHeaders(call.APIKey),
		body,
		&envelope,
	); err != nil {
		return nil, err
	}

	if err := checkEmbeddings(call, envelope.Embeddings); err != nil {
		return nil, err
	}

	return &EmbedResponse{
		Vectors:         envelope.Embeddings,
		ModelIdentifier: stringutils.FirstNonEmpty(envelope.Model, call.Provider.Model),
		InputTokens:     envelope.PromptEvalCount,
	}, nil
}
