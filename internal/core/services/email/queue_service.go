package email

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type queueService struct {
	l *zerolog.Logger
	r repositories.EmailQueueRepository
}

type QueueServiceParams struct {
	fx.In

	Logger     *logger.Logger
	Repository repositories.EmailQueueRepository
}

func NewQueueService(p QueueServiceParams) services.EmailQueueService {
	log := p.Logger.With().
		Str("service", "email_queue").
		Logger()

	return &queueService{
		l: &log,
		r: p.Repository,
	}
}

func (s *queueService) Create(ctx context.Context, queue *email.Queue) (*email.Queue, error) {
	// TODO: Implement queue creation
	return nil, nil
}

func (s *queueService) Update(ctx context.Context, queue *email.Queue) (*email.Queue, error) {
	// TODO: Implement queue update
	return nil, nil
}

func (s *queueService) Get(ctx context.Context, id pulid.ID) (*email.Queue, error) {
	// TODO: Implement get queue by ID
	return nil, nil
}

func (s *queueService) List(
	ctx context.Context,
	filter *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*email.Queue], error) {
	// TODO: Implement list queues
	return nil, nil
}

func (s *queueService) GetPending(ctx context.Context, limit int) ([]*email.Queue, error) {
	// TODO: Implement get pending emails
	return nil, nil
}

func (s *queueService) GetScheduled(ctx context.Context, limit int) ([]*email.Queue, error) {
	// TODO: Implement get scheduled emails
	return nil, nil
}

func (s *queueService) MarkAsSent(ctx context.Context, queueID pulid.ID, messageID string) error {
	// TODO: Implement mark as sent
	return nil
}

func (s *queueService) MarkAsFailed(
	ctx context.Context,
	queueID pulid.ID,
	errorMessage string,
) error {
	// TODO: Implement mark as failed
	return nil
}

func (s *queueService) IncrementRetryCount(ctx context.Context, queueID pulid.ID) error {
	// TODO: Implement increment retry count
	return nil
}
