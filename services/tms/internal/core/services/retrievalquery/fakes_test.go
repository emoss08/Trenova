package retrievalquery

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/pgvector/pgvector-go"
)

const testModelKey = "localhost:11434/nomic-embed-text@768"

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func activeSettings(tenant pagination.TenantInfo) *airetrieval.Settings {
	settings := airetrieval.DefaultSettings(tenant.OrgID, tenant.BuID)
	settings.ActiveModelKey = testModelKey
	settings.Dimensions = airetrieval.Dimensions768

	return settings
}

type fakeRepository struct {
	repositories.AIRetrievalRepository

	mu            sync.Mutex
	availability  airetrieval.Availability
	availErr      error
	settings      *airetrieval.Settings
	settingsErr   error
	stored        map[string]*airetrieval.CatalogEmbedding
	getRequests   []repositories.GetCatalogEmbeddingsRequest
	putRequests   []repositories.PutCatalogEmbeddingsRequest
	pruneRequests []repositories.PruneCatalogEmbeddingsRequest
}

func newFakeRepository(settings *airetrieval.Settings) *fakeRepository {
	return &fakeRepository{
		availability: airetrieval.Availability{Available: true, ExtensionInstalled: true},
		settings:     settings,
		stored:       make(map[string]*airetrieval.CatalogEmbedding),
	}
}

func storedKey(modelKey string, corpus airetrieval.CatalogCorpus, item, hash string) string {
	return modelKey + "|" + corpus.String() + "|" + item + "|" + hash
}

func (r *fakeRepository) VectorAvailability(context.Context) (airetrieval.Availability, error) {
	return r.availability, r.availErr
}

func (r *fakeRepository) GetSettings(
	context.Context,
	pagination.TenantInfo,
) (*airetrieval.Settings, error) {
	return r.settings, r.settingsErr
}

func (r *fakeRepository) GetCatalogEmbeddings(
	_ context.Context,
	req repositories.GetCatalogEmbeddingsRequest,
) ([]*airetrieval.CatalogEmbedding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.getRequests = append(r.getRequests, req)
	out := make([]*airetrieval.CatalogEmbedding, 0, len(req.Items))
	for _, ref := range req.Items {
		key := storedKey(req.ModelKey, req.Corpus, ref.ItemKey, ref.ContentHash)
		if row, ok := r.stored[key]; ok {
			out = append(out, row)
		}
	}

	return out, nil
}

func (r *fakeRepository) PutCatalogEmbeddings(
	_ context.Context,
	req repositories.PutCatalogEmbeddingsRequest,
) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.putRequests = append(r.putRequests, req)
	for _, item := range req.Items {
		r.stored[storedKey(req.ModelKey, req.Corpus, item.ItemKey, item.ContentHash)] =
			&airetrieval.CatalogEmbedding{
				ModelKey:    req.ModelKey,
				Corpus:      req.Corpus,
				ItemKey:     item.ItemKey,
				ContentHash: item.ContentHash,
				Dimensions:  req.Dimensions,
				Embedding:   pgvector.NewHalfVector(item.Vector),
			}
	}

	return len(req.Items), nil
}

func (r *fakeRepository) PruneCatalogEmbeddings(
	_ context.Context,
	req repositories.PruneCatalogEmbeddingsRequest,
) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneRequests = append(r.pruneRequests, req)

	return 0, nil
}

type fakeEmbeddings struct {
	mu         sync.Mutex
	requests   []serviceports.EmbedRequest
	err        error
	configured error
	vectorFor  func(input string) []float32
	modelKey   string
}

func (e *fakeEmbeddings) Embed(
	_ context.Context,
	req *serviceports.EmbedRequest,
) (serviceports.EmbedResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.requests = append(e.requests, *req)
	if e.err != nil {
		return serviceports.EmbedResult{}, e.err
	}

	vectors := make([][]float32, 0, len(req.Inputs))
	for _, input := range req.Inputs {
		vectors = append(vectors, e.vectorFor(input))
	}

	modelKey := e.modelKey
	if modelKey == "" {
		modelKey = req.ModelKey
	}

	return serviceports.EmbedResult{
		Vectors:    vectors,
		ModelKey:   modelKey,
		Dimensions: airetrieval.Dimensions768,
	}, nil
}

func (e *fakeEmbeddings) ConfiguredModelKey(
	context.Context,
	pagination.TenantInfo,
) (string, error) {
	if e.configured != nil {
		return "", e.configured
	}

	return testModelKey, nil
}

func (e *fakeEmbeddings) calls() []serviceports.EmbedRequest {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]serviceports.EmbedRequest(nil), e.requests...)
}

func axis(index int) []float32 {
	vector := make([]float32, airetrieval.Dimensions768)
	vector[index%airetrieval.Dimensions768] = 1

	return vector
}

func blend(weights map[int]float32) []float32 {
	vector := make([]float32, airetrieval.Dimensions768)
	for index, weight := range weights {
		vector[index] = weight
	}

	return vector
}
