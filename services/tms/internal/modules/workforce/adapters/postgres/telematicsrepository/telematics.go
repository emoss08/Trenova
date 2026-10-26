package telematicsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	hosStateErrors  = dberror.Mapping{Entity: "WorkerHOSState"}
	feedStateErrors = dberror.Mapping{Entity: "FeedState"}
	webhookErrors   = dberror.Mapping{Entity: "TelematicsWebhook"}
	eventErrors     = dberror.Mapping{Entity: "Telematics event"}
)

type webhookRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	WebhookSecret  string
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) telematicsports.TelematicsRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.telematics-repository"),
	}
}

func positionIsNewer() keel.Pred {
	vp := telematicsdb.VehiclePositions

	return keel.Int64Of(vp.RecordedAt).LteExpr(keel.Int64Of(vp.RecordedAt.Excluded()))
}

func newerText(col keel.StringCol) keel.StringExpr {
	return keel.CaseString().When(positionIsNewer(), keel.StringOf(col.Excluded())).Else(col)
}

func newerFloat(col keel.Float64Col) keel.Float64Expr {
	return keel.CaseFloat64().When(positionIsNewer(), keel.Float64Of(col.Excluded())).Else(col)
}

func newerInt64(col keel.Int64Col) keel.Int64Expr {
	return keel.CaseInt64().When(positionIsNewer(), keel.Int64Of(col.Excluded())).Else(col)
}

func positionConflict() keel.Conflict {
	vp := telematicsdb.VehiclePositions

	return keel.OnConflict(vp.OrganizationID, vp.BusinessUnitID, vp.TractorID).Set(
		vp.Provider.ToExpr(newerText(vp.Provider)),
		vp.ProviderVehicleID.ToExpr(newerText(vp.ProviderVehicleID)),
		vp.Latitude.ToExpr(newerFloat(vp.Latitude)),
		vp.Longitude.ToExpr(newerFloat(vp.Longitude)),
		vp.HeadingDegrees.ToExpr(newerFloat(vp.HeadingDegrees)),
		vp.SpeedMph.ToExpr(newerFloat(vp.SpeedMph)),
		vp.EngineState.ToExpr(newerText(vp.EngineState.StringCol)),
		vp.FuelPercent.ToExpr(newerFloat(vp.FuelPercent)),
		vp.OdometerMeters.ToExpr(newerInt64(vp.OdometerMeters)),
		vp.FormattedLocation.ToExpr(newerText(vp.FormattedLocation)),
		vp.ReceivedAt.ToExpr(newerInt64(vp.ReceivedAt)),
		vp.RecordedAt.ToExpr(vp.RecordedAt.Greatest(keel.Int64Of(vp.RecordedAt.Excluded()))),
	)
}

func (r *repository) UpsertVehiclePositions(
	ctx context.Context,
	positions []*telematics.VehiclePosition,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(positions) == 0 {
			return nil
		}

		_, err := telematicsdb.VehiclePositions.Upsert(
			ctx,
			r.db.Keel(ctx),
			positions,
			positionConflict(),
		)
		if err != nil {
			return fmt.Errorf("upsert vehicle positions: %w", err)
		}

		return nil
	})
}

func vehiclePositionsQuery(
	req *telematicsports.ListVehiclePositionsRequest,
) *keel.Query[*telematics.VehiclePosition] {
	vp := telematicsdb.VehiclePositions
	tr := vp.Tractor()

	var loads []keel.Loader
	switch {
	case req.IncludeWorker:
		loads = append(loads, tr.PrimaryWorker())
	case req.IncludeTractor:
		loads = append(loads, tr)
	}

	return vp.Where(
		keel.When(len(req.TractorIDs) > 0, vp.TractorID.In(req.TractorIDs...)),
		keel.When(
			req.MaxAgeSeconds > 0,
			vp.RecordedAt.Gte(timeutils.NowUnix()-req.MaxAgeSeconds),
		),
	).
		Load(loads...).
		Order(vp.RecordedAt.Desc())
}

