package runtimetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type FixedVectorizer struct {
	Vector retrievalports.QueryVector
	Calls  []retrievalports.QueryVectorRequest
}

func (v *FixedVectorizer) Vectorize(
	_ context.Context,
	req *retrievalports.QueryVectorRequest,
) (retrievalports.QueryVector, error) {
	v.Calls = append(v.Calls, *req)

	return v.Vector, nil
}

func (v *FixedVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: v.Vector.Available}, nil
}

type FixedSimilarities struct {
	ByKey    map[string]float64
	Requests []retrievalports.CatalogSimilarityRequest
}

func (f *FixedSimilarities) Similarities(
	_ context.Context,
	req *retrievalports.CatalogSimilarityRequest,
) (retrievalports.CatalogSimilarities, error) {
	f.Requests = append(f.Requests, *req)

	return retrievalports.CatalogSimilarities{Available: true, ByKey: f.ByKey}, nil
}

func UsableQuery() retrievalports.QueryVector {
	vector := make([]float32, airetrieval.Dimensions768)
	vector[0] = 1
	vector[767] = 0.5

	return retrievalports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   "localhost/nomic-embed-text@768",
		Dimensions: airetrieval.Dimensions768,
	}
}
