package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type failingVectorizer struct{ err error }

func (v failingVectorizer) Vectorize(
	context.Context,
	*retrievalports.QueryVectorRequest,
) (retrievalports.QueryVector, error) {
	return retrievalports.QueryVector{}, v.err
}

func (v failingVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: true}, nil
}

type failingSimilarities struct{ err error }

func (f failingSimilarities) Similarities(
	context.Context,
	*retrievalports.CatalogSimilarityRequest,
) (retrievalports.CatalogSimilarities, error) {
	return retrievalports.CatalogSimilarities{}, f.err
}

func observedRuntime(t *testing.T, err error) (*Service, *observer.ObservedLogs) {
	t.Helper()

	service, _ := wideRuntime(t)
	core, logs := observer.New(zap.WarnLevel)
	service.logger = zap.New(core)
	service.vectorizer = failingVectorizer{err: err}
	service.vectors = failingSimilarities{err: err}

	return service, logs
}

func embedAndCompare(t *testing.T, service *Service) {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	query := service.vectorize(t.Context(), &retrievalports.QueryVectorRequest{
		TenantInfo: tenant,
		Text:       "which invoices are past due",
	})
	assert.False(t, query.Usable())
	assert.Nil(t, service.toolSemantic(t.Context(), tenant, usableQuery()))
}

func TestSemantic_StaysQuietWhenThePersonTypedOn(t *testing.T) {
	t.Parallel()

	service, logs := observedRuntime(t, context.Canceled)

	embedAndCompare(t, service)

	assert.Zero(t, logs.Len(), "a cancelled request is not a failure worth a warning")
}

func TestSemantic_WarnsWhenEmbeddingFails(t *testing.T) {
	t.Parallel()

	service, logs := observedRuntime(t, errors.New("provider unavailable"))

	embedAndCompare(t, service)

	assert.Equal(t, 2, logs.Len(), "the embedding and the catalog comparison each warn once")
}
