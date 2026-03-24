package documentservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func NewTestService(
	logger *zap.Logger,
	repo repositories.DocumentRepository,
	cacheRepo repositories.DocumentCacheRepository,
	storageClient storage.Client,
	validator *Validator,
	auditService services.AuditService,
	cfg *config.StorageConfig,
	thumbnailGenerator *thumbnailservice.Generator,
	temporalClient client.Client,
) *Service {
	return &Service{
		l:                  logger.Named("service.document"),
		repo:               repo,
		cacheRepo:          cacheRepo,
		storage:            storageClient,
		validator:          validator,
		auditService:       auditService,
		config:             cfg,
		thumbnailGenerator: thumbnailGenerator,
		temporalClient:     temporalClient,
	}
}
