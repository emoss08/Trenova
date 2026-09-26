package documentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingCorrections struct {
	requests []*services.CaptureShipmentDraftCorrectionRequest
	err      error
}

func (r *recordingCorrections) CaptureShipmentDraft(
	_ context.Context,
	req *services.CaptureShipmentDraftCorrectionRequest,
) (*aicorrection.Correction, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return nil, r.err
	}
	return &aicorrection.Correction{}, nil
}

func (r *recordingCorrections) PurgeExpired(
	context.Context,
	services.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	return 0, nil
}

func TestAttachLineageToResourceUpdatesDraftAttachmentMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		corrections *recordingCorrections
	}{
		{name: "without correction capture"},
		{name: "captures the correction", corrections: &recordingCorrections{}},
		{
			name:        "a failed capture does not fail the attach",
			corrections: &recordingCorrections{err: assert.AnError},
		},
		{
			name:        "a draft with nothing predicted is skipped quietly",
			corrections: &recordingCorrections{err: aicorrection.ErrNothingPredicted},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			attachLineageToShipment(t, tt.corrections)
		})
	}
}

func attachLineageToShipment(t *testing.T, corrections *recordingCorrections) {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	documentID := pulid.MustNew("doc_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	}

	current := &document.Document{
		ID:             documentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ResourceType:   "worker",
		ResourceID:     "wrk_123",
	}
	updated := &document.Document{
		ID:             documentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ResourceType:   "shipment",
		ResourceID:     shipmentID.String(),
	}

	getByIDCalls := 0
	repo := &mockDocRepo{
		GetByIDFn: func(_ context.Context, req repositories.GetDocumentByIDRequest) (*document.Document, error) {
			assert.Equal(t, tenantInfo, req.TenantInfo)
			getByIDCalls++
			if getByIDCalls == 1 {
				return current, nil
			}
			return updated, nil
		},
		MoveLineageToResourceFn: func(_ context.Context, req *repositories.MoveDocumentLineageRequest) error {
			assert.Equal(t, documentID, req.DocumentID)
			assert.Equal(t, shipmentID.String(), req.ResourceID)
			assert.Equal(t, "shipment", req.ResourceType)
			assert.Equal(t, tenantInfo, req.TenantInfo)
			return nil
		},
	}

	cacheRepo := mocks.NewMockDocumentCacheRepository(t)
	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	contentService := mocks.NewMockDocumentContentService(t)
	contentService.EXPECT().
		GetContent(mock.Anything, documentID, tenantInfo).
		Return(nil, assert.AnError)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	searchProjection.EXPECT().Upsert(mock.Anything, updated, "").Return(nil)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	shipments := mocks.NewMockShipmentRepository(t)
	shipments.EXPECT().
		ListSummariesByIDs(mock.Anything, &repositories.ListShipmentSummariesRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: []pulid.ID{shipmentID},
		}).
		Return([]*repositories.ShipmentSummary{{ShipmentID: shipmentID}}, nil)

	draft := &documentshipmentdraft.DocumentShipmentDraft{
		ID:             pulid.MustNew("dsd_"),
		DocumentID:     documentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         documentshipmentdraft.StatusReady,
	}
	draftRepo.EXPECT().GetByDocumentID(mock.Anything, documentID, tenantInfo).Return(draft, nil)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.MatchedBy(func(entity *documentshipmentdraft.DocumentShipmentDraft) bool {
			return entity.AttachedShipmentID != nil &&
				*entity.AttachedShipmentID == shipmentID &&
				entity.AttachedByID != nil &&
				*entity.AttachedByID == userID &&
				entity.AttachedAt != nil &&
				*entity.AttachedAt > 0
		})).
		Return(draft, nil)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			AllowedMIMETypes:   []string{"application/pdf"},
			MaxFileSize:        50 * 1024 * 1024,
			PresignedURLExpiry: 15 * time.Minute,
		},
	}
	validator := documentservice.NewValidator(documentservice.ValidatorParams{Config: cfg})

	service := documentservice.New(documentservice.Params{
		Logger:               zap.NewNop(),
		Repo:                 repo,
		DraftRepo:            draftRepo,
		Shipments:            shipments,
		CacheRepo:            cacheRepo,
		SessionRepo:          sessionRepo,
		Storage:              &mockStorageClient{},
		Validator:            validator,
		AuditService:         &mocks.NoopAuditService{},
		DocumentIntelligence: contentService,
		SearchProjection:     searchProjection,
		Config:               cfg,
		ThumbnailGenerator:   thumbnailservice.NewGenerator(),
		WorkflowStarter:      workflowstarter.New(workflowstarter.Params{}),
		AICorrections:        correctionService(corrections),
	})

	result, err := service.AttachLineageToResource(
		t.Context(),
		documentID,
		"shipment",
		shipmentID.String(),
		tenantInfo,
		userID,
	)
	require.NoError(t, err)
	require.Equal(t, updated, result)

	if corrections != nil {
		require.Len(t, corrections.requests, 1)
		captured := corrections.requests[0]
		assert.Same(t, draft, captured.Draft)
		assert.Equal(t, shipmentID, captured.ShipmentID)
		assert.Equal(t, userID, captured.CapturedByID)
		assert.Equal(t, tenantInfo, captured.TenantInfo)
	}
}

func correctionService(corrections *recordingCorrections) services.AICorrectionService {
	if corrections == nil {
		return nil
	}
	return corrections
}

// The lineage columns hold the shipment id as text with no foreign key, so a
// document moved onto another tenant's shipment, or one that never existed,
// would disappear from every place a person looks for it.
func TestAttachLineageToResourceRefusesAShipmentTheTenantDoesNotHave(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	documentID := pulid.MustNew("doc_")

	moved := false
	repo := &mockDocRepo{
		GetByIDFn: func(context.Context, repositories.GetDocumentByIDRequest) (*document.Document, error) {
			return &document.Document{
				ID:           documentID,
				ResourceType: "worker",
				ResourceID:   "wrk_1",
			}, nil
		},
		MoveLineageToResourceFn: func(context.Context, *repositories.MoveDocumentLineageRequest) error {
			moved = true
			return nil
		},
	}
	shipments := mocks.NewMockShipmentRepository(t)
	shipments.EXPECT().
		ListSummariesByIDs(mock.Anything, mock.Anything).
		Return([]*repositories.ShipmentSummary{}, nil)

	cfg := &config.Config{}
	service := documentservice.New(documentservice.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Shipments:    shipments,
		Storage:      &mockStorageClient{},
		Validator:    documentservice.NewValidator(documentservice.ValidatorParams{Config: cfg}),
		AuditService: &mocks.NoopAuditService{},
		Config:       cfg,
	})

	_, err := service.AttachLineageToResource(
		t.Context(),
		documentID,
		"shipment",
		pulid.MustNew("shp_").String(),
		tenantInfo,
		tenantInfo.UserID,
	)

	require.Error(t, err)
	assert.False(t, moved, "the document must stay where it was")
}
