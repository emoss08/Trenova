package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	accumulationPrefix = `WITH mc AS (
	SELECT sm.id, sm.organization_id, sm.business_unit_id, sm.loaded,
		CASE WHEN sm.distance_units = $1 THEN sm.distance / 1.609344 ELSE sm.distance END AS distance,
		COALESCE(MAX(COALESCE(stp.actual_departure, stp.actual_arrival)), sm.updated_at) AS completed_at
	FROM shipment_moves AS sm
	LEFT JOIN stops AS stp
		ON stp.shipment_move_id = sm.id
		AND stp.organization_id = sm.organization_id
		AND stp.business_unit_id = sm.business_unit_id
	WHERE sm.organization_id = $2 AND sm.business_unit_id = $3 AND sm.status = $4
	GROUP BY sm.id, sm.organization_id, sm.business_unit_id, sm.loaded, sm.distance,
		sm.distance_units, sm.updated_at
), inp AS (
	SELECT mc.* FROM mc WHERE mc.completed_at >= $5 AND mc.completed_at < $6
), overridden AS (
	SELECT DISTINCT ifme.shipment_move_id
	FROM ifta_jurisdiction_mileage_entries AS ifme
	WHERE ifme.organization_id = $2 AND ifme.business_unit_id = $3
		AND ifme.shipment_move_id IS NOT NULL
), jm AS (
	SELECT smjm.shipment_move_id, smjm.organization_id, smjm.business_unit_id,
		smjm.country_code, smjm.jurisdiction_code, inp.loaded AS loaded,
		inp.distance AS move_distance,
		CASE WHEN smjm.distance_units = $7 THEN smjm.distance / 1.609344 ELSE smjm.distance END AS miles
	FROM shipment_move_jurisdiction_miles AS smjm
	JOIN inp ON inp.id = smjm.shipment_move_id
		AND inp.organization_id = smjm.organization_id
		AND inp.business_unit_id = smjm.business_unit_id
	WHERE smjm.organization_id = $2 AND smjm.business_unit_id = $3
		AND NOT EXISTS (SELECT 1 FROM overridden o WHERE o.shipment_move_id = smjm.shipment_move_id)
) `

	routeRowsSQL = accumulationPrefix + `SELECT a.tractor_id AS tractor_id,
	jm.country_code AS country_code, jm.jurisdiction_code AS jurisdiction_code,
	ifj.id AS jurisdiction_id,
	ROUND((COALESCE(SUM(jm.miles), 0))::numeric, 3) AS miles,
	ROUND((COALESCE(SUM(jm.miles) FILTER (WHERE jm.loaded), 0))::numeric, 3) AS loaded_miles,
	ROUND((COALESCE(SUM(jm.miles) FILTER (WHERE NOT jm.loaded), 0))::numeric, 3) AS empty_miles,
	COUNT(DISTINCT jm.shipment_move_id) AS move_count
FROM jm
LEFT JOIN assignments AS a
	ON a.shipment_move_id = jm.shipment_move_id
	AND a.organization_id = jm.organization_id
	AND a.business_unit_id = jm.business_unit_id
	AND a.archived_at IS NULL
LEFT JOIN ifta_jurisdictions AS ifj
	ON ifj.country_code = jm.country_code
	AND ifj.code = jm.jurisdiction_code
GROUP BY a.tractor_id, jm.country_code, jm.jurisdiction_code, ifj.id
ORDER BY a.tractor_id NULLS FIRST, jm.country_code, jm.jurisdiction_code`

	noTractorSQL = accumulationPrefix + `SELECT jm.shipment_move_id AS move_id,
	ROUND((COALESCE(SUM(jm.miles), 0))::numeric, 3) AS miles
FROM jm
LEFT JOIN assignments AS a
	ON a.shipment_move_id = jm.shipment_move_id
	AND a.organization_id = jm.organization_id
	AND a.business_unit_id = jm.business_unit_id
	AND a.archived_at IS NULL
WHERE a.tractor_id IS NULL
GROUP BY jm.shipment_move_id
ORDER BY jm.shipment_move_id`

	unattributedSQL = accumulationPrefix + `SELECT inp.id AS move_id,
	ROUND((inp.distance)::numeric, 3) AS miles
FROM inp
WHERE inp.distance > 0
	AND NOT EXISTS (SELECT 1 FROM shipment_move_jurisdiction_miles AS smjm
		WHERE smjm.shipment_move_id = inp.id
		AND smjm.organization_id = inp.organization_id
		AND smjm.business_unit_id = inp.business_unit_id)
	AND NOT EXISTS (SELECT 1 FROM overridden o WHERE o.shipment_move_id = inp.id)
ORDER BY inp.completed_at, inp.id`

	mismatchSQL = accumulationPrefix + `SELECT jm.shipment_move_id AS move_id,
	ROUND((jm.move_distance)::numeric, 3) AS move_distance,
	ROUND((SUM(jm.miles))::numeric, 3) AS attributed_miles
FROM jm
WHERE jm.move_distance IS NOT NULL
GROUP BY jm.shipment_move_id, jm.move_distance
HAVING ABS(SUM(jm.miles) - jm.move_distance) > GREATEST(1, jm.move_distance * 0.02)
ORDER BY jm.shipment_move_id`
)

