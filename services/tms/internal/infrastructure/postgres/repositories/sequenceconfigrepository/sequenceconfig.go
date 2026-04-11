package sequenceconfigrepository

import (
	"context"
	"fmt"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.SequenceConfigRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sequenceconfig-repository"),
	}
}

func (r *repository) GetByTenant(
	ctx context.Context,
	req repositories.GetSequenceConfigRequest,
) (*tenant.SequenceConfigDocument, error) {
	if err := r.ensureDefaults(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID); err != nil {
		return nil, err
	}

	configs := make([]*tenant.SequenceConfig, 0, len(tenant.RequiredSequenceTypes()))
	requiredTypes := tenant.RequiredSequenceTypes()
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&configs).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Where("sequence_type IN (?)", bun.List(requiredTypes)).
		Scan(ctx); err != nil {
		return nil, err
	}

	sort.Slice(configs, func(i, j int) bool {
		return tenant.SequenceTypeSortOrder(
			configs[i].SequenceType,
		) < tenant.SequenceTypeSortOrder(
			configs[j].SequenceType,
		)
	})

	return &tenant.SequenceConfigDocument{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Configs:        configs,
	}, nil
}

func (r *repository) UpdateByTenant(
	ctx context.Context,
	doc *tenant.SequenceConfigDocument,
) (*tenant.SequenceConfigDocument, error) {
	now := timeutils.NowUnix()

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		for _, cfg := range doc.Configs {
			if cfg == nil {
				continue
			}

			cfg.OrganizationID = doc.OrganizationID
			cfg.BusinessUnitID = doc.BusinessUnitID
			if cfg.ID.IsNil() {
				cfg.ID = pulid.MustNew("sqcfg_")
			}
			if cfg.CreatedAt == 0 {
				cfg.CreatedAt = now
			}
			cfg.UpdatedAt = now

			_, err := r.db.DBForContext(c).NewInsert().
				Model(cfg).
				On(`CONFLICT (sequence_type, organization_id, business_unit_id) DO UPDATE`).
				Set("prefix = EXCLUDED.prefix").
				Set("include_year = EXCLUDED.include_year").
				Set("year_digits = EXCLUDED.year_digits").
				Set("include_month = EXCLUDED.include_month").
				Set("include_week_number = EXCLUDED.include_week_number").
				Set("include_day = EXCLUDED.include_day").
				Set("sequence_digits = EXCLUDED.sequence_digits").
				Set("include_location_code = EXCLUDED.include_location_code").
				Set("include_random_digits = EXCLUDED.include_random_digits").
				Set("random_digits_count = EXCLUDED.random_digits_count").
				Set("include_check_digit = EXCLUDED.include_check_digit").
				Set("include_business_unit_code = EXCLUDED.include_business_unit_code").
				Set("use_separators = EXCLUDED.use_separators").
				Set("separator_char = EXCLUDED.separator_char").
				Set("allow_custom_format = EXCLUDED.allow_custom_format").
				Set("custom_format = EXCLUDED.custom_format").
				Set("version = sequence_configs.version + 1").
				Set("updated_at = EXCLUDED.updated_at").
				Returning("*").
				Exec(c)
			if err != nil {
				return fmt.Errorf("upsert sequence config %s: %w", cfg.SequenceType, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Sequence configuration is busy. Retry the request.",
		)
	}

	return r.GetByTenant(ctx, repositories.GetSequenceConfigRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: doc.OrganizationID,
			BuID:  doc.BusinessUnitID,
		},
	})
}

func (r *repository) ensureDefaults(ctx context.Context, orgID, buID pulid.ID) error {
	now := timeutils.NowUnix()
	defaults := make([]*tenant.SequenceConfig, 0, len(tenant.RequiredSequenceTypes()))
	for _, sequenceType := range tenant.RequiredSequenceTypes() {
		cfg := tenant.DefaultSequenceConfig(orgID, buID, sequenceType)
		if cfg == nil {
			continue
		}

		defaults = append(defaults, cfg)
	}

	for _, cfg := range defaults {
		cfg.CreatedAt = now
		cfg.UpdatedAt = now
		_, err := r.db.DBForContext(ctx).NewInsert().
			Model(cfg).
			On("CONFLICT (sequence_type, organization_id, business_unit_id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("seed default sequence config: %w", err)
		}
	}

	return nil
}
