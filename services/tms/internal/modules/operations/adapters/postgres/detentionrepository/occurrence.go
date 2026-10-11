package detentionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/detentioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type OccurrenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type occurrenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOccurrenceRepository(
	p OccurrenceParams,
) detentionports.DetentionOccurrenceRepository {
	return &occurrenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-occurrence-repository"),
	}
}

func withOccurrenceNames(q *bun.SelectQuery) *bun.SelectQuery {
	occCols := detentioncol.DetentionOccurrenceColumns
	locCols := locationcol.LocationColumns
	cusCols := customercol.CustomerColumns
	shipCols := shipmentcol.ShipmentColumns

	return q.
		ColumnExpr("dto.*").
		ColumnExpr("COALESCE(?, '') AS location_name", bun.Safe(locCols.Name.Qualified())).
		ColumnExpr("COALESCE(?, '') AS customer_name", bun.Safe(cusCols.Name.Qualified())).
		ColumnExpr(
			"COALESCE(?, '') AS shipment_pro_number",
			bun.Safe(shipCols.ProNumber.Qualified()),
		).
		Join("LEFT JOIN locations AS loc ON ?", bun.Safe(
			locCols.ID.EqColumn(occCols.LocationID)+
				" AND "+locCols.OrganizationID.EqColumn(occCols.OrganizationID),
		)).
		Join("LEFT JOIN customers AS cus ON ?", bun.Safe(
			cusCols.ID.EqColumn(occCols.CustomerID)+
				" AND "+cusCols.OrganizationID.EqColumn(occCols.OrganizationID),
		)).
		Join("LEFT JOIN shipments AS sp ON ?", bun.Safe(
			shipCols.ID.EqColumn(occCols.ShipmentID)+
				" AND "+shipCols.OrganizationID.EqColumn(occCols.OrganizationID),
		))
}

func (r *occurrenceRepository) filterQuery(
	q *bun.SelectQuery,
	req *detentionports.ListDetentionOccurrencesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		detentioncol.DetentionOccurrenceTable.Alias,
		req.Filter,
		(*detention.DetentionOccurrence)(nil),
	)

	cols := detentioncol.DetentionOccurrenceColumns

	if req.ShipmentID != nil && !req.ShipmentID.IsNil() {
		q = q.Where(cols.ShipmentID.Eq(), *req.ShipmentID)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	}

	if req.LocationID != nil && !req.LocationID.IsNil() {
		q = q.Where(cols.LocationID.Eq(), *req.LocationID)
	}

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if req.OpenOnly {
		q = q.Where(cols.IsOpen.IsTrue())
	}

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *occurrenceRepository) List(
	ctx context.Context,
	req *detentionports.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
			log := r.l.With(zap.String("operation", "List"))

			entities := make([]*detention.DetentionOccurrence, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count detention occurrences", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*detention.DetentionOccurrence]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *occurrenceRepository) GetByID(
	ctx context.Context,
	req *detentionports.GetDetentionOccurrenceByIDRequest,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.OccurrenceID.String()),
		)

		cols := detentioncol.DetentionOccurrenceColumns
		entity := new(detention.DetentionOccurrence)

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(withOccurrenceNames).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.OccurrenceID)
			})

		if req.IncludeEvidence {
			q = q.Relation(detentioncol.DetentionOccurrenceRelations.Evidence,
				func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(detentioncol.DetentionEvidenceColumns.Sequence.OrderAsc())
				})
		}

		if req.IncludeNotices {
			q = q.Relation(detentioncol.DetentionOccurrenceRelations.Notices,
				func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(detentioncol.DetentionNoticeColumns.CreatedAt.OrderAsc())
				})
		}

		if err := q.Scan(ctx); err != nil {
			log.Error("failed to get detention occurrence", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "DetentionOccurrence")
		}

		return entity, nil
	})
}

func (r *occurrenceRepository) GetByStop(
	ctx context.Context,
	req *detentionports.GetOccurrenceByStopRequest,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		cols := detentioncol.DetentionOccurrenceColumns
		entity := new(detention.DetentionOccurrence)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
					Where(cols.StopID.Eq(), req.StopID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to get detention occurrence by stop", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *occurrenceRepository) GetByShipment(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, 4)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.ShipmentID.Eq(), req.ShipmentID)
				})
			if req.IncludeEvidence {
				q = q.Relation("Evidence", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order("dte.sequence ASC")
				})
			}
			err := q.Order(cols.ClockStartAt.OrderAsc()).Scan(ctx)
			if err != nil {
				r.l.Error("failed to list detention occurrences for shipment", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, 32)

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, tenantInfo).
						Where(cols.IsOpen.IsTrue()).
						Where(cols.Status.Ne(), detention.OccurrenceStatusBilled)
				}).
				Order(cols.FreeTimeExpiresAt.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list open detention occurrences", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListBillingHoldsByShipments(
	ctx context.Context,
	req *detentionports.ListDetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			if req == nil || len(req.ShipmentIDs) == 0 {
				return []*detention.DetentionOccurrence{}, nil
			}

			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, len(req.ShipmentIDs))

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyBillingHold(
						detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo),
					).
						Where(cols.ShipmentID.In(), bun.List(req.ShipmentIDs))
				}).
				Order(cols.ShipmentID.OrderAsc(), cols.ClockStartAt.OrderAsc(), cols.ID.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list detention billing holds by shipment", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListBillingHolds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, 16)

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyBillingHold(
						detentioncol.DetentionOccurrenceScopeTenant(sq, tenantInfo),
					)
				}).
				Order(cols.ClockStartAt.OrderAsc(), cols.ID.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list detention billing holds", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func applyBillingHold(q *bun.SelectQuery) *bun.SelectQuery {
	cols := detentioncol.DetentionOccurrenceColumns

	return q.
		Where(cols.Status.Eq(), detention.OccurrenceStatusPending).
		Where(cols.RequiresApproval.IsTrue())
}

