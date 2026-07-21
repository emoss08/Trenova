package recurringshipmentrepository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	// missedGraceSeconds is how far past its scheduled pickup time an
	// occurrence may still be materialized before it is recorded as missed.
	missedGraceSeconds = int64(6 * 60 * 60)

	maxConsecutiveGenerationFailures = 5

	maxShipmentBOLLength = 100
)

func (r *repository) applyDerivedFields(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) error {
	source, err := shipmentrepository.LoadShipmentGraphSource(
		ctx,
		r.db.DBForContext(ctx),
		pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		entity.SourceShipmentID,
	)
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"sourceShipmentId",
			errortypes.ErrInvalid,
			"Source shipment could not be found in your organization",
		)
		return multiErr
	}

	if firstStop := shipment.FirstShipperStop(source.Moves); firstStop == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"sourceShipmentId",
			errortypes.ErrInvalid,
			"Source shipment must have at least one pickup stop with a scheduled window",
		)
		return multiErr
	}

	entity.CustomerID = source.CustomerID
	entity.OriginLocationID = originLocationID(source.Moves)
	entity.DestinationLocationID = destinationLocationID(source.Moves)

	if entity.Status == recurringshipment.StatusActive {
		next, occErr := entity.NextOccurrence(timeutils.NowUnix())
		if occErr != nil {
			multiErr := errortypes.NewMultiError()
			multiErr.Add(
				"cronExpression",
				errortypes.ErrInvalid,
				"Schedule produces no valid occurrences",
			)
			return multiErr
		}

		if next == nil {
			multiErr := errortypes.NewMultiError()
			multiErr.Add(
				"endDate",
				errortypes.ErrInvalid,
				"Schedule has no future occurrences before its end date",
			)
			return multiErr
		}

		entity.NextOccurrenceAt = &next.At
		entity.NextOccurrenceSourceAt = &next.OriginalAt
	} else {
		entity.NextOccurrenceAt = nil
		entity.NextOccurrenceSourceAt = nil
	}

	return nil
}

func originLocationID(moves []*shipment.ShipmentMove) pulid.ID {
	if firstStop := shipment.FirstShipperStop(moves); firstStop != nil {
		return firstStop.LocationID
	}

	return pulid.Nil
}

func destinationLocationID(moves []*shipment.ShipmentMove) pulid.ID {
	var best *shipment.Stop
	var bestMoveSeq, bestStopSeq int64

	for _, move := range moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || !stop.IsDestinationStop() {
				continue
			}
			if best == nil ||
				move.Sequence > bestMoveSeq ||
				(move.Sequence == bestMoveSeq && stop.Sequence >= bestStopSeq) {
				best = stop
				bestMoveSeq = move.Sequence
				bestStopSeq = stop.Sequence
			}
		}
	}

	if best == nil {
		return pulid.Nil
	}

	return best.LocationID
}

func (r *repository) Generate(
	ctx context.Context,
	req *repositories.GenerateRecurringShipmentRequest,
) (*repositories.GenerateRecurringShipmentResult, error) {
	log := r.l.With(
		zap.String("operation", "Generate"),
		zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
		zap.String("trigger", string(req.Trigger)),
	)

	result := new(repositories.GenerateRecurringShipmentResult)
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		series, err := r.lockSeries(c, tx, req.TenantInfo, req.RecurringShipmentID)
		if err != nil {
			return err
		}
		result.Series = series

		if validationErr := validateGenerationEligibility(series, req.Trigger); validationErr != nil {
			return validationErr
		}

		occurrence, occErr := resolveOccurrence(series, req)
		if occErr != nil {
			return occErr
		}

		now := timeutils.NowUnix()

		if req.Trigger == recurringshipment.RunTriggerAuto &&
			occurrence.At < now-missedGraceSeconds {
			return r.recordMissedOccurrence(c, tx, series, occurrence, result)
		}

		alreadyGenerated, dupErr := r.occurrenceAlreadyGenerated(c, tx, series, occurrence.At)
		if dupErr != nil {
			return dupErr
		}

		if alreadyGenerated && req.Trigger == recurringshipment.RunTriggerAuto {
			return r.advanceSeriesOnly(c, tx, series, occurrence)
		}

		generated, genErr := r.materializeOccurrence(c, tx, series, occurrence, req)
		if genErr != nil {
			return genErr
		}

		run := &recurringshipment.RecurringShipmentRun{
			BusinessUnitID:      series.BusinessUnitID,
			OrganizationID:      series.OrganizationID,
			RecurringShipmentID: series.ID,
			GeneratedShipmentID: generated.ID,
			TriggeredByID:       req.RequestedBy,
			Status:              recurringshipment.RunStatusGenerated,
			Trigger:             req.Trigger,
			OccurrenceAt:        occurrence.At,
		}
		if occurrence.Shifted {
			run.OriginalOccurrenceAt = &occurrence.OriginalAt
		}

		if _, insertErr := tx.NewInsert().Model(run).Returning("*").Exec(c); insertErr != nil {
			return insertErr
		}

		series.GenerationCount++
		series.LastOccurrenceAt = &occurrence.At
		series.LastRunAt = &now
		series.LastGeneratedShipmentID = generated.ID
		series.ConsecutiveFailures = 0

		if shouldAdvancePointer(series, req, occurrence) {
			if advanceErr := advanceSeries(series, occurrence); advanceErr != nil {
				return advanceErr
			}
		}

		if updateErr := r.persistSeries(c, tx, series); updateErr != nil {
			return updateErr
		}

		result.Run = run
		result.Shipment = generated

		return nil
	})
	if err != nil {
		log.Error("failed to generate recurring shipment occurrence", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Recurring shipment is busy. Retry the request.",
		)
	}

	return result, nil
}

