package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultDigestLimit   = 200
	maxDigestLimit       = 1000
	eventEntityName      = "CarrierIntelEvent"
	eventDetectedAtField = "detectedAt"
)

var (
	eventErrors = dberror.Mapping{Entity: eventEntityName}

	openEventStatuses = []carrierintelevent.EventStatus{
		carrierintelevent.EventStatusOpen,
		carrierintelevent.EventStatusAcknowledged,
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
)

type eventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

type eventStatusSeverityCount struct {
	Status   carrierintelevent.EventStatus
	Severity carrierintel.Severity
	Count    int64
}

type carrierEventCount struct {
	CarrierID pulid.ID
	Count     int64
}

func NewEventRepository(p Params) carrierintelports.CarrierIntelEventRepository {
	return &eventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-event-repository"),
	}
}

func (r *eventRepository) InsertIgnoreDuplicates(
	ctx context.Context,
	entities []*carrierintelevent.CarrierIntelEvent,
) ([]*carrierintelevent.CarrierIntelEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintelevent.CarrierIntelEvent, error) {
			if len(entities) == 0 {
				return []*carrierintelevent.CarrierIntelEvent{}, nil
			}

			e := carrierinteldb.CarrierIntelEvents
			db := r.db.Keel(ctx)
			inserted, err := e.CreateOrSkip(
				ctx,
				db,
				entities,
				e.OrganizationID,
				e.BusinessUnitID,
				e.Provider,
				e.SubjectType,
				e.SubjectID,
				e.Fingerprint,
			)
			if err != nil {
				r.l.Error("failed to insert carrier intel events", zap.Error(err))
				return nil, fmt.Errorf("insert carrier intel events: %w", err)
			}

			switch inserted {
			case int64(len(entities)):
				return entities, nil
			case 0:
				return []*carrierintelevent.CarrierIntelEvent{}, nil
			}

			ids := make([]pulid.ID, 0, len(entities))
			for _, entity := range entities {
				ids = append(ids, entity.ID)
			}
			stored, err := keel.Into[pulid.ID](
				ctx,
				db,
				e.Pick(e.ID).Where(e.ID.In(ids...)),
			)
			if err != nil {
				return nil, fmt.Errorf("read inserted carrier intel events: %w", err)
			}

			kept := make(map[pulid.ID]struct{}, len(stored))
			for _, id := range stored {
				kept[id] = struct{}{}
			}
			result := make([]*carrierintelevent.CarrierIntelEvent, 0, len(stored))
			for _, entity := range entities {
				if _, ok := kept[entity.ID]; ok {
					result = append(result, entity)
				}
			}

			return result, nil
		},
	)
}

func eventFilters(req *carrierintelports.ListCarrierIntelEventsRequest) []keel.Pred {
	e := carrierinteldb.CarrierIntelEvents

	return []keel.Pred{
		keel.When(len(req.Statuses) > 0, e.Status.In(req.Statuses...)),
		keel.When(req.OpenOnly, e.Status.In(openEventStatuses...)),
		keel.When(len(req.Severities) > 0, e.Severity.In(req.Severities...)),
		keel.When(len(req.Categories) > 0, e.Category.In(req.Categories...)),
		keel.When(req.SubjectType != "", e.SubjectType.Eq(req.SubjectType)),
		keel.When(req.SubjectID != "", e.SubjectID.Eq(req.SubjectID)),
		keel.When(!req.CarrierID.IsNil(), e.CarrierID.Eq(req.CarrierID)),
		keel.When(req.DetectedFrom > 0, e.DetectedAt.Gte(req.DetectedFrom)),
		keel.When(req.DetectedTo > 0, e.DetectedAt.Lte(req.DetectedTo)),
	}
}

func (r *eventRepository) ListConnection(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelEventsRequest,
) (*pagination.CursorListResult[*carrierintelevent.CarrierIntelEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carrierintelevent.CarrierIntelEvent], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: eventDetectedAtField, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				carrierinteldb.CarrierIntelEvents.Where(eventFilters(req)...),
				&listquery.Request[carrierintelevent.CarrierIntelEvent]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier intel events", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *eventRepository) GetByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*carrierintelevent.CarrierIntelEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintelevent.CarrierIntelEvent, error) {
			if len(ids) == 0 {
				return []*carrierintelevent.CarrierIntelEvent{}, nil
			}

			e := carrierinteldb.CarrierIntelEvents
			entities, err := e.Where(e.ID.In(ids...)).Limit(len(ids)).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get carrier intel events", zap.Error(err))
				return nil, fmt.Errorf("get carrier intel events: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *eventRepository) Update(
	ctx context.Context,
	entity *carrierintelevent.CarrierIntelEvent,
) (*carrierintelevent.CarrierIntelEvent, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelevent.CarrierIntelEvent, error) {
			err := carrierinteldb.CarrierIntelEvents.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update carrier intel event", zap.Error(err))
				return nil, saveError(
					eventErrors,
					err,
					entity.ID.String(),
					"update carrier intel event",
				)
			}

			return entity, nil
		},
	)
}

