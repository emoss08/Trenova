package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
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

	Logger              *zap.Logger
	DB                  kernelports.DBConnection
	SettlementRepo      driversettlementports.DriverSettlementRepository
	DisputeRepo         driversettlementports.SettlementDisputeRepository
	ExpenseRepo         driverpayports.DriverExpenseRepository
	BatchRepo           driversettlementports.SettlementBatchRepository
	PayEventRepo        driversettlementports.PayEventRepository
	AssignmentRepo      driverpayports.WorkerPayAssignmentRepository
	DeductionRepo       driverpayports.RecurringDeductionRepository
	EarningRepo         driverpayports.RecurringEarningRepository
	PayCodeRepo         driverpayports.PayCodeRepository
	AdvanceRepo         driverpayports.PayAdvanceRepository
	EscrowRepo          driverpayports.EscrowAccountRepository
	SettlementControl   tenantports.SettlementControlRepository
	DashControlRepo     tenantports.DashControlRepository
	ShipmentRepo        shipmentports.ShipmentRepository
	WorkerRepo          workerports.WorkerRepository
	AccountingRepo      tenantports.AccountingControlRepository
	Stamper             *exchangeratestamp.Stamper
	JournalRepo         journalentryports.JournalPostingRepository
	FiscalPeriodRepo    fiscalperiodports.FiscalPeriodRepository
	Generator           seqgen.Generator
	PayService          *driverpayservice.Service
	AuditService        auditports.AuditService
	Realtime            agentports.RealtimeService
	DriverNotify        *drivernotificationservice.Service
	NotificationService *notificationservice.Service
	Sync                accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	db                  kernelports.DBConnection
	settlementRepo      driversettlementports.DriverSettlementRepository
	disputeRepo         driversettlementports.SettlementDisputeRepository
	expenseRepo         driverpayports.DriverExpenseRepository
	batchRepo           driversettlementports.SettlementBatchRepository
	payEventRepo        driversettlementports.PayEventRepository
	assignmentRepo      driverpayports.WorkerPayAssignmentRepository
	deductionRepo       driverpayports.RecurringDeductionRepository
	earningRepo         driverpayports.RecurringEarningRepository
	payCodeRepo         driverpayports.PayCodeRepository
	advanceRepo         driverpayports.PayAdvanceRepository
	escrowRepo          driverpayports.EscrowAccountRepository
	settlementControl   tenantports.SettlementControlRepository
	dashControlRepo     tenantports.DashControlRepository
	shipmentRepo        shipmentports.ShipmentRepository
	workerRepo          workerports.WorkerRepository
	accountingRepo      tenantports.AccountingControlRepository
	stamper             *exchangeratestamp.Stamper
	journalRepo         journalentryports.JournalPostingRepository
	fiscalPeriodRepo    fiscalperiodports.FiscalPeriodRepository
	generator           seqgen.Generator
	payService          *driverpayservice.Service
	auditService        auditports.AuditService
	realtime            agentports.RealtimeService
	driverNotify        *drivernotificationservice.Service
	notificationService *notificationservice.Service
	sync                accountingsyncports.AccountingSyncEnqueuer
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
		stamper:             p.Stamper,
		journalRepo:         p.JournalRepo,
		fiscalPeriodRepo:    p.FiscalPeriodRepo,
		generator:           p.Generator,
		payService:          p.PayService,
		auditService:        p.AuditService,
		realtime:            p.Realtime,
		driverNotify:        p.DriverNotify,
		notificationService: p.NotificationService,
		sync:                p.Sync,
	}
}

func (s *Service) queueSync(
	ctx context.Context,
	entity *driversettlement.Settlement,
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
	source accountingsync.SyncSourceEvent,
) error {
	if entity.Classification != driverpay.PayeeClassificationOwnerOperator {
		return nil
	}
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
	req *driversettlementports.ListDriverSettlementsRequest,
) (*pagination.ListResult[*driversettlement.Settlement], error) {
	return s.settlementRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListDriverSettlementConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Settlement], error) {
	return s.settlementRepo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req driversettlementports.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	return s.settlementRepo.GetByID(ctx, req)
}

func (s *Service) ListBatches(
	ctx context.Context,
	req *driversettlementports.ListSettlementBatchesRequest,
) (*pagination.ListResult[*driversettlement.SettlementBatch], error) {
	return s.batchRepo.List(ctx, req)
}

func (s *Service) ListBatchesConnection(
	ctx context.Context,
	req *driversettlementports.ListSettlementBatchConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error) {
	return s.batchRepo.ListConnection(ctx, req)
}

func (s *Service) GetBatch(
	ctx context.Context,
	req driversettlementports.GetSettlementBatchByIDRequest,
) (*driversettlement.SettlementBatch, error) {
	return s.batchRepo.GetByID(ctx, req)
}

func (s *Service) ListPayEvents(
	ctx context.Context,
	req *driversettlementports.ListPayEventsRequest,
) (*pagination.ListResult[*driversettlement.PayEvent], error) {
	return s.payEventRepo.List(ctx, req)
}

func (s *Service) ListPayEventsConnection(
	ctx context.Context,
	req *driversettlementports.ListPayEventConnectionRequest,
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
		driversettlementports.GetAccruedTotalsForWorkerRequest{
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
		driverpayports.ListOutstandingAdvancesForWorkerRequest{
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
		driverpayports.GetActiveEscrowAccountForWorkerRequest{
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
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDriverSettlement,
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
		s.l.Error("failed to log driver settlement audit action", zap.Error(err))
	}
}
