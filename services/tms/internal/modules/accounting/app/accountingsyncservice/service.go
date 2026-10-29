package accountingsyncservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncactions"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncdrain"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncenqueue"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncledger"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncparty"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpayables"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncreads"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncsales"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncsetup"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncsweep"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.Transactor
	Connections       accountingconnectionports.AccountingConnectionRepository
	Records           accountingsyncports.AccountingSyncRecordRepository
	Backfills         accountingsyncports.AccountingBackfillRepository
	MappingService    accountingmappingports.AccountingMappingService
	ConnectionService accountingconnectionports.AccountingConnectionService
	Invoices          invoiceports.InvoiceRepository
	Adjustments       invoiceadjustmentports.InvoiceAdjustmentRepository
	Payments          customerpaymentports.CustomerPaymentRepository
	Organizations     tenantports.OrganizationRepository
	Controls          tenantports.AccountingControlRepository
	Payables          accountingpayablesports.AccountingPayablesSource
	Ledger            journalentryports.AccountingLedgerSource
	References        accountingmappingports.AccountingReferenceObjectRepository
	Rates             exchangerateports.ExchangeRateService
	Policy            *accountingcontrolpolicyservice.Service
	AuditService      auditports.AuditService
	Enqueuer          *accountingsyncenqueue.Enqueuer
	Dispatcher        accountingsyncports.AccountingSyncDispatcher `optional:"true"`
	Publisher         agentports.AgentEventPublisher               `optional:"true"`
	Watchtower        watchtowerports.WatchtowerProjector          `optional:"true"`
	Realtime          realtimeports.RealtimeService                `optional:"true"`
}

type Service struct {
	drain     *accountingsyncdrain.Service
	reads     *accountingsyncreads.Service
	setup     *accountingsyncsetup.Service
	actions   *accountingsyncactions.Service
	backfills *accountingsyncbackfill.Service
	sweep     *accountingsyncsweep.Service
}

var _ accountingsyncports.AccountingSyncService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	l := p.Logger.Named("service.accounting-sync")
	effects := accountingsynceffects.New(accountingsynceffects.Deps{
		Logger:        l,
		Records:       p.Records,
		Connections:   p.Connections,
		Organizations: p.Organizations,
		Controls:      p.Controls,
		Audit:         p.AuditService,
		Dispatcher:    p.Dispatcher,
		Publisher:     p.Publisher,
		Watchtower:    p.Watchtower,
		Realtime:      p.Realtime,
	})
	kit := accountingsyncpush.New(accountingsyncpush.Deps{
		Logger:         l,
		Records:        p.Records,
		MappingService: p.MappingService,
		Rates:          p.Rates,
		Policy:         p.Policy,
		Effects:        effects,
	})
	backfills := accountingsyncbackfill.New(accountingsyncbackfill.Deps{
		Records:    p.Records,
		Backfills:  p.Backfills,
		Enqueuer:   p.Enqueuer,
		Dispatcher: p.Dispatcher,
		Effects:    effects,
	})
	return &Service{
		drain: accountingsyncdrain.New(accountingsyncdrain.Deps{
			Logger:            l,
			Records:           p.Records,
			ConnectionService: p.ConnectionService,
			Controls:          p.Controls,
			Effects:           effects,
			Parties: accountingsyncparty.New(
				accountingsyncparty.Deps{Kit: kit, MappingService: p.MappingService},
			),
			Sales: accountingsyncsales.New(accountingsyncsales.Deps{
				Kit:         kit,
				Invoices:    p.Invoices,
				Adjustments: p.Adjustments,
				Payments:    p.Payments,
			}),
			Payables: accountingsyncpayables.New(
				accountingsyncpayables.Deps{Kit: kit, Payables: p.Payables},
			),
			Ledger: accountingsyncledger.New(accountingsyncledger.Deps{
				Kit:        kit,
				Ledger:     p.Ledger,
				References: p.References,
			}),
		}),
		reads: accountingsyncreads.New(accountingsyncreads.Deps{
			Connections: p.Connections,
			Records:     p.Records,
			Backfills:   p.Backfills,
		}),
		setup: accountingsyncsetup.New(accountingsyncsetup.Deps{
			Logger:         l,
			DB:             p.DB,
			Connections:    p.Connections,
			MappingService: p.MappingService,
			Enqueuer:       p.Enqueuer,
			Effects:        effects,
			Backfills:      backfills,
		}),
		actions: accountingsyncactions.New(
			accountingsyncactions.Deps{Records: p.Records, Effects: effects},
		),
		backfills: backfills,
		sweep: accountingsyncsweep.New(accountingsyncsweep.Deps{
			Records:  p.Records,
			Enqueuer: p.Enqueuer,
			Effects:  effects,
		}),
	}
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingsyncports.AccountingSyncSummary, error) {
	return s.reads.Summary(ctx, tenantInfo, integrationType)
}

