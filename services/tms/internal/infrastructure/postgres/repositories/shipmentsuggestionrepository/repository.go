package shipmentsuggestionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxListedDecisions = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.ShipmentSuggestionDecisionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-suggestion-decision-repository"),
	}
}

func scoped(
	q *bun.SelectQuery,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) *bun.SelectQuery {
	cols := buncolgen.DecisionRecordColumns

	return q.
		Where(cols.OrganizationID.Eq(), tenant.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenant.BuID).
		Where(cols.UserID.Eq(), userID)
}

func (r *repository) ListSince(
	ctx context.Context,
	req *repositories.ListSuggestionDecisionsRequest,
) ([]*shipmentsuggestion.DecisionRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentsuggestion.DecisionRecord, error) {
			cols := buncolgen.DecisionRecordColumns
			records := make([]*shipmentsuggestion.DecisionRecord, 0)
			err := scoped(
				r.db.DBForContext(ctx).NewSelect().Model(&records),
				req.TenantInfo,
				req.UserID,
			).
				Where(cols.DecidedAt.Gte(), req.Since).
				Order(cols.DecidedAt.OrderDesc()).
				Limit(maxListedDecisions).
				Scan(ctx)
			if err != nil {
				return nil, fmt.Errorf("list suggestion decisions: %w", err)
			}

			return records, nil
		},
	)
}

func (r *repository) Upsert(
	ctx context.Context,
	record *shipmentsuggestion.DecisionRecord,
) (*shipmentsuggestion.DecisionRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentsuggestion.DecisionRecord, error) {
			cols := buncolgen.DecisionRecordColumns
			_, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(record).
				On(fmt.Sprintf(
					"CONFLICT (%s, %s, %s, %s) DO UPDATE",
					cols.OrganizationID.Bare(),
					cols.BusinessUnitID.Bare(),
					cols.UserID.Bare(),
					cols.SuggestionKey.Bare(),
				)).
				Set(cols.Decision.Bare() + " = EXCLUDED." + cols.Decision.Bare()).
				Set(cols.DecidedAt.Bare() + " = EXCLUDED." + cols.DecidedAt.Bare()).
				Set(cols.UpdatedAt.Bare() + " = EXCLUDED." + cols.UpdatedAt.Bare()).
				Returning("*").
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("save suggestion decision: %w", err)
			}

			return record, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteSuggestionDecisionRequest,
) (*shipmentsuggestion.DecisionRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentsuggestion.DecisionRecord, error) {
			cols := buncolgen.DecisionRecordColumns
			removed := make([]*shipmentsuggestion.DecisionRecord, 0, 1)
			_, err := r.db.DBForContext(ctx).
				NewDelete().
				Model(&removed).
				Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(cols.UserID.Eq(), req.UserID).
				Where(cols.SuggestionKey.Eq(), req.Key).
				Returning("*").
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("delete suggestion decision: %w", err)
			}
			if len(removed) == 0 {
				return nil, nil //nolint:nilnil // nothing was decided for this key
			}

			return removed[0], nil
		},
	)
}

func (r *repository) Prune(
	ctx context.Context,
	req *repositories.PruneSuggestionDecisionsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.DecisionRecordColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*shipmentsuggestion.DecisionRecord)(nil)).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(cols.UserID.Eq(), req.UserID).
			Where(cols.DecidedAt.Lt(), req.Before).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("prune suggestion decisions: %w", err)
		}

		return nil
	})
}
