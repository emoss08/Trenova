package services

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const DefaultQueryEmbeddingTimeout = 1500 * time.Millisecond

var (
	ErrEmbeddingDimensionMismatch = errors.New(
		"embedding provider returned vectors of a different dimension than configured",
	)
	ErrEmbeddingResponseInvalid = errors.New("embedding provider returned an unusable response")
)

type EmbeddingPurpose string

const (
	EmbeddingPurposeDocument = EmbeddingPurpose("Document")
	EmbeddingPurposeQuery    = EmbeddingPurpose("Query")
)

func (p EmbeddingPurpose) IsValid() bool {
	switch p {
	case EmbeddingPurposeDocument, EmbeddingPurposeQuery:
		return true
	default:
		return false
	}
}

func (p EmbeddingPurpose) DefaultSurface() aiusage.Surface {
	if p == EmbeddingPurposeQuery {
		return aiusage.SurfaceRetrieval
	}

	return aiusage.SurfaceIndexing
}

type EmbedRequest struct {
	TenantInfo   pagination.TenantInfo
	Purpose      EmbeddingPurpose
	Inputs       []string
	ModelKey     string
	QueryTimeout time.Duration
	Surface      aiusage.Surface
	Attribution  AIUsageAttribution
}

func (r *EmbedRequest) ResolvedSurface() aiusage.Surface {
	if r.Surface != "" {
		return r.Surface
	}

	return r.Purpose.DefaultSurface()
}

func (r *EmbedRequest) ResolvedQueryTimeout() time.Duration {
	if r.QueryTimeout > 0 {
		return r.QueryTimeout
	}

	return DefaultQueryEmbeddingTimeout
}

type EmbedResult struct {
	Vectors      [][]float32
	ModelKey     string
	Dimensions   int
	InputTokens  int
	CostUSD      *decimal.Decimal
	ProviderID   pulid.ID
	ProviderKind aiprovider.Kind
	Model        string
	LatencyMs    int64
}

type EmbeddingService interface {
	Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error)
	ConfiguredModelKey(ctx context.Context, tenant pagination.TenantInfo) (string, error)
}
