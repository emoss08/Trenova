package documentqualityconfig

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/documentqualityconfig"
	"github.com/trenova-app/transport/internal/core/domain/permission"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger      *logger.Logger
	Repo        repositories.DocumentQualityConfigRepository
	PermService services.PermissionService
}

type Service struct {
	repo repositories.DocumentQualityConfigRepository
	l    *zerolog.Logger
	ps   services.PermissionService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "documentqualityconfig").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		l:    &log,
	}
}

func (s *Service) Get(ctx context.Context, opts *repositories.GetDocumentQualityConfigOptions) (*documentqualityconfig.DocumentQualityConfig, error) {
	log := s.l.With().Str("operation", "Get").Str("orgID", opts.OrgID.String()).Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         opts.UserID,
			Resource:       permission.ResourceDocumentQualityConfig,
			Action:         permission.ActionRead,
			BusinessUnitID: opts.BuID,
			OrganizationID: opts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check read document quality config permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read document quality config")
	}

	entity, err := s.repo.Get(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get document quality config")
		return nil, eris.Wrap(err, "failed to get document quality config")
	}

	return entity, nil
}