type moveMiles struct {
	MoveID pulid.ID
	Miles  decimal.Decimal
}

func accumulationArgs(req *iftaports.AccumulateMilesRequest) []any {
	kilometers := string(shipment.JurisdictionDistanceUnitsKilometers)

	return []any{
		kilometers,
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		string(shipment.MoveStatusCompleted),
		req.Start,
		req.End,
		kilometers,
	}
}

func (r *repository) AccumulateMiles(
	ctx context.Context,
	req *iftaports.AccumulateMilesRequest,
) (*iftaports.MileAccumulation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iftaports.MileAccumulation, error) {
		db := r.db.Keel(ctx)
		args := accumulationArgs(req)

		routeRows, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToAddrOfStructByPos[iftaports.MileRow],
			routeRowsSQL,
			args...,
		)
		if err != nil {
			r.l.Error("failed to accumulate route miles", zap.Error(err))
			return nil, fmt.Errorf("accumulate route miles: %w", err)
		}

		noTractor, err := keel.QueryAll(ctx, db, pgx.RowToStructByPos[moveMiles], noTractorSQL, args...)
		if err != nil {
			r.l.Error("failed to accumulate no-tractor miles", zap.Error(err))
			return nil, fmt.Errorf("accumulate no-tractor miles: %w", err)
		}

		unattributed, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToStructByPos[moveMiles],
			unattributedSQL,
			args...,
		)
		if err != nil {
			r.l.Error("failed to list unattributed moves", zap.Error(err))
			return nil, fmt.Errorf("list unattributed moves: %w", err)
		}

		mismatches, err := keel.QueryAll(
			ctx,
			db,
			pgx.RowToStructByPos[iftaports.MoveMismatch],
			mismatchSQL,
			args...,
		)
		if err != nil {
			r.l.Error("failed to detect jurisdiction mileage mismatches", zap.Error(err))
			return nil, fmt.Errorf("detect mileage mismatches: %w", err)
		}

		manual, err := r.manualRows(ctx, req)
		if err != nil {
			return nil, err
		}

		out := &iftaports.MileAccumulation{
			RouteRows:  nonNil(routeRows),
			ManualRows: manual,
			Mismatches: nonNil(mismatches),
		}
		for i := range noTractor {
			out.NoTractor.Add(noTractor[i].MoveID, noTractor[i].Miles)
		}
		for i := range unattributed {
			out.Unattributed.Add(unattributed[i].MoveID, unattributed[i].Miles)
		}

		return out, nil
	})
}

func (r *repository) manualRows(
	ctx context.Context,
	req *iftaports.AccumulateMilesRequest,
) ([]*iftaports.MileRow, error) {
	me := iftadb.JurisdictionMileageEntries
	j := iftadb.Jurisdictions
	rows, err := keel.Into[iftaports.MileRow](
		ctx,
		r.db.Keel(ctx),
		me.Join(j, j.ID.EqCol(me.JurisdictionID)).
			Pick(
				me.TractorID,
				j.CountryCode,
				j.Code,
				me.JurisdictionID,
				keel.Raw("coalesce(?, 0)", me.Miles.Sum()),
				keel.Raw("coalesce(?, 0)", me.Miles.Sum().Filter(me.Loaded.IsTrue())),
				keel.Raw("coalesce(?, 0)", me.Miles.Sum().Filter(me.Loaded.IsFalse())),
				keel.Count(),
			).
			Where(
				me.Year.Eq(int64(req.Year)),
				me.Quarter.Eq(int64(req.Quarter)),
			).
			Group(me.TractorID, j.CountryCode, j.Code, me.JurisdictionID).
			Order(me.TractorID.Asc(), j.CountryCode.Asc(), j.Code.Asc()),
	)
	if err != nil {
		r.l.Error("failed to accumulate manual miles", zap.Error(err))
		return nil, fmt.Errorf("accumulate manual miles: %w", err)
	}

	out := make([]*iftaports.MileRow, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out, nil
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}

	return items
}
