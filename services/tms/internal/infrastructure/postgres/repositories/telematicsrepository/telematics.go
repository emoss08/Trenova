package telematicsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TelematicsRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.telematics-repository"),
	}
}

func (r *repository) UpsertVehiclePositions(
	ctx context.Context,
	positions []*telematics.VehiclePosition,
) error {
	if len(positions) == 0 {
		return nil
	}

	cols := buncolgen.VehiclePositionColumns
	_, err := r.db.DB().NewInsert().
		Model(&positions).
		On("CONFLICT (organization_id, business_unit_id, tractor_id) DO UPDATE").
		Set(cols.Provider.SetExcluded()).
		Set(cols.ProviderVehicleID.SetExcluded()).
		Set(cols.Latitude.SetExcluded()).
		Set(cols.Longitude.SetExcluded()).
		Set(cols.HeadingDegrees.SetExcluded()).
		Set(cols.SpeedMph.SetExcluded()).
		Set(cols.EngineState.SetExcluded()).
		Set(cols.FuelPercent.SetExcluded()).
		Set(cols.OdometerMeters.SetExcluded()).
		Set(cols.FormattedLocation.SetExcluded()).
		Set(cols.RecordedAt.SetExcluded()).
		Set(cols.ReceivedAt.SetExcluded()).
		Where(cols.RecordedAt.Qualified() + " <= EXCLUDED.recorded_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert vehicle positions: %w", err)
	}
	return nil
}

func (r *repository) ListVehiclePositions(
	ctx context.Context,
	req *repositories.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	cols := buncolgen.VehiclePositionColumns
	rel := buncolgen.VehiclePositionRelations

	entities := make([]*telematics.VehiclePosition, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.VehiclePositionScopeTenant(sq, req.TenantInfo)
			if len(req.TractorIDs) > 0 {
				sq = sq.Where(cols.TractorID.In(), bun.List(req.TractorIDs))
			}
			if req.MaxAgeSeconds > 0 {
				sq = sq.Where(cols.RecordedAt.Gte(), timeutils.NowUnix()-req.MaxAgeSeconds)
			}
			return sq
		}).
		Order(cols.RecordedAt.OrderDesc())

	if req.IncludeTractor {
		q = q.Relation(rel.Tractor)
	}
	if req.IncludeWorker {
		q = q.Relation(buncolgen.Rel(rel.Tractor, buncolgen.TractorRelations.PrimaryWorker))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list vehicle positions: %w", err)
	}
	return entities, nil
}

func (r *repository) UpsertWorkerHOSStates(
	ctx context.Context,
	states []*telematics.WorkerHOSState,
) error {
	if len(states) == 0 {
		return nil
	}

	cols := buncolgen.WorkerHOSStateColumns
	_, err := r.db.DB().NewInsert().
		Model(&states).
		On("CONFLICT (organization_id, business_unit_id, worker_id) DO UPDATE").
		Set(cols.Provider.SetExcluded()).
		Set(cols.ProviderDriverID.SetExcluded()).
		Set(cols.DutyStatus.SetExcluded()).
		Set(cols.DriveRemainingMs.SetExcluded()).
		Set(cols.ShiftRemainingMs.SetExcluded()).
		Set(cols.CycleRemainingMs.SetExcluded()).
		Set(cols.CycleTomorrowMs.SetExcluded()).
		Set(cols.BreakRemainingMs.SetExcluded()).
		Set(cols.CycleStartedAt.SetExcluded()).
		Set(cols.ShiftDrivingViolationMs.SetExcluded()).
		Set(cols.CycleViolationMs.SetExcluded()).
		Set(cols.CurrentVehicleID.SetExcluded()).
		Set(cols.CurrentTractorID.SetExcluded()).
		Set(cols.RecordedAt.SetExcluded()).
		Set(cols.ReceivedAt.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert worker hos states: %w", err)
	}
	return nil
}

func (r *repository) ListWorkerHOSStates(
	ctx context.Context,
	req *repositories.ListWorkerHOSStatesRequest,
) ([]*telematics.WorkerHOSState, error) {
	cols := buncolgen.WorkerHOSStateColumns

	entities := make([]*telematics.WorkerHOSState, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerHOSStateScopeTenant(sq, req.TenantInfo)
			if len(req.WorkerIDs) > 0 {
				sq = sq.Where(cols.WorkerID.In(), bun.List(req.WorkerIDs))
			}
			return sq
		}).
		Order(cols.DriveRemainingMs.OrderAsc())

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerHOSStateRelations.Worker)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list worker hos states: %w", err)
	}
	return entities, nil
}

