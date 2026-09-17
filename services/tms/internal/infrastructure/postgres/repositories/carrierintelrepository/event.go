package carrierintelrepository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultDigestLimit   = 200
	maxDigestLimit       = 1000
	eventEntityName      = "CarrierIntelEvent"
	eventDetectedAtField = "detectedAt"
)

var (
	openEventStatuses = []carrierintel.EventStatus{
		carrierintel.EventStatusOpen,
		carrierintel.EventStatusAcknowledged,
	}
	digestEventActions = []carrierintel.RuleAction{
		carrierintel.RuleActionBlock,
		carrierintel.RuleActionWarn,
		carrierintel.RuleActionNotify,
	}
	eventSeverityOrder = []carrierintel.Severity{
		carrierintel.SeverityCritical,
		carrierintel.SeverityHigh,
		carrierintel.SeverityMedium,
		carrierintel.SeverityLow,
		carrierintel.SeverityInfo,
	}
	eventSeverityRankExpr = buncolgen.CarrierIntelEventColumns.Severity.Expr(
		"CASE {}" + strings.Repeat(" WHEN ? THEN ?", len(eventSeverityOrder)) + " ELSE ? END",
	)
)

type eventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewEventRepository(p Params) repositories.CarrierIntelEventRepository {
	return &eventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-event-repository"),
	}
}

func buildInsertIgnoreDuplicateEvents(
	db bun.IDB,
	rows *[]*carrierintel.CarrierIntelEvent,
) *bun.InsertQuery {
	return db.NewInsert().
		Model(rows).
		On(
			"CONFLICT (organization_id, business_unit_id, provider, subject_type, subject_id, fingerprint) DO NOTHING",
		).
		Returning("*")
}

func (r *eventRepository) InsertIgnoreDuplicates(
	ctx context.Context,
	entities []*carrierintel.CarrierIntelEvent,
) ([]*carrierintel.CarrierIntelEvent, error) {
	if len(entities) == 0 {
		return []*carrierintel.CarrierIntelEvent{}, nil
	}

	rows := entities
	inserted := make([]*carrierintel.CarrierIntelEvent, 0, len(entities))
	if _, err := buildInsertIgnoreDuplicateEvents(r.db.DBForContext(ctx), &rows).
		Exec(ctx, &inserted); err != nil {
		r.l.Error("failed to insert carrier intel events", zap.Error(err))
		return nil, fmt.Errorf("insert carrier intel events: %w", err)
	}

	return inserted, nil
}

func (r *eventRepository) applyEventFilters(
	q *bun.SelectQuery,
	req *repositories.ListCarrierIntelEventsRequest,
) *bun.SelectQuery {
	cols := buncolgen.CarrierIntelEventColumns
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if req.OpenOnly {
		q = q.Where(cols.Status.In(), bun.List(openEventStatuses))
	}
	if len(req.Severities) > 0 {
		q = q.Where(cols.Severity.In(), bun.List(req.Severities))
	}
	if len(req.Categories) > 0 {
		q = q.Where(cols.Category.In(), bun.List(req.Categories))
	}
	if req.SubjectType != "" {
		q = q.Where(cols.SubjectType.Eq(), req.SubjectType)
	}
	if req.SubjectID != "" {
		q = q.Where(cols.SubjectID.Eq(), req.SubjectID)
	}
	if !req.CarrierID.IsNil() {
		q = q.Where(cols.CarrierID.Eq(), req.CarrierID)
	}

	return q
}

func (r *eventRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierIntelEventsRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierIntelEvent], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: eventDetectedAtField, Direction: dbtype.SortDirectionDesc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*carrierintel.CarrierIntelEvent)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.CarrierIntelEventTable.Alias,
					req.Filter,
					(*carrierintel.CarrierIntelEvent)(nil),
				)
				return r.applyEventFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count carrier intel events", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carrierintel.CarrierIntelEvent]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*carrierintel.CarrierIntelEvent) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.CarrierIntelEventTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CarrierIntelEventTable.Alias,
					req.Filter,
					req.Cursor,
					(*carrierintel.CarrierIntelEvent)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyEventFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list carrier intel events", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *eventRepository) GetByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*carrierintel.CarrierIntelEvent, error) {
	if len(ids) == 0 {
		return []*carrierintel.CarrierIntelEvent{}, nil
	}

	entities := make([]*carrierintel.CarrierIntelEvent, 0, len(ids))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelEventScopeTenant(sq, tenantInfo).
				Where(buncolgen.CarrierIntelEventColumns.ID.In(), bun.List(ids))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get carrier intel events", zap.Error(err))
		return nil, fmt.Errorf("get carrier intel events: %w", err)
	}

	return entities, nil
}

