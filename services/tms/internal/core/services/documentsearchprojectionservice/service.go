package documentsearchprojectionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
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

	Logger  *zap.Logger
	Repo    repositories.DocumentSearchProjectionRepository
	Indexer serviceports.RetrievalIndexer `optional:"true"`
}

type Service struct {
	logger  *zap.Logger
	repo    repositories.DocumentSearchProjectionRepository
	indexer serviceports.RetrievalIndexer
}

var _ serviceports.DocumentSearchProjectionService = (*Service)(nil)

func New(p Params) serviceports.DocumentSearchProjectionService {
	return &Service{
		logger:  p.Logger.Named("service.document-search-projection"),
		repo:    p.Repo,
		indexer: p.Indexer,
	}
}

func (s *Service) Upsert(ctx context.Context, doc *document.Document, contentText string) error {
	if _, err := s.repo.Upsert(ctx, documentsearchprojection.Build(doc, contentText)); err != nil {
		return err
	}

	s.markStale(ctx, doc)

	return nil
}

func (s *Service) Delete(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if err := s.repo.Delete(ctx, documentID, tenantInfo); err != nil {
		return err
	}

	s.deleteFromIndex(ctx, documentID, tenantInfo)

	return nil
}

func (s *Service) markStale(ctx context.Context, doc *document.Document) {
	if s.indexer == nil || doc == nil {
		return
	}

	tenantInfo := pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID}
	if err := s.indexer.MarkStale(
		ctx,
		tenantInfo,
		airetrieval.SourceTypeDocument,
		doc.ID,
	); err != nil {
		s.logger.Warn("document could not be queued for semantic indexing",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) deleteFromIndex(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) {
	if s.indexer == nil {
		return
	}

	if err := s.indexer.DeleteSource(
		ctx,
		tenantInfo,
		airetrieval.SourceTypeDocument,
		documentID,
	); err != nil {
		s.logger.Warn("document could not be removed from the semantic index",
			zap.String("documentId", documentID.String()),
			zap.Error(err),
		)
	}
}