func (r *repository) GetWorkerHOSState(
	ctx context.Context,
	req repositories.GetWorkerHOSStateRequest,
) (*telematics.WorkerHOSState, error) {
	entity := new(telematics.WorkerHOSState)
	err := r.db.DB().NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerHOSStateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerHOSStateColumns.WorkerID.Eq(), req.WorkerID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerHOSState")
	}
	return entity, nil
}

func (r *repository) UpsertWorkerHOSViolations(
	ctx context.Context,
	violations []*telematics.WorkerHOSViolation,
) error {
	if len(violations) == 0 {
		return nil
	}

	cols := buncolgen.WorkerHOSViolationColumns
	_, err := r.db.DB().NewInsert().
		Model(&violations).
		On("CONFLICT (organization_id, business_unit_id, worker_id, violation_type, violation_start_at) DO UPDATE").
		Set(cols.Description.SetExcluded()).
		Set(cols.DurationMs.SetExcluded()).
		Set(cols.DayStartAt.SetExcluded()).
		Set(cols.DayEndAt.SetExcluded()).
		Set(cols.DetectedAt.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert worker hos violations: %w", err)
	}
	return nil
}

func (r *repository) ListWorkerHOSViolations(
	ctx context.Context,
	req *repositories.ListWorkerHOSViolationsRequest,
) ([]*telematics.WorkerHOSViolation, error) {
	cols := buncolgen.WorkerHOSViolationColumns

	entities := make([]*telematics.WorkerHOSViolation, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerHOSViolationScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.Since > 0 {
				sq = sq.Where(cols.ViolationStartAt.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.ViolationStartAt.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list worker hos violations: %w", err)
	}
	return entities, nil
}

func (r *repository) UpsertVehicleInspections(
	ctx context.Context,
	inspections []*telematics.VehicleInspection,
) error {
	if len(inspections) == 0 {
		return nil
	}

	cols := buncolgen.VehicleInspectionColumns
	_, err := r.db.DB().NewInsert().
		Model(&inspections).
		On("CONFLICT (organization_id, business_unit_id, provider, provider_dvir_id) DO UPDATE").
		Set(cols.TractorID.SetExcluded()).
		Set(cols.WorkerID.SetExcluded()).
		Set(cols.InspectionType.SetExcluded()).
		Set(cols.SafetyStatus.SetExcluded()).
		Set(cols.StartedAt.SetExcluded()).
		Set(cols.EndedAt.SetExcluded()).
		Set(cols.OdometerMeters.SetExcluded()).
		Set(cols.Location.SetExcluded()).
		Set(cols.Signed.SetExcluded()).
		Set(cols.DefectCount.SetExcluded()).
		Set(cols.UnresolvedDefectCount.SetExcluded()).
		Set(cols.Defects.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert vehicle inspections: %w", err)
	}
	return nil
}

func (r *repository) ListVehicleInspections(
	ctx context.Context,
	req *repositories.ListVehicleInspectionsRequest,
) ([]*telematics.VehicleInspection, error) {
	cols := buncolgen.VehicleInspectionColumns

	entities := make([]*telematics.VehicleInspection, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.VehicleInspectionScopeTenant(sq, req.TenantInfo)
			if !req.TractorID.IsNil() {
				sq = sq.Where(cols.TractorID.Eq(), req.TractorID)
			}
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.Since > 0 {
				sq = sq.Where(cols.StartedAt.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.StartedAt.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list vehicle inspections: %w", err)
	}
	return entities, nil
}

func (r *repository) InsertEvent(
	ctx context.Context,
	event *telematics.TelematicsEvent,
) (bool, error) {
	result, err := r.db.DB().NewInsert().
		Model(event).
		On("CONFLICT (organization_id, business_unit_id, provider, event_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("insert telematics event: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert telematics event rows affected: %w", err)
	}
	return rows > 0, nil
}

func (r *repository) GetFeedState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider string,
	feedType telematics.FeedType,
) (*telematics.FeedState, error) {
	entity := new(telematics.FeedState)
	err := r.db.DB().NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FeedStateScopeTenant(sq, tenantInfo).
				Where(buncolgen.FeedStateColumns.Provider.Eq(), provider).
				Where(buncolgen.FeedStateColumns.FeedType.Eq(), feedType)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FeedState")
	}
	return entity, nil
}

func (r *repository) UpsertFeedState(
	ctx context.Context,
	state *telematics.FeedState,
) error {
	cols := buncolgen.FeedStateColumns
	_, err := r.db.DB().NewInsert().
		Model(state).
		On("CONFLICT (organization_id, business_unit_id, provider, feed_type) DO UPDATE").
		Set(cols.Cursor.SetExcluded()).
		Set(cols.LastPolledAt.SetExcluded()).
		Set(cols.LastSuccessAt.SetExcluded()).
		Set(cols.FailureCount.SetExcluded()).
		Set(cols.LastError.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert feed state: %w", err)
	}
	return nil
}

func (r *repository) ListWorkerMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.WorkerTelematicsMapping, error) {
	cols := buncolgen.WorkerColumns

	rows := make([]repositories.WorkerTelematicsMapping, 0)
	err := r.db.DB().NewSelect().
		TableExpr(buncolgen.WorkerTable.Name+" AS "+buncolgen.WorkerTable.Alias).
		ColumnExpr(cols.ID.As("worker_id")).
		ColumnExpr(cols.ExternalID.As("external_id")).
		ColumnExpr(cols.FirstName.As("first_name")).
		ColumnExpr(cols.LastName.As("last_name")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
				Where(cols.ExternalID.Expr("NULLIF(BTRIM({}), '') IS NOT NULL"))
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list worker telematics mappings: %w", err)
	}
	return rows, nil
}

func (r *repository) ListTractorMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.TractorTelematicsMapping, error) {
	cols := buncolgen.TractorColumns

	rows := make([]repositories.TractorTelematicsMapping, 0)
	err := r.db.DB().NewSelect().
		TableExpr(buncolgen.TractorTable.Name+" AS "+buncolgen.TractorTable.Alias).
		ColumnExpr(cols.ID.As("tractor_id")).
		ColumnExpr(buncolgen.Coalesce(cols.ExternalID, "''", "external_id")).
		ColumnExpr(buncolgen.Coalesce(cols.Vin, "''", "vin")).
		ColumnExpr(cols.Code.As("code")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list tractor telematics mappings: %w", err)
	}
	return rows, nil
}