func (r *eventRepository) Update(
	ctx context.Context,
	entity *carrierintel.CarrierIntelEvent,
) (*carrierintel.CarrierIntelEvent, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CarrierIntelEventColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update carrier intel event", zap.Error(err))
		return nil, fmt.Errorf("update carrier intel event: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, eventEntityName, entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *eventRepository) BulkAcknowledge(
	ctx context.Context,
	req *repositories.BulkAcknowledgeCarrierIntelEventsRequest,
) (int, error) {
	if len(req.EventIDs) == 0 {
		return 0, nil
	}

	cols := buncolgen.CarrierIntelEventColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrierintel.CarrierIntelEvent)(nil)).
		Set(cols.Status.Set(), carrierintel.EventStatusAcknowledged).
		Set(cols.AcknowledgedByID.Set(), req.UserID).
		Set(cols.AcknowledgedAt.Set(), req.AcknowledgedAt).
		Set(cols.UpdatedAt.Set(), req.AcknowledgedAt).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierIntelEventScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.EventIDs)).
				Where(cols.Status.Eq(), carrierintel.EventStatusOpen)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to acknowledge carrier intel events", zap.Error(err))
		return 0, fmt.Errorf("acknowledge carrier intel events: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("acknowledge carrier intel events rows affected: %w", err)
	}

	return int(affected), nil
}

func (r *eventRepository) CountOpenByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	if len(carrierIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.CarrierIntelEventColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelEvent)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelEventScopeTenant(sq, tenantInfo).
				Where(cols.Status.In(), bun.List(openEventStatuses)).
				Where(cols.CarrierID.In(), bun.List(carrierIDs))
		})

	counts, err := dbhelper.CountByID(ctx, q, cols.CarrierID, len(carrierIDs))
	if err != nil {
		r.l.Error("failed to count open carrier intel events by carrier", zap.Error(err))
		return nil, fmt.Errorf("count open carrier intel events by carrier: %w", err)
	}

	return counts, nil
}

func (r *eventRepository) CountOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.CarrierIntelEventCounts, error) {
	cols := buncolgen.CarrierIntelEventColumns
	counts := &repositories.CarrierIntelEventCounts{
		BySeverity: make(map[carrierintel.Severity]int, len(eventSeverityOrder)),
	}
	severityCounts := make([]int, len(eventSeverityOrder))
	dest := make([]any, 0, len(eventSeverityOrder)+2)
	dest = append(dest, &counts.Open, &counts.Acknowledged)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierIntelEvent)(nil)).
		ColumnExpr(buncolgen.CountFilter("open_count", cols.Status.Eq()), carrierintel.EventStatusOpen).
		ColumnExpr(
			buncolgen.CountFilter("acknowledged_count", cols.Status.Eq()),
			carrierintel.EventStatusAcknowledged,
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelEventScopeTenant(sq, tenantInfo).
				Where(cols.Status.In(), bun.List(openEventStatuses))
		})
	for i, severity := range eventSeverityOrder {
		q = q.ColumnExpr(
			buncolgen.CountFilter("severity_"+strconv.Itoa(i), cols.Severity.Eq()),
			severity,
		)
		dest = append(dest, &severityCounts[i])
	}

	if err := q.Scan(ctx, dest...); err != nil {
		r.l.Error("failed to count open carrier intel events", zap.Error(err))
		return nil, fmt.Errorf("count open carrier intel events: %w", err)
	}

	for i, severity := range eventSeverityOrder {
		counts.BySeverity[severity] = severityCounts[i]
	}

	return counts, nil
}

func (r *eventRepository) ListForDigest(
	ctx context.Context,
	req *repositories.ListCarrierIntelDigestEventsRequest,
) ([]*carrierintel.CarrierIntelEvent, error) {
	cols := buncolgen.CarrierIntelEventColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultDigestLimit),
		1,
		maxDigestLimit,
	)

	severityRank := make([]any, 0, len(eventSeverityOrder)*2+1)
	for i, severity := range eventSeverityOrder {
		severityRank = append(severityRank, severity, i)
	}
	severityRank = append(severityRank, len(eventSeverityOrder))

	entities := make([]*carrierintel.CarrierIntelEvent, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelEventScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), carrierintel.EventStatusOpen).
				Where(cols.DetectedAt.Gte(), req.Since).
				Where(cols.Action.In(), bun.List(digestEventActions))
		}).
		OrderExpr(eventSeverityRankExpr, severityRank...).
		Order(cols.DetectedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier intel digest events", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel digest events: %w", err)
	}

	return entities, nil
}

func (r *eventRepository) MarkNotified(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	notifiedAt int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	cols := buncolgen.CarrierIntelEventColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrierintel.CarrierIntelEvent)(nil)).
		Set(cols.NotifiedAt.Set(), notifiedAt).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierIntelEventScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.In(), bun.List(ids))
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark carrier intel events notified", zap.Error(err))
		return fmt.Errorf("mark carrier intel events notified: %w", err)
	}

	return nil
}
