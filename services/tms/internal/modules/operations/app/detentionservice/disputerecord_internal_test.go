package detentionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/detentiontemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/generateddocument"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttypeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type listedNotices struct {
	stubNoticeRepo
	notices []*detention.DetentionNotice
}

func (r *listedNotices) List(
	context.Context,
	*detentionports.ListDetentionNoticesRequest,
) ([]*detention.DetentionNotice, error) {
	return r.notices, nil
}

type storeSpy struct {
	stored []*documentports.StoreDocumentRequest
}

func (s *storeSpy) Store(
	_ context.Context,
	req *documentports.StoreDocumentRequest,
) (*document.Document, error) {
	s.stored = append(s.stored, req)

	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: req.FileName,
		FileSize:     int64(len(req.Content)),
		StoragePath:  "org/shipment/" + req.FileName,
	}, nil
}

type disputeRig struct {
	svc        *Service
	occurrence *detention.DetentionOccurrence
	tenant     pagination.TenantInfo
	notices    *listedNotices
	documents  *documentportsmock.MockDocumentRepository
	store      *storeSpy
	templates  *documenttemplateportsmock.MockDocumentTemplateResolver
	types      *documenttypeportsmock.MockDocumentTypeRepository
}

func newDisputeRig(t *testing.T) *disputeRig {
	t.Helper()

	arrived := int64(1_791_000_000)
	departed := arrived + 5*3600
	occurrence := pdfOccurrence()
	occurrence.ShipmentProNumber = "SEED-DET-008"
	occurrence.LocationName = "Louisville DC"
	occurrence.ArrivedAt = &arrived
	occurrence.DepartedAt = &departed
	occurrence.BillableAmount = decimal.RequireFromString("1200")
	occurrence.Evidence = []*detention.DetentionEvidence{
		{Kind: detention.EvidenceKindArrival, Source: detention.EvidenceSourceGeofence},
	}

	organizations := tenantportsmock.NewMockOrganizationRepository(t)
	organizations.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Trenova Freight", Timezone: "America/Chicago"}, nil).
		Maybe()

	rig := &disputeRig{
		occurrence: occurrence,
		tenant: pagination.TenantInfo{
			OrgID: occurrence.OrganizationID,
			BuID:  occurrence.BusinessUnitID,
		},
		notices:   &listedNotices{},
		documents: documentportsmock.NewMockDocumentRepository(t),
		store:     &storeSpy{},
		templates: documenttemplateportsmock.NewMockDocumentTemplateResolver(t),
		types:     documenttypeportsmock.NewMockDocumentTypeRepository(t),
	}
	rig.svc = &Service{
		l:                zap.NewNop(),
		noticeRepo:       rig.notices,
		organizations:    organizations,
		documentTypeRepo: rig.types,
		templates:        rig.templates,
		contextBuilder:   &ContextBuilder{orgRepo: organizations},
		documentStore:    rig.store,
		documents:        rig.documents,
	}

	return rig
}

func (r *disputeRig) record(t *testing.T) *DisputeRecord {
	t.Helper()

	record, err := r.svc.DisputeRecord(t.Context(), &DisputeRecordParams{
		TenantInfo: r.tenant,
		Occurrence: r.occurrence,
	})
	require.NoError(t, err)

	return record
}

func TestDisputeRecordPrefersTheFinalNoticeThatWasSent(t *testing.T) {
	t.Parallel()

	rig := newDisputeRig(t)
	warning := pulid.MustNew("doc_")
	final := pulid.MustNew("doc_")
	bounced := pulid.MustNew("doc_")
	rig.notices.notices = []*detention.DetentionNotice{
		{Kind: detention.NoticeKindWarning, PDFDocumentID: &warning, CreatedAt: 300},
		{Kind: detention.NoticeKindFinal, PDFDocumentID: &final, CreatedAt: 200},
		{
			Kind:           detention.NoticeKindFinal,
			PDFDocumentID:  &bounced,
			CreatedAt:      400,
			DeliveryStatus: detention.NoticeDeliveryStatusBounced,
		},
	}

	record := rig.record(t)

	require.NotNil(t, record)
	assert.Equal(t, final, record.DocumentID)
	assert.False(t, record.Rendered)
	assert.Empty(t, rig.store.stored)
}