func (r *repository) ListVehiclePositions(
	ctx context.Context,
	req *telematicsports.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.VehiclePosition, error) {
		entities, err := vehiclePositionsQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list vehicle positions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) UpsertWorkerHOSStates(
	ctx context.Context,
	states []*telematics.WorkerHOSState,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(states) == 0 {
			return nil
		}

		s := telematicsdb.WorkerHOSStates
		_, err := s.Upsert(ctx, r.db.Keel(ctx), states,
			keel.OnConflict(s.OrganizationID, s.BusinessUnitID, s.WorkerID).Update(
				s.Provider,
				s.ProviderDriverID,
				s.DutyStatus,
				s.DriveRemainingMs,
				s.ShiftRemainingMs,
				s.CycleRemainingMs,
				s.CycleTomorrowMs,
				s.BreakRemainingMs,
				s.CycleStartedAt,
				s.ShiftDrivingViolationMs,
				s.CycleViolationMs,
				s.CurrentVehicleID,
				s.CurrentTractorID,
				s.RecordedAt,
				s.ReceivedAt,
			))
		if err != nil {
			return fmt.Errorf("upsert worker hos states: %w", err)
		}

		return nil
	})
}

func (r *repository) ListWorkerHOSStates(
	ctx context.Context,
	req *telematicsports.ListWorkerHOSStatesRequest,
) ([]*telematics.WorkerHOSState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.WorkerHOSState, error) {
		s := telematicsdb.WorkerHOSStates
		q := s.Where(keel.When(len(req.WorkerIDs) > 0, s.WorkerID.In(req.WorkerIDs...))).
			Load(listquery.When(req.IncludeWorker, s.Worker())...).
			Order(s.DriveRemainingMs.Asc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list worker hos states: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetWorkerHOSState(
	ctx context.Context,
	req telematicsports.GetWorkerHOSStateRequest,
) (*telematics.WorkerHOSState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*telematics.WorkerHOSState, error) {
		s := telematicsdb.WorkerHOSStates
		entity, err := s.Where(s.WorkerID.Eq(req.WorkerID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, hosStateErrors.Map(err, req.WorkerID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpsertWorkerHOSViolations(
	ctx context.Context,
	violations []*telematics.WorkerHOSViolation,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(violations) == 0 {
			return nil
		}

		v := telematicsdb.WorkerHOSViolations
		_, err := v.Upsert(ctx, r.db.Keel(ctx), violations,
			keel.OnConflict(
				v.OrganizationID,
				v.BusinessUnitID,
				v.WorkerID,
				v.ViolationType,
				v.ViolationStartAt,
			).Update(v.Description, v.DurationMs, v.DayStartAt, v.DayEndAt, v.DetectedAt))
		if err != nil {
			return fmt.Errorf("upsert worker hos violations: %w", err)
		}

		return nil
	})
}

func (r *repository) ListWorkerHOSViolations(
	ctx context.Context,
	req *telematicsports.ListWorkerHOSViolationsRequest,
) ([]*telematics.WorkerHOSViolation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*telematics.WorkerHOSViolation, error) {
			v := telematicsdb.WorkerHOSViolations
			q := v.Where(
				keel.When(req.WorkerID.IsNotNil(), v.WorkerID.Eq(req.WorkerID)),
				keel.When(req.Since > 0, v.ViolationStartAt.Gte(req.Since)),
			).Order(v.ViolationStartAt.Desc())
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			entities, err := q.All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list worker hos violations: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListWorkerHOSViolationStats(
	ctx context.Context,
	req *telematicsports.ListWorkerHOSViolationStatsRequest,
) ([]*telematicsports.WorkerHOSViolationStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*telematicsports.WorkerHOSViolationStats, error) {
			v := telematicsdb.WorkerHOSViolations
			rows, err := keel.Into[telematicsports.WorkerHOSViolationStats](ctx, r.db.Keel(ctx), v.
				Group(v.WorkerID).
				Pick(v.WorkerID, keel.Count(), v.ViolationStartAt.Max()).
				Where(
					keel.When(len(req.WorkerIDs) > 0, v.WorkerID.In(req.WorkerIDs...)),
					keel.When(req.Since > 0, v.ViolationStartAt.Gte(req.Since)),
				))
			if err != nil {
				return nil, fmt.Errorf("list worker hos violation stats: %w", err)
			}

			stats := make([]*telematicsports.WorkerHOSViolationStats, 0, len(rows))
			for i := range rows {
				stats = append(stats, &rows[i])
			}

			return stats, nil
		},
	)
}

func (r *repository) SyncWorkerHOSLogs(
	ctx context.Context,
	req *telematicsports.SyncWorkerHOSLogsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.WorkerIDs) == 0 {
			return 0, nil
		}

		db := r.db.Keel(ctx)
		hl := telematicsdb.WorkerHOSLogs
		_, err := hl.Where(hl.WorkerID.In(req.WorkerIDs...), hl.LogStartAt.Gte(req.WindowStart)).
			Delete(ctx, db)
		if err != nil {
			return 0, fmt.Errorf("sync worker hos logs: %w", err)
		}

		if len(req.Logs) == 0 {
			return 0, nil
		}

		inserted, err := hl.Upsert(
			ctx,
			db,
			req.Logs,
			keel.OnConflict(hl.OrganizationID, hl.BusinessUnitID, hl.WorkerID, hl.LogStartAt).
				Update(
					hl.DutyStatus,
					hl.LogEndAt,
					hl.Remark,
					hl.Provider,
					hl.ProviderVehicleID,
					hl.ReceivedAt,
				),
		)
		if err != nil {
			return 0, fmt.Errorf("sync worker hos logs: %w", err)
		}

		return len(inserted), nil
	})
}

