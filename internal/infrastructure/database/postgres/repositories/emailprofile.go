package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/encryption"
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

// EmailProfileRepositoryParams defines dependencies for the email profile repository
type EmailProfileRepositoryParams struct {
	fx.In

	DB                db.Connection
	Logger            *logger.Logger
	EncryptionService encryption.Service
}

// emailProfileRepository implements the EmailProfileRepository interface
type emailProfileRepository struct {
	db                db.Connection
	l                 *zerolog.Logger
	encryptionService encryption.Service
}

// NewEmailProfileRepository creates a new email profile repository instance
func NewEmailProfileRepository(p EmailProfileRepositoryParams) repositories.EmailProfileRepository {
	log := p.Logger.With().
		Str("repository", "email_profile").
		Logger()

	return &emailProfileRepository{
		db:                p.DB,
		l:                 &log,
		encryptionService: p.EncryptionService,
	}
}

func (r *emailProfileRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEmailProfileRequest,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"ep",
		repositories.EmailProfileFieldConfig,
		(*email.Profile)(nil),
	)

	qb.ApplyTenantFilters(req.Filter.TenantOpts)

	if req.Filter != nil {
		qb.ApplyFilters(req.Filter.FieldFilters)

		if len(req.Filter.Sort) > 0 {
			qb.ApplySort(req.Filter.Sort)
		}

		if req.Filter.Query != "" {
			qb.ApplyTextSearch(req.Filter.Query, []string{"name", "host", "description"})
		}

		q = qb.GetQuery()
	}

	if req.ExcludeInactive {
		q = q.Where("ep.status = ?", domain.StatusActive)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List retrieves a list of email profiles with pagination
func (r *emailProfileRepository) List(
	ctx context.Context,
	req *repositories.ListEmailProfileRequest,
) (*ports.ListResult[*email.Profile], error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	profiles := make([]*email.Profile, 0)

	q := dba.NewSelect().Model(&profiles)

	q = r.filterQuery(q, req)

	log.Info().Interface("req", req).Msg("req")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan email profiles")
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "list").
			Time(time.Now()).
			Wrapf(err, "failed to list email profiles")
	}

	// Decrypt sensitive fields for all profiles
	for _, profile := range profiles {
		if dErr := r.decryptSensitiveFields(profile); dErr != nil {
			log.Error().
				Err(dErr).
				Str("profileID", profile.ID.String()).
				Msg("failed to decrypt profile fields")
			// Continue with other profiles rather than failing entirely
		}
	}

	return &ports.ListResult[*email.Profile]{
		Items: profiles,
		Total: total,
	}, nil
}

// Create creates a new email profile with encrypted credentials
func (r *emailProfileRepository) Create(
	ctx context.Context,
	profile *email.Profile,
) (*email.Profile, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", profile.OrganizationID.String()).
		Str("name", profile.Name).
		Logger()

	// Encrypt sensitive fields
	if err = r.encryptSensitiveFields(profile); err != nil {
		log.Error().Err(err).Msg("failed to encrypt sensitive fields")
		return nil, err
	}

	// If this is marked as default, unset any existing default
	if profile.IsDefault {
		if err = r.unsetExistingDefault(ctx, dba, profile.OrganizationID); err != nil {
			log.Error().Err(err).Msg("failed to unset existing default")
			return nil, err
		}
	}

	if _, err = dba.NewInsert().Model(profile).Returning("*").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to insert email profile")
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "create").
			Time(time.Now()).
			Wrapf(err, "failed to insert email profile")
	}

	log.Info().
		Str("profileID", profile.ID.String()).
		Bool("isDefault", profile.IsDefault).
		Msg("email profile created successfully")

	return profile, nil
}

// Update updates an existing email profile
func (r *emailProfileRepository) Update(
	ctx context.Context,
	profile *email.Profile,
) (*email.Profile, error) {
	dba, err := r.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "update").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("profileID", profile.ID.String()).
		Int64("version", profile.Version).
		Logger()

	// Encrypt sensitive fields
	if err = r.encryptSensitiveFields(profile); err != nil {
		log.Error().Err(err).Msg("failed to encrypt sensitive fields")
		return nil, err
	}

	// If this is marked as default, unset any existing default
	if profile.IsDefault {
		if err = r.unsetExistingDefault(ctx, dba, profile.OrganizationID); err != nil {
			log.Error().Err(err).Msg("failed to unset existing default")
			return nil, err
		}
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if profile.IsDefault {
			if tErr := r.unsetExistingDefaultTx(c, tx, profile.OrganizationID, profile.ID); tErr != nil {
				return tErr
			}
		}

		ov := profile.Version
		profile.Version++

		results, rErr := tx.NewUpdate().
			Model(profile).
			WherePK().
			Where("ep.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update email profile")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"Email profile has been updated or deleted since last request",
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update email profile")
		return nil, err
	}

	log.Info().
		Str("profileID", profile.ID.String()).
		Bool("isDefault", profile.IsDefault).
		Msg("email profile updated successfully")

	return profile, nil
}

