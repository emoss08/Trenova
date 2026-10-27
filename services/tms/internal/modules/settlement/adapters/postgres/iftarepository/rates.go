package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const taxRateEntity = "IFTA tax rate"

var taxRateErrors = dberror.Mapping{Entity: taxRateEntity}

func taxRateFilters(req *iftaports.ListTaxRatesRequest) []keel.Pred {
	tr := iftadb.TaxRates

	return []keel.Pred{
		when(req.Year > 0, tr.Year.Eq(int64(req.Year))),
		when(req.Quarter > 0, tr.Quarter.Eq(int64(req.Quarter))),
		when(!req.JurisdictionID.IsNil(), tr.JurisdictionID.Eq(req.JurisdictionID)),
		when(req.FuelType != "", tr.FuelType.Eq(req.FuelType)),
	}
}

func (r *repository) ListTaxRates(
	ctx context.Context,
	req *iftaports.ListTaxRatesRequest,
) (*pagination.CursorListResult[*ifta.TaxRate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ifta.TaxRate], error) {
			tr := iftadb.TaxRates
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				tr.Where(taxRateFilters(req)...),
				&listquery.Request[ifta.TaxRate]{
					Filter: withDefaultSort(
						req.Filter,
						desc("year"),
						desc("quarter"),
						asc("fuelType"),
					),
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*ifta.TaxRate]) *keel.Query[*ifta.TaxRate] {
						if req.IncludeJurisdiction {
							return q.Load(tr.Jurisdiction())
						}
						return q
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list ifta tax rates", zap.Error(err))
				return nil, fmt.Errorf("list ifta tax rates: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetTaxRateByID(ctx context.Context, id pulid.ID) (*ifta.TaxRate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ifta.TaxRate, error) {
		tr := iftadb.TaxRates
		entity, err := tr.Where(tr.ID.Eq(id)).Load(tr.Jurisdiction()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, taxRateErrors.Map(err, id.String())
		}

		return entity, nil
	})
}

func (r *repository) UpsertTaxRates(
	ctx context.Context,
	rates []*ifta.TaxRate,
) ([]*ifta.TaxRate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*ifta.TaxRate, error) {
		if len(rates) == 0 {
			return []*ifta.TaxRate{}, nil
		}

		db := r.db.Keel(ctx)
		tr := iftadb.TaxRates
		inserted, err := tr.Upsert(
			ctx,
			db,
			rates,
			keel.OnConflict(tr.JurisdictionID, tr.Year, tr.Quarter, tr.FuelType).
				Update(tr.RatePerGallon, tr.SurchargeRatePerGallon, tr.SourceNote, tr.SourceURL),
		)
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return nil, errortypes.NewValidationError(
					"jurisdictionId",
					errortypes.ErrInvalid,
					"One of the rates references a jurisdiction that does not exist",
				)
			}
			r.l.Error("failed to upsert ifta tax rates", zap.Error(err))
			return nil, fmt.Errorf("upsert ifta tax rates: %w", err)
		}

		if err = r.readBackUpdatedRates(ctx, rates, inserted); err != nil {
			return nil, err
		}

		return rates, nil
	})
}

type taxRatePeriodKey struct {
	jurisdictionID pulid.ID
	year           int
	quarter        int
	fuelType       string
}

func periodKeyOf(rate *ifta.TaxRate) taxRatePeriodKey {
	return taxRatePeriodKey{
		jurisdictionID: rate.JurisdictionID,
		year:           rate.Year,
		quarter:        rate.Quarter,
		fuelType:       string(rate.FuelType),
	}
}

func (r *repository) readBackUpdatedRates(
	ctx context.Context,
	rates []*ifta.TaxRate,
	inserted []bool,
) error {
	updated := make(map[taxRatePeriodKey]*ifta.TaxRate, len(rates))
	jurisdictions := make([]pulid.ID, 0, len(rates))
	years := make([]int64, 0, len(rates))
	for i, rate := range rates {
		if inserted[i] {
			continue
		}
		updated[periodKeyOf(rate)] = rate
		jurisdictions = append(jurisdictions, rate.JurisdictionID)
		years = append(years, int64(rate.Year))
	}
	if len(updated) == 0 {
		return nil
	}

	tr := iftadb.TaxRates
	stored, err := tr.Where(
		tr.JurisdictionID.In(jurisdictions...),
		tr.Year.In(years...),
	).All(ctx, r.db.Keel(ctx))
	if err != nil {
		return fmt.Errorf("read upserted ifta tax rates: %w", err)
	}

	for _, row := range stored {
		if rate, ok := updated[periodKeyOf(row)]; ok {
			*rate = *row
		}
	}

	return nil
}

func (r *repository) DeleteTaxRate(ctx context.Context, id pulid.ID, version int64) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tr := iftadb.TaxRates
		n, err := tr.Where(tr.ID.Eq(id), tr.Version.Eq(version)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete ifta tax rate", zap.Error(err))
			return fmt.Errorf("delete ifta tax rate: %w", err)
		}

		return dberror.CheckAffected(n, taxRateEntity, id.String())
	})
}

func (r *repository) ResolveRates(
	ctx context.Context,
	req *iftaports.ResolveRatesRequest,
) (map[ifta.RateKey]*ifta.TaxRate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[ifta.RateKey]*ifta.TaxRate, error) {
		tr := iftadb.TaxRates
		entities, err := tr.Where(
			tr.Year.Eq(int64(req.Year)),
			tr.Quarter.Eq(int64(req.Quarter)),
		).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to resolve ifta tax rates", zap.Error(err))
			return nil, fmt.Errorf("resolve ifta tax rates: %w", err)
		}

		out := make(map[ifta.RateKey]*ifta.TaxRate, len(entities))
		for _, entity := range entities {
			out[entity.Key()] = entity
		}

		return out, nil
	})
}