func (r *repository) RecordGenerationFailure(
	ctx context.Context,
	req *repositories.RecordRecurringGenerationFailureRequest,
) (*recurringshipment.RecurringShipment, error) {
	log := r.l.With(
		zap.String("operation", "RecordGenerationFailure"),
		zap.String("recurringShipmentId", req.RecurringShipmentID.String()),
	)

	var series *recurringshipment.RecurringShipment
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		locked, err := r.lockSeries(c, tx, req.TenantInfo, req.RecurringShipmentID)
		if err != nil {
			return err
		}
		series = locked

		run := &recurringshipment.RecurringShipmentRun{
			BusinessUnitID:      series.BusinessUnitID,
			OrganizationID:      series.OrganizationID,
			RecurringShipmentID: series.ID,
			Status:              recurringshipment.RunStatusFailed,
			Trigger:             recurringshipment.RunTriggerAuto,
			OccurrenceAt:        req.OccurrenceAt,
			Detail:              req.Detail,
		}
		if _, insertErr := tx.NewInsert().Model(run).Returning("*").Exec(c); insertErr != nil {
			return insertErr
		}

		now := timeutils.NowUnix()
		series.ConsecutiveFailures++
		series.LastRunAt = &now

		// Advance first so a persistently failing series can never spin on
		// every dispatch tick.
		occurrence := &recurringshipment.Occurrence{
			At:         req.OccurrenceAt,
			OriginalAt: seriesSourceSlot(series, req.OccurrenceAt),
		}
		if advanceErr := advanceSeries(series, occurrence); advanceErr != nil {
			return advanceErr
		}

		if series.ConsecutiveFailures >= maxConsecutiveGenerationFailures &&
			series.Status == recurringshipment.StatusActive {
			series.Status = recurringshipment.StatusPaused
		}

		return r.persistSeries(c, tx, series)
	})
	if err != nil {
		log.Error("failed to record recurring generation failure", zap.Error(err))
		return nil, err
	}

	return series, nil
}

func (r *repository) lockSeries(
	ctx context.Context,
	tx bun.Tx,
	tenantInfo pagination.TenantInfo,
	seriesID pulid.ID,
) (*recurringshipment.RecurringShipment, error) {
	rsh := buncolgen.RecurringShipmentColumns
	series := new(recurringshipment.RecurringShipment)
	err := tx.NewSelect().
		Model(series).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RecurringShipmentScopeTenant(sq, tenantInfo).
				Where(rsh.ID.Eq(), seriesID)
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "RecurringShipment")
	}

	return series, nil
}