// Get retrieves an email profile by ID
func (r *emailProfileRepository) Get(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "get").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Get").
		Str("profileID", req.ProfileID.String()).
		Logger()

	profile := new(email.Profile)

	err = dba.NewSelect().
		Model(profile).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ep.id = ?", req.ProfileID).
				Where("ep.organization_id = ?", req.OrgID).
				Where("ep.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("email profile not found")
			return nil, errors.NewNotFoundError("Email profile not found")
		}
		log.Error().Err(err).Msg("failed to get email profile")
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "get").
			Tags("profile_id", req.ProfileID.String()).
			Time(time.Now()).
			Wrap(err)
	}

	// Decrypt sensitive fields
	if err = r.decryptSensitiveFields(profile); err != nil {
		log.Error().Err(err).Msg("failed to decrypt sensitive fields")
		return nil, err
	}

	return profile, nil
}

// GetDefault retrieves the default email profile for an organization
func (r *emailProfileRepository) GetDefault(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*email.Profile, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "get_default").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetDefault").
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	profile := new(email.Profile)

	err = dba.NewSelect().
		Model(profile).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ep.organization_id = ?", orgID).
				Where("ep.business_unit_id = ?", buID).
				Where("ep.status = ?", domain.StatusActive).
				Where("ep.is_default = ?", true)
		}).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Debug().Msg("no default email profile found")
			return nil, errors.NewNotFoundError("No active default email profile configured")
		}
		log.Error().Err(err).Msg("failed to get default email profile")
		return nil, oops.
			In("email_profile_repository").
			Tags("operation", "get_default").
			Tags("orgID", orgID.String()).
			Tags("buID", buID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get default email profile")
	}

	// Decrypt sensitive fields
	if err = r.decryptSensitiveFields(profile); err != nil {
		log.Error().Err(err).Msg("failed to decrypt sensitive fields")
		return nil, err
	}

	return profile, nil
}

// Helper methods

// encryptSensitiveFields encrypts password and API key fields
func (r *emailProfileRepository) encryptSensitiveFields(profile *email.Profile) error {
	// Encrypt password if present in transient field
	if profile.Password != "" {
		encrypted, err := r.encryptionService.Encrypt(profile.Password)
		if err != nil {
			return oops.
				In("email_profile_repository").
				Tags("operation", "encrypt_password").
				Time(time.Now()).
				Wrapf(err, "failed to encrypt password")
		}
		profile.EncryptedPassword = encrypted
		profile.Password = "" // Clear the plain text
	}

	// Encrypt API key if present in transient field
	if profile.APIKey != "" {
		encrypted, err := r.encryptionService.Encrypt(profile.APIKey)
		if err != nil {
			return oops.
				In("email_profile_repository").
				Tags("operation", "encrypt_api_key").
				Time(time.Now()).
				Wrapf(err, "failed to encrypt API key")
		}
		profile.EncryptedAPIKey = encrypted
		profile.APIKey = "" // Clear the plain text
	}

	// Encrypt OAuth2 client secret if present
	if profile.OAuth2ClientSecret != "" {
		encrypted, err := r.encryptionService.Encrypt(profile.OAuth2ClientSecret)
		if err != nil {
			return oops.
				In("email_profile_repository").
				Tags("operation", "encrypt_oauth2_secret").
				Time(time.Now()).
				Wrapf(err, "failed to encrypt OAuth2 client secret")
		}
		profile.OAuth2ClientSecret = encrypted
	}

	return nil
}

// decryptSensitiveFields decrypts password and API key fields
func (r *emailProfileRepository) decryptSensitiveFields(profile *email.Profile) error {
	// Decrypt password if present
	if profile.EncryptedPassword != "" {
		decrypted, err := r.encryptionService.Decrypt(profile.EncryptedPassword)
		if err != nil {
			return oops.
				In("email_profile_repository").
				Tags("operation", "decrypt_password").
				Time(time.Now()).
				Wrapf(err, "failed to decrypt password")
		}
		profile.Password = decrypted
		// Keep encrypted version for updates
	}

	// Decrypt API key if present
	if profile.EncryptedAPIKey != "" {
		decrypted, err := r.encryptionService.Decrypt(profile.EncryptedAPIKey)
		if err != nil {
			return oops.
				In("email_profile_repository").
				Tags("operation", "decrypt_api_key").
				Time(time.Now()).
				Wrapf(err, "failed to decrypt API key")
		}
		profile.APIKey = decrypted
		// Keep encrypted version for updates
	}

	// Note: OAuth2ClientSecret is typically not decrypted for display
	// It's only decrypted when needed for authentication

	return nil
}

// unsetExistingDefault unsets any existing default profile for the organization
func (r *emailProfileRepository) unsetExistingDefault(
	ctx context.Context,
	dba bun.IDB,
	orgID pulid.ID,
) error {
	_, err := dba.NewUpdate().
		Model((*email.Profile)(nil)).
		Set("is_default = ?", false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("ep.organization_id = ?", orgID).
				Where("ep.is_default = ?", true)
		}).
		Exec(ctx)

	return err
}

// unsetExistingDefaultTx unsets any existing default profile in a transaction
func (r *emailProfileRepository) unsetExistingDefaultTx(
	ctx context.Context,
	tx bun.Tx,
	orgID pulid.ID,
	excludeID pulid.ID,
) error {
	_, err := tx.NewUpdate().
		Model((*email.Profile)(nil)).
		Set("is_default = ?", false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("ep.organization_id = ?", orgID).
				Where("ep.is_default = ?", true).
				Where("ep.id != ?", excludeID)
		}).
		Exec(ctx)

	return err
}
