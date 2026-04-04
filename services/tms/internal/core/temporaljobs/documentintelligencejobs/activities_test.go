package documentintelligencejobs

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type asyncRouteOnlyAIDocumentService struct{}

func (asyncRouteOnlyAIDocumentService) RouteDocument(
	context.Context,
	*services.AIRouteRequest,
) (*services.AIRouteResult, error) {
	return &services.AIRouteResult{
		ShouldExtract:       true,
		DocumentKind:        "RateConfirmation",
		Confidence:          0.95,
		Signals:             []string{"ai route"},
		ReviewStatus:        "Ready",
		ClassifierSource:    "ai-route",
		ProviderFingerprint: "provider=CHRobinson",
		Reason:              "AI route matched rate confirmation",
	}, nil
}

func (asyncRouteOnlyAIDocumentService) ExtractRateConfirmation(
	context.Context,
	*services.AIExtractRequest,
) (*services.AIExtractResult, error) {
	return nil, nil
}

func (asyncRouteOnlyAIDocumentService) SubmitRateConfirmationBackgroundExtraction(
	context.Context,
	*services.AIExtractRequest,
) (*services.AIBackgroundExtractSubmission, error) {
	return nil, nil
}

func (asyncRouteOnlyAIDocumentService) PollRateConfirmationBackgroundExtraction(
	context.Context,
	*services.AIBackgroundExtractPollRequest,
) (*services.AIBackgroundExtractPollResult, error) {
	return nil, nil
}

func TestProcessDocumentIntelligenceActivity_SkipsWhenControlDisabled(t *testing.T) {
	t.Parallel()

	docRepo := mocks.NewMockDocumentRepository(t)
	controlRepo := mocks.NewMockDocumentControlRepository(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docID := pulid.MustNew("doc_")

	doc := &document.Document{
		ID:             docID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ContentStatus:  document.ContentStatusPending,
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
			ID: docID,
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
		}).
		Return(doc, nil)
	controlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&tenant.DocumentControl{
			OrganizationID:             orgID,
			BusinessUnitID:             buID,
			EnableDocumentIntelligence: false,
		}, nil)

	promRegistry := prometheus.NewRegistry()
	activities := &Activities{
		logger: zap.NewNop(),
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(promRegistry, zap.NewNop(), true),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
	}

	result, err := activities.ProcessDocumentIntelligenceActivity(
		context.Background(),
		&ProcessDocumentIntelligencePayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, string(document.ContentStatusPending), result.Status)
	assert.Equal(t, "", result.Kind)
}

func TestProcessDocumentIntelligenceActivity_SuppressesShipmentDraftOutsideShipments(t *testing.T) {
	t.Parallel()

	docRepo := mocks.NewMockDocumentRepository(t)
	controlRepo := mocks.NewMockDocumentControlRepository(t)
	contentRepo := mocks.NewMockDocumentContentRepository(t)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	storageClient := mocks.NewMockClient(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")

	doc := &document.Document{
		ID:             docID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OriginalName:   "rate-confirmation.txt",
		FileType:       "text/plain",
		StoragePath:    "documents/rate-confirmation.txt",
		ResourceType:   "trailer",
		ResourceID:     "trl_123",
		ContentStatus:  document.ContentStatusPending,
		UploadedByID:   userID,
	}

	control := tenant.NewDefaultDocumentControl(orgID, buID)
	contentWrites := make([]documentcontent.Content, 0, 2)
	docUpdates := make([]repositories.UpdateDocumentIntelligenceRequest, 0, 2)
	draftWrites := make([]documentshipmentdraft.DocumentShipmentDraft, 0, 1)
	metricsRegistry := &metrics.Registry{
		Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(doc, nil)
	controlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(control, nil)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("dc_")
			}
			contentWrites = append(contentWrites, *entity)
			return entity, nil
		}).
		Twice()
	contentRepo.EXPECT().
		ReplacePages(mock.Anything, mock.AnythingOfType("*documentcontent.Content"), mock.AnythingOfType("[]*documentcontent.Page")).
		Return(nil)
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*repositories.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Twice()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Twice()
	storageClient.EXPECT().
		Download(mock.Anything, doc.StoragePath).
		Return(&storage.DownloadResult{
			Body: io.NopCloser(
				strings.NewReader(
					"Rate Confirmation\nRate: $1,200\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
				),
			),
		}, nil)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		RunAndReturn(func(_ context.Context, entity *documentshipmentdraft.DocumentShipmentDraft) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			draftWrites = append(draftWrites, *entity)
			return entity, nil
		})

	activities := &Activities{
		logger:              zap.NewNop(),
		cfg:                 &config.DocumentIntelligenceConfig{},
		metrics:             metricsRegistry,
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		storage:             storageClient,
		workflowStarter:     noopWorkflowStarter{},
		parsingRuleRuntime:  noopDocumentParsingRuleRuntime{},
	}

	result, err := activities.ProcessDocumentIntelligenceActivity(
		context.Background(),
		&ProcessDocumentIntelligencePayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, draftWrites, 1)
	require.Len(t, docUpdates, 2)
	assert.Equal(t, kindRateConfirmation, result.Kind)
	assert.Equal(t, documentshipmentdraft.StatusUnavailable, draftWrites[0].Status)
	assert.Equal(t, document.ShipmentDraftStatusUnavailable, docUpdates[1].ShipmentDraftStatus)
	assert.Nil(t, docUpdates[1].DocumentTypeID)
}

