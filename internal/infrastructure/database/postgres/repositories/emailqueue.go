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
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// EmailQueueRepositoryParams defines dependencies for the email queue repository
type EmailQueueRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// emailQueueRepository implements the EmailQueueRepository interface
type emailQueueRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewEmailQueueRepository creates a new email queue repository instance
func NewEmailQueueRepository(p EmailQueueRepositoryParams) repositories.EmailQueueRepository {
	log := p.Logger.With().
		Str("repository", "email_queue").
		Logger()

	return &emailQueueRepository{
		db: p.DB,
		l:  &log,
	}
}

// Create creates a new email queue entry
func (r *emailQueueRepository) Create(
	ctx context.Context,
	queue *email.Queue,
) (*email.Queue, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", queue.OrganizationID.String()).
		Strs("to", queue.ToAddresses).
		Logger()

	if _, err = dba.NewInsert().Model(queue).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to insert email queue entry")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "failed to insert email queue entry")
	}

	log.Info().
		Str("queueID", queue.ID.String()).
		Str("priority", string(queue.Priority)).
		Msg("email queue entry created successfully")

	return queue, nil
}

// Update updates an email queue entry
func (r *emailQueueRepository) Update(
	ctx context.Context,
	queue *email.Queue,
) (*email.Queue, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("queueID", queue.ID.String()).
		Str("status", string(queue.Status)).
		Logger()

	// Update sent timestamp if status is sent
	if queue.Status == email.QueueStatusSent && (queue.SentAt == nil || *queue.SentAt == 0) {
		now := time.Now().Unix()
		queue.SentAt = &now
	}

	results, err := dba.NewUpdate().
		Model(queue).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to update email queue entry")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "failed to update email queue entry")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return nil, errors.NewNotFoundError("Email queue entry not found")
	}

	log.Info().
		Str("queueID", queue.ID.String()).
		Str("status", string(queue.Status)).
		Msg("email queue entry updated successfully")

	return queue, nil
}

// Get retrieves an email queue entry by ID
func (r *emailQueueRepository) Get(ctx context.Context, id pulid.ID) (*email.Queue, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Get").
		Str("queueID", id.String()).
		Logger()

	queue := new(email.Queue)

	err = dba.NewSelect().
		Model(queue).
		Where("eq.id = ?", id).
		Relation("Profile").
		Relation("Template").
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email queue entry not found")
			return nil, errors.NewNotFoundError("Email queue entry not found")
		}
		log.Error().Err(err).Msg("failed to get email queue entry")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get").
			Tags("queue_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email queue entry")
	}

	return queue, nil
}

func (r *emailQueueRepository) filterQuery(
	q *bun.SelectQuery,
	filter *ports.QueryOptions,
) *bun.SelectQuery {
	// Apply filters using query builder
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"eq",
		repositories.EmailQueueFieldConfig,
		(*email.Queue)(nil),
	)

	qb.ApplyTenantFilters(filter.TenantOpts)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if filter.Query != "" {
		qb.ApplyTextSearch(filter.Query, []string{"to", "subject", "error_message"})
	}

	if len(filter.Sort) > 0 {
		qb.ApplySort(filter.Sort)
	}

	q.Relation("Profile").
		Relation("Template")

	q = qb.GetQuery()

	return q.Limit(filter.Limit).Offset(filter.Offset)
}

// List retrieves a list of email queue entries with pagination
func (r *emailQueueRepository) List(
	ctx context.Context,
	filter *ports.QueryOptions,
) (*ports.ListResult[*email.Queue], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("orgID", filter.TenantOpts.OrgID.String()).
		Str("buID", filter.TenantOpts.BuID.String()).
		Logger()

	queues := make([]*email.Queue, 0)

	q := dba.NewSelect().Model(&queues)

	q = r.filterQuery(q, filter)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan email queue entries")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "failed to list email queue entries")
	}

	return &ports.ListResult[*email.Queue]{
		Items: queues,
		Total: total,
	}, nil
}

