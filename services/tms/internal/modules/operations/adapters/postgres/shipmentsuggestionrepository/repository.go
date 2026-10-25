package shipmentsuggestionrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentsuggestion/shipmentsuggestiondb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentsuggestionports"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) shipmentsuggestionports.ShipmentSuggestionDecisionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-suggestion-decision-repository"),
	}
}

func (r *repository) ListSince(
	ctx context.Context,
	req *shipmentsuggestionports.ListSuggestionDecisionsRequest,
) ([]*shipmentsuggestion.DecisionRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentsuggestion.DecisionRecord, error) {
			d := shipmentsuggestiondb.DecisionRecords
			records, err := d.Where(d.UserID.Eq(req.UserID), d.DecidedAt.Gte(req.Since)).
				Order(d.DecidedAt.Desc()).
				Limit(maxListedDecisions).
				List(ctx, r.db.Keel(ctx))
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
			d := shipmentsuggestiondb.DecisionRecords
			inserted, err := d.Upsert(
				ctx,
				r.db.Keel(ctx),
				[]*shipmentsuggestion.DecisionRecord{record},
				keel.OnConflict(d.OrganizationID, d.BusinessUnitID, d.UserID, d.SuggestionKey).
					Update(d.Decision, d.DecidedAt),
			)
			if err != nil {
				return nil, fmt.Errorf("save suggestion decision: %w", err)
			}
			if len(inserted) == 1 && inserted[0] {
				return record, nil
			}

			stored, err := byKey(record.UserID, record.SuggestionKey).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("read saved suggestion decision: %w", err)
			}

			return stored, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req *shipmentsuggestionports.DeleteSuggestionDecisionRequest,
) (*shipmentsuggestion.DecisionRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentsuggestion.DecisionRecord, error) {
			d := shipmentsuggestiondb.DecisionRecords
			removed, err := byKey(req.UserID, req.Key).
				Lock(keel.ForUpdate).
				One(ctx, r.db.Keel(ctx))
			if errors.Is(err, keel.ErrNotFound) {
				return nil, nil
			}
			if err != nil {
				return nil, fmt.Errorf("delete suggestion decision: %w", err)
			}

			if err = d.DeleteRow(ctx, r.db.Keel(ctx), removed); err != nil {
				return nil, fmt.Errorf("delete suggestion decision: %w", err)
			}

			return removed, nil
		},
	)
}

func (r *repository) Prune(
	ctx context.Context,
	req *shipmentsuggestionports.PruneSuggestionDecisionsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := shipmentsuggestiondb.DecisionRecords
		if _, err := d.Where(d.UserID.Eq(req.UserID), d.DecidedAt.Lt(req.Before)).
			Delete(ctx, r.db.Keel(ctx)); err != nil {
			return fmt.Errorf("prune suggestion decisions: %w", err)
		}

		return nil
	})
}

func byKey(userID pulid.ID, key string) *keel.Query[*shipmentsuggestion.DecisionRecord] {
	d := shipmentsuggestiondb.DecisionRecords
	return d.Where(d.UserID.Eq(userID), d.SuggestionKey.Eq(key))
}
