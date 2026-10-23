package formulatemplatejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ExpireStaleSubmissionsPayload struct {
	Limit int `json:"limit"`
}

type ExpireStaleSubmissionsResult struct {
	Expired          int `json:"expired"`
	LibrariesExpired int `json:"librariesExpired"`
	MatricesExpired  int `json:"matricesExpired"`
}

type ActivitiesParams struct {
	fx.In

	TemplateService *formulatemplateservice.Service
	LibraryService  *formulalibraryservice.Service
	MatrixService   *ratematrixservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	templates *formulatemplateservice.Service
	libraries *formulalibraryservice.Service
	matrices  *ratematrixservice.Service
	logger    *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		templates: p.TemplateService,
		libraries: p.LibraryService,
		matrices:  p.MatrixService,
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

	matrices, err := a.matrices.ExpireStaleSubmissions(
		ctx,
		&ratematrixservice.ExpireStaleSubmissionsRequest{Limit: limit},
	)
	if err != nil {
		return nil, err
	}

	if len(result.Expired) > 0 || len(libraries.Expired) > 0 || len(matrices.Expired) > 0 {
		a.logger.Info("expired stale formula submissions",
			zap.Int("templates", len(result.Expired)),
			zap.Int("libraries", len(libraries.Expired)),
			zap.Int("matrices", len(matrices.Expired)),
		)
	}

	return &ExpireStaleSubmissionsResult{
		Expired:          len(result.Expired),
		LibrariesExpired: len(libraries.Expired),
		MatricesExpired:  len(matrices.Expired),
	}, nil
}
