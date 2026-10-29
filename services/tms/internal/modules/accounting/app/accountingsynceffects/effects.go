package accountingsynceffects

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const AttentionGroups = 20

type Params struct {
	fx.In

	Logger        *zap.Logger
	Records       accountingsyncports.AccountingSyncRecordRepository
	Connections   accountingconnectionports.AccountingConnectionRepository
	Organizations tenantports.OrganizationRepository
	Controls      tenantports.AccountingControlRepository
	Audit         auditports.AuditService
	Dispatcher    accountingsyncports.AccountingSyncDispatcher `optional:"true"`
	Publisher     agenteventports.AgentEventPublisher          `optional:"true"`
	Watchtower    watchtowerports.WatchtowerProjector          `optional:"true"`
	Realtime      realtimeports.RealtimeService                `optional:"true"`
}

type Effects struct {
	l             *zap.Logger
	records       accountingsyncports.AccountingSyncRecordRepository
	connections   accountingconnectionports.AccountingConnectionRepository
	organizations tenantports.OrganizationRepository
	controls      tenantports.AccountingControlRepository
	audit         auditports.AuditService
	dispatcher    accountingsyncports.AccountingSyncDispatcher
	publisher     agenteventports.AgentEventPublisher
	watchtower    watchtowerports.WatchtowerProjector
	realtime      realtimeports.RealtimeService
}

//nolint:gocritic // dependency injection
func New(p Params) *Effects {
	return &Effects{
		l:             p.Logger.Named("service.accounting-sync"),
		records:       p.Records,
		connections:   p.Connections,
		organizations: p.Organizations,
		controls:      p.Controls,
		audit:         p.Audit,
		dispatcher:    p.Dispatcher,
		publisher:     p.Publisher,
		watchtower:    p.Watchtower,
		realtime:      p.Realtime,
	}
}