func TestProcessDocumentIntelligenceActivity_AutoCreatesAndAssociatesDocumentTypeForShipment(
	t *testing.T,
) {
	t.Parallel()

	docRepo := mocks.NewMockDocumentRepository(t)
	controlRepo := mocks.NewMockDocumentControlRepository(t)
	typeRepo := mocks.NewMockDocumentTypeRepository(t)
	contentRepo := mocks.NewMockDocumentContentRepository(t)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	storageClient := mocks.NewMockClient(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:             docID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OriginalName:   "rate-confirmation.txt",
		FileType:       "text/plain",
		StoragePath:    "documents/rate-confirmation.txt",
		ResourceType:   "shipment",
		ResourceID:     "shp_123",
		ContentStatus:  document.ContentStatusPending,
		UploadedByID:   userID,
	}

	control := tenant.NewDefaultDocumentControl(orgID, buID)
	docUpdates := make([]repositories.UpdateDocumentIntelligenceRequest, 0, 2)
	draftWrites := make([]documentshipmentdraft.DocumentShipmentDraft, 0, 1)
	metricsRegistry := &metrics.Registry{
		Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(doc, nil)
	controlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(control, nil)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("dc_")
			}
			return entity, nil
		}).
		Twice()
	contentRepo.EXPECT().
		ReplacePages(mock.Anything, mock.AnythingOfType("*documentcontent.Content"), mock.AnythingOfType("[]*documentcontent.Page")).
		Return(nil)
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*repositories.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Twice()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Twice()
	storageClient.EXPECT().
		Download(mock.Anything, doc.StoragePath).
		Return(&storage.DownloadResult{
			Body: io.NopCloser(
				strings.NewReader(
					"Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $1,200\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
				),
			),
		}, nil)
	typeRepo.EXPECT().
		GetByCode(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("DocumentType not found"))
	typeRepo.EXPECT().
		GetByName(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("DocumentType not found"))
	typeRepo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*documenttype.DocumentType")).
		Return(&documenttype.DocumentType{ID: typeID}, nil)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		RunAndReturn(func(_ context.Context, entity *documentshipmentdraft.DocumentShipmentDraft) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			draftWrites = append(draftWrites, *entity)
			return entity, nil
		})

	activities := &Activities{
		logger:              zap.NewNop(),
		cfg:                 &config.DocumentIntelligenceConfig{},
		metrics:             metricsRegistry,
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		documentTypeRepo:    typeRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		storage:             storageClient,
		workflowStarter:     noopWorkflowStarter{},
		parsingRuleRuntime:  noopDocumentParsingRuleRuntime{},
	}

	result, err := activities.ProcessDocumentIntelligenceActivity(
		context.Background(),
		&ProcessDocumentIntelligencePayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, draftWrites, 1)
	require.Len(t, docUpdates, 2)
	require.NotNil(t, docUpdates[1].DocumentTypeID)
	assert.Equal(t, typeID, *docUpdates[1].DocumentTypeID)
	assert.Equal(t, document.ShipmentDraftStatusReady, docUpdates[1].ShipmentDraftStatus)
	assert.Equal(t, documentshipmentdraft.StatusReady, draftWrites[0].Status)
	require.NotNil(t, draftWrites[0].DraftData["stops"])
}

