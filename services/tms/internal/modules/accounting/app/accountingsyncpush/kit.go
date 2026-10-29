package accountingsyncpush

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"go.uber.org/zap"
)

const dependencyWait = time.Minute

type Deps struct {
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

//nolint:gocritic // dependency injection
func New(d Deps) *Kit {
	return &Kit{
		l:         d.Logger,
		records:   d.Records,
		decisions: d.Decisions,
		rates:     d.Rates,
		policy:    d.Policy,
		effects:   d.Effects,
	}
}

func (k *Kit) Kick(ctx context.Context, sess *Session) {
	k.effects.Kick(ctx, sess.Tenant, sess.Conn.ID)
}
