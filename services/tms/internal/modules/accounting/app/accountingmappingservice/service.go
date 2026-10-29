package accountingmappingservice

import (
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
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

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
	Completion         llmports.CompletionService                             `optional:"true"`
	Refresher          accountingconnectionports.AccountingReferenceRefresher `optional:"true"`
	Realtime           realtimeports.RealtimeService                          `optional:"true"`
	SyncRecords        accountingsyncports.AccountingSyncRecordRepository     `optional:"true"`
	Dispatcher         accountingsyncports.AccountingSyncDispatcher           `optional:"true"`
	Workers            workerports.WorkerRepository                           `optional:"true"`
	SettlementControls tenantports.CarrierSettlementControlRepository         `optional:"true"`
	PayCodes           driverpayports.PayCodeRepository                       `optional:"true"`
	Ledger             journalentryports.AccountingLedgerSource               `optional:"true"`
}
