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
	"go.uber.org/zap"
)

const mileageEntryEntity = "IFTA mileage entry"

var mileageEntryErrors = dberror.Mapping{Entity: mileageEntryEntity}

func invalidMileageReference() error {
	return errortypes.NewValidationError(
		"tractorId",
		errortypes.ErrInvalid,
		"The tractor, jurisdiction or move referenced by this entry does not exist",
	)
}

func mileageEntryFilters(req *iftaports.ListMileageEntriesRequest) []keel.Pred {
	me := iftadb.JurisdictionMileageEntries

	return []keel.Pred{
		when(!req.TractorID.IsNil(), me.TractorID.Eq(req.TractorID)),
		when(!req.JurisdictionID.IsNil(), me.JurisdictionID.Eq(req.JurisdictionID)),
		when(req.Year > 0, me.Year.Eq(int64(req.Year))),
		when(req.Quarter > 0, me.Quarter.Eq(int64(req.Quarter))),
		when(len(req.Sources) > 0, me.Source.In(req.Sources...)),
	}
}

func mileageEntryLoads(includeTractor, includeJurisdiction bool) []keel.Loader {
	me := iftadb.JurisdictionMileageEntries
	loads := make([]keel.Loader, 0, 2)
	if includeTractor {
		loads = append(loads, me.Tractor())
	}
	if includeJurisdiction {
		loads = append(loads, me.Jurisdiction())
	}

	return loads
}

func (r *repository) ListMileageEntries(
	ctx context.Context,
	req *iftaports.ListMileageEntriesRequest,
) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
			loads := mileageEntryLoads(req.IncludeTractor, req.IncludeJurisdiction)
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				iftadb.JurisdictionMileageEntries.Where(mileageEntryFilters(req)...),
				&listquery.Request[ifta.JurisdictionMileageEntry]{
					Filter: withDefaultSort(req.Filter, desc("traveledAt")),
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[ifta.JurisdictionMileageEntry],
					) *keel.Query[ifta.JurisdictionMileageEntry] {
						return q.Load(loads...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list ifta mileage entries", zap.Error(err))
				return nil, fmt.Errorf("list ifta mileage entries: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetMileageEntryByID(
	ctx context.Context,
	req *iftaports.GetMileageEntryByIDRequest,
) (*ifta.JurisdictionMileageEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ifta.JurisdictionMileageEntry, error) {
		me := iftadb.JurisdictionMileageEntries
		entity, err := me.Where(me.ID.Eq(req.ID)).
			Load(mileageEntryLoads(req.IncludeTractor, req.IncludeJurisdiction)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, mileageEntryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ifta.JurisdictionMileageEntry, error) {
		err := iftadb.JurisdictionMileageEntries.Create(ctx, r.db.Keel(ctx), entity)
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return nil, invalidMileageReference()
			}
			r.l.Error("failed to create ifta mileage entry", zap.Error(err))
			return nil, mileageEntryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ifta.JurisdictionMileageEntry, error) {
		err := iftadb.JurisdictionMileageEntries.Save(ctx, r.db.Keel(ctx), entity)
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return nil, invalidMileageReference()
			}
			r.l.Error("failed to update ifta mileage entry", zap.Error(err))
			return nil, mileageEntryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteMileageEntry(
	ctx context.Context,
	req *iftaports.DeleteMileageEntryRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		me := iftadb.JurisdictionMileageEntries
		n, err := me.Where(me.ID.Eq(req.ID), me.Version.Eq(req.Version)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete ifta mileage entry", zap.Error(err))
			return fmt.Errorf("delete ifta mileage entry: %w", err)
		}

		return dberror.CheckAffected(n, mileageEntryEntity, req.ID.String())
	})
}
