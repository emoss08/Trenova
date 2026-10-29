package documentextractionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentcontentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentsearchprojectionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentshipmentdraftportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
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
		AI: aiconfig.Config{},
	}
	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
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

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(doc, nil)
	documentRepo.EXPECT().UpdateIntelligence(
		mock.Anything,
		mock.MatchedBy(func(req *documentports.UpdateDocumentIntelligenceRequest) bool {
			return req.ID == documentID &&
				req.TenantInfo == tenantInfo &&
				req.ContentStatus == document.ContentStatusPending &&
				req.ContentError == "" &&
				req.DetectedKind == "" &&
				!req.HasExtractedText &&
				req.ShipmentDraftStatus == document.ShipmentDraftStatusUnavailable
		}),
	).Return(nil)

	contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.MatchedBy(func(content *documentcontent.Content) bool {
			return content.DocumentID == documentID &&
				content.OrganizationID == orgID &&
				content.BusinessUnitID == buID &&
				content.Status == documentcontent.StatusPending
		})).
		Return(&documentcontent.Content{
			ID:             pulid.MustNew("dcc_"),
			DocumentID:     documentID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         documentcontent.StatusPending,
		}, nil)
	contentRepo.EXPECT().ReplacePages(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	documentControlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	documentControlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&documentcontrol.DocumentControl{
			EnableDocumentIntelligence: true,
		}, nil)

	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)
	searchProjection.EXPECT().Upsert(mock.Anything, doc, "").Return(nil)

	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(
			mock.Anything,
			mock.Anything,
			documentextractioncontract.ProcessWorkflowName,
			mock.MatchedBy(func(args []any) bool {
				if len(args) != 1 {
					return false
				}
				payload, ok := args[0].(*documentextractioncontract.ProcessDocumentExtractionPayload)
				return ok && payload.DocumentID == documentID && payload.UserID == userID
			}),
		).
		Return(nil, nil)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &cfg.DocumentConversion,
		SearchConfig:             &cfg.Search,
		Metrics:                  metricRegistry,
		DocumentControlRepo:      documentControlRepo,
		DocumentRepo:             documentRepo,
		ContentRepo:              contentRepo,
		DraftRepo: documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(
			t,
		),
		SearchProjection: searchProjection,
		WorkflowStarter:  workflowStarter,
	})

	err = service.Reextract(t.Context(), documentID, tenantInfo)
	require.NoError(t, err)
}

func TestEnqueueExtractionSkipsWhenDocumentIntelligenceDisabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		AI: aiconfig.Config{},
	}
	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	documentID := pulid.MustNew("doc_")
	userID := pulid.MustNew("usr_")

	documentControlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	documentControlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&documentcontrol.DocumentControl{
			EnableDocumentIntelligence: false,
		}, nil)

	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &cfg.DocumentConversion,
		SearchConfig:             &cfg.Search,
		Metrics:                  metricRegistry,
		DocumentControlRepo:      documentControlRepo,
		DocumentRepo:             documentportsmock.NewMockDocumentRepository(t),
		ContentRepo:              documentcontentportsmock.NewMockDocumentContentRepository(t),
		DraftRepo: documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(
			t,
		),
		WorkflowStarter: workflowStarter,
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
		AI: aiconfig.Config{},
	}
	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &cfg.DocumentConversion,
		SearchConfig:             &cfg.Search,
		Metrics:                  metricRegistry,
		DocumentControlRepo:      tenantportsmock.NewMockDocumentControlRepository(t),
		DocumentRepo:             documentportsmock.NewMockDocumentRepository(t),
		ContentRepo:              documentcontentportsmock.NewMockDocumentContentRepository(t),
		DraftRepo: documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(
			t,
		),
		WorkflowStarter: kernelportsmock.NewMockWorkflowStarter(t),
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
		AI: aiconfig.Config{},
	}
	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}).Return(&document.Document{
		ID:                documentID,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		ProcessingProfile: document.ProcessingProfileNone,
	}, nil)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &cfg.DocumentConversion,
		SearchConfig:             &cfg.Search,
		Metrics:                  metricRegistry,
		DocumentControlRepo:      tenantportsmock.NewMockDocumentControlRepository(t),
		DocumentRepo:             documentRepo,
		ContentRepo:              documentcontentportsmock.NewMockDocumentContentRepository(t),
		DraftRepo: documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(
			t,
		),
		WorkflowStarter: kernelportsmock.NewMockWorkflowStarter(t),
	})

	err = service.Reextract(t.Context(), documentID, tenantInfo)
	require.Error(t, err)
	require.True(t, errortypes.IsConflictError(err))
}