func validateGenerationEligibility(
	series *recurringshipment.RecurringShipment,
	trigger recurringshipment.RunTrigger,
) error {
	multiErr := errortypes.NewMultiError()

	switch trigger {
	case recurringshipment.RunTriggerAuto:
		if series.Status != recurringshipment.StatusActive || !series.AutoGenerate {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"Series is not eligible for automatic generation",
			)
		}
	case recurringshipment.RunTriggerManual:
		if series.Status == recurringshipment.StatusExpired {
			multiErr.Add(
				"status",
				errortypes.ErrInvalid,
				"An expired series can no longer generate shipments",
			)
		}
	}

	if series.ReachedOccurrenceLimit() {
		multiErr.Add(
			"maxOccurrences",
			errortypes.ErrInvalid,
			"Series has reached its maximum number of occurrences",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func resolveOccurrence(
	series *recurringshipment.RecurringShipment,
	req *repositories.GenerateRecurringShipmentRequest,
) (*recurringshipment.Occurrence, error) {
	if req.OccurrenceAt != nil {
		return &recurringshipment.Occurrence{
			At:         *req.OccurrenceAt,
			OriginalAt: *req.OccurrenceAt,
		}, nil
	}

	if series.NextOccurrenceAt != nil {
		return &recurringshipment.Occurrence{
			At:         *series.NextOccurrenceAt,
			OriginalAt: seriesSourceSlot(series, *series.NextOccurrenceAt),
			Shifted: series.NextOccurrenceSourceAt != nil &&
				*series.NextOccurrenceSourceAt != *series.NextOccurrenceAt,
		}, nil
	}

	next, err := series.NextOccurrence(timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	if next == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"occurrenceAt",
			errortypes.ErrInvalid,
			"Series has no upcoming occurrence to generate",
		)
		return nil, multiErr
	}

	return next, nil
}

func seriesSourceSlot(series *recurringshipment.RecurringShipment, fallback int64) int64 {
	if series.NextOccurrenceSourceAt != nil {
		return *series.NextOccurrenceSourceAt
	}

	return fallback
}

func shouldAdvancePointer(
	series *recurringshipment.RecurringShipment,
	req *repositories.GenerateRecurringShipmentRequest,
	occurrence *recurringshipment.Occurrence,
) bool {
	if req.Trigger == recurringshipment.RunTriggerAuto || req.OccurrenceAt == nil {
		return true
	}

	return series.NextOccurrenceAt != nil && occurrence.At == *series.NextOccurrenceAt
}

func advanceSeries(
	series *recurringshipment.RecurringShipment,
	occurrence *recurringshipment.Occurrence,
) error {
	if series.ReachedOccurrenceLimit() {
		series.Status = recurringshipment.StatusExpired
		series.NextOccurrenceAt = nil
		series.NextOccurrenceSourceAt = nil
		return nil
	}

	base := max(occurrence.OriginalAt, occurrence.At)

	next, err := series.NextOccurrence(base)
	if err != nil {
		return err
	}

	if next == nil {
		series.Status = recurringshipment.StatusExpired
		series.NextOccurrenceAt = nil
		series.NextOccurrenceSourceAt = nil
		return nil
	}

	series.NextOccurrenceAt = &next.At
	series.NextOccurrenceSourceAt = &next.OriginalAt

	return nil
}

func (r *repository) recordMissedOccurrence(
	ctx context.Context,
	tx bun.Tx,
	series *recurringshipment.RecurringShipment,
	occurrence *recurringshipment.Occurrence,
	result *repositories.GenerateRecurringShipmentResult,
) error {
	run := &recurringshipment.RecurringShipmentRun{
		BusinessUnitID:      series.BusinessUnitID,
		OrganizationID:      series.OrganizationID,
		RecurringShipmentID: series.ID,
		Status:              recurringshipment.RunStatusSkipped,
		Trigger:             recurringshipment.RunTriggerAuto,
		OccurrenceAt:        occurrence.At,
		Detail:              "Occurrence was missed and skipped because its pickup window had already passed",
	}
	if occurrence.Shifted {
		run.OriginalOccurrenceAt = &occurrence.OriginalAt
	}

	if _, err := tx.NewInsert().Model(run).Returning("*").Exec(ctx); err != nil {
		return err
	}

	now := timeutils.NowUnix()
	series.LastRunAt = &now

	if err := advanceSeries(series, occurrence); err != nil {
		return err
	}

	if err := r.persistSeries(ctx, tx, series); err != nil {
		return err
	}

	result.Run = run

	return nil
}

func (r *repository) advanceSeriesOnly(
	ctx context.Context,
	tx bun.Tx,
	series *recurringshipment.RecurringShipment,
	occurrence *recurringshipment.Occurrence,
) error {
	now := timeutils.NowUnix()
	series.LastRunAt = &now

	if err := advanceSeries(series, occurrence); err != nil {
		return err
	}

	return r.persistSeries(ctx, tx, series)
}

func (r *repository) occurrenceAlreadyGenerated(
	ctx context.Context,
	tx bun.Tx,
	series *recurringshipment.RecurringShipment,
	occurrenceAt int64,
) (bool, error) {
	rsr := buncolgen.RecurringShipmentRunColumns

	return tx.NewSelect().
		Model((*recurringshipment.RecurringShipmentRun)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RecurringShipmentRunScopeTenant(sq, pagination.TenantInfo{
				OrgID: series.OrganizationID,
				BuID:  series.BusinessUnitID,
			}).
				Where(rsr.RecurringShipmentID.Eq(), series.ID).
				Where(rsr.OccurrenceAt.Eq(), occurrenceAt).
				Where(rsr.Status.Eq(), recurringshipment.RunStatusGenerated)
		}).
		Exists(ctx)
}

