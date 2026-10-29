package documentextractionjobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs/documentextractioncontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentcontentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentsearchprojectionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentshipmentdraftportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttypeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
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

const (
	rateConfirmationText = "Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $1,200\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026"
	billOfLadingText     = "Bill of Lading\nShipper: ACME Foods\nConsignee: Blue Market\nCommodity: Produce\nBOL #: B12345"
)

type asyncRouteOnlyAIDocumentService struct{}

func (asyncRouteOnlyAIDocumentService) ExtractRateConfirmationForEvaluation(
	context.Context,
	*documentaiextractionports.AIEvaluationExtractRequest,
) (*documentaiextractionports.AIEvaluationExtractResult, error) {
	return nil, errors.New("evaluation extraction is not used by these tests")
}

func (asyncRouteOnlyAIDocumentService) ExtractRateConfirmationForShadow(
	context.Context,
	*documentaiextractionports.AIShadowExtractRequest,
) (*documentaiextractionports.AIEvaluationExtractResult, error) {
	return nil, errors.New("shadow extraction is not used by these tests")
}

func (asyncRouteOnlyAIDocumentService) RouteDocument(
	context.Context,
	*documentaiextractionports.AIRouteRequest,
) (*documentaiextractionports.AIRouteResult, error) {
	return &documentaiextractionports.AIRouteResult{
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
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIExtractResult, error) {
	return nil, nil
}

func (asyncRouteOnlyAIDocumentService) SubmitRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIBackgroundExtractSubmission, error) {
	return nil, nil
}

func (asyncRouteOnlyAIDocumentService) ExtractionAvailability(
	context.Context,
	pagination.TenantInfo,
) (documentaiextractionports.ExtractionAvailability, error) {
	return documentaiextractionports.ExtractionAvailable, nil
}

func (asyncRouteOnlyAIDocumentService) PollRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIBackgroundExtractPollRequest,
) (*documentaiextractionports.AIBackgroundExtractPollResult, error) {
	return nil, nil
}

func expectStoredText(
	contentRepo *documentcontentportsmock.MockDocumentContentRepository,
	doc *document.Document,
	text string,
) {
	contentRepo.EXPECT().
		GetByDocumentID(mock.Anything, doc.ID, mock.Anything).
		Return(&documentcontent.Content{
			ID:             pulid.MustNew("dc_"),
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			Status:         documentcontent.StatusExtracted,
			ContentText:    text,
			PageCount:      1,
			SourceKind:     documentcontent.SourceKindNative,
		}, nil)
	contentRepo.EXPECT().
		ListPagesByDocumentID(mock.Anything, doc.ID, mock.Anything).
		Return([]*documentcontent.Page{{
			DocumentID:     doc.ID,
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			PageNumber:     1,
			SourceKind:     documentcontent.SourceKindNative,
			ExtractedText:  text,
		}}, nil)
}

func analyzeInput(doc *document.Document) *AnalyzeConvertedDocumentInput {
	return &AnalyzeConvertedDocumentInput{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
			UserID:         doc.UploadedByID,
		},
		DocumentID: doc.ID,
	}
}

func TestPlanDocumentExtractionActivity_SkipsWhenControlDisabled(t *testing.T) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docID := pulid.MustNew("doc_")

	doc := &document.Document{
		ID:                docID,
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		ContentStatus:     document.ContentStatusPending,
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
			ID: docID,
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
		}).
		Return(doc, nil)
	controlRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&documentcontrol.DocumentControl{
			OrganizationID:             orgID,
			BusinessUnitID:             buID,
			EnableDocumentIntelligence: false,
		}, nil)

	activities := &Activities{
		logger: zap.NewNop(),
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), true),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
	}

	plan, err := activities.PlanDocumentExtractionActivity(
		t.Context(),
		&documentextractioncontract.ProcessDocumentExtractionPayload{
			DocumentID: docID,
			BasePayload: temporaltype.BasePayload{
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, ExtractionRouteSkip, plan.Route)
	require.NotNil(t, plan.Result)
	assert.Equal(t, string(document.ContentStatusPending), plan.Result.Status)
	assert.Empty(t, plan.Result.Kind)
}

/*
The profile is the one gate that cannot be gone around.

EnqueueExtraction checks it, but the reconcile sweep starts this workflow
directly, so the plan refuses the document itself. The document control is not
reached at all here: a document that is not to be read is not read, whatever
the tenant has switched on.
*/
func TestPlanDocumentExtractionActivity_RefusesAProfileThatIsNotRead(t *testing.T) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	docID := pulid.MustNew("doc_")

	doc := &document.Document{
		ID:                docID,
		ProcessingProfile: document.ProcessingProfileNone,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		ContentStatus:     document.ContentStatusPending,
	}

	docRepo.EXPECT().
		GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
			ID:         docID,
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(doc, nil)

	activities := &Activities{
		logger: zap.NewNop(),
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), true),
		},
		documentRepo: docRepo,
	}

	plan, err := activities.PlanDocumentExtractionActivity(
		t.Context(),
		&documentextractioncontract.ProcessDocumentExtractionPayload{
			DocumentID:  docID,
			BasePayload: temporaltype.BasePayload{OrganizationID: orgID, BusinessUnitID: buID},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, ExtractionRouteSkip, plan.Route)
	require.NotNil(t, plan.Result)
	assert.Equal(t, string(document.ContentStatusPending), plan.Result.Status)
}

