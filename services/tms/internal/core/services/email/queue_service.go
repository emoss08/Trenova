/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
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
	return s.r.Create(ctx, queue)
}

func (s *queueService) Update(ctx context.Context, queue *email.Queue) (*email.Queue, error) {
	return s.r.Update(ctx, queue)
}

func (s *queueService) Get(ctx context.Context, id pulid.ID) (*email.Queue, error) {
	return s.r.Get(ctx, id)
}

func (s *queueService) List(
	ctx context.Context,
	filter *ports.QueryOptions,
) (*ports.ListResult[*email.Queue], error) {
	return s.r.List(ctx, filter)
}

func (s *queueService) GetPending(ctx context.Context, limit int) ([]*email.Queue, error) {
	return s.r.GetPending(ctx, limit)
}

func (s *queueService) GetScheduled(ctx context.Context, limit int) ([]*email.Queue, error) {
	return s.r.GetScheduled(ctx, limit)
}

func (s *queueService) MarkAsFailed(
	ctx context.Context,
	queueID pulid.ID,
	errorMessage string,
) error {
	return s.r.MarkAsFailed(ctx, queueID, errorMessage)
}

func (s *queueService) IncrementRetryCount(ctx context.Context, queueID pulid.ID) error {
	return s.r.IncrementRetryCount(ctx, queueID)
}

func (s *queueService) MarkAsSent(ctx context.Context, queueID pulid.ID, messageID string) error {
	return s.r.MarkAsSent(ctx, queueID, messageID)
}