func (r *repository) ListWorkerHOSLogs(
	ctx context.Context,
	req *telematicsports.ListWorkerHOSLogsRequest,
) ([]*telematics.WorkerHOSLog, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.WorkerHOSLog, error) {
		hl := telematicsdb.WorkerHOSLogs
		entities, err := hl.Where(
			keel.When(len(req.WorkerIDs) > 0, hl.WorkerID.In(req.WorkerIDs...)),
			keel.When(req.Since > 0, hl.LogStartAt.Gte(req.Since)),
		).
			Order(hl.WorkerID.Asc(), hl.LogStartAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list worker hos logs: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) UpsertVehicleInspections(
	ctx context.Context,
	inspections []*telematics.VehicleInspection,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(inspections) == 0 {
			return nil
		}

		vi := telematicsdb.VehicleInspections
		_, err := vi.Upsert(ctx, r.db.Keel(ctx), inspections,
			keel.OnConflict(vi.OrganizationID, vi.BusinessUnitID, vi.Provider, vi.ProviderDvirID).
				Update(
					vi.TractorID,
					vi.WorkerID,
					vi.InspectionType,
					vi.SafetyStatus,
					vi.StartedAt,
					vi.EndedAt,
					vi.OdometerMeters,
					vi.Location,
					vi.Signed,
					vi.DefectCount,
					vi.UnresolvedDefectCount,
					vi.Defects,
				))
		if err != nil {
			return fmt.Errorf("upsert vehicle inspections: %w", err)
		}

		return nil
	})
}

func inspectionsBefore(req *telematicsports.ListVehicleInspectionsRequest) keel.Pred {
	if req.Before <= 0 {
		return nil
	}

	vi := telematicsdb.VehicleInspections
	if req.BeforeID.IsNil() {
		return vi.StartedAt.Lt(req.Before)
	}

	return keel.Or(
		vi.StartedAt.Lt(req.Before),
		keel.And(vi.StartedAt.Eq(req.Before), vi.ID.Lt(req.BeforeID)),
	)
}

func vehicleInspectionsQuery(
	req *telematicsports.ListVehicleInspectionsRequest,
) *keel.Query[*telematics.VehicleInspection] {
	vi := telematicsdb.VehicleInspections
	q := vi.Where(
		keel.When(req.TractorID.IsNotNil(), vi.TractorID.Eq(req.TractorID)),
		keel.When(req.WorkerID.IsNotNil(), vi.WorkerID.Eq(req.WorkerID)),
		keel.When(req.Since > 0, vi.StartedAt.Gte(req.Since)),
		inspectionsBefore(req),
	).Order(vi.StartedAt.Desc(), vi.ID.Desc())
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	return q
}