func TestProcessDocumentIntelligenceActivity_AssociatesExistingDocumentTypeByName(t *testing.T) {
	t.Parallel()

	docRepo := mocks.NewMockDocumentRepository(t)
	controlRepo := mocks.NewMockDocumentControlRepository(t)
	typeRepo := mocks.NewMockDocumentTypeRepository(t)
	contentRepo := mocks.NewMockDocumentContentRepository(t)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	storageClient := mocks.NewMockClient(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:             docID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OriginalName:   "bill-of-lading.txt",
		FileType:       "text/plain",
		StoragePath:    "documents/bill-of-lading.txt",
		ResourceType:   "shipment",
		ResourceID:     "shp_123",
		ContentStatus:  document.ContentStatusPending,
		UploadedByID:   userID,
	}

	control := tenant.NewDefaultDocumentControl(orgID, buID)
	docUpdates := make([]repositories.UpdateDocumentIntelligenceRequest, 0, 2)
	metricsRegistry := &metrics.Registry{
		Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(doc, nil)
	controlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(control, nil)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("dc_")
			}
			return entity, nil
		}).
		Twice()
	contentRepo.EXPECT().
		ReplacePages(mock.Anything, mock.AnythingOfType("*documentcontent.Content"), mock.AnythingOfType("[]*documentcontent.Page")).
		Return(nil)
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*repositories.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Twice()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Twice()
	storageClient.EXPECT().
		Download(mock.Anything, doc.StoragePath).
		Return(&storage.DownloadResult{
			Body: io.NopCloser(
				strings.NewReader(
					"Bill of Lading\nShipper: ACME Foods\nConsignee: Blue Market\nCommodity: Produce\nBOL #: B12345",
				),
			),
		}, nil)
	typeRepo.EXPECT().
		GetByCode(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("DocumentType not found"))
	typeRepo.EXPECT().
		GetByName(mock.Anything, mock.Anything).
		Return(&documenttype.DocumentType{ID: typeID, Name: "Bill of Lading"}, nil)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		Return(&documentshipmentdraft.DocumentShipmentDraft{}, nil)

	activities := &Activities{
		logger:              zap.NewNop(),
		cfg:                 &config.DocumentIntelligenceConfig{},
		metrics:             metricsRegistry,
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		documentTypeRepo:    typeRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		storage:             storageClient,
		workflowStarter:     noopWorkflowStarter{},
		parsingRuleRuntime:  noopDocumentParsingRuleRuntime{},
	}

	result, err := activities.ProcessDocumentIntelligenceActivity(
		context.Background(),
		&ProcessDocumentIntelligencePayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, docUpdates, 2)
	require.NotNil(t, docUpdates[1].DocumentTypeID)
	assert.Equal(t, typeID, *docUpdates[1].DocumentTypeID)
	assert.Equal(t, "BillOfLading", result.Kind)
}

