package modeladapter

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
)

const (
	nomicDocumentPrefix = "search_document: "
	nomicQueryPrefix    = "search_query: "
	voyageDocumentType  = "document"
	voyageQueryType     = "query"
)

type EmbedCall struct {
	Provider *aiprovider.Provider
	APIKey   string
	Client   *http.Client
	Purpose  aiusageports.EmbeddingPurpose
	Inputs   []string
}

type EmbedResponse struct {
	Vectors         [][]float32
	ModelIdentifier string
	InputTokens     int
}

type Embedder interface {
	Embed(ctx context.Context, call *EmbedCall) (*EmbedResponse, error)
}

func (c *EmbedCall) Dimensions() int {
	return c.Provider.ResolvedEmbeddingDimensions()
}

func (c *EmbedCall) InputStyle() aiprovider.EmbeddingInputStyle {
	return c.Provider.ResolvedEmbeddingInputStyle()
}

func (c *EmbedCall) WireInputs() []string {
	if c.InputStyle() != aiprovider.EmbeddingInputStyleNomicPrefix {
		return c.Inputs
	}

	prefix := nomicDocumentPrefix
	if c.Purpose == aiusageports.EmbeddingPurposeQuery {
		prefix = nomicQueryPrefix
	}

	prefixed := make([]string, len(c.Inputs))
	for idx, input := range c.Inputs {
		prefixed[idx] = prefix + input
	}

	return prefixed
}

func (c *EmbedCall) VoyageInputType() string {
	if c.Purpose == aiusageports.EmbeddingPurposeQuery {
		return voyageQueryType
	}

	return voyageDocumentType
}

func (c *EmbedCall) ExtraBodyNames(key string) bool {
	if c.Provider == nil || c.Provider.ExtraBody == nil {
		return false
	}

	_, ok := c.Provider.ExtraBody[key]

	return ok
}
