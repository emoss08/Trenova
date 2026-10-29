package accountingsyncpush

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const dependencyWait = time.Minute

type Params struct {
	fx.In

	Logger    *zap.Logger
	Records   accountingsyncports.AccountingSyncRecordRepository
	Decisions accountingmappingports.AccountingMappingDecisionService
	Rates     exchangerateports.ExchangeRateService
	Policy    *accountingcontrolpolicyservice.Service
	Effects   *accountingsynceffects.Effects
}

type Kit struct {
	l         *zap.Logger
	records   accountingsyncports.AccountingSyncRecordRepository
	decisions accountingmappingports.AccountingMappingDecisionService
	rates     exchangerateports.ExchangeRateService
	policy    *accountingcontrolpolicyservice.Service
	effects   *accountingsynceffects.Effects
}

func New(p Params) *Kit {
	return &Kit{
		l:         p.Logger.Named("service.accounting-sync"),
		records:   p.Records,
		decisions: p.Decisions,
		rates:     p.Rates,
		policy:    p.Policy,
		effects:   p.Effects,
	}
}

func (k *Kit) Kick(ctx context.Context, sess *Session) {
	k.effects.Kick(ctx, sess.Tenant, sess.Conn.ID)
}