func (r *repository) ListVehicleInspections(
	ctx context.Context,
	req *telematicsports.ListVehicleInspectionsRequest,
) ([]*telematics.VehicleInspection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.VehicleInspection, error) {
		entities, err := vehicleInspectionsQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list vehicle inspections: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) InsertEvent(
	ctx context.Context,
	event *telematics.TelematicsEvent,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		e := telematicsdb.TelematicsEvents
		inserted, err := e.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*telematics.TelematicsEvent{event},
			e.OrganizationID,
			e.BusinessUnitID,
			e.Provider,
			e.EventID,
		)
		if err != nil {
			return false, fmt.Errorf("insert telematics event: %w", err)
		}

		return inserted > 0, nil
	})
}

func (r *repository) RecordStopOutcome(
	ctx context.Context,
	event *telematics.TelematicsEvent,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		e := telematicsdb.TelematicsEvents
		err := e.Save(
			ctx,
			r.db.Keel(ctx),
			event,
			e.StopOutcome,
			e.StopVisit,
			e.ShipmentMoveID,
			e.StopID,
			e.StopOutcomeReason,
		)
		if err != nil {
			return fmt.Errorf(
				"record telematics stop outcome: %w",
				eventErrors.Map(err, event.ID.String()),
			)
		}

		return nil
	})
}

func openStopReviewsQuery(
	req *telematicsports.ListOpenStopReviewsRequest,
) *keel.Query[*telematics.TelematicsEvent] {
	e := telematicsdb.TelematicsEvents
	m := shipmentdb.ShipmentMoves
	st := shipmentdb.Stops

	missingStamp := keel.Or(
		keel.And(e.StopVisit.Eq(shipment.VisitArrival), st.ActualArrival.IsNull()),
		keel.And(e.StopVisit.Eq(shipment.VisitDeparture), st.ActualDeparture.IsNull()),
	)

	q := e.Join(m, m.ID.EqCol(e.ShipmentMoveID)).
		LeftJoin(st, st.ID.EqCol(e.StopID)).
		Compute(e.MoveShipmentID, m.ShipmentID).
		Where(
			e.StopOutcome.In(telematics.StopOutcomeUnmatched, telematics.StopOutcomeRefused),
			m.Status.NotIn(shipment.MoveStatusCompleted, shipment.MoveStatusCanceled),
			keel.When(req.Since > 0, e.OccurredAt.Gt(req.Since)),
			keel.Or(
				e.StopID.IsNull(),
				keel.And(st.Status.Ne(shipment.StopStatusCanceled), missingStamp),
			),
		).
		Order(e.OccurredAt.Desc())
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	return q
}

func (r *repository) ListOpenStopReviews(
	ctx context.Context,
	req *telematicsports.ListOpenStopReviewsRequest,
) ([]*telematicsports.StopReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematicsports.StopReview, error) {
		events, err := openStopReviewsQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list open telematics stop reviews: %w", err)
		}

		reviews := make([]*telematicsports.StopReview, 0, len(events))
		for _, event := range events {
			reviews = append(reviews, &telematicsports.StopReview{
				Event:      event,
				ShipmentID: event.MoveShipmentID,
			})
		}

		return reviews, nil
	})
}

func (r *repository) GetFeedState(
	ctx context.Context,
	_ pagination.TenantInfo,
	provider string,
	feedType telematics.FeedType,
) (*telematics.FeedState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*telematics.FeedState, error) {
		fs := telematicsdb.FeedStates
		entity, err := fs.Where(fs.Provider.Eq(provider), fs.FeedType.Eq(feedType)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, feedStateErrors.Map(err, provider)
		}

		return entity, nil
	})
}

