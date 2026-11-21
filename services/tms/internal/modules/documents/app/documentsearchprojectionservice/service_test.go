package documentsearchprojectionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeProjectionRepo struct {
	documentsearchprojectionports.DocumentSearchProjectionRepository

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

func (f *fakeIndexer) Reindex(
	context.Context,
	pagination.TenantInfo,
	airetrieval.SourceType,
) error {
	return nil
}

type fakeControls struct {
	tenantports.DocumentControlRepository

	fullText bool
	reads    int
}

func (f *fakeControls) GetOrCreate(
	_ context.Context,
	orgID, buID pulid.ID,
) (*documentcontrol.DocumentControl, error) {
	f.reads++
	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	control.EnableFullTextIndexing = f.fullText

	return control, nil
}

func newTestService(repo *fakeProjectionRepo, indexer *fakeIndexer) *Service {
	return newTestServiceWithControls(repo, indexer, &fakeControls{fullText: true})
}

func newTestServiceWithControls(
	repo *fakeProjectionRepo,
	indexer *fakeIndexer,
	controls *fakeControls,
) *Service {
	return New(Params{
		Logger:   zap.NewNop(),
		Repo:     repo,
		Indexer:  indexer,
		Controls: controls,
	}).(*Service)
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

	require.NoError(
		t,
		newTestService(repo, indexer).Delete(t.Context(), id, pagination.TenantInfo{}),
	)
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

func TestUpsert_LeavesConversationFilesOutOfTheProjection(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	indexer := &fakeIndexer{}
	service := &Service{logger: zap.NewNop(), repo: repo, indexer: indexer}
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		ResourceType: document.ConversationResourceType,
		ResourceID:   pulid.MustNew("athr_").String(),
	}

	require.NoError(t, service.Upsert(t.Context(), doc, "rate confirmation"))

	assert.Empty(t, repo.upserted, "global search has no owner check to protect a private file")
	assert.Equal(t, []pulid.ID{doc.ID}, indexer.stale, "agent retrieval checks the rule itself")
}

func TestUpsert_StoresTheTextWhenFullTextIndexingIsOn(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	svc := newTestServiceWithControls(repo, &fakeIndexer{}, &fakeControls{fullText: true})

	require.NoError(t, svc.Upsert(t.Context(), testDocument(), "signed POD"))

	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "signed POD", repo.upserted[0].ContentText)
}

/*
Every caller that indexes a document, from extraction to promoting a version
or attaching it to a shipment, passes the text it has. Whether that text is
searchable is the organization's choice, so the projection drops it here,
where every caller passes, rather than at each caller.
*/
func TestUpsert_DropsTheTextWhenFullTextIndexingIsOff(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	controls := &fakeControls{fullText: false}
	doc := testDocument()
	svc := newTestServiceWithControls(repo, &fakeIndexer{}, controls)

	require.NoError(t, svc.Upsert(t.Context(), doc, "signed POD"))

	require.Len(t, repo.upserted, 1)
	assert.Empty(t, repo.upserted[0].ContentText)
	assert.Equal(t, doc.ID, repo.upserted[0].ID)
	assert.Equal(t, 1, controls.reads)
}

func TestUpsert_ReadsNoSettingsWhenThereIsNoText(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectionRepo{}
	controls := &fakeControls{fullText: false}
	svc := newTestServiceWithControls(repo, &fakeIndexer{}, controls)

	require.NoError(t, svc.Upsert(t.Context(), testDocument(), ""))

	require.Len(t, repo.upserted, 1)
	assert.Zero(t, controls.reads)
}
