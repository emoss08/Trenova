package triagebriefingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type InvalidatorParams struct {
	fx.In

	Logger *zap.Logger
	Cache  triageports.TriageCache `optional:"true"`
}

type Invalidator struct {
	l     *zap.Logger
	cache triageports.TriageCache
}

func NewInvalidator(p InvalidatorParams) *Invalidator {
	return &Invalidator{
		l:     p.Logger.Named("service.triage-briefing-invalidator"),
		cache: p.Cache,
	}
}

var _ inboundmessageports.TriageBriefingInvalidator = (*Invalidator)(nil)

func (i *Invalidator) InvalidateBriefing(ctx context.Context, tenantInfo pagination.TenantInfo) {
	triagesupport.Invalidate(ctx, i.cache, i.l, tenantInfo)
}
