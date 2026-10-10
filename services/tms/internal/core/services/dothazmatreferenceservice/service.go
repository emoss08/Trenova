package dothazmatreferenceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/modules/reference/ports/dothazmatreferenceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   dothazmatreferenceports.DotHazmatReferenceRepository
}

type Service struct {
	l    *zap.Logger
	repo dothazmatreferenceports.DotHazmatReferenceRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.dothazmatreference"),
		repo: p.Repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req dothazmatreferenceports.GetDotHazmatReferenceByIDRequest,
) (*dothazmatreference.DotHazmatReference, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error) {
	return s.repo.SelectOptions(ctx, req)
}
