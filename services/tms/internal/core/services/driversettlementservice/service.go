package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
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

	Logger              *zap.Logger
	DB                  ports.DBConnection
	SettlementRepo      repositories.DriverSettlementRepository
	DisputeRepo         repositories.SettlementDisputeRepository
	ExpenseRepo         repositories.DriverExpenseRepository
	BatchRepo           repositories.SettlementBatchRepository
	PayEventRepo        repositories.PayEventRepository
	AssignmentRepo      repositories.WorkerPayAssignmentRepository
	DeductionRepo       repositories.RecurringDeductionRepository
	EarningRepo         repositories.RecurringEarningRepository
	PayCodeRepo         repositories.PayCodeRepository
	AdvanceRepo         repositories.PayAdvanceRepository
	EscrowRepo          repositories.EscrowAccountRepository
	SettlementControl   repositories.SettlementControlRepository
	DashControlRepo     repositories.DashControlRepository
	ShipmentRepo        repositories.ShipmentRepository
	WorkerRepo          repositories.WorkerRepository
	AccountingRepo      repositories.AccountingControlRepository
	JournalRepo         repositories.JournalPostingRepository
	FiscalPeriodRepo    repositories.FiscalPeriodRepository
	Generator           seqgen.Generator
	PayService          *driverpayservice.Service
	AuditService        serviceports.AuditService
	Realtime            serviceports.RealtimeService
	DriverNotify        *drivernotificationservice.Service
	NotificationService *notificationservice.Service
}

type Service struct {
	l                   *zap.Logger
	db                  ports.DBConnection
	settlementRepo      repositories.DriverSettlementRepository
	disputeRepo         repositories.SettlementDisputeRepository
	expenseRepo         repositories.DriverExpenseRepository
	batchRepo           repositories.SettlementBatchRepository
	payEventRepo        repositories.PayEventRepository
	assignmentRepo      repositories.WorkerPayAssignmentRepository
	deductionRepo       repositories.RecurringDeductionRepository
	earningRepo         repositories.RecurringEarningRepository
	payCodeRepo         repositories.PayCodeRepository
	advanceRepo         repositories.PayAdvanceRepository
	escrowRepo          repositories.EscrowAccountRepository
	settlementControl   repositories.SettlementControlRepository
	dashControlRepo     repositories.DashControlRepository
	shipmentRepo        repositories.ShipmentRepository
	workerRepo          repositories.WorkerRepository
	accountingRepo      repositories.AccountingControlRepository
	journalRepo         repositories.JournalPostingRepository
	fiscalPeriodRepo    repositories.FiscalPeriodRepository
	generator           seqgen.Generator
	payService          *driverpayservice.Service
	auditService        serviceports.AuditService
	realtime            serviceports.RealtimeService
	driverNotify        *drivernotificationservice.Service
	notificationService *notificationservice.Service
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                   p.Logger.Named("service.driver-settlement"),
		db:                  p.DB,
		settlementRepo:      p.SettlementRepo,
		disputeRepo:         p.DisputeRepo,
		expenseRepo:         p.ExpenseRepo,
		batchRepo:           p.BatchRepo,
		payEventRepo:        p.PayEventRepo,
		assignmentRepo:      p.AssignmentRepo,
		deductionRepo:       p.DeductionRepo,
		earningRepo:         p.EarningRepo,
		payCodeRepo:         p.PayCodeRepo,
		advanceRepo:         p.AdvanceRepo,
		escrowRepo:          p.EscrowRepo,
		settlementControl:   p.SettlementControl,
		dashControlRepo:     p.DashControlRepo,
		shipmentRepo:        p.ShipmentRepo,
		workerRepo:          p.WorkerRepo,
		accountingRepo:      p.AccountingRepo,
		journalRepo:         p.JournalRepo,
		fiscalPeriodRepo:    p.FiscalPeriodRepo,
		generator:           p.Generator,
		payService:          p.PayService,
		auditService:        p.AuditService,
		realtime:            p.Realtime,
		driverNotify:        p.DriverNotify,
		notificationService: p.NotificationService,
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
	req *repositories.ListDriverSettlementsRequest,
) (*pagination.ListResult[*driversettlement.Settlement], error) {
	return s.settlementRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListDriverSettlementConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Settlement], error) {
	return s.settlementRepo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	return s.settlementRepo.GetByID(ctx, req)
}

