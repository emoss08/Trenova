package documentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

func TestAttachLineageToResourceUpdatesDraftAttachmentMetadata(t *testing.T) {
	t.Parallel()

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
	cacheRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Maybe().Return(nil, repositories.ErrCacheMiss)
	sessionRepo := mocks.NewMockDocumentUploadSessionRepository(t)
	sessionRepo.EXPECT().ClearDocumentReference(mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.EXPECT().ClearDocumentReferences(mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	contentService := mocks.NewMockDocumentContentService(t)
	contentService.EXPECT().GetContent(mock.Anything, documentID, tenantInfo).Return(nil, assert.AnError)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	searchProjection.EXPECT().Upsert(mock.Anything, updated, "").Return(nil)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)

	draft := &documentshipmentdraft.Draft{
		ID:             pulid.MustNew("dsd_"),
		DocumentID:     documentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         documentshipmentdraft.StatusReady,
	}
	draftRepo.EXPECT().GetByDocumentID(mock.Anything, documentID, tenantInfo).Return(draft, nil)
	draftRepo.EXPECT().Upsert(mock.Anything, mock.MatchedBy(func(entity *documentshipmentdraft.Draft) bool {
		return entity.AttachedShipmentID != nil &&
			*entity.AttachedShipmentID == shipmentID &&
			entity.AttachedByID != nil &&
			*entity.AttachedByID == userID &&
			entity.AttachedAt != nil &&
			*entity.AttachedAt > 0
	})).Return(draft, nil)

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
}
