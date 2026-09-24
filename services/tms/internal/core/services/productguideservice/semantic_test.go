package productguideservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	rateMatricesPath = "/billing/configuration-files/rate-matrices"
	shipmentsPath    = "/shipment-management/shipments"
	carriersPath     = "/dispatch/carrier-sourcing"
)

type stubVectorizer struct {
	vector   serviceports.QueryVector
	requests []serviceports.QueryVectorRequest
}

func (v *stubVectorizer) Vectorize(
	_ context.Context,
	req *serviceports.QueryVectorRequest,
) (serviceports.QueryVector, error) {
	v.requests = append(v.requests, *req)

	return v.vector, nil
}

func (v *stubVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: v.vector.Available}, nil
}

type stubVectors struct {
	similarity map[string]float64
	requests   []serviceports.CatalogSimilarityRequest
}

func (s *stubVectors) Similarities(
	_ context.Context,
	req *serviceports.CatalogSimilarityRequest,
) (serviceports.CatalogSimilarities, error) {
	s.requests = append(s.requests, *req)

	return serviceports.CatalogSimilarities{Available: true, ByKey: s.similarity}, nil
}

func guideQueryVector() serviceports.QueryVector {
	vector := make([]float32, airetrieval.Dimensions768)
	vector[5] = 1

	return serviceports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   "localhost/nomic-embed-text@768",
		Dimensions: airetrieval.Dimensions768,
	}
}

func semanticHarness(
	t *testing.T,
	query serviceports.QueryVector,
	similarity map[string]float64,
) (*harness, *stubVectorizer, *stubVectors) {
	t.Helper()

	h := newHarness(t, "rate_matrix:read", "shipment:read")
	vectorizer := &stubVectorizer{vector: query}
	vectors := &stubVectors{similarity: similarity}
	h.service.vectorizer = vectorizer
	h.service.vectors = vectors

	return h, vectorizer, vectors
}

func tenantSearch(
	t *testing.T,
	h *harness,
	query string,
	attribution serviceports.AIUsageAttribution,
) []serviceports.ProductGuideMatch {
	t.Helper()

	requester := actor()
	matches, err := h.service.Search(t.Context(), &serviceports.ProductGuideSearchRequest{
		Actor:       requester,
		TenantInfo:  requester.TenantInfo(),
		Query:       query,
		Attribution: attribution,
	})
	require.NoError(t, err)

	return matches
}

func paths(matches []serviceports.ProductGuideMatch) []string {
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.Page.Path)
	}

	return out
}

func TestSearch_KeywordAnswersAreUnchangedWithoutAVector(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"how do I add a rate matrix",
		"where do I enter a new load",
		"quantum banana",
		"",
	} {
		keyword := newHarness(t, "rate_matrix:read", "shipment:read")
		unavailable, vectorizer, vectors := semanticHarness(t,
			serviceports.UnavailableQueryVector(airetrieval.UnavailableReasonNoProvider),
			map[string]float64{carriersPath: 0.99},
		)

		want := tenantSearch(t, keyword, query, serviceports.AIUsageAttribution{})
		got := tenantSearch(t, unavailable, query, serviceports.AIUsageAttribution{})

		assert.Equal(t, want, got, "query %q", query)
		assert.Empty(t, vectors.requests, "an unavailable vector is never compared")
		if query == "" {
			assert.Empty(t, vectorizer.requests, "a blank question is not embedded")
		}
	}
}