func TestDisputeRecordReusesARecordAlreadyStoredOnTheShipment(t *testing.T) {
	t.Parallel()

	rig := newDisputeRig(t)
	existing := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: disputeRecordFileName(rig.occurrence),
		Status:       document.StatusActive,
		StoragePath:  "org/shipment/record.pdf",
	}
	rig.documents.EXPECT().GetByResourceID(mock.Anything, mock.MatchedBy(
		func(req *documentports.GetDocumentsByResourceRequest) bool {
			return req.ResourceType == "shipment" &&
				req.ResourceID == rig.occurrence.ShipmentID.String()
		},
	)).Return([]*document.Document{existing}, nil).Once()

	record := rig.record(t)

	require.NotNil(t, record)
	assert.Equal(t, existing.ID, record.DocumentID)
	assert.Empty(t, rig.store.stored)
}

func TestDisputeRecordRendersTheFinalNoticeAndStoresItOnTheShipment(t *testing.T) {
	t.Parallel()

	rig := newDisputeRig(t)
	typeID := pulid.MustNew("dt_")
	rig.documents.EXPECT().GetByResourceID(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).Once()
	rig.templates.EXPECT().RenderDocument(mock.Anything, mock.MatchedBy(
		func(req *documenttemplateports.RenderDocumentRequest) bool {
			data, ok := req.Data.(*detentiontemplate.DetentionNoticeContext)

			return ok && req.Kind == documenttemplate.KindDetentionNoticePDF &&
				data.Kind == string(detention.NoticeKindFinal) &&
				data.ArrivalSource == string(detention.EvidenceSourceGeofence) &&
				data.ShipmentProNumber == "SEED-DET-008"
		},
	)).Return(&documenttemplateports.RenderedDocument{
		PDF:    []byte("%PDF-1.7 record"),
		Source: documenttemplate.SourceBuiltIn,
	}, nil).Once()
	rig.types.EXPECT().GetByCode(mock.Anything, documenttypeports.GetDocumentTypeByCodeRequest{
		Code:       documenttype.CodeDetentionNotice,
		TenantInfo: rig.tenant,
	}).Return(&documenttype.DocumentType{ID: typeID}, nil).Once()
	rig.templates.EXPECT().RecordGeneratedDocument(mock.Anything, mock.MatchedBy(
		func(req *documenttemplateports.RecordGeneratedDocumentRequest) bool {
			return req.ReferenceID == rig.occurrence.ID && req.DocumentID != nil
		},
	)).Return(&generateddocument.GeneratedDocument{}, nil).Once()

	record := rig.record(t)

	require.NotNil(t, record)
	assert.True(t, record.Rendered)
	require.Len(t, rig.store.stored, 1)
	stored := rig.store.stored[0]
	assert.Equal(t, "shipment", stored.ResourceType)
	assert.Equal(t, rig.occurrence.ShipmentID.String(), stored.ResourceID)
	assert.Equal(t, typeID, stored.DocumentTypeID)
	assert.Equal(t, []byte("%PDF-1.7 record"), stored.Content)
	assert.Equal(t, permissionports.PrincipalTypeSystem, stored.Actor.PrincipalType)
	assert.Equal(t, disputeRecordFileName(rig.occurrence), record.FileName)
	assert.Contains(t, record.FileName, "SEED-DET-008")
}

func TestDisputeRecordWithoutADocumentStoreFindsNothingToRender(t *testing.T) {
	t.Parallel()

	rig := newDisputeRig(t)
	rig.svc.documentStore = nil
	rig.documents.EXPECT().GetByResourceID(mock.Anything, mock.Anything).
		Return([]*document.Document{}, nil).Once()

	record := rig.record(t)

	assert.Nil(t, record)
}