func (r *repository) AssignTractorExternalIDs(
	ctx context.Context,
	req repositories.AssignTractorExternalIDsRequest,
) (int, error) {
	if len(req.Assignments) == 0 {
		return 0, nil
	}

	cols := buncolgen.TractorColumns
	assigned := 0
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, assignment := range req.Assignments {
			result, execErr := tx.NewUpdate().
				Model((*tractor.Tractor)(nil)).
				Set(cols.ExternalID.Set(), assignment.ExternalID).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.
						Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(cols.ID.Eq(), assignment.TractorID)
				}).
				Exec(ctx)
			if execErr != nil {
				return execErr
			}
			rows, raErr := result.RowsAffected()
			if raErr != nil {
				return raErr
			}
			assigned += int(rows)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("assign tractor external ids: %w", err)
	}
	return assigned, nil
}

func (r *repository) ListTrailerMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.TrailerTelematicsMapping, error) {
	cols := buncolgen.TrailerColumns

	rows := make([]repositories.TrailerTelematicsMapping, 0)
	err := r.db.DB().NewSelect().
		TableExpr(buncolgen.TrailerTable.Name+" AS "+buncolgen.TrailerTable.Alias).
		ColumnExpr(cols.ID.As("trailer_id")).
		ColumnExpr(buncolgen.Coalesce(cols.ExternalID, "''", "external_id")).
		ColumnExpr(buncolgen.Coalesce(cols.Vin, "''", "vin")).
		ColumnExpr(cols.Code.As("code")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list trailer telematics mappings: %w", err)
	}
	return rows, nil
}

