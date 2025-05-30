package favorite

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.FavoriteRepository
	PermService  services.PermissionService
	AuditService services.AuditService
}

type Service struct {
	repo repositories.FavoriteRepository
	l    *zerolog.Logger
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "favorite").
		Logger()

	return &Service{
		repo: p.Repo,
		l:    &log,
		ps:   p.PermService,
		as:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
) ([]*pagefavorite.PageFavorite, error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser, // Users can manage their own favorites
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read favorites")
	}

	favorites, err := s.repo.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list favorites")
		return nil, err
	}

	return favorites, nil
}

func (s *Service) Get(
	ctx context.Context,
	orgID, buID, userID, favoriteID pulid.ID,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("favoriteID", favoriteID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read favorites")
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetFavoriteByIDOptions{
		OrgID:      orgID,
		BuID:       buID,
		UserID:     userID,
		FavoriteID: favoriteID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get favorite")
		return nil, err
	}

	return entity, nil
}

func (s *Service) GetByURL(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
	pageURL string,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With().
		Str("operation", "GetByURL").
		Str("pageURL", pageURL).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read favorites")
	}

	entity, err := s.repo.GetByURL(ctx, repositories.GetFavoriteByURLOptions{
		OrgID:   orgID,
		BuID:    buID,
		UserID:  userID,
		PageURL: pageURL,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get favorite by URL")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
	fav *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With().Str("operation", "Create").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionCreate,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create favorites")
	}

	// Set the required IDs
	fav.OrganizationID = orgID
	fav.BusinessUnitID = buID
	fav.UserID = userID

	// Validate the favorite
	multiErr := errors.NewMultiError()
	fav.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		log.Error().Interface("errors", multiErr.Errors).Msg("failed to validate favorite")
		return nil, multiErr
	}

	// Check if the user already has this page favorited
	existing, err := s.repo.GetByURL(ctx, repositories.GetFavoriteByURLOptions{
		OrgID:   orgID,
		BuID:    buID,
		UserID:  userID,
		PageURL: fav.PageURL,
	})
	if err == nil && existing != nil {
		return nil, errors.NewValidationError(
			"pageUrl",
			errors.ErrDuplicate,
			"This page is already in your favorites",
		)
	}

	entity, err := s.repo.Create(ctx, fav)
	if err != nil {
		log.Error().Err(err).Msg("failed to create favorite")
		return nil, err
	}

	// Audit the creation
	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePageFavorite,
		ResourceID:     entity.ID.String(),
		Action:         permission.ActionCreate,
		CurrentState:   jsonutils.MustToJSON(entity),
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to log audit action")
	}

	return entity, nil
}

func (s *Service) Update(
	ctx context.Context,
	orgID, buID, userID, favoriteID pulid.ID,
	fav *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("favoriteID", favoriteID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionUpdate,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update favorites")
	}

	// Get the existing favorite for audit purposes
	existing, err := s.repo.GetByID(ctx, repositories.GetFavoriteByIDOptions{
		OrgID:      orgID,
		BuID:       buID,
		UserID:     userID,
		FavoriteID: favoriteID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing favorite")
		return nil, err
	}

	// Set the required IDs
	fav.ID = favoriteID
	fav.OrganizationID = orgID
	fav.BusinessUnitID = buID
	fav.UserID = userID

	// Validate the favorite
	multiErr := errors.NewMultiError()
	fav.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		log.Error().Interface("errors", multiErr.Errors).Msg("failed to validate favorite")
		return nil, multiErr
	}

	entity, err := s.repo.Update(ctx, fav)
	if err != nil {
		log.Error().Err(err).Msg("failed to update favorite")
		return nil, err
	}

	// Audit the update
	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePageFavorite,
		ResourceID:     entity.ID.String(),
		Action:         permission.ActionUpdate,
		CurrentState:   jsonutils.MustToJSON(entity),
		PreviousState:  jsonutils.MustToJSON(existing),
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to log audit action")
	}

	return entity, nil
}

func (s *Service) Delete(ctx context.Context, orgID, buID, userID, favoriteID pulid.ID) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("favoriteID", favoriteID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionDelete,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to delete favorites")
	}

	// Get the existing favorite for audit purposes
	existing, err := s.repo.GetByID(ctx, repositories.GetFavoriteByIDOptions{
		OrgID:      orgID,
		BuID:       buID,
		UserID:     userID,
		FavoriteID: favoriteID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing favorite")
		return err
	}

	err = s.repo.Delete(ctx, repositories.DeleteFavoriteOptions{
		OrgID:      orgID,
		BuID:       buID,
		UserID:     userID,
		FavoriteID: favoriteID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete favorite")
		return err
	}

	// Audit the deletion
	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePageFavorite,
		ResourceID:     favoriteID.String(),
		Action:         permission.ActionDelete,
		PreviousState:  jsonutils.MustToJSON(existing),
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to log audit action")
	}

	return nil
}

func (s *Service) ToggleFavorite(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
	pageURL, pageTitle string,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With().
		Str("operation", "ToggleFavorite").
		Str("pageURL", pageURL).
		Logger()

	// Check if the page is already favorited
	existing, err := s.repo.GetByURL(ctx, repositories.GetFavoriteByURLOptions{
		OrgID:   orgID,
		BuID:    buID,
		UserID:  userID,
		PageURL: pageURL,
	})

	if err == nil && existing != nil {
		// Page is favorited, remove it
		if deleteErr := s.Delete(ctx, orgID, buID, userID, existing.ID); deleteErr != nil {
			log.Error().Err(deleteErr).Msg("failed to remove favorite")
			return nil, deleteErr
		}
		return nil, nil //nolint:nilnil // This is a special case where we return nil to indicate the page was unfavorited
	}

	// Page is not favorited, add it
	newFav := &pagefavorite.PageFavorite{
		PageURL:   pageURL,
		PageTitle: pageTitle,
	}

	created, err := s.Create(ctx, orgID, buID, userID, newFav)
	if err != nil {
		log.Error().Err(err).Msg("failed to create favorite")
		return nil, err
	}

	return created, nil
}
