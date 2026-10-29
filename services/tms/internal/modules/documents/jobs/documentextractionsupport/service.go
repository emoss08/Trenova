package documentextractionsupport

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Metrics             *metrics.Registry
	DocumentControlRepo tenantports.DocumentControlRepository
	DraftRepo           documentshipmentdraftports.DocumentShipmentDraftRepository
	SearchProjection    documentsearchprojectionports.DocumentSearchProjectionService
	AgentEvents         agenteventports.AgentEventPublisher `optional:"true"`
}

type Service struct {
	logger              *zap.Logger
	metrics             *metrics.Registry
	documentControlRepo tenantports.DocumentControlRepository
	draftRepo           documentshipmentdraftports.DocumentShipmentDraftRepository
	searchProjection    documentsearchprojectionports.DocumentSearchProjectionService
	agentEvents         agenteventports.AgentEventPublisher
}

func New(p Params) *Service {
	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	return &Service{
		logger:              p.Logger.Named("temporal.document-extraction"),
		metrics:             p.Metrics,
		documentControlRepo: p.DocumentControlRepo,
		draftRepo:           p.DraftRepo,
		searchProjection:    searchProjection,
		agentEvents:         p.AgentEvents,
	}
}

func (a *Service) SyncSearchProjection(
	ctx context.Context,
	doc *document.Document,
	contentText string,
) {
	if err := a.searchProjection.Upsert(ctx, doc, contentText); err != nil {
		a.metrics.Document.RecordSearchProjectionSync(false)
		a.logger.Warn("failed to sync document search projection",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
		return
	}
	a.metrics.Document.RecordSearchProjectionSync(true)
}

func (a *Service) GetDocumentControl(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*documentcontrol.DocumentControl, error) {
	return a.documentControlRepo.GetOrCreate(ctx, orgID, buID)
}

func (a *Service) UpsertDraft(
	ctx context.Context,
	doc *document.Document,
	kind string,
	intelligence *documentanalysis.DocumentIntelligenceAnalysis,
	isReady bool,
) (document.ShipmentDraftStatus, error) {
	draftState := documentshipmentdraft.StatusUnavailable
	draftStatus := document.ShipmentDraftStatusUnavailable
	if isReady {
		draftState = documentshipmentdraft.StatusReady
		draftStatus = document.ShipmentDraftStatusReady
	}

	if _, err := a.draftRepo.Upsert(ctx, &documentshipmentdraft.DocumentShipmentDraft{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		Status:         draftState,
		DocumentKind:   kind,
		Confidence:     intelligence.OverallConfidence,
		DraftData:      intelligence.ToMap(),
	}); err != nil {
		return "", err
	}

	agenteventports.PublishAgentEvent(ctx, a.agentEvents, agenteventports.AgentEvent{
		Kind:      agenttrigger.EventDocumentExtracted,
		SubjectID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: doc.OrganizationID,
			BuID:  doc.BusinessUnitID,
		},
	})

	return draftStatus, nil
}
