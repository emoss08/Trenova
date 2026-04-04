package documentsearchprojectionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.DocumentSearchProjectionRepository
}

type Service struct {
	logger *zap.Logger
	repo   repositories.DocumentSearchProjectionRepository
}

var _ serviceports.DocumentSearchProjectionService = (*Service)(nil)

func New(p Params) serviceports.DocumentSearchProjectionService {
	return &Service{
		logger: p.Logger.Named("service.document-search-projection"),
		repo:   p.Repo,
	}
}

func (s *Service) Upsert(ctx context.Context, doc *document.Document, contentText string) error {
	_, err := s.repo.Upsert(ctx, documentsearchprojection.Build(doc, contentText))
	return err
}

func (s *Service) Delete(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return s.repo.Delete(ctx, documentID, tenantInfo)
}
