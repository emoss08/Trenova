/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// EmailLogRepositoryParams defines dependencies for the email log repository
type EmailLogRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// emailLogRepository implements the EmailLogRepository interface
type emailLogRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewEmailLogRepository creates a new email log repository instance
func NewEmailLogRepository(p EmailLogRepositoryParams) repositories.EmailLogRepository {
	log := p.Logger.With().
		Str("repository", "email_log").
		Logger()

	return &emailLogRepository{
		db: p.DB,
		l:  &log,
	}
}

// Create creates a new email log entry
func (r *emailLogRepository) Create(ctx context.Context, log *email.Log) (*email.Log, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	rLog := r.l.With().
		Str("operation", "Create").
		Str("queueID", log.QueueID.String()).
		Str("status", string(log.Status)).
		Logger()

	if _, err = dba.NewInsert().Model(log).Returning("*").Exec(ctx); err != nil {
		rLog.Error().Err(err).Msg("failed to insert email log entry")
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "failed to insert email log entry")
	}

	rLog.Info().
		Str("logID", log.ID.String()).
		Msg("email log entry created successfully")

	return log, nil
}

// Get retrieves an email log entry by ID
func (r *emailLogRepository) Get(ctx context.Context, id pulid.ID) (*email.Log, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Get").
		Str("logID", id.String()).
		Logger()

	emailLog := new(email.Log)

	err = dba.NewSelect().
		Model(emailLog).
		Where("el.id = ?", id).
		Relation("Queue").
		Relation("Queue.Profile").
		Relation("Queue.Template").
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email log entry not found")
			return nil, errors.NewNotFoundError("Email log entry not found")
		}
		log.Error().Err(err).Msg("failed to get email log entry")
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get").
			Tags("log_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email log entry")
	}

	return emailLog, nil
}

// GetByQueueID retrieves logs for a specific queue entry
func (r *emailLogRepository) GetByQueueID(
	ctx context.Context,
	queueID pulid.ID,
) ([]*email.Log, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get_by_queue_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByQueueID").
		Str("queueID", queueID.String()).
		Logger()

	var logs []*email.Log

	err = dba.NewSelect().
		Model(&logs).
		Where("el.queue_id = ?", queueID).
		Order("el.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get email logs by queue ID")
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get_by_queue_id").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email logs by queue ID")
	}

	log.Info().
		Int("count", len(logs)).
		Msg("retrieved email logs for queue")

	return logs, nil
}

// GetByMessageID retrieves a log by provider message ID
func (r *emailLogRepository) GetByMessageID(
	ctx context.Context,
	messageID string,
) (*email.Log, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get_by_message_id").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByMessageID").
		Str("messageID", messageID).
		Logger()

	emailLog := new(email.Log)

	err = dba.NewSelect().
		Model(emailLog).
		Where("el.message_id = ?", messageID).
		Relation("Queue").
		Relation("Queue.Profile").
		Relation("Queue.Template").
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email log entry not found by message ID")
			return nil, errors.NewNotFoundError("Email log entry not found")
		}
		log.Error().Err(err).Msg("failed to get email log by message ID")
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "get_by_message_id").
			Tags("message_id", messageID).
			Time(time.Now()).
			Wrapf(err, "failed to get email log by message ID")
	}

	return emailLog, nil
}

// List retrieves a list of email logs with pagination
func (r *emailLogRepository) List(
	ctx context.Context,
	filter *ports.QueryOptions,
) (*ports.ListResult[*email.Log], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Logger()

	logs := make([]*email.Log, 0)

	q := dba.NewSelect().Model(&logs).
		Relation("Queue").
		Relation("Queue.Profile").
		Relation("Queue.Template")

	// Apply filters using query builder
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"el",
		repositories.EmailLogFieldConfig,
		(*email.Log)(nil),
	)

	if filter != nil {
		qb.ApplyFilters(filter.FieldFilters)

		if len(filter.Sort) > 0 {
			qb.ApplySort(filter.Sort)
		}

		if filter.Query != "" {
			qb.ApplyTextSearch(filter.Query, []string{"message_id", "provider_response"})
		}

		q = qb.GetQuery()
	}

	q = q.Limit(filter.Limit).Offset(filter.Offset)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan email log entries")
		return nil, oops.
			In("email_log_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "failed to list email log entries")
	}

	return &ports.ListResult[*email.Log]{
		Items: logs,
		Total: total,
	}, nil
}
