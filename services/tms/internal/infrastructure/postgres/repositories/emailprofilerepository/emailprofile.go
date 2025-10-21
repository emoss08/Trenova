package emailprofilerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/encryption"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                *postgres.Connection
	Logger            *zap.Logger
	EncryptionService encryption.Service
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
	es encryption.Service
}

func NewRepository(p Params) repositories.EmailProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.emailprofile-repository"),
		es: p.EncryptionService,
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEmailProfileRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ep",
		req.Filter,
		(*email.EmailProfile)(nil),
	)

	if req.ExcludeInactive {
		q = q.Where("ep.status = ?", domain.StatusActive)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEmailProfileRequest,
) (*pagination.ListResult[*email.EmailProfile], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgID", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buID", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*email.EmailProfile, 0, req.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan email profiles", zap.Error(err))
		return nil, err
	}

	for _, profile := range entities {
		if dErr := r.decryptSensitiveFields(profile); dErr != nil {
			log.Error(
				"failed to decrypt profile fields",
				zap.Error(dErr),
				zap.String("profileID", profile.ID.String()),
			)
			// ! Continue with other profiles rather than failing entirely
		}
	}

	return &pagination.ListResult[*email.EmailProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *email.EmailProfile,
) (*email.EmailProfile, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("name", entity.Name),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if err = r.encryptSensitiveFields(entity); err != nil {
		log.Error("failed to encrypt sensitive fields", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if err = r.unsetExistingDefault(c, tx, entity.OrganizationID); err != nil {
				log.Error("failed to unset existing default", zap.Error(err))
				return err
			}
		}

		if _, err = tx.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert email profile", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *email.EmailProfile,
) (*email.EmailProfile, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("profileID", entity.ID.String()),
		zap.Int64("version", entity.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if err = r.encryptSensitiveFields(entity); err != nil {
		log.Error("failed to encrypt sensitive fields", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if tErr := r.unsetExistingDefaultTx(c, tx, entity.OrganizationID, entity.ID); tErr != nil {
				return tErr
			}
		}

		ov := entity.Version
		entity.Version++

		results, rErr := tx.NewUpdate().
			Model(entity).
			WherePK().
			Where("ep.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update email profile", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Email Profile", entity.ID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		log.Error("failed to update email profile", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.EmailProfile, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("profileID", req.ProfileID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(email.EmailProfile)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ep.id = ?", req.ProfileID).
				Where("ep.organization_id = ?", req.OrgID).
				Where("ep.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get email profile", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EmailProfile")
	}

	if err = r.decryptSensitiveFields(entity); err != nil {
		log.Error("failed to decrypt sensitive fields", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetDefault(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*email.EmailProfile, error) {
	log := r.l.With(
		zap.String("operation", "GetDefault"),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(email.EmailProfile)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ep.organization_id = ?", orgID).
				Where("ep.business_unit_id = ?", buID).
				Where("ep.status = ?", domain.StatusActive).
				Where("ep.is_default = ?", true)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get default email profile", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "EmailProfile")
	}

	if err = r.decryptSensitiveFields(entity); err != nil {
		log.Error("failed to decrypt sensitive fields", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) encryptSensitiveFields(profile *email.EmailProfile) error {
	if profile.Password != "" {
		encrypted, err := r.es.Encrypt(profile.Password)
		if err != nil {
			return err
		}
		profile.EncryptedPassword = encrypted
		profile.Password = ""
	}

	if profile.APIKey != "" {
		encrypted, err := r.es.Encrypt(profile.APIKey)
		if err != nil {
			return err
		}
		profile.EncryptedAPIKey = encrypted
		profile.APIKey = ""
	}

	if profile.OAuth2ClientSecret != "" {
		encrypted, err := r.es.Encrypt(profile.OAuth2ClientSecret)
		if err != nil {
			return err
		}
		profile.OAuth2ClientSecret = encrypted
	}

	return nil
}

func (r *repository) decryptSensitiveFields(profile *email.EmailProfile) error {
	if profile.EncryptedPassword != "" {
		decrypted, err := r.es.Decrypt(profile.EncryptedPassword)
		if err != nil {
			return err
		}
		profile.Password = decrypted
	}

	if profile.EncryptedAPIKey != "" {
		decrypted, err := r.es.Decrypt(profile.EncryptedAPIKey)
		if err != nil {
			return err
		}
		profile.APIKey = decrypted
	}

	return nil
}

func (r *repository) unsetExistingDefault(
	ctx context.Context,
	tx bun.IDB,
	orgID pulid.ID,
) error {
	_, err := tx.NewUpdate().
		Model((*email.EmailProfile)(nil)).
		Set("is_default = ?", false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("ep.organization_id = ?", orgID).
				Where("ep.is_default = ?", true)
		}).
		Exec(ctx)

	return err
}

func (r *repository) unsetExistingDefaultTx(
	ctx context.Context,
	tx bun.IDB,
	orgID pulid.ID,
	excludeID pulid.ID,
) error {
	_, err := tx.NewUpdate().
		Model((*email.EmailProfile)(nil)).
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