func TestAnalyzeConvertedDocumentActivity_SuppressesShipmentDraftOutsideShipments(t *testing.T) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
	draftRepo := documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	doc := &document.Document{
		ID:                pulid.MustNew("doc_"),
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "rate-confirmation.txt",
		FileType:          "text/plain",
		StoragePath:       "documents/rate-confirmation.txt",
		ResourceType:      "trailer",
		ResourceID:        "trl_123",
		ContentStatus:     document.ContentStatusExtracting,
		UploadedByID:      pulid.MustNew("usr_"),
	}

	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	docUpdates := make([]documentports.UpdateDocumentIntelligenceRequest, 0, 1)
	draftWrites := make([]documentshipmentdraft.DocumentShipmentDraft, 0, 1)

	docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)
	expectStoredText(contentRepo, doc, rateConfirmationText)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			return entity, nil
		}).
		Once()
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*documentports.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *documentports.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Once()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Once()
	draftRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentshipmentdraft.DocumentShipmentDraft")).
		RunAndReturn(func(_ context.Context, entity *documentshipmentdraft.DocumentShipmentDraft) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			draftWrites = append(draftWrites, *entity)
			return entity, nil
		})

	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		workflowStarter:     noopWorkflowStarter{},
	}

	result, err := activities.AnalyzeConvertedDocumentActivity(t.Context(), analyzeInput(doc))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, draftWrites, 1)
	require.Len(t, docUpdates, 1)
	assert.Equal(t, kindRateConfirmation, result.Kind)
	assert.Equal(t, documentshipmentdraft.StatusUnavailable, draftWrites[0].Status)
	assert.Equal(t, document.ShipmentDraftStatusUnavailable, docUpdates[0].ShipmentDraftStatus)
	assert.Nil(t, docUpdates[0].DocumentTypeID)
}

func TestAnalyzeConvertedDocumentActivity_AutoCreatesAndAssociatesDocumentTypeForShipment(
	t *testing.T,
) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	typeRepo := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
	draftRepo := documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:                pulid.MustNew("doc_"),
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "rate-confirmation.txt",
		FileType:          "text/plain",
		StoragePath:       "documents/rate-confirmation.txt",
		ResourceType:      "shipment",
		ResourceID:        "shp_123",
		ContentStatus:     document.ContentStatusExtracting,
		UploadedByID:      pulid.MustNew("usr_"),
	}

	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	docUpdates := make([]documentports.UpdateDocumentIntelligenceRequest, 0, 1)
	draftWrites := make([]documentshipmentdraft.DocumentShipmentDraft, 0, 1)

	docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)
	expectStoredText(contentRepo, doc, rateConfirmationText)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			return entity, nil
		}).
		Once()
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*documentports.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *documentports.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Once()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Once()
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
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		documentTypeRepo:    typeRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		workflowStarter:     noopWorkflowStarter{},
	}

	result, err := activities.AnalyzeConvertedDocumentActivity(t.Context(), analyzeInput(doc))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, draftWrites, 1)
	require.Len(t, docUpdates, 1)
	require.NotNil(t, docUpdates[0].DocumentTypeID)
	assert.Equal(t, typeID, *docUpdates[0].DocumentTypeID)
	assert.Equal(t, document.ShipmentDraftStatusReady, docUpdates[0].ShipmentDraftStatus)
	assert.Equal(t, documentshipmentdraft.StatusReady, draftWrites[0].Status)
	require.NotNil(t, draftWrites[0].DraftData["stops"])
}

