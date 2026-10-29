package documentextractionjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttypeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func signatureNow() int64 {
	return time.Date(2026, time.October, 16, 15, 0, 0, 0, time.UTC).Unix()
}

func TestReadSignatureMarksAConfidentSignatureSigned(t *testing.T) {
	t.Parallel()

	finding := readSignature(&documentaiextractionports.AISignatureReading{
		Status:     "Signed",
		Confidence: 0.93,
		SignedDate: "2026-10-14",
	}, signatureNow())

	assert.Equal(t, document.SignatureStatusSigned, finding.Status)
	require.NotNil(t, finding.SignedAt)
	assert.Equal(
		t,
		time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC).Unix(),
		*finding.SignedAt,
	)
}

func TestReadSignatureMarksAConfidentBlankLineUnsigned(t *testing.T) {
	t.Parallel()

	finding := readSignature(&documentaiextractionports.AISignatureReading{
		Status:     "unsigned",
		Confidence: 0.9,
		SignedDate: "2026-10-14",
	}, signatureNow())

	assert.Equal(t, document.SignatureStatusUnsigned, finding.Status)
	assert.Nil(t, finding.SignedAt)
}

func TestReadSignatureLeavesALowConfidenceReadingUnknown(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"Signed", "Unsigned"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			finding := readSignature(&documentaiextractionports.AISignatureReading{
				Status:     status,
				Confidence: documentanalysis.ReviewReadyConfidenceThreshold - 0.01,
				SignedDate: "2026-10-14",
			}, signatureNow())

			assert.Equal(t, document.SignatureStatusUnknown, finding.Status)
			assert.Nil(t, finding.SignedAt)
		})
	}
}

func TestReadSignatureLeavesAnUnreadOrUnrecognisedReadingUnknown(t *testing.T) {
	t.Parallel()

	assert.Equal(t, document.SignatureStatusUnknown, readSignature(nil, signatureNow()).Status)
	assert.Equal(t, document.SignatureStatusUnknown, readSignature(
		&documentaiextractionports.AISignatureReading{Status: "Maybe", Confidence: 0.99},
		signatureNow(),
	).Status)
}

func TestReadSignatureDropsASigningDateItCannotTrust(t *testing.T) {
	t.Parallel()

	for name, date := range map[string]string{
		"empty":     "",
		"malformed": "10/14/2026",
		"no such":   "2026-02-30",
		"future":    "2026-10-20",
		"ancient":   "1999-12-31",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			finding := readSignature(&documentaiextractionports.AISignatureReading{
				Status:     "Signed",
				Confidence: 0.95,
				SignedDate: date,
			}, signatureNow())

			assert.Equal(t, document.SignatureStatusSigned, finding.Status)
			assert.Nil(t, finding.SignedAt)
		})
	}
}

type signatureFixture struct {
	activities *Activities
	docRepo    *documentportsmock.MockDocumentRepository
	typeRepo   *documenttypeportsmock.MockDocumentTypeRepository
	realtime   *realtimeportsmock.MockRealtimeService
	doc        *document.Document
	tenant     pagination.TenantInfo
}

func newSignatureFixture(t *testing.T, requiresSignature bool) *signatureFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	typeID := pulid.MustNew("dt_")
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	typeRepo := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}

	typeRepo.EXPECT().
		GetByID(mock.Anything, documenttypeports.GetDocumentTypeByIDRequest{
			ID:         typeID,
			TenantInfo: tenantInfo,
		}).
		Return(&documenttype.DocumentType{
			ID:                typeID,
			Code:              "POD",
			RequiresSignature: requiresSignature,
		}, nil).
		Maybe()

	return &signatureFixture{
		activities: &Activities{
			logger: zap.NewNop(),
			metrics: &metrics.Registry{
				Document: documentmetrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), true),
			},
			documentRepo:     docRepo,
			documentTypeRepo: typeRepo,
			realtime:         realtime,
		},
		docRepo:  docRepo,
		typeRepo: typeRepo,
		realtime: realtime,
		doc: &document.Document{
			ID:              pulid.MustNew("doc_"),
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			ResourceID:      pulid.MustNew("shp_").String(),
			ResourceType:    "shipment",
			DocumentTypeID:  &typeID,
			SignatureStatus: document.SignatureStatusUnknown,
		},
		tenant: tenantInfo,
	}
}

