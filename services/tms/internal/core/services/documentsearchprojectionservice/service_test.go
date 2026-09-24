package documentsearchprojectionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeProjectionRepo struct {
	repositories.DocumentSearchProjectionRepository

	err      error
	upserted []*documentsearchprojection.Projection
	deleted  []pulid.ID
}

func (f *fakeProjectionRepo) Upsert(
	_ context.Context,
	entity *documentsearchprojection.Projection,
) (*documentsearchprojection.Projection, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.upserted = append(f.upserted, entity)

	return entity, nil
}

func (f *fakeProjectionRepo) Delete(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, documentID)

	return nil
}

type fakeIndexer struct {
	err     error
	stale   []pulid.ID
	deleted []pulid.ID
	types   []airetrieval.SourceType
}

func (f *fakeIndexer) MarkStale(
	_ context.Context,
	_ pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	ids ...pulid.ID,
) error {
	f.stale = append(f.stale, ids...)
	f.types = append(f.types, sourceType)

	return f.err
}

func (f *fakeIndexer) DeleteSource(
	_ context.Context,
	_ pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	id pulid.ID,
) error {
	f.deleted = append(f.deleted, id)
	f.types = append(f.types, sourceType)

	return f.err
}

func (f *fakeIndexer) Reindex(context.Context, pagination.TenantInfo, airetrieval.SourceType) error {
	return nil
}

func newTestService(repo *fakeProjectionRepo, indexer *fakeIndexer) *Service {
	params := Params{Logger: zap.NewNop(), Repo: repo}
	if indexer != nil {
		params.Indexer = indexer
	}

	return New(params).(*Service)
}

func testDocument() *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		FileName:       "pod.pdf",
		OriginalName:   "pod.pdf",
		ResourceType:   "shipment",
		ResourceID:     "shp_1",
		Status:         document.StatusActive,
	}
}

func TestUpsertQueuesTheDocumentForSemanticIndexing(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	indexer := &fakeIndexer{}
	doc := testDocument()

	require.NoError(t, newTestService(repo, indexer).Upsert(t.Context(), doc, "signed POD"))
	assert.Len(t, repo.upserted, 1)
	assert.Equal(t, []pulid.ID{doc.ID}, indexer.stale)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeDocument}, indexer.types)
}

func TestDeleteDropsTheDocumentFromTheSemanticIndex(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	indexer := &fakeIndexer{}
	id := pulid.MustNew("doc_")

	require.NoError(t, newTestService(repo, indexer).Delete(t.Context(), id, pagination.TenantInfo{}))
	assert.Equal(t, []pulid.ID{id}, repo.deleted)
	assert.Equal(t, []pulid.ID{id}, indexer.deleted)
}

func TestAnIndexingFailureNeverFailsTheProjection(t *testing.T) {
	t.Parallel()

	indexer := &fakeIndexer{err: errors.New("outbox unavailable")}
	svc := newTestService(&fakeProjectionRepo{}, indexer)

	require.NoError(t, svc.Upsert(t.Context(), testDocument(), ""))
	require.NoError(t, svc.Delete(t.Context(), pulid.MustNew("doc_"), pagination.TenantInfo{}))
}

func TestAFailedProjectionWriteIsNotIndexed(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{err: errors.New("database is down")}
	indexer := &fakeIndexer{}
	svc := newTestService(repo, indexer)

	require.Error(t, svc.Upsert(t.Context(), testDocument(), ""))
	assert.Empty(t, indexer.stale)
}

func TestWithoutAnIndexerTheProjectionStillWorks(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	require.NoError(t, newTestService(repo, nil).Upsert(t.Context(), testDocument(), ""))
	assert.Len(t, repo.upserted, 1)
}
