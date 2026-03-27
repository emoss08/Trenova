package tcaallowlistrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TCAAllowlistRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tca-allowlist-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*tablechangealert.TCAAllowlistedTable, error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*tablechangealert.TCAAllowlistedTable, 0)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Where("tcaw.organization_id = ?", tenantInfo.OrgID).
		Where("tcaw.business_unit_id = ?", tenantInfo.BuID).
		Where("tcaw.enabled = ?", true).
		Order("tcaw.display_name ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to list tca allowlisted tables", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) IsTableAllowed(
	ctx context.Context,
	tableName string,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	log := r.l.With(
		zap.String("operation", "IsTableAllowed"),
		zap.String("tableName", tableName),
	)

	exists, err := r.db.DB().
		NewSelect().
		Model((*tablechangealert.TCAAllowlistedTable)(nil)).
		Where("tcaw.organization_id = ?", tenantInfo.OrgID).
		Where("tcaw.business_unit_id = ?", tenantInfo.BuID).
		Where("tcaw.table_name = ?", tableName).
		Where("tcaw.enabled = ?", true).
		Exists(ctx)
	if err != nil {
		log.Error("failed to check if table is allowed", zap.Error(err))
		return false, err
	}

	return exists, nil
}
