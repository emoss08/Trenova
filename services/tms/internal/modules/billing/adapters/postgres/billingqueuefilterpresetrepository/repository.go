package billingqueuefilterpresetrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset/billingqueuefilterpresetdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueuefilterpresetports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var presetErrors = dberror.Mapping{Entity: "BillingQueueFilterPreset"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) billingqueuefilterpresetports.BillingQueueFilterPresetRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-filter-preset-repository"),
	}
}

func (r *repository) ListByUserID(
	ctx context.Context,
	req *billingqueuefilterpresetports.ListBillingQueueFilterPresetsRequest,
) ([]*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
			p := billingqueuefilterpresetdb.BillingQueueFilterPresets
			entities, err := p.Where(p.UserID.Eq(req.UserID)).
				Order(p.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list billing queue filter presets", zap.Error(err))
				return nil, err
			}
			if entities == nil {
				return []*billingqueuefilterpreset.BillingQueueFilterPreset{}, nil
			}

			return entities, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingqueuefilterpreset.BillingQueueFilterPreset,
) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
			p := billingqueuefilterpresetdb.BillingQueueFilterPresets
			if err := p.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to create billing queue filter preset", zap.Error(err))
				return nil, presetErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingqueuefilterpreset.BillingQueueFilterPreset,
) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
			p := billingqueuefilterpresetdb.BillingQueueFilterPresets
			rows, err := listquery.UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				p.Where(p.ID.Eq(entity.ID), p.UserID.Eq(entity.UserID)),
				p.Name.To(entity.Name),
				p.Filters.To(entity.Filters),
				p.IsDefault.To(entity.IsDefault),
			)
			if err != nil {
				r.l.Error("failed to update billing queue filter preset", zap.Error(err))
				return nil, presetErrors.Map(err, entity.ID.String())
			}
			if err = dberror.CheckAffected(
				int64(len(rows)),
				"BillingQueueFilterPreset",
				entity.ID.String(),
			); err != nil {
				return nil, err
			}

			return rows[0], nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req *billingqueuefilterpresetports.DeleteBillingQueueFilterPresetRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := billingqueuefilterpresetdb.BillingQueueFilterPresets
		n, err := p.Where(p.ID.Eq(req.PresetID), p.UserID.Eq(req.UserID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete billing queue filter preset", zap.Error(err))
			return presetErrors.Map(err, req.PresetID.String())
		}

		return dberror.CheckAffected(n, "BillingQueueFilterPreset", req.PresetID.String())
	})
}