func (r *eventRepository) BulkAcknowledge(
	ctx context.Context,
	req *carrierintelports.BulkAcknowledgeCarrierIntelEventsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.EventIDs) == 0 {
			return 0, nil
		}

		e := carrierinteldb.CarrierIntelEvents
		affected, err := e.Where(
			e.ID.In(req.EventIDs...),
			e.Status.Eq(carrierintelevent.EventStatusOpen),
		).Update(
			ctx,
			r.db.Keel(ctx),
			e.Status.To(carrierintelevent.EventStatusAcknowledged),
			e.AcknowledgedByID.To(req.UserID),
			e.AcknowledgedAt.To(req.AcknowledgedAt),
			e.UpdatedAt.To(req.AcknowledgedAt),
		)
		if err != nil {
			r.l.Error("failed to acknowledge carrier intel events", zap.Error(err))
			return 0, fmt.Errorf("acknowledge carrier intel events: %w", err)
		}

		return int(affected), nil
	})
}

func (r *eventRepository) CountOpenByCarrierIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(carrierIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		e := carrierinteldb.CarrierIntelEvents
		rows, err := keel.Into[carrierEventCount](
			ctx,
			r.db.Keel(ctx),
			e.Where(e.Status.In(openEventStatuses...), e.CarrierID.In(carrierIDs...)).
				Group(e.CarrierID).
				Pick(e.CarrierID, keel.Count()),
		)
		if err != nil {
			r.l.Error("failed to count open carrier intel events by carrier", zap.Error(err))
			return nil, fmt.Errorf("count open carrier intel events by carrier: %w", err)
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.CarrierID] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *eventRepository) CountOpen(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*carrierintelports.CarrierIntelEventCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelports.CarrierIntelEventCounts, error) {
			e := carrierinteldb.CarrierIntelEvents
			rows, err := keel.Into[eventStatusSeverityCount](
				ctx,
				r.db.Keel(ctx),
				e.Where(e.Status.In(openEventStatuses...)).
					Group(e.Status, e.Severity).
					Pick(e.Status, e.Severity, keel.Count()),
			)
			if err != nil {
				r.l.Error("failed to count open carrier intel events", zap.Error(err))
				return nil, fmt.Errorf("count open carrier intel events: %w", err)
			}

			return eventCounts(rows), nil
		},
	)
}

func eventCounts(rows []eventStatusSeverityCount) *carrierintelports.CarrierIntelEventCounts {
	counts := &carrierintelports.CarrierIntelEventCounts{
		BySeverity: make(map[carrierintel.Severity]int, len(eventSeverityOrder)),
	}
	for _, severity := range eventSeverityOrder {
		counts.BySeverity[severity] = 0
	}

	for _, row := range rows {
		switch row.Status {
		case carrierintelevent.EventStatusOpen:
			counts.Open += int(row.Count)
		case carrierintelevent.EventStatusAcknowledged:
			counts.Acknowledged += int(row.Count)
		case carrierintelevent.EventStatusResolved, carrierintelevent.EventStatusDismissed:
		}
		if _, ranked := counts.BySeverity[row.Severity]; ranked {
			counts.BySeverity[row.Severity] += int(row.Count)
		}
	}

	return counts
}

func eventSeverityRank() keel.Int64Expr {
	severity := keel.CastText(carrierinteldb.CarrierIntelEvents.Severity)
	rank := keel.CaseInt64()
	for i, level := range eventSeverityOrder {
		rank = rank.When(severity.Eq(string(level)), keel.Int64(int64(i)))
	}

	return rank.Else(keel.Int64(int64(len(eventSeverityOrder))))
}

func (r *eventRepository) ListForDigest(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelDigestEventsRequest,
) ([]*carrierintelevent.CarrierIntelEvent, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintelevent.CarrierIntelEvent, error) {
			e := carrierinteldb.CarrierIntelEvents
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultDigestLimit),
				1,
				maxDigestLimit,
			)

			entities, err := e.Where(
				e.Status.Eq(carrierintelevent.EventStatusOpen),
				e.DetectedAt.Gte(req.Since),
				e.Action.In(digestEventActions...),
			).
				Order(keel.Asc(eventSeverityRank()), e.DetectedAt.Desc(), e.ID.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel digest events", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel digest events: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *eventRepository) MarkNotified(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
	notifiedAt int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(ids) == 0 {
			return nil
		}

		e := carrierinteldb.CarrierIntelEvents
		_, err := e.Where(e.ID.In(ids...)).Update(
			ctx,
			r.db.Keel(ctx),
			e.NotifiedAt.To(notifiedAt),
			e.Version.ToExpr(e.Version),
			e.UpdatedAt.ToExpr(e.UpdatedAt),
		)
		if err != nil {
			r.l.Error("failed to mark carrier intel events notified", zap.Error(err))
			return fmt.Errorf("mark carrier intel events notified: %w", err)
		}

		return nil
	})
}