func TestSearch_FindsAPageByMeaning(t *testing.T) {
	t.Parallel()

	attribution := serviceports.AIUsageAttribution{
		UserID:            pulid.MustNew("usr_"),
		AgentDefinitionID: pulid.MustNew("agd_"),
	}
	h, vectorizer, vectors := semanticHarness(t, guideQueryVector(), map[string]float64{
		carriersPath:     0.74,
		shipmentsPath:    0.41,
		rateMatricesPath: 0.2,
	})

	matches := tenantSearch(t, h, "who can haul this for us", attribution)

	assert.Equal(t, []string{carriersPath}, paths(matches),
		"only the page close in meaning clears the floor")
	require.Len(t, vectorizer.requests, 1)
	assert.Equal(t, "who can haul this for us", vectorizer.requests[0].Text)
	assert.Equal(t, attribution, vectorizer.requests[0].Attribution)
	require.Len(t, vectors.requests, 1)
	assert.Equal(t, airetrieval.CatalogCorpusProductGuide, vectors.requests[0].Corpus)
	assert.Len(t, vectors.requests[0].Items, 3)
}

func TestSearch_FusesWordsAndMeaning(t *testing.T) {
	t.Parallel()

	h, _, _ := semanticHarness(t, guideQueryVector(), map[string]float64{
		carriersPath:     0.9,
		rateMatricesPath: 0.3,
	})

	matches := tenantSearch(t, h, "how do I add a rate matrix", serviceports.AIUsageAttribution{})

	require.Len(t, matches, 2)
	assert.Equal(t, rateMatricesPath, matches[0].Page.Path,
		"the page both lists found leads")
	require.NotNil(t, matches[0].Task)
	assert.Equal(t, "Add a rate matrix", matches[0].Task.Title)
	assert.Equal(t, carriersPath, matches[1].Page.Path)
	assert.Nil(t, matches[1].Task, "a page found by meaning alone names no task it did not match")
}

func TestSearch_NonsenseStillFindsNothing(t *testing.T) {
	t.Parallel()

	h, _, _ := semanticHarness(t, guideQueryVector(), map[string]float64{
		carriersPath:     serviceports.DefaultCatalogSimilarityFloor - 0.01,
		shipmentsPath:    0.3,
		rateMatricesPath: 0.1,
	})

	assert.Empty(t, tenantSearch(t, h, "quantum banana", serviceports.AIUsageAttribution{}))
}

func TestSearch_OnePageQuestionIgnoresMeaning(t *testing.T) {
	t.Parallel()

	h, vectorizer, _ := semanticHarness(t, guideQueryVector(), map[string]float64{
		carriersPath: 0.99,
	})
	requester := actor()
	matches, err := h.service.Search(t.Context(), &serviceports.ProductGuideSearchRequest{
		Actor:      requester,
		TenantInfo: requester.TenantInfo(),
		Query:      "retire it",
		Page:       rateMatricesPath,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{rateMatricesPath}, paths(matches))
	assert.Empty(t, vectorizer.requests)
}

func TestCatalogItems_KeyEachPageByItsPath(t *testing.T) {
	t.Parallel()

	catalog, err := productguide.Load([]byte(fixture))
	require.NoError(t, err)

	items := CatalogItems(catalog)
	require.Len(t, items, len(catalog.Pages))
	assert.Equal(t, rateMatricesPath, items[0].Key)
	assert.Equal(t,
		"Rate matrices (Billing › Configuration files › Rate matrices)\n"+
			"Published tariffs as grids.\nRates by lane, zone and weight break.\n"+
			"Also called: tariff\nPeople do here: Add a rate matrix; Retire a rate matrix",
		items[0].Text,
	)
	assert.Len(t, items[0].ContentHash, airetrieval.ContentHashChars)
}

func TestNewService_WiresTheVectorsItIsGiven(t *testing.T) {
	t.Parallel()

	catalog, err := productguide.Load([]byte(fixture))
	require.NoError(t, err)
	vectorizer := &stubVectorizer{}
	vectors := &stubVectors{}

	service := newService(Params{
		Logger:         zap.NewNop(),
		Vectorizer:     vectorizer,
		CatalogVectors: vectors,
	}, catalog)

	assert.Same(t, vectorizer, service.vectorizer)
	assert.Same(t, vectors, service.vectors)
	assert.Len(t, service.items, len(catalog.Pages))
}