// GetPending retrieves pending emails to process
func (r *emailQueueRepository) GetPending(ctx context.Context, limit int) ([]*email.Queue, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get_pending").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetPending").
		Int("limit", limit).
		Logger()

	var queues []*email.Queue

	err = dba.NewSelect().
		Model(&queues).
		Where("eq.status = ?", email.QueueStatusPending).
		OrderExpr("CASE eq.priority WHEN ? THEN 1 WHEN ? THEN 2 WHEN ? THEN 3 END",
			email.PriorityHigh, email.PriorityMedium, email.PriorityLow).
		Order("eq.created_at ASC").
		Limit(limit).
		Relation("Profile").
		Relation("Template").
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pending email queue entries")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get_pending").
			Time(time.Now()).
			Wrapf(err, "failed to get pending email queue entries")
	}

	log.Info().
		Int("count", len(queues)).
		Msg("retrieved pending email queue entries")

	return queues, nil
}

// GetScheduled retrieves scheduled emails that are due
func (r *emailQueueRepository) GetScheduled(
	ctx context.Context,
	limit int,
) ([]*email.Queue, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get_scheduled").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetScheduled").
		Int("limit", limit).
		Logger()

	var queues []*email.Queue
	now := time.Now().Unix()

	err = dba.NewSelect().
		Model(&queues).
		Where("eq.status = ?", email.QueueStatusScheduled).
		Where("eq.scheduled_at <= ?", now).
		OrderExpr("CASE eq.priority WHEN ? THEN 1 WHEN ? THEN 2 WHEN ? THEN 3 END",
			email.PriorityHigh, email.PriorityMedium, email.PriorityLow).
		Order("eq.scheduled_at ASC").
		Limit(limit).
		Relation("Profile").
		Relation("Template").
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get scheduled email queue entries")
		return nil, oops.
			In("email_queue_repository").
			Tags("operation", "get_scheduled").
			Time(time.Now()).
			Wrapf(err, "failed to get scheduled email queue entries")
	}

	log.Info().
		Int("count", len(queues)).
		Msg("retrieved scheduled email queue entries")

	return queues, nil
}

func (r *emailQueueRepository) MarkAsSent(
	ctx context.Context,
	queueID pulid.ID,
	messageID string,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_sent").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "MarkAsSent").
		Str("queueID", queueID.String()).
		Str("messageID", messageID).
		Logger()

	results, err := dba.NewUpdate().
		Model(&email.Queue{}).
		Where("eq.id = ?", queueID).
		Set("status = ?", email.QueueStatusSent).
		Set("sent_at = ?", time.Now().Unix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to mark email as sent")
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_sent").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to mark email as sent")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_sent").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Email queue entry not found")
	}

	log.Info().
		Str("queueID", queueID.String()).
		Str("messageID", messageID).
		Msg("email marked as sent")

	return nil
}

func (r *emailQueueRepository) MarkAsFailed(
	ctx context.Context,
	queueID pulid.ID,
	errorMessage string,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_failed").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "MarkAsFailed").
		Str("queueID", queueID.String()).
		Str("errorMessage", errorMessage).
		Logger()

	results, err := dba.NewUpdate().
		Model(&email.Queue{}).
		Where("eq.id = ?", queueID).
		Set("status = ?", email.QueueStatusFailed).
		Set("error_message = ?", errorMessage).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to mark email as failed")
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_failed").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to mark email as failed")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("email_queue_repository").
			Tags("operation", "mark_as_failed").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Email queue entry not found")
	}

	log.Info().
		Str("queueID", queueID.String()).
		Str("errorMessage", errorMessage).
		Msg("email marked as failed")

	return nil
}

func (r *emailQueueRepository) IncrementRetryCount(ctx context.Context, queueID pulid.ID) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.
			In("email_queue_repository").
			Tags("operation", "increment_retry_count").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "IncrementRetryCount").
		Str("queueID", queueID.String()).
		Logger()

	results, err := dba.NewUpdate().
		Model(&email.Queue{}).
		Where("eq.id = ?", queueID).
		Set("retry_count = retry_count + 1").
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to increment retry count")
		return oops.
			In("email_queue_repository").
			Tags("operation", "increment_retry_count").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to increment retry count")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("email_queue_repository").
			Tags("operation", "increment_retry_count").
			Tags("queue_id", queueID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Email queue entry not found")
	}

	log.Info().
		Str("queueID", queueID.String()).
		Msg("retry count incremented")

	return nil
}