func (r *repository) UpsertFeedState(
	ctx context.Context,
	state *telematics.FeedState,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		fs := telematicsdb.FeedStates
		_, err := fs.Upsert(ctx, r.db.Keel(ctx), []*telematics.FeedState{state},
			keel.OnConflict(fs.OrganizationID, fs.BusinessUnitID, fs.Provider, fs.FeedType).Update(
				fs.Cursor,
				fs.LastPolledAt,
				fs.LastSuccessAt,
				fs.FailureCount,
				fs.LastError,
			))
		if err != nil {
			return fmt.Errorf("upsert feed state: %w", err)
		}

		return nil
	})
}

func (r *repository) ListWorkerMappings(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]telematicsports.WorkerTelematicsMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]telematicsports.WorkerTelematicsMapping, error) {
			w := workerdb.Workers
			rows, err := keel.Into[telematicsports.WorkerTelematicsMapping](ctx, r.db.Keel(ctx), w.
				Pick(w.ID, w.ExternalID, w.FirstName, w.LastName).
				Where(w.ExternalID.Trim().NullIf("").IsNotNull()))
			if err != nil {
				return nil, fmt.Errorf("list worker telematics mappings: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) ListTractorMappings(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]telematicsports.TractorTelematicsMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]telematicsports.TractorTelematicsMapping, error) {
			t := tractordb.Tractors
			rows, err := keel.Into[telematicsports.TractorTelematicsMapping](ctx, r.db.Keel(ctx), t.
				Pick(t.ID, t.ExternalID.Coalesce(""), t.Vin.Coalesce(""), t.Code))
			if err != nil {
				return nil, fmt.Errorf("list tractor telematics mappings: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) AssignTractorExternalIDs(
	ctx context.Context,
	req telematicsports.AssignTractorExternalIDsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.Assignments) == 0 {
			return 0, nil
		}

		ids := make([]string, 0, len(req.Assignments))
		externalIDs := make([]string, 0, len(req.Assignments))
		for _, assignment := range req.Assignments {
			ids = append(ids, assignment.TractorID.String())
			externalIDs = append(externalIDs, assignment.ExternalID)
		}

		t := tractordb.Tractors
		data := keel.Unnest("assignment", []string{"id", "external_id"}, ids, externalIDs)
		n, err := t.With(data).
			Join(data, data.Col("id").EqCol(t.ID)).
			Where(keel.True()).
			Update(ctx, r.db.Keel(ctx), t.ExternalID.ToExpr(data.Col("external_id")), t.Version.Increment(0))
		if err != nil {
			return 0, fmt.Errorf("assign tractor external ids: %w", err)
		}

		return int(n), nil
	})
}

func (r *repository) ListTrailerMappings(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]telematicsports.TrailerTelematicsMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]telematicsports.TrailerTelematicsMapping, error) {
			t := trailerdb.Trailers
			rows, err := keel.Into[telematicsports.TrailerTelematicsMapping](ctx, r.db.Keel(ctx), t.
				Pick(t.ID, t.ExternalID.Coalesce(""), t.Vin.Coalesce(""), t.Code))
			if err != nil {
				return nil, fmt.Errorf("list trailer telematics mappings: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) AssignTrailerExternalIDs(
	ctx context.Context,
	req telematicsports.AssignTrailerExternalIDsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.Assignments) == 0 {
			return 0, nil
		}

		ids := make([]string, 0, len(req.Assignments))
		externalIDs := make([]string, 0, len(req.Assignments))
		for _, assignment := range req.Assignments {
			ids = append(ids, assignment.TrailerID.String())
			externalIDs = append(externalIDs, assignment.ExternalID)
		}

		t := trailerdb.Trailers
		data := keel.Unnest("assignment", []string{"id", "external_id"}, ids, externalIDs)
		n, err := t.With(data).
			Join(data, data.Col("id").EqCol(t.ID)).
			Where(keel.True()).
			Update(ctx, r.db.Keel(ctx), t.ExternalID.ToExpr(data.Col("external_id")), t.Version.Increment(0))
		if err != nil {
			return 0, fmt.Errorf("assign trailer external ids: %w", err)
		}

		return int(n), nil
	})
}

