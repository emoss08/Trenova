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
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// EmailTemplateRepositoryParams defines dependencies for the email template repository
type EmailTemplateRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// emailTemplateRepository implements the EmailTemplateRepository interface
type emailTemplateRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewEmailTemplateRepository creates a new email template repository instance
func NewEmailTemplateRepository(
	p EmailTemplateRepositoryParams,
) repositories.EmailTemplateRepository {
	log := p.Logger.With().
		Str("repository", "email_template").
		Logger()

	return &emailTemplateRepository{
		db: p.DB,
		l:  &log,
	}
}

// Create creates a new email template
func (r *emailTemplateRepository) Create(
	ctx context.Context,
	template *email.Template,
) (*email.Template, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", template.OrganizationID.String()).
		Str("name", template.Name).
		Str("slug", template.Slug).
		Logger()

	// Check if slug already exists
	exists, err := r.slugExists(ctx, dba, template.Slug, template.OrganizationID, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to check slug existence")
		return nil, err
	}
	if exists {
		return nil, errors.NewValidationError(
			"slug",
			errors.ErrDuplicate,
			"Template slug already exists",
		)
	}

	if _, err = dba.NewInsert().Model(template).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to insert email template")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "failed to insert email template")
	}

	log.Info().
		Str("templateID", template.ID.String()).
		Msg("email template created successfully")

	return template, nil
}

// Update updates an existing email template
func (r *emailTemplateRepository) Update(
	ctx context.Context,
	template *email.Template,
) (*email.Template, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("templateID", template.ID.String()).
		Int64("version", template.Version).
		Logger()

	// Check if slug already exists (excluding current template)
	exists, err := r.slugExists(ctx, dba, template.Slug, template.OrganizationID, &template.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to check slug existence")
		return nil, err
	}
	if exists {
		return nil, errors.NewValidationError(
			"slug",
			errors.ErrDuplicate,
			"Template slug already exists",
		)
	}

	ov := template.Version
	template.Version++

	results, err := dba.NewUpdate().
		Model(template).
		WherePK().
		Where("et.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to update email template")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "failed to update email template")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return nil, errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			"Email template has been updated or deleted since last request",
		)
	}

	log.Info().
		Str("templateID", template.ID.String()).
		Msg("email template updated successfully")

	return template, nil
}

// Get retrieves an email template by ID
func (r *emailTemplateRepository) Get(ctx context.Context, id pulid.ID) (*email.Template, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Get").
		Str("templateID", id.String()).
		Logger()

	template := new(email.Template)

	err = dba.NewSelect().
		Model(template).
		Where("et.id = ?", id).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email template not found")
			return nil, errors.NewNotFoundError("Email template not found")
		}
		log.Error().Err(err).Msg("failed to get email template")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "get").
			Tags("template_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email template")
	}

	return template, nil
}

// GetBySlug retrieves an email template by slug
func (r *emailTemplateRepository) GetBySlug(
	ctx context.Context,
	slug string,
	organizationID pulid.ID,
) (*email.Template, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "get_by_slug").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetBySlug").
		Str("slug", slug).
		Str("orgID", organizationID.String()).
		Logger()

	template := new(email.Template)

	err = dba.NewSelect().
		Model(template).
		Where("et.slug = ?", slug).
		Where("et.organization_id = ?", organizationID).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email template not found by slug")
			return nil, errors.NewNotFoundError("Email template not found")
		}
		log.Error().Err(err).Msg("failed to get email template by slug")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "get_by_slug").
			Tags("slug", slug).
			Tags("org_id", organizationID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get email template by slug")
	}

	return template, nil
}

func (r *emailTemplateRepository) filterQuery(
	q *bun.SelectQuery,
	filter *ports.QueryOptions,
) *bun.SelectQuery {
	// Apply filters using query builder
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"et",
		repositories.EmailTemplateFieldConfig,
		(*email.Template)(nil),
	)
	qb.ApplyTenantFilters(filter.TenantOpts)

	if len(filter.FieldFilters) > 0 {
		qb.ApplyFilters(filter.FieldFilters)
	}

	if len(filter.Sort) > 0 {
		qb.ApplySort(filter.Sort)
	}

	if filter.Query != "" {
		qb.ApplyTextSearch(filter.Query, []string{"name", "slug", "description"})
	}

	q = qb.GetQuery()

	return q.Limit(filter.Limit).Offset(filter.Offset)
}

// List retrieves a list of email templates with pagination
func (r *emailTemplateRepository) List(
	ctx context.Context,
	filter *ports.QueryOptions,
) (*ports.ListResult[*email.Template], error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("orgID", filter.TenantOpts.OrgID.String()).
		Str("buID", filter.TenantOpts.BuID.String()).
		Logger()

	templates := make([]*email.Template, 0)

	q := dba.NewSelect().Model(&templates)

	q = r.filterQuery(q, filter)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan email templates")
		return nil, oops.
			In("email_template_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "failed to list email templates")
	}

	return &ports.ListResult[*email.Template]{
		Items: templates,
		Total: total,
	}, nil
}

// Delete deletes an email template
func (r *emailTemplateRepository) Delete(ctx context.Context, id pulid.ID) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.
			In("email_template_repository").
			Tags("operation", "delete").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Delete").
		Str("templateID", id.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*email.Template)(nil)).
		Where("et.id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete email template")
		return oops.
			In("email_template_repository").
			Tags("operation", "delete").
			Tags("template_id", id.String()).
			Time(time.Now()).
			Wrapf(err, "failed to delete email template")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("email_template_repository").
			Tags("operation", "delete").
			Time(time.Now()).
			Wrapf(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Email template not found")
	}

	log.Info().Msg("email template deleted successfully")
	return nil
}

// Helper methods

// slugExists checks if a slug already exists for the organization
func (r *emailTemplateRepository) slugExists(
	ctx context.Context,
	dba bun.IDB,
	slug string,
	orgID pulid.ID,
	excludeID *pulid.ID,
) (bool, error) {
	q := dba.NewSelect().
		Model((*email.Template)(nil)).
		Where("et.slug = ?", slug).
		Where("et.organization_id = ?", orgID)

	if excludeID != nil {
		q = q.Where("et.id != ?", *excludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		return false, oops.
			In("email_template_repository").
			Tags("operation", "check_slug_exists").
			Tags("slug", slug).
			Time(time.Now()).
			Wrapf(err, "failed to check slug existence")
	}

	return exists, nil
}
