package documentintelligenceservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentintelligenceservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReextractResetsStateAndRequeuesWorkflow(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DocumentIntelligence: config.DocumentIntelligenceConfig{Enabled: true},
	}
	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	doc := &document.Document{
		ID:                  documentID,
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		ProcessingProfile:   document.ProcessingProfileRateConfirmationImport,
		ContentStatus:       document.ContentStatusFailed,
		ContentError:        "old error",
		DetectedKind:        "RateConfirmation",
		HasExtractedText:    true,
		ShipmentDraftStatus: document.ShipmentDraftStatusReady,
	}

	documentRepo := mocks.NewMockDocumentRepository(t)
	documentRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(doc, nil)
	documentRepo.EXPECT().UpdateIntelligence(
		mock.Anything,
		mock.MatchedBy(func(req *repositories.UpdateDocumentIntelligenceRequest) bool {
			return req.ID == documentID &&
				req.TenantInfo == tenantInfo &&
				req.ContentStatus == document.ContentStatusPending &&
				req.ContentError == "" &&
				req.DetectedKind == "" &&
				!req.HasExtractedText &&
				req.ShipmentDraftStatus == document.ShipmentDraftStatusUnavailable
		}),
	).Return(nil)

	contentRepo := mocks.NewMockDocumentContentRepository(t)
	contentRepo.EXPECT().Upsert(mock.Anything, mock.MatchedBy(func(content *documentcontent.Content) bool {
		return content.DocumentID == documentID &&
			content.OrganizationID == orgID &&
			content.BusinessUnitID == buID &&
			content.Status == documentcontent.StatusPending
	})).Return(&documentcontent.Content{
		ID:             pulid.MustNew("dcc_"),
		DocumentID:     documentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         documentcontent.StatusPending,
	}, nil)
	contentRepo.EXPECT().ReplacePages(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	documentControlRepo := mocks.NewMockDocumentControlRepository(t)
	documentControlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(&tenant.DocumentControl{
		EnableDocumentIntelligence: true,
	}, nil)

	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	searchProjection.EXPECT().Upsert(mock.Anything, doc, "").Return(nil)

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(
			mock.Anything,
			mock.Anything,
			"ProcessDocumentIntelligenceWorkflow",
			mock.MatchedBy(func(args []any) bool {
				if len(args) != 1 {
					return false
				}
				payload, ok := args[0].(*documentintelligencejobs.ProcessDocumentIntelligencePayload)
				return ok && payload.DocumentID == documentID && payload.UserID == userID
			}),
		).
		Return(nil, nil)

	service := documentintelligenceservice.New(documentintelligenceservice.Params{
		Logger:              zap.NewNop(),
		Config:              cfg,
		Metrics:             metricRegistry,
		DocumentControlRepo: documentControlRepo,
		DocumentRepo:        documentRepo,
		ContentRepo:         contentRepo,
		DraftRepo:           mocks.NewMockDocumentShipmentDraftRepository(t),
		SearchProjection:    searchProjection,
		WorkflowStarter:     workflowStarter,
	})

	err = service.Reextract(t.Context(), documentID, tenantInfo)
	require.NoError(t, err)
}

func TestEnqueueExtractionSkipsWhenDocumentIntelligenceDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DocumentIntelligence: config.DocumentIntelligenceConfig{Enabled: true},
	}
	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	documentID := pulid.MustNew("doc_")
	userID := pulid.MustNew("usr_")

	documentControlRepo := mocks.NewMockDocumentControlRepository(t)
	documentControlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(&tenant.DocumentControl{
		EnableDocumentIntelligence: false,
	}, nil)

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)

	service := documentintelligenceservice.New(documentintelligenceservice.Params{
		Logger:              zap.NewNop(),
		Config:              cfg,
		Metrics:             metricRegistry,
		DocumentControlRepo: documentControlRepo,
		DocumentRepo:        mocks.NewMockDocumentRepository(t),
		ContentRepo:         mocks.NewMockDocumentContentRepository(t),
		DraftRepo:           mocks.NewMockDocumentShipmentDraftRepository(t),
		WorkflowStarter:     workflowStarter,
	})

	err = service.EnqueueExtraction(t.Context(), &document.Document{
		ID:                documentID,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
	}, userID)
	require.NoError(t, err)
}

func TestEnqueueExtractionSkipsWhenProcessingProfileIsNone(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DocumentIntelligence: config.DocumentIntelligenceConfig{Enabled: true},
	}
	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	service := documentintelligenceservice.New(documentintelligenceservice.Params{
		Logger:              zap.NewNop(),
		Config:              cfg,
		Metrics:             metricRegistry,
		DocumentControlRepo: mocks.NewMockDocumentControlRepository(t),
		DocumentRepo:        mocks.NewMockDocumentRepository(t),
		ContentRepo:         mocks.NewMockDocumentContentRepository(t),
		DraftRepo:           mocks.NewMockDocumentShipmentDraftRepository(t),
		WorkflowStarter:     mocks.NewMockWorkflowStarter(t),
	})

	err = service.EnqueueExtraction(t.Context(), &document.Document{
		ID:                pulid.MustNew("doc_"),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		ProcessingProfile: document.ProcessingProfileNone,
	}, pulid.MustNew("usr_"))
	require.NoError(t, err)
}

func TestReextractRejectsDocumentsWithoutTargetedProcessing(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DocumentIntelligence: config.DocumentIntelligenceConfig{Enabled: true},
	}
	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	documentRepo := mocks.NewMockDocumentRepository(t)
	documentRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:                documentID,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		ProcessingProfile: document.ProcessingProfileNone,
	}, nil)

	service := documentintelligenceservice.New(documentintelligenceservice.Params{
		Logger:              zap.NewNop(),
		Config:              cfg,
		Metrics:             metricRegistry,
		DocumentControlRepo: mocks.NewMockDocumentControlRepository(t),
		DocumentRepo:        documentRepo,
		ContentRepo:         mocks.NewMockDocumentContentRepository(t),
		DraftRepo:           mocks.NewMockDocumentShipmentDraftRepository(t),
		WorkflowStarter:     mocks.NewMockWorkflowStarter(t),
	})

	err = service.Reextract(t.Context(), documentID, tenantInfo)
	require.Error(t, err)
	require.True(t, errortypes.IsConflictError(err))
}
