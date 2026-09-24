package modeladapter

import (
	"context"
	"fmt"
	"math"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	Purpose  serviceports.EmbeddingPurpose
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

func (c *EmbedCall) dimensions() int {
	return c.Provider.ResolvedEmbeddingDimensions()
}

func (c *EmbedCall) inputStyle() aiprovider.EmbeddingInputStyle {
	return c.Provider.ResolvedEmbeddingInputStyle()
}

func (c *EmbedCall) wireInputs() []string {
	if c.inputStyle() != aiprovider.EmbeddingInputStyleNomicPrefix {
		return c.Inputs
	}

	prefix := nomicDocumentPrefix
	if c.Purpose == serviceports.EmbeddingPurposeQuery {
		prefix = nomicQueryPrefix
	}

	prefixed := make([]string, len(c.Inputs))
	for idx, input := range c.Inputs {
		prefixed[idx] = prefix + input
	}

	return prefixed
}

func (c *EmbedCall) voyageInputType() string {
	if c.Purpose == serviceports.EmbeddingPurposeQuery {
		return voyageQueryType
	}

	return voyageDocumentType
}

func (c *EmbedCall) extraBodyNames(key string) bool {
	if c.Provider == nil || c.Provider.ExtraBody == nil {
		return false
	}

	_, ok := c.Provider.ExtraBody[key]

	return ok
}

func checkVectorCount(call *EmbedCall, got int) error {
	if got == len(call.Inputs) {
		return nil
	}

	return errortypes.NewBusinessError(
		"Embedding provider {0} returned {1} vectors for {2} texts",
		call.Provider.Name,
		got,
		len(call.Inputs),
	).WithInternal(serviceports.ErrEmbeddingResponseInvalid)
}

func checkEmbeddings(call *EmbedCall, vectors [][]float32) error {
	if err := checkVectorCount(call, len(vectors)); err != nil {
		return err
	}

	want := call.dimensions()
	for idx, vector := range vectors {
		if len(vector) != want {
			return errortypes.NewBusinessError(
				"Embedding provider {0} returned {1} dimensions, but it is configured for {2}. Set the dimension the model actually returns, or choose a model that supports {2}",
				call.Provider.Name,
				len(vector),
				want,
			).WithInternal(serviceports.ErrEmbeddingDimensionMismatch)
		}
		if err := checkFinite(vector); err != nil {
			return errortypes.NewBusinessError(
				"Embedding provider {0} returned an unusable response",
				call.Provider.Name,
			).WithInternal(fmt.Errorf("vector %d: %w: %w", idx, err, serviceports.ErrEmbeddingResponseInvalid))
		}
	}

	return nil
}

func checkFinite(vector []float32) error {
	for pos, value := range vector {
		f := float64(value)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("component %d is not a finite number", pos)
		}
	}

	return nil
}

func (r *Registry) Embedder(kind aiprovider.Kind) (Embedder, error) {
	embedder, ok := r.embedders[kind]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"The {0} protocol has no embedding endpoint", string(kind),
		).WithInternal(serviceports.ErrNoProviderConfigured)
	}

	return embedder, nil
}
