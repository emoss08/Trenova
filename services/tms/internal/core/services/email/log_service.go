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
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type logService struct {
	l *zerolog.Logger
	r repositories.EmailLogRepository
}

type LogServiceParams struct {
	fx.In

	Logger     *logger.Logger
	Repository repositories.EmailLogRepository
}

func NewLogService(p LogServiceParams) services.EmailLogService {
	log := p.Logger.With().
		Str("service", "email_log").
		Logger()

	return &logService{
		l: &log,
		r: p.Repository,
	}
}

func (s *logService) Create(ctx context.Context, log *email.Log) (*email.Log, error) {
	// TODO: Implement log creation
	return nil, nil
}

func (s *logService) Get(ctx context.Context, id pulid.ID) (*email.Log, error) {
	// TODO: Implement get log by ID
	return nil, nil
}

func (s *logService) List(
	ctx context.Context,
	filter *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*email.Log], error) {
	// TODO: Implement list logs
	return nil, nil
}

func (s *logService) GetByQueueID(ctx context.Context, queueID pulid.ID) ([]*email.Log, error) {
	// TODO: Implement get logs by queue ID
	return nil, nil
}

func (s *logService) GetByMessageID(ctx context.Context, messageID string) (*email.Log, error) {
	// TODO: Implement get log by message ID
	return nil, nil
}