func (s *Service) ListRecords(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncRecordsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	return s.reads.ListRecords(ctx, req)
}

func (s *Service) GetRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.reads.GetRecord(ctx, tenantInfo, id)
}

func (s *Service) ListAttempts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	return s.reads.ListAttempts(ctx, tenantInfo, recordID)
}

func (s *Service) ObjectStates(
	ctx context.Context,
	req *accountingsyncports.AccountingSyncObjectStatesRequest,
) (map[pulid.ID]*accountingsyncports.AccountingSyncObjectState, error) {
	return s.reads.ObjectStates(ctx, req)
}

func (s *Service) ListBackfills(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) ([]*accountingbackfill.AccountingBackfill, error) {
	return s.reads.ListBackfills(ctx, tenantInfo, integrationType)
}

func (s *Service) EnableSync(
	ctx context.Context,
	req *accountingsyncports.EnableAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.setup.EnableSync(ctx, req)
}

func (s *Service) ChooseMode(
	ctx context.Context,
	req *accountingsyncports.ChooseAccountingSyncModeRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.setup.ChooseMode(ctx, req)
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *accountingsyncports.UpdateAccountingSyncSettingsRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.setup.UpdateSettings(ctx, req)
}

func (s *Service) Pause(
	ctx context.Context,
	req *accountingsyncports.PauseAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.setup.Pause(ctx, req)
}

func (s *Service) Resume(
	ctx context.Context,
	req *accountingsyncports.PauseAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	return s.setup.Resume(ctx, req)
}

func (s *Service) Retry(
	ctx context.Context,
	req *accountingsyncports.RetryAccountingSyncRequest,
) (int64, error) {
	return s.actions.Retry(ctx, req)
}

func (s *Service) Release(
	ctx context.Context,
	req *accountingsyncports.ReleaseAccountingSyncRequest,
) (int64, error) {
	return s.actions.Release(ctx, req)
}

func (s *Service) Skip(
	ctx context.Context,
	req *accountingsyncports.SkipAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.actions.Skip(ctx, req)
}

func (s *Service) PlanRedate(
	ctx context.Context,
	req *accountingsyncports.RedateAccountingSyncRequest,
) (*accountingsyncports.AccountingSyncRedatePlan, error) {
	return s.actions.PlanRedate(ctx, req)
}

func (s *Service) Redate(
	ctx context.Context,
	req *accountingsyncports.RedateAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.actions.Redate(ctx, req)
}

func (s *Service) RequestBackfill(
	ctx context.Context,
	req *accountingsyncports.RequestAccountingBackfillRequest,
) (*accountingbackfill.AccountingBackfill, error) {
	return s.backfills.RequestBackfill(ctx, req)
}

func (s *Service) ChangeBackfill(
	ctx context.Context,
	req *accountingsyncports.ChangeAccountingBackfillRequest,
) (*accountingbackfill.AccountingBackfill, error) {
	return s.backfills.ChangeBackfill(ctx, req)
}

func (s *Service) BackfillStep(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	backfillID pulid.ID,
) (*accountingsyncports.AccountingBackfillStepResult, error) {
	return s.backfills.BackfillStep(ctx, tenantInfo, backfillID)
}

func (s *Service) Drain(
	ctx context.Context,
	req *accountingsyncports.DrainAccountingSyncRequest,
) (*accountingsyncports.AccountingSyncDrainResult, error) {
	return s.drain.Drain(ctx, req)
}

func (s *Service) ListDueConnections(
	ctx context.Context,
	limit int,
) ([]accountingsyncports.AccountingSyncDueConnection, error) {
	return s.sweep.ListDueConnections(ctx, limit)
}

func (s *Service) SafetyNet(
	ctx context.Context,
	ref accountingsyncports.AccountingSyncConnectionRef,
) (*accountingsyncports.AccountingSafetyNetResult, error) {
	return s.sweep.SafetyNet(ctx, ref)
}

func (s *Service) PurgeHistory(
	ctx context.Context,
) (*accountingsyncports.PurgeAccountingSyncHistoryResult, error) {
	return s.sweep.PurgeHistory(ctx)
}
