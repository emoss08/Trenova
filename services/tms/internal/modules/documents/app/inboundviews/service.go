package inboundviews

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Now      timeutils.Clock
	ViewRepo inboundmessageports.TriageViewRepository
}

type Service struct {
	now      func() int64
	viewRepo inboundmessageports.TriageViewRepository
}

func New(p Params) *Service {
	now := p.Now
	return &Service{
		now:      now,
		viewRepo: p.ViewRepo,
	}
}
