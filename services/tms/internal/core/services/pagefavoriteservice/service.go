package pagefavoriteservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ToggleResult struct {
	Favorited bool                       `json:"favorited"`
	Favorite  *pagefavorite.PageFavorite `json:"favorite,omitempty"`
}

type ToggleRequest struct {
	PageURL    string
	PageTitle  string
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.PageFavoriteRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.PageFavoriteRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.pagefavorite"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListPageFavoritesRequest,
) ([]*pagefavorite.PageFavorite, error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Toggle(
	ctx context.Context,
	req *ToggleRequest,
) (*ToggleResult, error) {
	log := s.l.With(
		zap.String("operation", "Toggle"),
		zap.String("pageURL", req.PageURL),
		zap.String("userID", req.UserID.String()),
	)

	existing, exists, err := s.repo.GetByURL(ctx, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    req.PageURL,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to check existing favorite", zap.Error(err))
		return nil, err
	}

	if exists {
		if err = s.repo.Delete(ctx, existing.ID, req.TenantInfo); err != nil {
			log.Error("failed to delete favorite", zap.Error(err))
			return nil, err
		}
		return &ToggleResult{Favorited: false}, nil
	}

	entity := &pagefavorite.PageFavorite{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.UserID,
		PageURL:        req.PageURL,
		PageTitle:      req.PageTitle,
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create favorite", zap.Error(err))
		return nil, err
	}

	return &ToggleResult{
		Favorited: true,
		Favorite:  created,
	}, nil
}

func (s *Service) IsFavorited(
	ctx context.Context,
	pageURL string,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	_, exists, err := s.repo.GetByURL(ctx, &repositories.GetPageFavoriteByURLRequest{
		PageURL:    pageURL,
		UserID:     userID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}