func TestProcessDocumentIntelligenceActivity_EnqueuesAsyncAIExtraction(t *testing.T) {
	t.Parallel()

	docRepo := mocks.NewMockDocumentRepository(t)
	controlRepo := mocks.NewMockDocumentControlRepository(t)
	typeRepo := mocks.NewMockDocumentTypeRepository(t)
	contentRepo := mocks.NewMockDocumentContentRepository(t)
	draftRepo := mocks.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := mocks.NewMockDocumentSearchProjectionService(t)
	storageClient := mocks.NewMockClient(t)
	workflowStarter := mocks.NewMockWorkflowStarter(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	docID := pulid.MustNew("doc_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:             docID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OriginalName:   "rate-confirmation.txt",
		FileType:       "text/plain",
		StoragePath:    "documents/rate-confirmation.txt",
		ResourceType:   "shipment",
		ResourceID:     "shp_123",
		ContentStatus:  document.ContentStatusPending,
		UploadedByID:   userID,
	}

	control := tenant.NewDefaultDocumentControl(orgID, buID)
	contentWrites := make([]documentcontent.Content, 0, 3)

	docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("dc_")
			}
			contentWrites = append(contentWrites, *entity)
			return entity, nil
		}).
		Twice()
	contentRepo.EXPECT().
		ReplacePages(mock.Anything, mock.AnythingOfType("*documentcontent.Content"), mock.AnythingOfType("[]*documentcontent.Page")).
		Return(nil)
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*repositories.UpdateDocumentIntelligenceRequest")).
		Return(nil).Twice()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Twice()
	storageClient.EXPECT().
		Download(mock.Anything, doc.StoragePath).
		Return(&storage.DownloadResult{
			Body: io.NopCloser(
				strings.NewReader(
					"Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $1,200\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
				),
			),
		}, nil)
	typeRepo.EXPECT().
		GetByCode(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("DocumentType not found"))
	typeRepo.EXPECT().
		GetByName(mock.Anything, mock.Anything).
		Return(&documenttype.DocumentType{ID: typeID, Name: "Rate Confirmation"}, nil)
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		Return(&documentshipmentdraft.DocumentShipmentDraft{}, nil)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(mock.Anything, mock.MatchedBy(func(options client.StartWorkflowOptions) bool {
			return strings.HasPrefix(options.ID, "document-ai-extraction-"+docID.String()+"-")
		}), "ProcessDocumentAIExtractionWorkflow", mock.Anything).
		Return(nil, nil)

	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &config.DocumentIntelligenceConfig{EnableAI: true},
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		documentTypeRepo:    typeRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   asyncRouteOnlyAIDocumentService{},
		searchProjection:    searchProjection,
		storage:             storageClient,
		workflowStarter:     workflowStarter,
	}

	_, err := activities.ProcessDocumentIntelligenceActivity(
		context.Background(),
		&ProcessDocumentIntelligencePayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UserID:         userID,
			},
		},
	)

	require.NoError(t, err)
	require.Len(t, contentWrites, 2)
	diagnostics := contentWrites[1].StructuredData["aiDiagnostics"].(map[string]any)
	assert.Equal(t, aiAcceptanceStatusPending, diagnostics["acceptanceStatus"])
}

func TestHasUsableShipmentDraft_WithReviewableStops(t *testing.T) {
	t.Parallel()

	intelligence := &DocumentIntelligenceAnalysis{
		ReviewStatus: "NeedsReview",
		Fields: map[string]*ReviewField{
			"shipper": {
				Value: "Anyco Clothes #425",
			},
			"consignee": {
				Value: "Anyco Clothes #255",
			},
			"rate": {
				Value: "4500.00",
			},
		},
		Stops: []*IntelligenceStop{
			{
				Role:         stopRolePickup,
				Name:         "Anyco Clothes #425",
				AddressLine1: "Main Drive",
				City:         "Houston",
				State:        "TX",
				PostalCode:   "78705",
				Date:         "2021-07-13",
				TimeWindow:   "04:00",
			},
			{
				Role:         stopRoleDelivery,
				Name:         "Anyco Clothes #255",
				AddressLine1: "1234 E 1st Ave",
				City:         "Dallas",
				State:        "TX",
				PostalCode:   "76103",
				Date:         "2021-07-15",
				TimeWindow:   "08:00-22:00",
			},
		},
	}

	require.True(t, hasUsableShipmentDraft(intelligence))
}

func TestHasUsableShipmentDraft_FalseForIncompleteDraft(t *testing.T) {
	t.Parallel()

	intelligence := &DocumentIntelligenceAnalysis{
		ReviewStatus: "NeedsReview",
		Fields:       map[string]*ReviewField{},
		Stops: []*IntelligenceStop{
			{
				Role: stopRolePickup,
				Name: "",
			},
		},
	}

	require.False(t, hasUsableShipmentDraft(intelligence))
}