func (r *repository) materializeOccurrence(
	ctx context.Context,
	tx bun.Tx,
	series *recurringshipment.RecurringShipment,
	occurrence *recurringshipment.Occurrence,
	req *repositories.GenerateRecurringShipmentRequest,
) (*shipment.Shipment, error) {
	source, err := shipmentrepository.LoadShipmentGraphSource(
		ctx,
		tx,
		pagination.TenantInfo{
			OrgID: series.OrganizationID,
			BuID:  series.BusinessUnitID,
		},
		series.SourceShipmentID,
	)
	if err != nil {
		return nil, err
	}

	locationCode, businessUnitCode, err := shipmentrepository.ResolveSequenceCodes(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return nil, err
	}

	proNumbers, err := r.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
		Type:             tenant.SequenceTypeProNumber,
		OrgID:            series.OrganizationID,
		BuID:             series.BusinessUnitID,
		Count:            1,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
	if err != nil {
		return nil, err
	}

	orderNumbers, err := r.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
		Type:             tenant.SequenceTypeOrder,
		OrgID:            series.OrganizationID,
		BuID:             series.BusinessUnitID,
		Count:            1,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
	if err != nil {
		return nil, err
	}

	requestedBy := req.RequestedBy
	if requestedBy.IsNil() {
		requestedBy = series.EnteredByID
	}

	generated := shipmentrepository.CopyShipmentGraph(source, shipmentrepository.ShipmentCopySpec{
		ProNumber:   proNumbers[0],
		BOL:         deriveRecurringBOL(source.BOL, occurrence.At, series.Timezone),
		RequestedBy: requestedBy,
		DateAnchor:  &occurrence.At,
	})

	autoOrder := shipmentrepository.BuildAutoOrder(generated, orderNumbers[0])
	generated.OrderID = autoOrder.ID

	if _, err = tx.NewInsert().Model(autoOrder).Returning("NULL").Exec(ctx); err != nil {
		return nil, err
	}

	if _, err = tx.NewInsert().Model(generated).Returning("NULL").Exec(ctx); err != nil {
		return nil, err
	}

	if len(generated.Moves) > 0 {
		if _, err = tx.NewInsert().Model(&generated.Moves).Returning("NULL").Exec(ctx); err != nil {
			return nil, err
		}

		stops := make([]*shipment.Stop, 0, len(generated.Moves))
		for _, move := range generated.Moves {
			stops = append(stops, move.Stops...)
		}

		if len(stops) > 0 {
			if _, err = tx.NewInsert().Model(&stops).Returning("NULL").Exec(ctx); err != nil {
				return nil, err
			}
		}
	}

	if len(generated.AdditionalCharges) > 0 {
		if _, err = tx.NewInsert().
			Model(&generated.AdditionalCharges).
			Returning("NULL").
			Exec(ctx); err != nil {
			return nil, err
		}
	}

	if len(generated.Commodities) > 0 {
		if _, err = tx.NewInsert().
			Model(&generated.Commodities).
			Returning("NULL").
			Exec(ctx); err != nil {
			return nil, err
		}
	}

	return generated, nil
}

func (r *repository) persistSeries(
	ctx context.Context,
	tx bun.Tx,
	series *recurringshipment.RecurringShipment,
) error {
	series.Version++

	results, err := tx.NewUpdate().
		Model(series).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, "RecurringShipment", series.ID.String())
}

func deriveRecurringBOL(sourceBOL string, occurrenceAt int64, timezone string) string {
	base := strings.TrimSpace(sourceBOL)
	if base == "" {
		return ""
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	suffix := fmt.Sprintf("-%s", time.Unix(occurrenceAt, 0).In(loc).Format("20060102"))
	baseRunes := []rune(base)
	maxBaseLength := max(maxShipmentBOLLength-len(suffix), 1)
	if len(baseRunes) > maxBaseLength {
		baseRunes = baseRunes[:maxBaseLength]
	}

	return string(baseRunes) + suffix
}
