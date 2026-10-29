package accountingmappingflows

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingcreate"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingdecisions"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingreads"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscoring"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingtargets"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type wiring struct {
	Logger             *zap.Logger
	DB                 kernelports.Transactor
	Connections        accountingconnectionports.AccountingConnectionRepository
	References         accountingmappingports.AccountingReferenceObjectRepository
	Mappings           accountingmappingports.AccountingMappingRepository
	Sessions           accountingconnectionports.AccountingSessionProvider
	AccountingControls tenantports.AccountingControlRepository
	GLAccounts         glaccountports.GLAccountRepository
	Customers          customerports.CustomerRepository
	Carriers           carrierports.CarrierRepository
	Accessorials       accessorialchargeports.AccessorialChargeRepository
	AuditService       auditports.AuditService
	Completion         llmports.CompletionService
	Refresher          accountingconnectionports.AccountingReferenceRefresher
	Realtime           realtimeports.RealtimeService
	SyncRecords        accountingsyncports.AccountingSyncRecordRepository
	Dispatcher         accountingsyncports.AccountingSyncDispatcher
	Workers            workerports.WorkerRepository
	SettlementControls tenantports.CarrierSettlementControlRepository
	PayCodes           driverpayports.PayCodeRepository
	Ledger             journalentryports.AccountingLedgerSource
}

type services struct {
	kit       *accountingmappingkit.Kit
	reads     *accountingmappingreads.Service
	scoring   *accountingmappingscoring.Service
	creator   *accountingmappingcreate.Service
	decisions *accountingmappingdecisions.Service
}

func wire(p *wiring) services {
	kit := accountingmappingkit.New(accountingmappingkit.Params{
		Logger:      p.Logger,
		Audit:       p.AuditService,
		Realtime:    p.Realtime,
		SyncRecords: p.SyncRecords,
		Dispatcher:  p.Dispatcher,
	})
	catalog := accountingmappingtargets.New(accountingmappingtargets.Params{
		Mappings:           p.Mappings,
		AccountingControls: p.AccountingControls,
		GLAccounts:         p.GLAccounts,
		Ledger:             p.Ledger,
		Customers:          p.Customers,
		Carriers:           p.Carriers,
		Accessorials:       p.Accessorials,
		Workers:            p.Workers,
		SettlementControls: p.SettlementControls,
		PayCodes:           p.PayCodes,
	})
	reads := accountingmappingreads.New(accountingmappingreads.Params{
		Connections:        p.Connections,
		References:         p.References,
		Mappings:           p.Mappings,
		Ledger:             p.Ledger,
		AccountingControls: p.AccountingControls,
	})
	return services{
		kit:   kit,
		reads: reads,
		scoring: accountingmappingscoring.New(accountingmappingscoring.Params{
			Logger:      p.Logger,
			Connections: p.Connections,
			References:  p.References,
			Mappings:    p.Mappings,
			Sessions:    p.Sessions,
			Completion:  p.Completion,
			Kit:         kit,
			Catalog:     catalog,
		}),
		creator: accountingmappingcreate.New(accountingmappingcreate.Params{
			Logger:       p.Logger,
			References:   p.References,
			Mappings:     p.Mappings,
			Sessions:     p.Sessions,
			Accessorials: p.Accessorials,
			Customers:    p.Customers,
			Carriers:     p.Carriers,
			Kit:          kit,
			Catalog:      catalog,
			Reads:        reads,
		}),
		decisions: accountingmappingdecisions.New(accountingmappingdecisions.Params{
			Logger:      p.Logger,
			DB:          p.DB,
			Connections: p.Connections,
			References:  p.References,
			Mappings:    p.Mappings,
			Refresher:   p.Refresher,
			Audit:       p.AuditService,
			Kit:         kit,
			Catalog:     catalog,
			Reads:       reads,
		}),
	}
}
