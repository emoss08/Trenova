package accountingsyncflows

import (
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
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type wiring struct {
	Logger        *zap.Logger
	DB            kernelports.Transactor
	Connections   accountingconnectionports.AccountingConnectionRepository
	Records       accountingsyncports.AccountingSyncRecordRepository
	Backfills     accountingsyncports.AccountingBackfillRepository
	Decisions     accountingmappingports.AccountingMappingDecisionService
	Creator       accountingmappingports.AccountingReferenceCreator
	Scorer        accountingmappingports.AccountingMappingScorer
	Sessions      accountingconnectionports.AccountingSessionProvider
	Invoices      invoiceports.InvoiceRepository
	Adjustments   invoiceadjustmentports.InvoiceAdjustmentRepository
	Payments      customerpaymentports.CustomerPaymentRepository
	Organizations tenantports.OrganizationRepository
	Controls      tenantports.AccountingControlRepository
	Payables      accountingpayablesports.AccountingPayablesSource
	Ledger        journalentryports.AccountingLedgerSource
	References    accountingmappingports.AccountingReferenceObjectRepository
	Rates         exchangerateports.ExchangeRateService
	Policy        *accountingcontrolpolicyservice.Service
	AuditService  auditports.AuditService
	Enqueuer      *accountingsyncenqueue.Enqueuer
	Dispatcher    accountingsyncports.AccountingSyncDispatcher
	Publisher     agenteventports.AgentEventPublisher
	Watchtower    watchtowerports.WatchtowerProjector
	Realtime      realtimeports.RealtimeService
}

type services struct {
	drain     *accountingsyncdrain.Service
	reads     *accountingsyncreads.Service
	setup     *accountingsyncsetup.Service
	actions   *accountingsyncactions.Service
	backfills *accountingsyncbackfill.Service
	sweep     *accountingsyncsweep.Service
}

func wire(p *wiring) services {
	effects := accountingsynceffects.New(accountingsynceffects.Params{
		Logger:        p.Logger,
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
	kit := accountingsyncpush.New(accountingsyncpush.Params{
		Logger:    p.Logger,
		Records:   p.Records,
		Decisions: p.Decisions,
		Rates:     p.Rates,
		Policy:    p.Policy,
		Effects:   effects,
	})
	backfills := accountingsyncbackfill.New(accountingsyncbackfill.Params{
		Records:    p.Records,
		Backfills:  p.Backfills,
		Enqueuer:   p.Enqueuer,
		Dispatcher: p.Dispatcher,
		Effects:    effects,
	})
	return services{
		drain: accountingsyncdrain.New(accountingsyncdrain.Params{
			Logger:   p.Logger,
			Records:  p.Records,
			Sessions: p.Sessions,
			Controls: p.Controls,
			Effects:  effects,
			Parties: accountingsyncparty.New(
				accountingsyncparty.Params{Kit: kit, Creator: p.Creator},
			),
			Sales: accountingsyncsales.New(accountingsyncsales.Params{
				Kit:         kit,
				Invoices:    p.Invoices,
				Adjustments: p.Adjustments,
				Payments:    p.Payments,
			}),
			Payables: accountingsyncpayables.New(
				accountingsyncpayables.Params{Kit: kit, Payables: p.Payables},
			),
			Ledger: accountingsyncledger.New(accountingsyncledger.Params{
				Kit:        kit,
				Ledger:     p.Ledger,
				References: p.References,
			}),
		}),
		reads: accountingsyncreads.New(accountingsyncreads.Params{
			Connections: p.Connections,
			Records:     p.Records,
			Backfills:   p.Backfills,
		}),
		setup: accountingsyncsetup.New(accountingsyncsetup.Params{
			Logger:      p.Logger,
			DB:          p.DB,
			Connections: p.Connections,
			Scorer:      p.Scorer,
			Enqueuer:    p.Enqueuer,
			Effects:     effects,
			Backfills:   backfills,
		}),
		actions: accountingsyncactions.New(
			accountingsyncactions.Params{Records: p.Records, Effects: effects},
		),
		backfills: backfills,
		sweep: accountingsyncsweep.New(accountingsyncsweep.Params{
			Records:  p.Records,
			Enqueuer: p.Enqueuer,
			Effects:  effects,
		}),
	}
}
