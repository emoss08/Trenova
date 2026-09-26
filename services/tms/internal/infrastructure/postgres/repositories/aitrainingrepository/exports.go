package aitrainingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	exportEntity        = "AITrainingExport"
	defaultListLimit    = 20
	maxListLimit        = 200
	defaultPeopleLimit  = 500
	defaultConsentLimit = 100
	maxConsentLimit     = 1000
	maxPeopleLimit      = 5000
	grantedColumn       = "granted"
	grantedAtColumn     = "granted_at"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type exportRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewExports(p Params) repositories.AITrainingExportRepository {
	return &exportRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aitraining-export-repository"),
	}
}

func (r *exportRepository) Create(
	ctx context.Context,
	entity *aitraining.TrainingExport,
) (*aitraining.TrainingExport, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewBusinessError(
				"A training export is already running; wait for it to finish or cancel it",
			).WithInternal(err)
		}
		r.l.Error("failed to create training export", zap.Error(err))

		return nil, fmt.Errorf("create training export: %w", err)
	}

	return entity, nil
}

func (r *exportRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*aitraining.TrainingExport, error) {
	entity := new(aitraining.TrainingExport)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(buncolgen.TrainingExportColumns.ID.Eq(), id).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, exportEntity)
	}

	return entity, nil
}

func (r *exportRepository) List(
	ctx context.Context,
	req repositories.ListAITrainingExportsRequest,
) ([]*aitraining.TrainingExport, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit)

	entities := make([]*aitraining.TrainingExport, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Order(
			buncolgen.TrainingExportColumns.CreatedAt.OrderDesc(),
			buncolgen.TrainingExportColumns.ID.OrderDesc(),
		).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list training exports", zap.Error(err))

		return nil, fmt.Errorf("list training exports: %w", err)
	}

	return entities, nil
}

func (r *exportRepository) Update(
	ctx context.Context,
	entity *aitraining.TrainingExport,
) (*aitraining.TrainingExport, error) {
	cols := buncolgen.TrainingExportColumns
	previous := entity.Version

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where(cols.ID.Eq(), entity.ID).
		Where(cols.Version.Eq(), previous).
		ExcludeColumn(cols.ID.Bare(), cols.CreatedAt.Bare(), cols.Version.Bare()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update training export", zap.Error(err))

		return nil, fmt.Errorf("update training export: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, exportEntity, entity.ID.String()); err != nil {
		return nil, err
	}
	entity.Version = previous + 1

	return entity, nil
}

func (r *exportRepository) consentQuery(ctx context.Context) *bun.SelectQuery {
	cols := buncolgen.AgentControlColumns

	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.AgentControl)(nil)).
		Column(cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		ColumnExpr(cols.AITrainingConsent.As(grantedColumn)).
		ColumnExpr("COALESCE(?, 0) AS "+grantedAtColumn, bun.Safe(cols.AITrainingConsentChangedAt.Qualified()))
}

func (r *exportRepository) ListConsentingOrganizations(
	ctx context.Context,
	req repositories.ListConsentingOrganizationsRequest,
) ([]repositories.TrainingConsent, error) {
	cols := buncolgen.AgentControlColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultConsentLimit
	}
	limit = min(limit, maxConsentLimit)

	consents := make([]repositories.TrainingConsent, 0, limit)
	query := r.consentQuery(ctx).Where(cols.AITrainingConsent.IsTrue())
	if req.AfterOrganizationID.IsNotNil() {
		query = query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.OrganizationID.Gt(), req.AfterOrganizationID).
				WhereGroup(" OR ", func(tq *bun.SelectQuery) *bun.SelectQuery {
					return tq.Where(cols.OrganizationID.Eq(), req.AfterOrganizationID).
						Where(cols.BusinessUnitID.Gt(), req.AfterBusinessUnitID)
				})
		})
	}
	err := query.
		Order(cols.OrganizationID.OrderAsc(), cols.BusinessUnitID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &consents)
	if err != nil {
		r.l.Error("failed to list consenting organizations", zap.Error(err))

		return nil, fmt.Errorf("list consenting organizations: %w", err)
	}

	return consents, nil
}

func (r *exportRepository) GetConsent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (repositories.TrainingConsent, error) {
	consents := make([]repositories.TrainingConsent, 0, 1)
	err := r.consentQuery(ctx).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentControlScopeTenant(sq, tenantInfo)
		}).
		Limit(1).
		Scan(ctx, &consents)
	if err != nil {
		r.l.Error("failed to read training consent", zap.Error(err))

		return repositories.TrainingConsent{}, fmt.Errorf("read training consent: %w", err)
	}
	if len(consents) == 0 {
		return repositories.TrainingConsent{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
		}, nil
	}

	return consents[0], nil
}

func (r *exportRepository) ListOrganizationPeople(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]string, error) {
	if limit <= 0 {
		limit = defaultPeopleLimit
	}
	limit = min(limit, maxPeopleLimit)

	users := buncolgen.UserColumns
	memberships := buncolgen.OrganizationMembershipColumns
	table := buncolgen.OrganizationMembershipTable
	names := make([]string, 0, min(limit, defaultPeopleLimit))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.User)(nil)).
		Distinct().
		Column(users.Name.Bare()).
		Join(
			"JOIN ? AS ? ON ?",
			bun.Ident(table.Name),
			bun.Ident(table.Alias),
			bun.Safe(memberships.UserID.EqColumn(users.ID)),
		).
		Where(memberships.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(memberships.BusinessUnitID.Eq(), tenantInfo.BuID).
		Order(users.Name.OrderAsc()).
		Limit(limit).
		Scan(ctx, &names)
	if err != nil {
		r.l.Error("failed to list organization people", zap.Error(err))

		return nil, fmt.Errorf("list organization people: %w", err)
	}

	return names, nil
}
