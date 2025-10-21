package pagefavorite

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.PageFavoriteRepository
	AuditService services.AuditService
}

type Service struct {
	l    *zap.Logger
	repo repositories.PageFavoriteRepository
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.pagefavorite"),
		repo: p.Repo,
		as:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *pagination.QueryOptions,
) (*pagination.ListResult[*pagefavorite.PageFavorite], error) {
	return s.repo.List(ctx, opts)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetPageFavoriteByIDRequest,
) (*pagefavorite.PageFavorite, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByURL(
	ctx context.Context,
	req repositories.GetPageFavoriteByURLRequest,
) (*pagefavorite.PageFavorite, error) {
	return s.repo.GetByURL(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	req *repositories.CreatePageFavoriteRequest,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", req.BuID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("userID", req.UserID.String()),
	)

	req.Favorite.OrganizationID = req.OrgID
	req.Favorite.BusinessUnitID = req.BuID
	req.Favorite.UserID = req.UserID

	multiErr := errortypes.NewMultiError()
	req.Favorite.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	existing, err := s.repo.GetByURL(ctx, repositories.GetPageFavoriteByURLRequest{
		OrgID:   req.OrgID,
		BuID:    req.BuID,
		UserID:  req.UserID,
		PageURL: req.Favorite.PageURL,
	})
	if err == nil && existing != nil {
		return nil, errortypes.NewValidationError(
			"pageURL",
			errortypes.ErrDuplicate,
			"This page is already in your favorites",
		)
	}

	entity, err := s.repo.Create(ctx, req)
	if err != nil {
		log.Error("create page favorite", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePageFavorite,
		ResourceID:     entity.ID.String(),
		Operation:      permission.OpCreate,
		CurrentState:   jsonutils.MustToJSON(entity),
		UserID:         req.UserID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
	}, audit.WithComment("Page favorite created"))
	if err != nil {
		log.Error("failed to log page favorite create", zap.Error(err))
	}

	return entity, nil
}

func (s *Service) Delete(ctx context.Context, req repositories.DeletePageFavoriteRequest) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("favoriteID", req.FavoriteID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetPageFavoriteByIDRequest(req))
	if err != nil {
		log.Error("failed to get existing favorite", zap.Error(err))
		return err
	}

	err = s.repo.Delete(ctx, req)
	if err != nil {
		log.Error("failed to delete favorite", zap.Error(err))
		return err
	}

	// Audit the deletion
	err = s.as.LogAction(&services.LogActionParams{
		Resource:       permission.ResourcePageFavorite,
		ResourceID:     req.FavoriteID.String(),
		Operation:      permission.OpDelete,
		PreviousState:  jsonutils.MustToJSON(existing),
		UserID:         req.UserID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
	},
		audit.WithComment("Page favorite deleted"),
	)
	if err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

type TogglePageFavoriteRequest struct {
	OrgID     pulid.ID
	BuID      pulid.ID
	UserID    pulid.ID
	PageURL   string
	PageTitle string
}

func (s *Service) Toggle(
	ctx context.Context,
	req *TogglePageFavoriteRequest,
) (*pagefavorite.PageFavorite, error) {
	log := s.l.With(
		zap.String("operation", "Toggle"),
		zap.String("buID", req.BuID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("userID", req.UserID.String()),
	)

	existing, err := s.repo.GetByURL(ctx, repositories.GetPageFavoriteByURLRequest{
		OrgID:   req.OrgID,
		BuID:    req.BuID,
		UserID:  req.UserID,
		PageURL: req.PageURL,
	})
	if err == nil && existing != nil {
		if deleteErr := s.Delete(ctx, repositories.DeletePageFavoriteRequest{
			OrgID:      req.OrgID,
			BuID:       req.BuID,
			UserID:     req.UserID,
			FavoriteID: existing.ID,
		}); deleteErr != nil {
			return nil, deleteErr
		}

		return nil, nil //nolint:nilnil // This is a special case where we return nil to indicate the page was unfavorited
	}

	newFav := &pagefavorite.PageFavorite{
		PageURL:   req.PageURL,
		PageTitle: req.PageTitle,
	}

	created, err := s.Create(ctx, &repositories.CreatePageFavoriteRequest{
		OrgID:    req.OrgID,
		BuID:     req.BuID,
		UserID:   req.UserID,
		Favorite: newFav,
	})
	if err != nil {
		log.Error("failed to create page favorite", zap.Any("request", req), zap.Error(err))
		return nil, err
	}

	return created, nil
}
