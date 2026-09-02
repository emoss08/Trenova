package formulatemplatejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ExpireStaleSubmissionsPayload struct {
	Limit int `json:"limit"`
}

type ExpireStaleSubmissionsResult struct {
	Expired int `json:"expired"`
}

type ActivitiesParams struct {
	fx.In

	TemplateService *formulatemplateservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	templates *formulatemplateservice.Service
	logger    *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		templates: p.TemplateService,
		logger:    p.Logger.Named("temporal.formula-template"),
	}
}

// ExpireStaleSubmissionsActivity returns every template that has waited in
// review past the expiry to draft, across tenants, so a forgotten submission
// never gets approved months later against rates that have since moved.
func (a *Activities) ExpireStaleSubmissionsActivity(
	ctx context.Context,
	payload *ExpireStaleSubmissionsPayload,
) (*ExpireStaleSubmissionsResult, error) {
	result, err := a.templates.ExpireStaleSubmissions(
		ctx,
		&formulatemplateservice.ExpireStaleSubmissionsRequest{
			Limit: temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(result.Expired) > 0 {
		a.logger.Info("expired stale formula template submissions",
			zap.Int("count", len(result.Expired)))
	}

	return &ExpireStaleSubmissionsResult{Expired: len(result.Expired)}, nil
}
