package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.DBConnection
	SettlementRepo    carriersettlementports.CarrierSettlementRepository
	BatchRepo         carriersettlementports.CarrierSettlementBatchRepository
	CostEventRepo     carriersettlementports.CarrierCostEventRepository
	LedgerRepo        carriersettlementports.CarrierLedgerRepository
	InvoiceMatchRepo  carriersettlementports.CarrierInvoiceMatchRepository
	SettlementControl tenantports.CarrierSettlementControlRepository
	CarrierRepo       carrierports.CarrierRepository
	AssignmentRepo    shipmentports.CarrierAssignmentRepository
	ShipmentRepo      shipmentports.ShipmentRepository
	EDIInvoiceRepo    ediports.EDICarrierInvoiceRepository
	AccountingRepo    tenantports.AccountingControlRepository
	Stamper           *exchangeratestamp.Stamper
	JournalRepo       journalentryports.JournalPostingRepository
	FiscalPeriodRepo  fiscalperiodports.FiscalPeriodRepository
	Generator         seqgen.Generator
	AuditService      auditports.AuditService
	Realtime          agentports.RealtimeService
	Sync              accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l                 *zap.Logger
	db                kernelports.DBConnection
	settlementRepo    carriersettlementports.CarrierSettlementRepository
	batchRepo         carriersettlementports.CarrierSettlementBatchRepository
	costEventRepo     carriersettlementports.CarrierCostEventRepository
	ledgerRepo        carriersettlementports.CarrierLedgerRepository
	invoiceMatchRepo  carriersettlementports.CarrierInvoiceMatchRepository
	settlementControl tenantports.CarrierSettlementControlRepository
	carrierRepo       carrierports.CarrierRepository
	assignmentRepo    shipmentports.CarrierAssignmentRepository
	shipmentRepo      shipmentports.ShipmentRepository
	ediInvoiceRepo    ediports.EDICarrierInvoiceRepository
	accountingRepo    tenantports.AccountingControlRepository
	stamper           *exchangeratestamp.Stamper
	journalRepo       journalentryports.JournalPostingRepository
	fiscalPeriodRepo  fiscalperiodports.FiscalPeriodRepository
	generator         seqgen.Generator
	auditService      auditports.AuditService
	realtime          agentports.RealtimeService
	sync              accountingsyncports.AccountingSyncEnqueuer
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                 p.Logger.Named("service.carrier-settlement"),
		db:                p.DB,
		settlementRepo:    p.SettlementRepo,
		batchRepo:         p.BatchRepo,
		costEventRepo:     p.CostEventRepo,
		ledgerRepo:        p.LedgerRepo,
		invoiceMatchRepo:  p.InvoiceMatchRepo,
		settlementControl: p.SettlementControl,
		carrierRepo:       p.CarrierRepo,
		assignmentRepo:    p.AssignmentRepo,
		shipmentRepo:      p.ShipmentRepo,
		ediInvoiceRepo:    p.EDIInvoiceRepo,
		accountingRepo:    p.AccountingRepo,
		stamper:           p.Stamper,
		journalRepo:       p.JournalRepo,
		fiscalPeriodRepo:  p.FiscalPeriodRepo,
		generator:         p.Generator,
		auditService:      p.AuditService,
		realtime:          p.Realtime,
		sync:              p.Sync,
	}
}

func (s *Service) queueSync(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
	source accountingsync.SyncSourceEvent,
) error {
	return accountingsyncports.EnqueueAccountingSync(
		ctx,
		s.sync,
		accountingsyncports.SettlementSync(&accountingsyncports.SettlementSyncRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			ObjectType:  objectType,
			ObjectID:    entity.ID,
			Number:      entity.SettlementNumber,
			Operation:   operation,
			SourceEvent: source,
			PostedAt:    entity.PostedAt,
		}),
	)
}

func requireActor(actor *permissionports.RequestActor, operation string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("{0} requires an authenticated user", operation)
	}
	return nil
}

func (s *Service) List(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementsRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error) {
	return s.settlementRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error) {
	return s.settlementRepo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req carriersettlementports.GetCarrierSettlementByIDRequest,
) (*carriersettlement.CarrierSettlement, error) {
	return s.settlementRepo.GetByID(ctx, req)
}

func (s *Service) ListBatches(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementBatchesRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlementBatch], error) {
	return s.batchRepo.List(ctx, req)
}

func (s *Service) ListBatchesConnection(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlementBatch], error) {
	return s.batchRepo.ListConnection(ctx, req)
}

func (s *Service) GetBatch(
	ctx context.Context,
	req carriersettlementports.GetCarrierSettlementBatchByIDRequest,
) (*carriersettlement.CarrierSettlementBatch, error) {
	return s.batchRepo.GetByID(ctx, req)
}

func (s *Service) ListCostEvents(
	ctx context.Context,
	req *carriersettlementports.ListCarrierCostEventsRequest,
) (*pagination.ListResult[*carriersettlement.CostEvent], error) {
	return s.costEventRepo.List(ctx, req)
}

func (s *Service) ListCostEventsConnection(
	ctx context.Context,
	req *carriersettlementports.ListCarrierCostEventConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CostEvent], error) {
	return s.costEventRepo.ListConnection(ctx, req)
}

func (s *Service) ListLedgerEntries(
	ctx context.Context,
	req *carriersettlementports.ListCarrierLedgerEntriesRequest,
) ([]*carriersettlement.LedgerEntry, error) {
	return s.ledgerRepo.ListByCarrier(ctx, req)
}

func (s *Service) publishRealtimeInvalidation(
	ctx context.Context,
	resource string,
	action permission.Operation,
	recordID, orgID, buID, userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ActorUserID:    userID,
		ActorType:      permissionports.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(action),
		RecordID:       recordID,
	})
	if err != nil {
		s.l.Warn("failed to publish realtime invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) publishCostEventInvalidation(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
	action permission.Operation,
	userID pulid.ID,
) {
	if entity == nil {
		return
	}
	s.publishRealtimeInvalidation(
		ctx,
		"carrier_cost_event",
		action,
		entity.ID,
		entity.OrganizationID,
		entity.BusinessUnitID,
		userID,
	)
}

func (s *Service) logSettlementAudit(
	ctx context.Context,
	current, previous *carriersettlement.CarrierSettlement,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	s.publishRealtimeInvalidation(
		ctx,
		permission.ResourceCarrierSettlement.String(),
		operation,
		current.ID,
		current.OrganizationID,
		current.BusinessUnitID,
		userID,
	)
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceCarrierSettlement,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log carrier settlement audit action", zap.Error(err))
	}
}