func TestRecordDetectedSignatureWritesAConfidentReadingAndSignalsTheChange(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, true)
	fx.docRepo.EXPECT().
		RecordDetectedSignature(
			mock.Anything,
			mock.MatchedBy(func(req *documentports.RecordDetectedSignatureRequest) bool {
				return req.ID == fx.doc.ID &&
					req.TenantInfo == fx.tenant &&
					req.Status == document.SignatureStatusSigned &&
					req.SignedAt != nil &&
					req.DetectedAt > 0
			}),
		).
		Return(true, nil)
	fx.realtime.EXPECT().
		PublishResourceInvalidation(
			mock.Anything,
			mock.MatchedBy(func(req *realtimeports.PublishResourceInvalidationRequest) bool {
				return req.Resource == permission.ResourceDocument.String() &&
					req.RecordID == fx.doc.ID
			}),
		).
		Return(nil)

	fx.activities.recordDetectedSignature(
		t.Context(),
		fx.doc,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{
			Status:     "Signed",
			Confidence: 0.94,
			SignedDate: "2026-01-05",
		},
	)

	assert.Equal(t, document.SignatureStatusSigned, fx.doc.SignatureStatus)
	assert.Equal(t, document.SignatureSourceExtraction, fx.doc.SignatureSource)
}

func TestRecordDetectedSignatureWritesUnknownForALowConfidenceReading(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, true)
	fx.docRepo.EXPECT().
		RecordDetectedSignature(
			mock.Anything,
			mock.MatchedBy(func(req *documentports.RecordDetectedSignatureRequest) bool {
				return req.Status == document.SignatureStatusUnknown && req.SignedAt == nil
			}),
		).
		Return(false, nil)

	fx.activities.recordDetectedSignature(
		t.Context(),
		fx.doc,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{Status: "Unsigned", Confidence: 0.4},
	)

	assert.Equal(t, document.SignatureStatusUnknown, fx.doc.SignatureStatus)
}

func TestRecordDetectedSignatureNeverOverwritesAPersonsAnswer(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, true)
	fx.doc.SignatureStatus = document.SignatureStatusUnsigned
	fx.doc.SignatureSource = document.SignatureSourcePerson
	fx.doc.SignatureUpdatedByID = pulid.MustNew("usr_")

	fx.activities.recordDetectedSignature(
		t.Context(),
		fx.doc,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{Status: "Signed", Confidence: 0.99},
	)

	fx.docRepo.AssertNotCalled(t, "RecordDetectedSignature", mock.Anything, mock.Anything)
	assert.Equal(t, document.SignatureStatusUnsigned, fx.doc.SignatureStatus)
	assert.Equal(t, document.SignatureSourcePerson, fx.doc.SignatureSource)
}

func TestRecordDetectedSignatureKeepsTheDocumentWhenAPersonAnsweredMeanwhile(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, true)
	fx.docRepo.EXPECT().
		RecordDetectedSignature(mock.Anything, mock.Anything).
		Return(false, nil)

	fx.activities.recordDetectedSignature(
		t.Context(),
		fx.doc,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{Status: "Signed", Confidence: 0.99},
	)

	assert.Equal(t, document.SignatureStatusUnknown, fx.doc.SignatureStatus)
	fx.realtime.AssertNotCalled(t, "PublishResourceInvalidation", mock.Anything, mock.Anything)
}

func TestRecordDetectedSignatureSkipsTypesThatNeedNoSignature(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, false)

	fx.activities.recordDetectedSignature(
		t.Context(),
		fx.doc,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{Status: "Signed", Confidence: 0.99},
	)

	fx.docRepo.AssertNotCalled(t, "RecordDetectedSignature", mock.Anything, mock.Anything)
}

func TestRecordDetectedSignatureSkipsADocumentWithNoReadingOrType(t *testing.T) {
	t.Parallel()

	fx := newSignatureFixture(t, true)
	fx.activities.recordDetectedSignature(t.Context(), fx.doc, fx.tenant, nil)

	untyped := *fx.doc
	untyped.DocumentTypeID = nil
	fx.activities.recordDetectedSignature(
		t.Context(),
		&untyped,
		fx.tenant,
		&documentaiextractionports.AISignatureReading{Status: "Signed", Confidence: 0.99},
	)

	fx.typeRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	fx.docRepo.AssertNotCalled(t, "RecordDetectedSignature", mock.Anything, mock.Anything)
}