func (r *repository) AssignTrailerExternalIDs(
	ctx context.Context,
	req repositories.AssignTrailerExternalIDsRequest,
) (int, error) {
	if len(req.Assignments) == 0 {
		return 0, nil
	}

	cols := buncolgen.TrailerColumns
	assigned := 0
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, assignment := range req.Assignments {
			result, execErr := tx.NewUpdate().
				Model((*trailer.Trailer)(nil)).
				Set(cols.ExternalID.Set(), assignment.ExternalID).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return uq.
						Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(cols.ID.Eq(), assignment.TrailerID)
				}).
				Exec(ctx)
			if execErr != nil {
				return execErr
			}
			rows, raErr := result.RowsAffected()
			if raErr != nil {
				return raErr
			}
			assigned += int(rows)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("assign trailer external ids: %w", err)
	}
	return assigned, nil
}

func (r *repository) UpdateWorkerRulesets(
	ctx context.Context,
	req repositories.UpdateWorkerRulesetsRequest,
) (int, error) {
	if len(req.Assignments) == 0 {
		return 0, nil
	}

	cols := buncolgen.WorkerHOSStateColumns
	updated := 0
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, assignment := range req.Assignments {
			result, execErr := tx.NewUpdate().
				Model((*telematics.WorkerHOSState)(nil)).
				Set(cols.RulesetCycle.Set(), assignment.Cycle).
				Set(cols.RulesetShift.Set(), assignment.Shift).
				Set(cols.RulesetRestart.Set(), assignment.Restart).
				Set(cols.RulesetBreak.Set(), assignment.Break).
				Set(cols.RulesetJurisdiction.Set(), assignment.Jurisdiction).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.WorkerHOSStateScopeTenantUpdate(uq, req.TenantInfo).
						Where(cols.WorkerID.Eq(), assignment.WorkerID)
				}).
				Exec(ctx)
			if execErr != nil {
				return execErr
			}
			rows, raErr := result.RowsAffected()
			if raErr != nil {
				return raErr
			}
			updated += int(rows)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("update worker rulesets: %w", err)
	}
	return updated, nil
}

func (r *repository) GetWebhookConfigByToken(
	ctx context.Context,
	typ integration.Type,
	token string,
) (*repositories.TelematicsWebhookConfig, error) {
	var row struct {
		OrganizationID string `bun:"organization_id"`
		BusinessUnitID string `bun:"business_unit_id"`
		WebhookSecret  string `bun:"webhook_secret"`
	}

	cols := buncolgen.IntegrationColumns
	err := r.db.DB().NewSelect().
		TableExpr(buncolgen.IntegrationTable.Name+" AS "+buncolgen.IntegrationTable.Alias).
		ColumnExpr(cols.OrganizationID.String()).
		ColumnExpr(cols.BusinessUnitID.String()).
		ColumnExpr(cols.Configuration.Expr("{} ->> 'webhookSecret'")+" AS webhook_secret").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.Type.Eq(), typ).
				Where(cols.Configuration.Expr("{} ->> 'webhookToken'")+" = ?", token).
				Where(cols.Enabled.IsTrue())
		}).
		Scan(ctx, &row)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "TelematicsWebhook")
	}

	orgID, err := pulid.Parse(row.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("parse webhook organization id: %w", err)
	}
	buID, err := pulid.Parse(row.BusinessUnitID)
	if err != nil {
		return nil, fmt.Errorf("parse webhook business unit id: %w", err)
	}

	return &repositories.TelematicsWebhookConfig{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		WebhookSecret: row.WebhookSecret,
	}, nil
}

func (r *repository) CleanupExpired(
	ctx context.Context,
	eventsOlderThan int64,
	violationsOlderThan int64,
) (int64, error) {
	total := int64(0)

	eventsResult, err := r.db.DB().NewDelete().
		Model((*telematics.TelematicsEvent)(nil)).
		Where(buncolgen.TelematicsEventColumns.OccurredAt.Lt(), eventsOlderThan).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleanup telematics events: %w", err)
	}
	if rows, raErr := eventsResult.RowsAffected(); raErr == nil {
		total += rows
	}

	violationsResult, err := r.db.DB().NewDelete().
		Model((*telematics.WorkerHOSViolation)(nil)).
		Where(buncolgen.WorkerHOSViolationColumns.ViolationStartAt.Lt(), violationsOlderThan).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleanup hos violations: %w", err)
	}
	if rows, raErr := violationsResult.RowsAffected(); raErr == nil {
		total += rows
	}

	return total, nil
}