func TestAnalyzeConvertedDocumentActivity_AssociatesExistingDocumentTypeByName(t *testing.T) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	typeRepo := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
	draftRepo := documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:                pulid.MustNew("doc_"),
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "bill-of-lading.txt",
		FileType:          "text/plain",
		StoragePath:       "documents/bill-of-lading.txt",
		ResourceType:      "shipment",
		ResourceID:        "shp_123",
		ContentStatus:     document.ContentStatusExtracting,
		UploadedByID:      pulid.MustNew("usr_"),
	}

	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	docUpdates := make([]documentports.UpdateDocumentIntelligenceRequest, 0, 1)

	docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)
	expectStoredText(contentRepo, doc, billOfLadingText)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			return entity, nil
		}).
		Once()
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*documentports.UpdateDocumentIntelligenceRequest")).
		RunAndReturn(func(_ context.Context, req *documentports.UpdateDocumentIntelligenceRequest) error {
			docUpdates = append(docUpdates, *req)
			return nil
		}).
		Once()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Once()
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
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
		metrics: &metrics.Registry{
			Document: metrics.NewDocument(prometheus.NewRegistry(), zap.NewNop(), false),
		},
		documentRepo:        docRepo,
		documentControlRepo: controlRepo,
		documentTypeRepo:    typeRepo,
		contentRepo:         contentRepo,
		draftRepo:           draftRepo,
		aiDocumentService:   noopAIDocumentService{},
		searchProjection:    searchProjection,
		workflowStarter:     noopWorkflowStarter{},
	}

	result, err := activities.AnalyzeConvertedDocumentActivity(t.Context(), analyzeInput(doc))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, docUpdates, 1)
	require.NotNil(t, docUpdates[0].DocumentTypeID)
	assert.Equal(t, typeID, *docUpdates[0].DocumentTypeID)
	assert.Equal(t, "BillOfLading", result.Kind)
}

func TestAnalyzeConvertedDocumentActivity_EnqueuesAsyncAIExtraction(t *testing.T) {
	t.Parallel()

	docRepo := documentportsmock.NewMockDocumentRepository(t)
	controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
	typeRepo := documenttypeportsmock.NewMockDocumentTypeRepository(t)
	contentRepo := documentcontentportsmock.NewMockDocumentContentRepository(t)
	draftRepo := documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t)
	searchProjection := documentsearchprojectionportsmock.NewMockDocumentSearchProjectionService(t)
	workflowStarter := kernelportsmock.NewMockWorkflowStarter(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	typeID := pulid.MustNew("dt_")

	doc := &document.Document{
		ID:                pulid.MustNew("doc_"),
		ProcessingProfile: document.ProcessingProfileRateConfirmationImport,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		OriginalName:      "rate-confirmation.txt",
		FileType:          "text/plain",
		StoragePath:       "documents/rate-confirmation.txt",
		ResourceType:      "shipment",
		ResourceID:        "shp_123",
		ContentStatus:     document.ContentStatusExtracting,
		UploadedByID:      pulid.MustNew("usr_"),
	}

	control := documentcontrol.NewDefaultDocumentControl(orgID, buID)
	contentWrites := make([]documentcontent.Content, 0, 1)

	docRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(doc, nil)
	controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)
	expectStoredText(contentRepo, doc, rateConfirmationText)
	contentRepo.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*documentcontent.Content")).
		RunAndReturn(func(_ context.Context, entity *documentcontent.Content) (*documentcontent.Content, error) {
			contentWrites = append(contentWrites, *entity)
			return entity, nil
		}).
		Once()
	docRepo.EXPECT().
		UpdateIntelligence(mock.Anything, mock.AnythingOfType("*documentports.UpdateDocumentIntelligenceRequest")).
		Return(nil).Once()
	searchProjection.EXPECT().
		Upsert(mock.Anything, mock.AnythingOfType("*document.Document"), mock.Anything).
		Return(nil).Once()
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
			return strings.HasPrefix(options.ID, "document-ai-extraction-"+doc.ID.String()+"-")
		}), "ProcessDocumentAIExtractionWorkflow", mock.Anything).
		Return(nil, nil)

	activities := &Activities{
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{DocumentExtraction: true},
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
		workflowStarter:     workflowStarter,
	}

	_, err := activities.AnalyzeConvertedDocumentActivity(t.Context(), analyzeInput(doc))

	require.NoError(t, err)
	require.Len(t, contentWrites, 1)
	diagnostics := contentWrites[0].StructuredData["aiDiagnostics"].(map[string]any)
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

func TestShipmentDraftDataFromAIExtract_MatchesTheProductionDraftShape(t *testing.T) {
	t.Parallel()

	data := ShipmentDraftDataFromAIExtract(&documentaiextractionports.AIExtractResult{
		DocumentKind: "RateConfirmation",
		Fields: map[string]documentaiextractionports.AIDocumentField{
			"Total Rate": {Label: "Total Rate", Value: "$1,850.00", Confidence: 0.9},
		},
		Stops: []*documentaiextractionports.AIDocumentStop{
			{Sequence: 0, Role: "Shipper", Name: "Northwind Foods", City: "Dallas"},
		},
	})

	fields, ok := data["fields"].(map[string]any)
	require.True(t, ok)
	rate, ok := fields["rate"].(map[string]any)
	require.True(t, ok, "field keys are normalized the way production normalizes them")
	assert.Equal(t, "$1,850.00", rate["value"])

	stops, ok := data["stops"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, stops, 1)
	assert.Equal(t, "pickup", stops[0]["role"])
	assert.Equal(t, "Northwind Foods", stops[0]["name"])

	empty := ShipmentDraftDataFromAIExtract(nil)
	assert.NotNil(t, empty["fields"])
}