func (r *repository) UpdateWorkerRulesets(
	ctx context.Context,
	req telematicsports.UpdateWorkerRulesetsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.Assignments) == 0 {
			return 0, nil
		}

		size := len(req.Assignments)
		workerIDs := make([]string, 0, size)
		cycles := make([]string, 0, size)
		shifts := make([]string, 0, size)
		restarts := make([]string, 0, size)
		breaks := make([]string, 0, size)
		jurisdictions := make([]string, 0, size)
		for _, assignment := range req.Assignments {
			workerIDs = append(workerIDs, assignment.WorkerID.String())
			cycles = append(cycles, assignment.Cycle)
			shifts = append(shifts, assignment.Shift)
			restarts = append(restarts, assignment.Restart)
			breaks = append(breaks, assignment.Break)
			jurisdictions = append(jurisdictions, assignment.Jurisdiction)
		}

		s := telematicsdb.WorkerHOSStates
		data := keel.Unnest(
			"ruleset",
			[]string{"worker_id", "cycle", "shift", "restart", "break", "jurisdiction"},
			workerIDs, cycles, shifts, restarts, breaks, jurisdictions,
		)
		n, err := s.With(data).
			Join(data, data.Col("worker_id").EqCol(s.WorkerID)).
			Where(keel.True()).
			Update(
				ctx,
				r.db.Keel(ctx),
				s.RulesetCycle.ToExpr(data.Col("cycle")),
				s.RulesetShift.ToExpr(data.Col("shift")),
				s.RulesetRestart.ToExpr(data.Col("restart")),
				s.RulesetBreak.ToExpr(data.Col("break")),
				s.RulesetJurisdiction.ToExpr(data.Col("jurisdiction")),
			)
		if err != nil {
			return 0, fmt.Errorf("update worker rulesets: %w", err)
		}

		return int(n), nil
	})
}

func (r *repository) GetWebhookConfigByToken(
	ctx context.Context,
	typ integration.Type,
	token string,
) (*telematicsports.TelematicsWebhookConfig, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve the telematics webhook a provider called before its tenant is known",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*telematicsports.TelematicsWebhookConfig, error) {
			in := integrationdb.Integrations
			row, err := keel.IntoOne[webhookRow](ctx, r.db.Keel(ctx), in.
				AllTenants().
				Pick(in.OrganizationID, in.BusinessUnitID, in.Configuration.Text("webhookSecret")).
				Where(
					in.Type.Eq(typ),
					in.Configuration.Text("webhookToken").Eq(token),
					in.Enabled.IsTrue(),
				).
				Limit(1))
			if err != nil {
				return nil, webhookErrors.Map(err, "")
			}

			return &telematicsports.TelematicsWebhookConfig{
				TenantInfo: pagination.TenantInfo{
					OrgID: row.OrganizationID,
					BuID:  row.BusinessUnitID,
				},
				WebhookSecret: row.WebhookSecret,
			}, nil
		},
	)
}

func (r *repository) CleanupExpired(
	ctx context.Context,
	eventsOlderThan int64,
	violationsOlderThan int64,
	hosLogsOlderThan int64,
) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "clean up expired telematics data across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		db := r.db.Keel(ctx)

		e := telematicsdb.TelematicsEvents
		events, err := e.AllTenants().Where(e.OccurredAt.Lt(eventsOlderThan)).Delete(ctx, db)
		if err != nil {
			return 0, fmt.Errorf("cleanup telematics events: %w", err)
		}

		v := telematicsdb.WorkerHOSViolations
		violations, err := v.AllTenants().
			Where(v.ViolationStartAt.Lt(violationsOlderThan)).
			Delete(ctx, db)
		if err != nil {
			return 0, fmt.Errorf("cleanup hos violations: %w", err)
		}

		hl := telematicsdb.WorkerHOSLogs
		logs, err := hl.AllTenants().Where(hl.LogStartAt.Lt(hosLogsOlderThan)).Delete(ctx, db)
		if err != nil {
			return 0, fmt.Errorf("cleanup hos logs: %w", err)
		}

		return events + violations + logs, nil
	})
}
