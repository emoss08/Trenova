package accountingsyncservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/core/services/accountingconnlookup"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultDrainLimit   = 25
	dependencyWait      = time.Minute
	safetyNetPage       = 200
	backfillPage        = 200
	purgeBatch          = 1000
	maxStateObjects     = 500
	maxAttemptsListed   = 50
	maxBackfillsListed  = 20
	attentionGroups     = 20
	pausedTooLongAfter  = 24 * time.Hour
	maxRetryIDs         = 500
	systemNoteMaxLength = 500
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.DBConnection
	Connections       accountingsyncports.AccountingConnectionRepository
	Records           accountingsyncports.AccountingSyncRecordRepository
	Backfills         accountingsyncports.AccountingBackfillRepository
	Mappings          accountingsyncports.AccountingMappingRepository
	MappingService    accountingsyncports.AccountingMappingService
	ConnectionService accountingsyncports.AccountingConnectionService
	Invoices          invoiceports.InvoiceRepository
	Adjustments       invoiceadjustmentports.InvoiceAdjustmentRepository
	Payments          customerpaymentports.CustomerPaymentRepository
	Organizations     tenantports.OrganizationRepository
	Controls          tenantports.AccountingControlRepository
	Payables          accountingsyncports.AccountingPayablesSource
	Ledger            journalentryports.AccountingLedgerSource
	References        accountingsyncports.AccountingReferenceObjectRepository
	Rates             exchangerateports.ExchangeRateService
	Policy            *accountingcontrolpolicyservice.Service
	AuditService      auditports.AuditService
	Enqueuer          *Enqueuer
	Dispatcher        accountingsyncports.AccountingSyncDispatcher `optional:"true"`
	Publisher         agentports.AgentEventPublisher               `optional:"true"`
	Watchtower        watchtowerports.WatchtowerProjector          `optional:"true"`
	Realtime          agentports.RealtimeService                   `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	db             kernelports.DBConnection
	connections    accountingsyncports.AccountingConnectionRepository
	records        accountingsyncports.AccountingSyncRecordRepository
	backfills      accountingsyncports.AccountingBackfillRepository
	mappings       accountingsyncports.AccountingMappingRepository
	mappingService accountingsyncports.AccountingMappingService
	connService    accountingsyncports.AccountingConnectionService
	invoices       invoiceports.InvoiceRepository
	adjustments    invoiceadjustmentports.InvoiceAdjustmentRepository
	payments       customerpaymentports.CustomerPaymentRepository
	organizations  tenantports.OrganizationRepository
	controls       tenantports.AccountingControlRepository
	payables       accountingsyncports.AccountingPayablesSource
	ledger         journalentryports.AccountingLedgerSource
	references     accountingsyncports.AccountingReferenceObjectRepository
	rates          exchangerateports.ExchangeRateService
	policy         *accountingcontrolpolicyservice.Service
	audit          auditports.AuditService
	enqueuer       *Enqueuer
	dispatcher     accountingsyncports.AccountingSyncDispatcher
	publisher      agentports.AgentEventPublisher
	watchtower     watchtowerports.WatchtowerProjector
	realtime       agentports.RealtimeService
}

var _ accountingsyncports.AccountingSyncService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.accounting-sync"),
		db:             p.DB,
		connections:    p.Connections,
		records:        p.Records,
		backfills:      p.Backfills,
		mappings:       p.Mappings,
		mappingService: p.MappingService,
		connService:    p.ConnectionService,
		invoices:       p.Invoices,
		adjustments:    p.Adjustments,
		payments:       p.Payments,
		organizations:  p.Organizations,
		controls:       p.Controls,
		payables:       p.Payables,
		ledger:         p.Ledger,
		references:     p.References,
		rates:          p.Rates,
		policy:         p.Policy,
		audit:          p.AuditService,
		enqueuer:       p.Enqueuer,
		dispatcher:     p.Dispatcher,
		publisher:      p.Publisher,
		watchtower:     p.Watchtower,
		realtime:       p.Realtime,
	}
}

func (s *Service) connectionFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingConnection, error) {
	return accountingconnlookup.ByType(ctx, s.connections, tenantInfo, typ)
}

func (s *Service) connectionByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingConnection, error) {
	return s.connections.GetByID(ctx, accountingsyncports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) kick(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) {
	if s.dispatcher == nil {
		return
	}
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		if err := s.dispatcher.Kick(committedCtx, tenantInfo, connectionID); err != nil {
			s.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
		}
	})
}
