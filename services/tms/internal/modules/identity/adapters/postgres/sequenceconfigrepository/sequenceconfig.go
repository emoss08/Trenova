package sequenceconfigrepository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig/sequenceconfigdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Sequence configuration is busy. Retry the request."

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.SequenceConfigRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sequenceconfig-repository"),
	}
}

func (r *repository) GetByTenant(
	ctx context.Context,
	req tenantports.GetSequenceConfigRequest,
) (*sequenceconfig.SequenceConfigDocument, error) {
	var found []*sequenceconfig.SequenceConfig

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*sequenceconfig.SequenceConfigDocument]{
		Find: func(ctx context.Context) (*sequenceconfig.SequenceConfigDocument, error) {
			configs, err := r.listRequired(ctx)
			if err != nil {
				return nil, err
			}
			found = configs
			if len(configs) < len(sequenceconfig.RequiredSequenceTypes()) {
				return nil, sql.ErrNoRows
			}

			return sequenceConfigDocument(req.TenantInfo, configs), nil
		},
		Create: func(ctx context.Context) error {
			return r.ensureDefaults(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
		},
		Default: func() *sequenceconfig.SequenceConfigDocument {
			return sequenceConfigDocument(
				req.TenantInfo,
				withDefaults(found, req.TenantInfo.OrgID, req.TenantInfo.BuID),
			)
		},
	})
}

func sequenceConfigDocument(
	tenantInfo pagination.TenantInfo,
	configs []*sequenceconfig.SequenceConfig,
) *sequenceconfig.SequenceConfigDocument {
	sort.Slice(configs, func(i, j int) bool {
		return sequenceconfig.SequenceTypeSortOrder(
			configs[i].SequenceType,
		) < sequenceconfig.SequenceTypeSortOrder(
			configs[j].SequenceType,
		)
	})
	normalizeSequenceConfigs(configs)

	return &sequenceconfig.SequenceConfigDocument{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Configs:        configs,
	}
}

func withDefaults(
	saved []*sequenceconfig.SequenceConfig,
	orgID, buID pulid.ID,
) []*sequenceconfig.SequenceConfig {
	bySequence := make(map[sequenceconfig.SequenceType]*sequenceconfig.SequenceConfig, len(saved))
	for _, cfg := range saved {
		bySequence[cfg.SequenceType] = cfg
	}

	configs := make(
		[]*sequenceconfig.SequenceConfig,
		0,
		len(sequenceconfig.RequiredSequenceTypes()),
	)
	for _, sequenceType := range sequenceconfig.RequiredSequenceTypes() {
		if cfg, ok := bySequence[sequenceType]; ok {
			configs = append(configs, cfg)
			continue
		}
		if cfg := sequenceconfig.DefaultSequenceConfig(orgID, buID, sequenceType); cfg != nil {
			configs = append(configs, cfg)
		}
	}

	return configs
}

func (r *repository) listRequired(ctx context.Context) ([]*sequenceconfig.SequenceConfig, error) {
	sc := sequenceconfigdb.SequenceConfigs
	requiredTypes := sequenceconfig.RequiredSequenceTypes()

	return sc.Where(sc.SequenceType.In(requiredTypes...)).
		Limit(len(requiredTypes)).
		Cache(resultcache.Settings()...).
		List(ctx, r.db.Keel(ctx))
}

func normalizeSequenceConfigs(configs []*sequenceconfig.SequenceConfig) {
	for _, cfg := range configs {
		if cfg == nil || cfg.SequenceType != sequenceconfig.SequenceTypeLocationCode {
			continue
		}

		cfg.LocationCodeStrategy = sequenceconfig.EffectiveLocationCodeStrategy(
			cfg.LocationCodeStrategy,
		)
	}
}

func (r *repository) UpdateByTenant(
	ctx context.Context,
	doc *sequenceconfig.SequenceConfigDocument,
) (*sequenceconfig.SequenceConfigDocument, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*sequenceconfig.SequenceConfigDocument, error) {
			configs := make([]*sequenceconfig.SequenceConfig, 0, len(doc.Configs))
			for _, cfg := range doc.Configs {
				if cfg == nil {
					continue
				}

				cfg.OrganizationID = doc.OrganizationID
				cfg.BusinessUnitID = doc.BusinessUnitID
				configs = append(configs, cfg)
			}

			if len(configs) > 0 {
				sc := sequenceconfigdb.SequenceConfigs
				if _, err := sc.Upsert(
					ctx,
					r.db.Keel(ctx),
					configs,
					keel.OnConflict(sc.SequenceType, sc.OrganizationID, sc.BusinessUnitID).Update(
						sc.Prefix,
						sc.IncludeYear,
						sc.YearDigits,
						sc.IncludeMonth,
						sc.IncludeWeekNumber,
						sc.IncludeDay,
						sc.SequenceDigits,
						sc.IncludeLocationCode,
						sc.IncludeRandomDigits,
						sc.RandomDigitsCount,
						sc.IncludeCheckDigit,
						sc.IncludeBusinessUnitCode,
						sc.UseSeparators,
						sc.SeparatorChar,
						sc.AllowCustomFormat,
						sc.CustomFormat,
						sc.LocationCodeStrategy,
					),
				); err != nil {
					return nil, dberror.MapRetryableTransactionError(
						fmt.Errorf("upsert sequence configs: %w", err),
						busyMessage,
					)
				}
			}

			return r.GetByTenant(ctx, tenantports.GetSequenceConfigRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: doc.OrganizationID,
					BuID:  doc.BusinessUnitID,
				},
			})
		},
	)
}

func (r *repository) ensureDefaults(ctx context.Context, orgID, buID pulid.ID) error {
	defaults := make(
		[]*sequenceconfig.SequenceConfig,
		0,
		len(sequenceconfig.RequiredSequenceTypes()),
	)
	for _, sequenceType := range sequenceconfig.RequiredSequenceTypes() {
		if cfg := sequenceconfig.DefaultSequenceConfig(orgID, buID, sequenceType); cfg != nil {
			defaults = append(defaults, cfg)
		}
	}
	if len(defaults) == 0 {
		return nil
	}

	sc := sequenceconfigdb.SequenceConfigs
	if _, err := sc.CreateOrSkip(
		ctx,
		r.db.Keel(ctx),
		defaults,
		sc.SequenceType,
		sc.OrganizationID,
		sc.BusinessUnitID,
	); err != nil {
		return fmt.Errorf("seed default sequence config: %w", err)
	}

	return nil
}
