package documentrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type storedDocuments struct {
	documentports.DocumentRepository
	byID     map[pulid.ID]*document.Document
	versions []*document.Document
}

func (s *storedDocuments) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return s.byID[req.ID], nil
}

func (s *storedDocuments) GetByIDs(
	_ context.Context,
	req documentports.GetDocumentsByIDsRequest,
) ([]*document.Document, error) {
	out := make([]*document.Document, 0, len(req.IDs))
	for _, id := range req.IDs {
		out = append(out, s.byID[id])
	}
	return out, nil
}

func (s *storedDocuments) ListVersions(
	context.Context,
	documentports.ListDocumentVersionsRequest,
) ([]*document.Document, error) {
	return s.versions, nil
}

func (s *storedDocuments) Delete(context.Context, documentports.DeleteDocumentRequest) error {
	return nil
}

func (s *storedDocuments) BulkDelete(
	context.Context,
	documentports.BulkDeleteDocumentRequest,
) error {
	return nil
}

func (s *storedDocuments) PromoteVersion(
	context.Context,
	*documentports.PromoteDocumentVersionRequest,
) error {
	return nil
}

func (s *storedDocuments) Update(
	_ context.Context,
	entity *document.Document,
) (*document.Document, error) {
	return entity, nil
}

type heard struct {
	changes []*documentports.DocumentChange
}

func (h *heard) DocumentsChanged(_ context.Context, change *documentports.DocumentChange) {
	h.changes = append(h.changes, change)
}

var testTenant = pagination.TenantInfo{
	OrgID: pulid.ID("org_01TESTOBSERVEDAAAAAAAAAAAAA"),
	BuID:  pulid.ID("bu_01TESTOBSERVEDAAAAAAAAAAAAA"),
}

func shipmentDocument(shipment string, version int64, current bool) *document.Document {
	return &document.Document{
		ID:               pulid.MustNew("doc_"),
		OrganizationID:   testTenant.OrgID,
		BusinessUnitID:   testTenant.BuID,
		ResourceType:     "shipment",
		ResourceID:       shipment,
		VersionNumber:    version,
		IsCurrentVersion: current,
		Status:           document.StatusActive,
	}
}

func observedOver(
	t *testing.T,
	stored *storedDocuments,
) (documentports.DocumentRepository, *heard) {
	t.Helper()
	listener := &heard{}
	repo := Observe(ObservedParams{
		Repo:      stored,
		Observers: []documentports.DocumentChangeObserver{listener},
		Logger:    zap.NewNop(),
	})
	return repo, listener
}

func TestDeletingADocumentRetiresItFromItsShipment(t *testing.T) {
	doc := shipmentDocument("shp_1", 1, true)
	repo, listener := observedOver(t, &storedDocuments{
		byID: map[pulid.ID]*document.Document{doc.ID: doc},
	})

	require.NoError(t, repo.Delete(t.Context(), documentports.DeleteDocumentRequest{
		ID:         doc.ID,
		TenantInfo: testTenant,
	}))

	require.Len(t, listener.changes, 1)
	assert.Equal(t, []string{"shp_1"}, listener.changes[0].ResourceIDs)
	assert.Equal(t, []documentports.RetiredDocument{{ID: doc.ID, ResourceID: "shp_1"}},
		listener.changes[0].Retired)
}

func TestDeletingManyDocumentsRetiresEachOne(t *testing.T) {
	first, second := shipmentDocument("shp_1", 1, true), shipmentDocument("shp_2", 1, true)
	repo, listener := observedOver(t, &storedDocuments{
		byID: map[pulid.ID]*document.Document{first.ID: first, second.ID: second},
	})

	require.NoError(t, repo.BulkDelete(t.Context(), documentports.BulkDeleteDocumentRequest{
		IDs:        []pulid.ID{first.ID, second.ID},
		TenantInfo: testTenant,
	}))

	require.Len(t, listener.changes, 1)
	assert.ElementsMatch(t, []documentports.RetiredDocument{
		{ID: first.ID, ResourceID: "shp_1"},
		{ID: second.ID, ResourceID: "shp_2"},
	}, listener.changes[0].Retired)
}

func TestANewVersionRetiresTheOneItReplaces(t *testing.T) {
	newer := shipmentDocument("shp_1", 2, false)
	older := shipmentDocument("shp_1", 1, true)
	repo, listener := observedOver(t, &storedDocuments{
		versions: []*document.Document{newer, older},
	})

	require.NoError(
		t,
		repo.PromoteVersion(t.Context(), &documentports.PromoteDocumentVersionRequest{
			LineageID:         pulid.MustNew("dln_"),
			CurrentDocumentID: newer.ID,
			TenantInfo:        testTenant,
		}),
	)

	require.Len(t, listener.changes, 1)
	assert.Equal(t, []documentports.RetiredDocument{
		{ID: older.ID, ResourceID: "shp_1", ReplacedBy: newer.ID},
	}, listener.changes[0].Retired)
}

func TestRejectingADocumentRetiresItAndAnOrdinaryEditDoesNot(t *testing.T) {
	doc := shipmentDocument("shp_1", 1, true)
	repo, listener := observedOver(t, &storedDocuments{})

	_, err := repo.Update(t.Context(), doc)
	require.NoError(t, err)
	rejected := *doc
	rejected.Status = document.StatusRejected
	_, err = repo.Update(t.Context(), &rejected)
	require.NoError(t, err)

	require.Len(t, listener.changes, 2)
	assert.Empty(t, listener.changes[0].Retired)
	assert.Equal(t, []documentports.RetiredDocument{{ID: doc.ID, ResourceID: "shp_1"}},
		listener.changes[1].Retired)
}
