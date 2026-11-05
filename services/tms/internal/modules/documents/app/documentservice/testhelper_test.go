package documentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/imagingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/pagerender"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type noopPaperwork struct{}

func (noopPaperwork) RequestRead(context.Context, *document.Document, pulid.ID) {}

type noopTransactor struct{}

func (noopTransactor) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

func NewTestService(
	logger *zap.Logger,
	repo documentports.DocumentRepository,
	cacheRepo customerports.DocumentCacheRepository,
	sessionRepo documentuploadports.DocumentUploadSessionRepository,
	storageClient kernelports.Client,
	validator *Validator,
	auditService auditports.AuditService,
	cfg *storageconfig.Config,
	thumbnailGenerator *thumbnailservice.Generator,
) *Service {
	return &Service{
		l:                    logger.Named("service.document"),
		db:                   noopTransactor{},
		repo:                 repo,
		cacheRepo:            cacheRepo,
		sessionRepo:          sessionRepo,
		storage:              storageClient,
		validator:            validator,
		auditService:         auditService,
		documentIntelligence: noopDocumentContentService{},
		searchProjection:     noopDocumentSearchProjectionService{},
		config:               cfg,
		encryption: encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
		),
		thumbnailGenerator: thumbnailGenerator,
		pageRenders: pagerender.New(pagerender.Params{
			Storage: storageClient,
			Encryption: encryptionservice.NewWithKeyManager(
				encryptionservice.NewLocalKeyManager(
					"unit-test-encryption-key-with-at-least-32-bytes",
				),
			),
			Generator:     thumbnailGenerator,
			ImagingConfig: &imagingconfig.Config{},
			Logger:        logger,
		}),
		paperwork:       noopPaperwork{},
		workflowStarter: &mocks.NoopWorkflowStarter{},
		realtime:        &mocks.NoopRealtimeService{},
	}
}
