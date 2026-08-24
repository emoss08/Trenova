package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
	DB                ports.DBConnection
	SettlementRepo    repositories.CarrierSettlementRepository
	BatchRepo         repositories.CarrierSettlementBatchRepository
	CostEventRepo     repositories.CarrierCostEventRepository
	LedgerRepo        repositories.CarrierLedgerRepository
	InvoiceMatchRepo  repositories.CarrierInvoiceMatchRepository
	SettlementControl repositories.CarrierSettlementControlRepository
	CarrierRepo       repositories.CarrierRepository
	AssignmentRepo    repositories.CarrierAssignmentRepository
	ShipmentRepo      repositories.ShipmentRepository
	EDIInvoiceRepo    repositories.EDICarrierInvoiceRepository
	AccountingRepo    repositories.AccountingControlRepository
	JournalRepo       repositories.JournalPostingRepository
	FiscalPeriodRepo  repositories.FiscalPeriodRepository
	Generator         seqgen.Generator
	AuditService      serviceports.AuditService
	Realtime          serviceports.RealtimeService
}

type Service struct {
	l                 *zap.Logger
	db                ports.DBConnection
	settlementRepo    repositories.CarrierSettlementRepository
	batchRepo         repositories.CarrierSettlementBatchRepository
	costEventRepo     repositories.CarrierCostEventRepository
	ledgerRepo        repositories.CarrierLedgerRepository
	invoiceMatchRepo  repositories.CarrierInvoiceMatchRepository
	settlementControl repositories.CarrierSettlementControlRepository
	carrierRepo       repositories.CarrierRepository
	assignmentRepo    repositories.CarrierAssignmentRepository
	shipmentRepo      repositories.ShipmentRepository
	ediInvoiceRepo    repositories.EDICarrierInvoiceRepository
	accountingRepo    repositories.AccountingControlRepository
	journalRepo       repositories.JournalPostingRepository
	fiscalPeriodRepo  repositories.FiscalPeriodRepository
	generator         seqgen.Generator
	auditService      serviceports.AuditService
	realtime          serviceports.RealtimeService
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
		journalRepo:       p.JournalRepo,
		fiscalPeriodRepo:  p.FiscalPeriodRepo,
		generator:         p.Generator,
		auditService:      p.AuditService,
		realtime:          p.Realtime,
	}
}

func requireActor(actor *serviceports.RequestActor, operation string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(operation + " requires an authenticated user")
	}
	return nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListCarrierSettlementsRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error) {
	return s.settlementRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierSettlementConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error) {
	return s.settlementRepo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetCarrierSettlementByIDRequest,
) (*carriersettlement.CarrierSettlement, error) {
	return s.settlementRepo.GetByID(ctx, req)
}

func (s *Service) ListBatches(
	ctx context.Context,
	req *repositories.ListCarrierSettlementBatchesRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlementBatch], error) {
	return s.batchRepo.List(ctx, req)
}

func (s *Service) ListBatchesConnection(
	ctx context.Context,
	req *repositories.ListCarrierSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlementBatch], error) {
	return s.batchRepo.ListConnection(ctx, req)
}

func (s *Service) GetBatch(
	ctx context.Context,
	req repositories.GetCarrierSettlementBatchByIDRequest,
) (*carriersettlement.CarrierSettlementBatch, error) {
	return s.batchRepo.GetByID(ctx, req)
}

func (s *Service) ListCostEvents(
	ctx context.Context,
	req *repositories.ListCarrierCostEventsRequest,
) (*pagination.ListResult[*carriersettlement.CostEvent], error) {
	return s.costEventRepo.List(ctx, req)
}

func (s *Service) ListCostEventsConnection(
	ctx context.Context,
	req *repositories.ListCarrierCostEventConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CostEvent], error) {
	return s.costEventRepo.ListConnection(ctx, req)
}

func (s *Service) ListLedgerEntries(
	ctx context.Context,
	req *repositories.ListCarrierLedgerEntriesRequest,
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
		ActorType:      serviceports.PrincipalTypeUser,
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
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceCarrierSettlement,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log carrier settlement audit action", zap.Error(err))
	}
}
