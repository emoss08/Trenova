package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type noopDBConnection struct{}

func (noopDBConnection) DB() *bun.DB                          { return nil }
func (noopDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (noopDBConnection) HealthCheck(context.Context) error    { return nil }
func (noopDBConnection) IsHealthy(context.Context) bool       { return true }
func (noopDBConnection) Close() error                         { return nil }
func (noopDBConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func NewTestService(
	logger *zap.Logger,
	repo repositories.DocumentRepository,
	cacheRepo repositories.DocumentCacheRepository,
	sessionRepo repositories.DocumentUploadSessionRepository,
	storageClient storage.Client,
	validator *Validator,
	auditService services.AuditService,
	cfg *config.StorageConfig,
	thumbnailGenerator *thumbnailservice.Generator,
	temporalClient client.Client,
) *Service {
	return &Service{
		l:                    logger.Named("service.document"),
		db:                   noopDBConnection{},
		repo:                 repo,
		cacheRepo:            cacheRepo,
		sessionRepo:          sessionRepo,
		storage:              storageClient,
		validator:            validator,
		auditService:         auditService,
		documentIntelligence: noopDocumentContentService{},
		searchProjection:     noopDocumentSearchProjectionService{},
		config:               cfg,
		thumbnailGenerator:   thumbnailGenerator,
		workflowStarter: workflowstarter.New(workflowstarter.Params{
			TemporalClient: temporalClient,
		}),
	}
}