func (r *occurrenceRepository) ListByAdditionalCharges(
	ctx context.Context,
	req *detentionports.ListOccurrencesByChargesRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			if req == nil || len(req.ChargeIDs) == 0 || len(req.ShipmentIDs) == 0 {
				return []*detention.DetentionOccurrence{}, nil
			}

			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, len(req.ChargeIDs))

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.ShipmentID.In(), bun.List(req.ShipmentIDs)).
						Where(cols.AdditionalChargeID.In(), bun.List(req.ChargeIDs)).
						Where(cols.Status.In(), bun.List([]detention.OccurrenceStatus{
							detention.OccurrenceStatusPending,
							detention.OccurrenceStatusApproved,
							detention.OccurrenceStatusBilled,
						}))
				}).
				Order(cols.ShipmentID.OrderAsc(), cols.ClockStartAt.OrderAsc(), cols.ID.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list detention occurrences by charge", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// ListNoticesDue powers the pre-breach alerting sweep: occurrences whose notice
// window has opened but whose notice has not gone out yet.
func (r *occurrenceRepository) ListNoticesDue(
	ctx context.Context,
	req *detentionports.ListNoticesDueRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			entities := make([]*detention.DetentionOccurrence, 0, 32)

			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(withOccurrenceNames).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.NotificationStatus.Eq(), detention.NotificationStatusPending).
						Where(cols.NoticeDueAt.IsNotNull()).
						Where(cols.NoticeDueAt.Lte(), req.Before)
				}).
				Order(cols.NoticeDueAt.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list detention notices due", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// Upsert writes the occurrence keyed on its stop. Recalculation runs on every
// shipment save, so this is the hot path and must stay idempotent.
func (r *occurrenceRepository) Upsert(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		log := r.l.With(
			zap.String("operation", "Upsert"),
			zap.String("stopId", entity.StopID.String()),
		)

		existing, err := r.GetByStop(ctx, &detentionports.GetOccurrenceByStopRequest{
			StopID: entity.StopID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err != nil {
			return nil, err
		}

		if existing == nil {
			if _, iErr := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); iErr != nil {
				log.Error("failed to insert detention occurrence", zap.Error(iErr))
				return nil, iErr
			}
			return entity, nil
		}

		entity.ID = existing.ID
		entity.CreatedAt = existing.CreatedAt
		entity.Version = existing.Version

		return r.Update(ctx, entity)
	})
}

func (r *occurrenceRepository) Update(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update detention occurrence", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			results, "DetentionOccurrence", entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

// ShipmentAccrued totals what detention has already been booked against a
// shipment so a per-shipment cap can be applied across stops.
func (r *occurrenceRepository) ShipmentAccrued(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) (decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (decimal.Decimal, error) {
		cols := detentioncol.DetentionOccurrenceColumns

		var total decimal.NullDecimal
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*detention.DetentionOccurrence)(nil)).
			ColumnExpr("COALESCE(SUM(?), 0)", bun.Safe(cols.BillableAmount.Qualified())).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID)
			}).
			Scan(ctx, &total)
		if err != nil {
			r.l.Error("failed to total shipment detention accrual", zap.Error(err))
			return decimal.Zero, err
		}

		if !total.Valid {
			return decimal.Zero, nil
		}

		return total.Decimal, nil
	})
}

func (r *occurrenceRepository) LinkCharges(
	ctx context.Context,
	tx bun.IDB,
	req *detentionports.LinkOccurrenceChargesRequest,
) error {
	if req == nil || req.ShipmentID.IsNil() {
		return nil
	}

	cols := detentioncol.DetentionOccurrenceColumns
	linked := make([]pulid.ID, 0, len(req.ChargeOccurrences)*2)

	for chargeID, occurrenceIDs := range req.ChargeOccurrences {
		if chargeID.IsNil() || len(occurrenceIDs) == 0 {
			continue
		}

		linked = append(linked, occurrenceIDs...)

		if _, err := tx.NewUpdate().
			Model((*detention.DetentionOccurrence)(nil)).
			Set(cols.AdditionalChargeID.Set(), chargeID).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return detentioncol.DetentionOccurrenceScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID).
					Where(cols.ID.In(), bun.In(occurrenceIDs))
			}).
			Exec(ctx); err != nil {
			return fmt.Errorf("link detention occurrences to charge %s: %w", chargeID, err)
		}
	}

	stale := tx.NewUpdate().
		Model((*detention.DetentionOccurrence)(nil)).
		Set(cols.AdditionalChargeID.SetNull()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = detentioncol.DetentionOccurrenceScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID).
				Where(cols.AdditionalChargeID.IsNotNull())
			if len(linked) > 0 {
				uq = uq.Where(cols.ID.NotIn(), bun.In(linked))
			}
			return uq
		})

	if _, err := stale.Exec(ctx); err != nil {
		return fmt.Errorf("release stale detention charge links: %w", err)
	}

	return nil
}
