package formulatemplatejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ExpireStaleSubmissionsPayload struct {
	Limit int `json:"limit"`
}

type ExpireStaleSubmissionsResult struct {
	Expired          int `json:"expired"`
	LibrariesExpired int `json:"librariesExpired"`
}

type ActivitiesParams struct {
	fx.In

	TemplateService *formulatemplateservice.Service
	LibraryService  *formulalibraryservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	templates *formulatemplateservice.Service
	libraries *formulalibraryservice.Service
	logger    *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		templates: p.TemplateService,
		libraries: p.LibraryService,
		logger:    p.Logger.Named("temporal.formula-template"),
	}
}

func (a *Activities) ExpireStaleSubmissionsActivity(
	ctx context.Context,
	payload *ExpireStaleSubmissionsPayload,
) (*ExpireStaleSubmissionsResult, error) {
	limit := temporaljobs.NormalizeLimit(payload.Limit, temporaljobs.DefaultTenantScanLimit)
	result, err := a.templates.ExpireStaleSubmissions(
		ctx,
		&formulatemplateservice.ExpireStaleSubmissionsRequest{Limit: limit},
	)
	if err != nil {
		return nil, err
	}

	libraries, err := a.libraries.ExpireStaleSubmissions(
		ctx,
		&formulalibraryservice.ExpireStaleSubmissionsRequest{Limit: limit},
	)
	if err != nil {
		return nil, err
	}

	if len(result.Expired) > 0 || len(libraries.Expired) > 0 {
		a.logger.Info("expired stale formula submissions",
			zap.Int("templates", len(result.Expired)),
			zap.Int("libraries", len(libraries.Expired)),
		)
	}

	return &ExpireStaleSubmissionsResult{
		Expired:          len(result.Expired),
		LibrariesExpired: len(libraries.Expired),
	}, nil
}
