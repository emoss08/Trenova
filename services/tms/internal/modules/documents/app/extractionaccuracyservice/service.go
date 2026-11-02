// Package extractionaccuracyservice measures production document extraction from the
// corrections people make when they confirm a draft, and warns an organization when a
// provider starts reading worse than it used to.
package extractionaccuracyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionaccuracyports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ extractionaccuracyports.ExtractionAccuracyService = (*Service)(nil)
	_ extractionaccuracyports.ExtractionDriftChecker    = (*Service)(nil)
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Corrections   aicorrectionports.AICorrectionRepository
	Providers     aiproviderports.AIProviderRepository
	Notifications *notificationservice.Service
	Clock         timeutils.Clock
}

type Service struct {
	l           *zap.Logger
	corrections aicorrectionports.AICorrectionRepository
	providers   aiproviderports.AIProviderRepository
	notifier    driftNotifier
	now         timeutils.Clock
}

type driftNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.extractionaccuracy"),
		corrections: p.Corrections,
		providers:   p.Providers,
		notifier:    p.Notifications,
		now:         p.Clock,
	}
}

func (s *Service) GetCorrection(
	ctx context.Context,
	req aicorrectionports.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	return s.corrections.GetByID(ctx, req)
}

func (s *Service) ListCorrections(
	ctx context.Context,
	req *aicorrectionports.ListAICorrectionConnectionRequest,
) (*pagination.CursorListResult[*aicorrection.Correction], error) {
	return s.corrections.ListConnection(ctx, req)
}
