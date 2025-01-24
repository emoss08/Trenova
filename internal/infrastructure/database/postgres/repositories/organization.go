package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type OrganizationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type organizationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewOrganizationRepository(p OrganizationRepositoryParams) repositories.OrganizationRepository {
	log := p.Logger.With().
		Str("repository", "organization").
		Logger()

	return &organizationRepository{
		db: p.DB,
		l:  &log,
	}
}

// TODO(Wolfred): Cache the organization because it should not change often.
// filterQuery returns a query that filters organizations by the given options.
func (or *organizationRepository) filterQuery(q *bun.SelectQuery, f *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	return q.Where("org.business_unit_id = ?", f.TenantOpts.BuID).
		Limit(f.Limit).
		Offset(f.Offset)
}

// List returns a list of organizations for a business unit.
func (or *organizationRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*organization.Organization], error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := or.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	organizations := make([]*organization.Organization, 0)

	q := dba.NewSelect().Model(&organizations)
	q = or.filterQuery(q, opts)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count organizations")
		return nil, eris.Wrap(err, "failed to scan and count organizations")
	}

	return &ports.ListResult[*organization.Organization]{
		Items: organizations,
		Total: count,
	}, nil
}

// GetByID returns an organization by its ID.
func (or *organizationRepository) GetByID(ctx context.Context, opts repositories.GetOrgByIDOptions) (*organization.Organization, error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := or.l.With().
		Str("operation", "GetByID").
		Str("buID", opts.BuID.String()).
		Str("orgID", opts.OrgID.String()).
		Logger()

	org := new(organization.Organization)

	q := dba.NewSelect().Model(org).
		Where("org.id = ?", opts.OrgID).
		Where("org.business_unit_id = ?", opts.BuID)

	// Include the state if requested
	if opts.IncludeState {
		q.Relation("State")
	}

	// Include the business unit if requested
	if opts.IncludeBu {
		q.Relation("BusinessUnit")
	}

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewValidationError("id", errors.ErrNotFound, "Organization not found within your business unit")
		}

		log.Error().Err(err).Msgf("failed to get organization by ID %s", opts.OrgID)
		return nil, eris.Wrapf(err, "failed to get organization by ID %s", opts.OrgID)
	}

	return org, nil
}

// Create creates an organization and audits the creation.
func (or *organizationRepository) Create(
	ctx context.Context, org *organization.Organization,
) (*organization.Organization, error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := or.l.With().
		Str("operation", "Create").
		Str("scacCode", org.ScacCode).
		Str("businessUnitID", org.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(org).Exec(c); err != nil {
			log.Error().
				Err(err).
				Interface("organization", org).
				Msg("failed to insert organization")
			return eris.Wrap(err, "insert organization")
		}

		log.Info().
			Str("id", org.ID.String()).
			Str("name", org.Name).
			Msg("organization created successfully")

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "create organization")
	}

	return org, nil
}

func (or *organizationRepository) Update(
	ctx context.Context, org *organization.Organization,
) (*organization.Organization, error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := or.l.With().
		Str("operation", "Update").
		Str("orgID", org.ID.String()).
		Int64("version", org.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := org.Version
		org.Version++

		results, rErr := tx.NewUpdate().Model(org).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).
				Interface("organization", org).
				Msg("failed to update organization")
			return eris.Wrap(rErr, "update organization")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The organization (%s) has either been updated or deleted since the last request.", org.ID),
			)
		}
		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "update organization")
	}

	return org, nil
}

// SetLogo sets the logo for an organization.
func (or *organizationRepository) SetLogo(ctx context.Context, org *organization.Organization) (*organization.Organization, error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := or.l.With().
		Str("operation", "SetLogo").
		Str("orgID", org.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		results, rErr := tx.NewUpdate().Model(org).
			WherePK().
			Set("logo_url = ?", org.LogoURL).
			Set("metadata = ?", org.Metadata).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update organization logo")
			return eris.Wrap(rErr, "update organization logo")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			log.Warn().Msg("organization not found")
			return errors.NewNotFoundError("Organization not found")
		}

		log.Info().
			Str("orgID", org.ID.String()).
			Str("logoURL", org.LogoURL).
			Msg("organization logo set successfully")

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to set organization logo")
		return nil, eris.Wrap(err, "set organization logo")
	}

	return org, nil
}

func (or *organizationRepository) ClearLogo(ctx context.Context, org *organization.Organization) (*organization.Organization, error) {
	log := or.l.With().Str("operation", "ClearLogo").Str("orgID", org.ID.String()).Logger()

	original, err := or.GetByID(ctx, repositories.GetOrgByIDOptions{
		OrgID: org.ID,
		BuID:  org.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get original organization")
		return nil, eris.Wrap(err, "get original organization")
	}

	if original.LogoURL == "" {
		log.Warn().Msg("organization logo already cleared")
		return nil, errors.NewValidationError("logo_url", errors.ErrAlreadyCleared, "Organization logo already cleared")
	}

	// Clear the logo URL and metadata before calling SetLogo
	org.LogoURL = ""
	org.Metadata = nil

	updatedOrg, err := or.SetLogo(ctx, org)
	if err != nil {
		return nil, eris.Wrap(err, "set organization logo")
	}

	return updatedOrg, nil
}

func (or *organizationRepository) GetUserOrganizations(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*organization.Organization], error) {
	dba, err := or.db.DB(ctx)
	if err != nil {
		or.l.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	orgs := make([]*organization.Organization, 0)

	q := dba.NewSelect().
		Model(&orgs).
		Relation("State").
		Join("INNER JOIN user_organizations AS uo ON uo.organization_id = org.id").
		Where("uo.user_id = ?", opts.TenantOpts.UserID)

	// Apply the filter query
	q = or.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		or.l.Error().Err(err).Msg("failed to scan organizations")
		return nil, eris.Wrap(err, "scan organizations")
	}

	return &ports.ListResult[*organization.Organization]{
		Items: orgs,
		Total: total,
	}, nil
}
