package accountingconnectionservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Connections   accountingconnectionports.AccountingConnectionRepository
	Apps          accountingconnectionports.AccountingAppCredentialRepository
	States        accountingconnectionports.AccountingOAuthStateRepository
	Integrations  integrationports.IntegrationRepository
	Connectors    accountingconnectorports.AccountingConnectorRegistry
	Encryption    *encryptionservice.Service
	AuditService  auditports.AuditService
	Realtime      realtimeports.RealtimeService                                     `optional:"true"`
	Watchtower    watchtowerports.WatchtowerProjector                               `optional:"true"`
	Publisher     agentports.AgentEventPublisher                                    `optional:"true"`
	Refresher     accountingconnectionports.AccountingReferenceRefresher            `optional:"true"`
	Poller        accountinginboundports.AccountingChangePoller                     `optional:"true"`
	Plans         platformplanports.PlanService                                     `optional:"true"`
	Subscriptions accountingconnectionports.AccountingWebhookSubscriptionRepository `optional:"true"`
	Certificates  accountingconnectionports.AccountingPendingCertificateRepository  `optional:"true"`
	Organizations tenantports.OrganizationRepository                                `optional:"true"`
}