func (s *Service) ListBatches(
	ctx context.Context,
	req *repositories.ListSettlementBatchesRequest,
) (*pagination.ListResult[*driversettlement.SettlementBatch], error) {
	return s.batchRepo.List(ctx, req)
}

func (s *Service) ListBatchesConnection(
	ctx context.Context,
	req *repositories.ListSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error) {
	return s.batchRepo.ListConnection(ctx, req)
}

func (s *Service) GetBatch(
	ctx context.Context,
	req repositories.GetSettlementBatchByIDRequest,
) (*driversettlement.SettlementBatch, error) {
	return s.batchRepo.GetByID(ctx, req)
}

func (s *Service) ListPayEvents(
	ctx context.Context,
	req *repositories.ListPayEventsRequest,
) (*pagination.ListResult[*driversettlement.PayEvent], error) {
	return s.payEventRepo.List(ctx, req)
}

func (s *Service) ListPayEventsConnection(
	ctx context.Context,
	req *repositories.ListPayEventConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.PayEvent], error) {
	return s.payEventRepo.ListConnection(ctx, req)
}

type WorkerEarningsSummary struct {
	WorkerID            pulid.ID `json:"workerId"`
	AccruedEventCount   int      `json:"accruedEventCount"`
	AccruedGrossMinor   int64    `json:"accruedGrossMinor"`
	OutstandingAdvances int64    `json:"outstandingAdvances"`
	EscrowBalanceMinor  int64    `json:"escrowBalanceMinor"`
}

func (s *Service) GetWorkerEarningsSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*WorkerEarningsSummary, error) {
	totals, err := s.payEventRepo.GetAccruedTotalsForWorker(
		ctx,
		repositories.GetAccruedTotalsForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err != nil {
		return nil, err
	}
	summary := &WorkerEarningsSummary{
		WorkerID:          workerID,
		AccruedEventCount: totals.EventCount,
		AccruedGrossMinor: totals.GrossAmountMinor,
	}

	advances, err := s.advanceRepo.ListOutstandingForWorker(
		ctx,
		repositories.ListOutstandingAdvancesForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, advance := range advances {
		summary.OutstandingAdvances += advance.OutstandingMinor()
	}

	escrow, err := s.escrowRepo.GetActiveForWorker(
		ctx,
		repositories.GetActiveEscrowAccountForWorkerRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err == nil && escrow != nil {
		summary.EscrowBalanceMinor = escrow.BalanceMinor
	}

	return summary, nil
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

func (s *Service) publishPayEventInvalidation(
	ctx context.Context,
	entity *driversettlement.PayEvent,
	action permission.Operation,
	userID pulid.ID,
) {
	if entity == nil {
		return
	}
	s.publishRealtimeInvalidation(
		ctx,
		"driver_pay_event",
		action,
		entity.ID,
		entity.OrganizationID,
		entity.BusinessUnitID,
		userID,
	)
}

func (s *Service) logSettlementAudit(
	ctx context.Context,
	current, previous *driversettlement.Settlement,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	s.publishRealtimeInvalidation(
		ctx,
		permission.ResourceDriverSettlement.String(),
		operation,
		current.ID,
		current.OrganizationID,
		current.BusinessUnitID,
		userID,
	)
	params := &serviceports.LogActionParams{
		Resource:       permission.ResourceDriverSettlement,
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
		s.l.Error("failed to log driver settlement audit action", zap.Error(err))
	}
}
