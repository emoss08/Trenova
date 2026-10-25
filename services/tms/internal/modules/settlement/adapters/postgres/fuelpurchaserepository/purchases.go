package fuelpurchaserepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type purchaseIncludes struct {
	tractor      bool
	worker       bool
	jurisdiction bool
	fuelCard     bool
}

func purchaseFilters(req *fuelpurchaseports.ListFuelPurchasesRequest) []keel.Pred {
	fp := fuelpurchasedb.FuelPurchases
	var taxPaid keel.Pred
	if req.TaxPaid != nil {
		taxPaid = fp.TaxPaid.Eq(*req.TaxPaid)
	}

	return []keel.Pred{
		when(!req.TractorID.IsNil(), fp.TractorID.Eq(req.TractorID)),
		when(!req.WorkerID.IsNil(), fp.WorkerID.Eq(req.WorkerID)),
		when(!req.JurisdictionID.IsNil(), fp.JurisdictionID.Eq(req.JurisdictionID)),
		when(!req.FuelCardID.IsNil(), fp.FuelCardID.Eq(req.FuelCardID)),
		when(!req.ImportBatchID.IsNil(), fp.ImportBatchID.Eq(req.ImportBatchID)),
		when(len(req.FuelTypes) > 0, fp.FuelType.In(req.FuelTypes...)),
		when(len(req.Sources) > 0, fp.Source.In(req.Sources...)),
		taxPaid,
		when(req.From > 0, fp.PurchasedAt.Gte(req.From)),
		when(req.To > 0, fp.PurchasedAt.Lt(req.To)),
	}
}

func purchaseLoads(includes purchaseIncludes) []keel.Loader {
	fp := fuelpurchasedb.FuelPurchases
	loads := make([]keel.Loader, 0, 4)
	if includes.tractor {
		loads = append(loads, fp.Tractor())
	}
	if includes.worker {
		loads = append(loads, fp.Worker())
	}
	if includes.jurisdiction {
		loads = append(loads, fp.Jurisdiction())
	}
	if includes.fuelCard {
		loads = append(loads, fp.FuelCard())
	}

	return loads
}

func (r *repository) ListPurchases(
	ctx context.Context,
	req *fuelpurchaseports.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
			loads := purchaseLoads(purchaseIncludes{
				tractor:      req.IncludeTractor,
				worker:       req.IncludeWorker,
				jurisdiction: req.IncludeJurisdiction,
				fuelCard:     req.IncludeFuelCard,
			})

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fuelpurchasedb.FuelPurchases.Where(purchaseFilters(req)...),
				&listquery.Request[fuelpurchase.FuelPurchase]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*fuelpurchase.FuelPurchase],
					) *keel.Query[*fuelpurchase.FuelPurchase] {
						return q.Load(loads...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel purchases", zap.Error(err))
				return nil, fmt.Errorf("list fuel purchases: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetPurchaseByID(
	ctx context.Context,
	req *fuelpurchaseports.GetFuelPurchaseByIDRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelPurchase, error) {
		fp := fuelpurchasedb.FuelPurchases
		entity, err := fp.Where(fp.ID.Eq(req.ID)).
			Load(purchaseLoads(purchaseIncludes{
				tractor:      req.IncludeTractor,
				worker:       req.IncludeWorker,
				jurisdiction: req.IncludeJurisdiction,
				fuelCard:     req.IncludeFuelCard,
			})...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, purchaseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetPurchasesByIDs(
	ctx context.Context,
	req *fuelpurchaseports.GetFuelPurchasesByIDsRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelpurchase.FuelPurchase, error) {
		if len(req.IDs) == 0 {
			return []*fuelpurchase.FuelPurchase{}, nil
		}

		fp := fuelpurchasedb.FuelPurchases
		entities, err := fp.Where(fp.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get fuel purchases by ids", zap.Error(err))
			return nil, fmt.Errorf("get fuel purchases by ids: %w", err)
		}

		return entities, nil
	})
}

type referenceRow struct {
	ID        pulid.ID
	Reference string
}

func (r *repository) FindReferences(
	ctx context.Context,
	req *fuelpurchaseports.FindFuelPurchaseReferencesRequest,
) (map[string]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]pulid.ID, error) {
		refs := make([]string, 0, len(req.References))
		for _, ref := range req.References {
			normalized := strings.ToUpper(strings.TrimSpace(ref))
			if normalized != "" {
				refs = append(refs, normalized)
			}
		}
		found := make(map[string]pulid.ID, len(refs))
		if len(refs) == 0 {
			return found, nil
		}

		fp := fuelpurchasedb.FuelPurchases
		rows, err := keel.Into[referenceRow](
			ctx,
			r.db.Keel(ctx),
			fp.Pick(fp.ID, fp.TransactionReference).Where(fp.TransactionReference.In(refs...)),
		)
		if err != nil {
			r.l.Error("failed to find fuel purchase references", zap.Error(err))
			return nil, fmt.Errorf("find fuel purchase references: %w", err)
		}

		for _, row := range rows {
			found[row.Reference] = row.ID
		}

		return found, nil
	})
}

func (r *repository) CreatePurchase(
	ctx context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelPurchase, error) {
		if err := fuelpurchasedb.FuelPurchases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateReference()
			}
			r.l.Error("failed to create fuel purchase", zap.Error(err))
			return nil, purchaseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePurchase(
	ctx context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelPurchase, error) {
		if err := fuelpurchasedb.FuelPurchases.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateReference()
			}
			r.l.Error("failed to update fuel purchase", zap.Error(err))
			return nil, purchaseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeletePurchase(
	ctx context.Context,
	req *fuelpurchaseports.DeleteFuelPurchaseRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fuelpurchasedb.FuelPurchases.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fuelpurchase.FuelPurchase{ID: req.ID},
		)
		if err != nil && !dberror.IsNotFoundError(err) {
			r.l.Error("failed to delete fuel purchase", zap.Error(err))
		}

		return purchaseErrors.Map(err, req.ID.String())
	})
}

func (r *repository) AccumulateFuel(
	ctx context.Context,
	req *fuelpurchaseports.AccumulateFuelRequest,
) ([]*fuelpurchaseports.FuelAccumulationRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*fuelpurchaseports.FuelAccumulationRow, error) {
			fp := fuelpurchasedb.FuelPurchases
			rows, err := keel.Into[fuelpurchaseports.FuelAccumulationRow](
				ctx,
				r.db.Keel(ctx),
				fp.Pick(
					fp.TractorID,
					fp.JurisdictionID,
					fp.FuelType,
					fp.Gallons.Sum(),
					keel.DynOf(fp.Gallons.Sum().Filter(fp.TaxPaid.IsTrue())).Coalesce(0),
					keel.Count(),
				).
					Where(
						fp.PurchasedAt.Gte(req.Start),
						fp.PurchasedAt.Lt(req.End),
						fp.FuelType.In(domaintypes.IFTAFuelTypes()...),
					).
					Group(fp.TractorID, fp.JurisdictionID, fp.FuelType).
					Order(fp.TractorID.Asc(), fp.JurisdictionID.Asc(), fp.FuelType.Asc()),
			)
			if err != nil {
				r.l.Error("failed to accumulate fuel purchases", zap.Error(err))
				return nil, fmt.Errorf("accumulate fuel purchases: %w", err)
			}

			return pointers(rows), nil
		},
	)
}

func pointers[T any](rows []T) []*T {
	out := make([]*T, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}
