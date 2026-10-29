package driversettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementjournal"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementrecurring"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	DB                kernelports.Transactor
	SettlementRepo    driversettlementports.DriverSettlementRepository
	BatchRepo         driversettlementports.SettlementBatchRepository
	PayEventRepo      driversettlementports.PayEventRepository
	AssignmentRepo    driverpayports.WorkerPayAssignmentRepository
	DeductionRepo     driverpayports.RecurringDeductionRepository
	EarningRepo       driverpayports.RecurringEarningRepository
	PayCodeRepo       driverpayports.PayCodeRepository
	AdvanceRepo       driverpayports.PayAdvanceRepository
	EscrowRepo        driverpayports.EscrowAccountRepository
	SettlementControl tenantports.SettlementControlRepository
	AccountingRepo    tenantports.AccountingControlRepository
	Stamper           *exchangeratestamp.Stamper
	JournalRepo       journalentryports.JournalPostingRepository
	FiscalPeriodRepo  fiscalperiodports.FiscalPeriodRepository
	Generator         seqgen.Generator
	AuditService      auditports.AuditService
	Realtime          realtimeports.RealtimeService
	DriverNotify      *drivernotificationservice.Service
	Sync              accountingsyncports.AccountingSyncEnqueuer `optional:"true"`
}

type Service struct {
	l                 *zap.Logger
	db                kernelports.Transactor
	settlementRepo    driversettlementports.DriverSettlementRepository
	batchRepo         driversettlementports.SettlementBatchRepository
	payEventRepo      driversettlementports.PayEventRepository
	assignmentRepo    driverpayports.WorkerPayAssignmentRepository
	payCodeRepo       driverpayports.PayCodeRepository
	advanceRepo       driverpayports.PayAdvanceRepository
	escrowRepo        driverpayports.EscrowAccountRepository
	settlementControl tenantports.SettlementControlRepository
	stamper           *exchangeratestamp.Stamper
	generator         seqgen.Generator
	auditService      auditports.AuditService
	realtime          realtimeports.RealtimeService
	driverNotify      *drivernotificationservice.Service
	sync              accountingsyncports.AccountingSyncEnqueuer
	recurring         *driversettlementrecurring.Cycle
	journals          *driversettlementjournal.Writer
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:                 p.Logger.Named("service.driver-settlement"),
		db:                p.DB,
		settlementRepo:    p.SettlementRepo,
		batchRepo:         p.BatchRepo,
		payEventRepo:      p.PayEventRepo,
		assignmentRepo:    p.AssignmentRepo,
		payCodeRepo:       p.PayCodeRepo,
		advanceRepo:       p.AdvanceRepo,
		escrowRepo:        p.EscrowRepo,
		settlementControl: p.SettlementControl,
		stamper:           p.Stamper,
		generator:         p.Generator,
		auditService:      p.AuditService,
		realtime:          p.Realtime,
		driverNotify:      p.DriverNotify,
		sync:              p.Sync,
		recurring: driversettlementrecurring.New(&driversettlementrecurring.Params{
			SettlementRepo: p.SettlementRepo,
			DeductionRepo:  p.DeductionRepo,
			EarningRepo:    p.EarningRepo,
			AdvanceRepo:    p.AdvanceRepo,
			EscrowRepo:     p.EscrowRepo,
		}),
		journals: driversettlementjournal.New(&driversettlementjournal.Params{
			AccountingRepo:   p.AccountingRepo,
			PayCodeRepo:      p.PayCodeRepo,
			JournalRepo:      p.JournalRepo,
			FiscalPeriodRepo: p.FiscalPeriodRepo,
			Generator:        p.Generator,
		}),
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

func (s *Service) PublishRealtimeInvalidation(
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

func (s *Service) PublishPayEventInvalidation(
	ctx context.Context,
	entity *driversettlement.PayEvent,
	action permission.Operation,
	userID pulid.ID,
) {
	if entity == nil {
		return
	}
	s.PublishRealtimeInvalidation(
		ctx,
		"driver_pay_event",
		action,
		entity.ID,
		entity.OrganizationID,
		entity.BusinessUnitID,
		userID,
	)
}

func (s *Service) LogSettlementAudit(
	ctx context.Context,
	current, previous *driversettlement.Settlement,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	s.PublishRealtimeInvalidation(
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
