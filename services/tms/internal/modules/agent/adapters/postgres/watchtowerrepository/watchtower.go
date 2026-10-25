package watchtowerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
	maxDeleteBatch   = 1000
)

var itemErrors = dberror.Mapping{Entity: "Exception"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) watchtowerports.WatchtowerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.watchtower-repository"),
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	item *watchtower.Item,
) (*watchtower.Item, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*watchtower.Item, bool, error) {
		db := r.db.Keel(ctx)
		existing, err := lockBySource(ctx, db, item.SourceKind, item.SourceID)
		if err != nil {
			return nil, false, err
		}

		plan := planUpsert(existing, item)
		if plan.unchanged {
			existing.Unchanged = true
			return existing, false, nil
		}

		if existing != nil {
			err = r.refresh(ctx, db, existing, item, plan)
		} else {
			err = r.raise(ctx, db, item)
		}
		if err != nil {
			r.l.Error("failed to upsert watchtower item",
				zap.String("kind", string(item.SourceKind)),
				zap.String("source", item.SourceID),
				zap.Error(err))

			return nil, false, fmt.Errorf("upsert watchtower item: %w", err)
		}

		item.Reopened = plan.reopen
		switch {
		case item.Inserted:
			err = r.insertEvents(ctx, db, []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventCreated, pulid.Nil).
					WithChange("", item.Severity.String()),
			})
		case plan.reopen:
			err = r.insertEvents(ctx, db, []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventSourceReopened, pulid.Nil),
			})
		}
		if err != nil {
			return nil, false, err
		}

		return item, item.Inserted, nil
	})
}

func (r *repository) refresh(
	ctx context.Context,
	db *keel.DB,
	existing, item *watchtower.Item,
	plan upsertPlan,
) error {
	merged := applyUpsert(existing, item, plan)
	if err := watchtowerdb.Items.Save(ctx, db, merged); err != nil {
		return itemErrors.Map(err, merged.ID.String())
	}

	*item = *merged
	item.Inserted = false

	return nil
}

func (r *repository) raise(ctx context.Context, db *keel.DB, item *watchtower.Item) error {
	if item.Status == "" {
		item.Status = watchtower.StatusOpen
	}

	it := watchtowerdb.Items
	inserted, err := it.Upsert(ctx, db, []*watchtower.Item{item}, keel.OnConflict(
		it.OrganizationID,
		it.BusinessUnitID,
		it.SourceKind,
		it.SourceID,
	).Update(
		it.Title,
		it.Summary,
		it.SubjectType,
		it.SubjectID,
		it.EventKind,
		it.Path,
		it.OccurredAt,
		it.RecordKind,
		it.RecordLabel,
		it.Facts,
		it.Impact,
		it.Trail,
		it.Severity,
	))
	if err != nil {
		return err
	}
	if len(inserted) > 0 && inserted[0] {
		item.Inserted = true
		return nil
	}

	stored, err := it.Where(it.SourceKind.Eq(item.SourceKind), it.SourceID.Eq(item.SourceID)).
		One(ctx, db)
	if err != nil {
		return fmt.Errorf("read the raced watchtower item back: %w", err)
	}
	*item = *stored
	item.Inserted = false

	return nil
}

type upsertPlan struct {
	reopen       bool
	keepSeverity bool
	takeDeadline bool
	dropDeadline bool
	unchanged    bool
}

func planUpsert(existing, incoming *watchtower.Item) upsertPlan {
	if existing == nil {
		return upsertPlan{}
	}
	plan := upsertPlan{}
	if existing.ResolvedAt != nil &&
		(existing.ResolvedByID.IsNil() || existing.SourceClearedAt != nil) {
		plan.reopen = true
	}
	if !plan.reopen && existing.EscalationLevel > 0 &&
		existing.Severity.Rank() > incoming.Severity.Rank() {
		plan.keepSeverity = true
	}

	incomingDeadline := incoming.DueSource == watchtower.DueSourceSource && incoming.DueAt != nil
	switch {
	case incomingDeadline:
		plan.takeDeadline = plan.reopen || existing.DueSource != watchtower.DueSourcePerson
	case !plan.reopen && existing.DueSource == watchtower.DueSourceSource:
		plan.dropDeadline = true
	}

	plan.unchanged = !plan.reopen &&
		!plan.dropDeadline &&
		(!plan.takeDeadline || existing.SameDeadline(incoming)) &&
		(plan.keepSeverity || existing.Severity == incoming.Severity) &&
		existing.SameDescription(incoming)
	return plan
}

func applyUpsert(existing, incoming *watchtower.Item, plan upsertPlan) *watchtower.Item {
	merged := *existing
	merged.Title = incoming.Title
	merged.Summary = incoming.Summary
	merged.SubjectType = incoming.SubjectType
	merged.SubjectID = incoming.SubjectID
	merged.EventKind = incoming.EventKind
	merged.Path = incoming.Path
	merged.OccurredAt = incoming.OccurredAt
	merged.RecordKind = incoming.RecordKind
	merged.RecordLabel = incoming.RecordLabel
	merged.Facts = incoming.Facts
	merged.Impact = incoming.Impact
	merged.Trail = incoming.Trail
	if !plan.keepSeverity {
		merged.Severity = incoming.Severity
	}
	if plan.reopen {
		merged.ResolvedAt = nil
		merged.Status = watchtower.StatusOpen
		merged.ResolutionCode = ""
		merged.ResolutionNote = ""
		merged.ResolvedByID = pulid.Nil
		merged.SourceClearedAt = nil
		merged.AcknowledgedAt = nil
		merged.SnoozedUntil = nil
		merged.EscalationLevel = 0
		merged.LastEscalatedAt = nil
	}
	switch {
	case plan.takeDeadline:
		merged.DueAt = incoming.DueAt
		merged.DueLabel = incoming.DueLabel
		merged.DueSource = incoming.DueSource
	case plan.reopen || plan.dropDeadline:
		merged.DueAt = nil
		merged.DueLabel = ""
		merged.DueSource = ""
	}

	return &merged
}

func lockBySource(
	ctx context.Context,
	db *keel.DB,
	kind watchtower.SourceKind,
	sourceID string,
) (*watchtower.Item, error) {
	it := watchtowerdb.Items
	existing, err := it.Where(it.SourceKind.Eq(kind), it.SourceID.Eq(sourceID)).
		Lock(keel.ForUpdate).
		One(ctx, db)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // no item watches this source yet
		}
		return nil, fmt.Errorf("lock watchtower item: %w", err)
	}
	return existing, nil
}

func (r *repository) Resolve(
	ctx context.Context,
	req watchtowerports.ResolveWatchtowerItemRequest,
) (*watchtower.Item, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		db := r.db.Keel(ctx)
		existing, err := lockBySource(ctx, db, req.SourceKind, req.SourceID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, nil
		}

		it := watchtowerdb.Items
		if existing.ResolvedAt != nil {
			if existing.ResolvedByID.IsNil() || existing.SourceClearedAt != nil {
				return nil, nil
			}
			_, err = it.Where(it.ID.Eq(existing.ID)).Update(ctx, db,
				it.SourceClearedAt.To(req.ResolvedAt),
				it.Version.ToExpr(it.Version),
			)
			if err != nil {
				return nil, fmt.Errorf("mark watchtower source cleared: %w", err)
			}
			return nil, nil
		}

		rows, err := it.Where(it.ID.Eq(existing.ID), it.ResolvedAt.IsNull()).
			UpdateReturning(ctx, db, sourceResolveSets(req.ResolvedAt)...)
		if err != nil {
			return nil, fmt.Errorf("resolve watchtower item: %w", err)
		}
		if len(rows) == 0 {
			return nil, nil
		}

		item := rows[0]
		if err = r.insertEvents(ctx, db, []*watchtower.Event{
			watchtower.NewEvent(item, watchtower.EventSourceResolved, pulid.Nil),
		}); err != nil {
			return nil, err
		}

		return item, nil
	})
}

func sourceResolveSets(resolvedAt int64) []keel.Assignment {
	it := watchtowerdb.Items
	return []keel.Assignment{
		it.ResolvedAt.To(resolvedAt),
		it.Status.To(watchtower.StatusResolved),
		it.ResolutionCode.To(watchtower.ResolutionCleared),
		it.ResolvedByID.ToNull(),
		it.SnoozedUntil.ToNull(),
	}
}

func notOpenSources(req *watchtowerports.ResolveMissingWatchtowerItemsRequest) keel.Pred {
	it := watchtowerdb.Items
	return keel.And(
		it.SourceKind.Eq(req.SourceKind),
		keel.When(len(req.OpenSourceIDs) > 0, it.SourceID.NotIn(req.OpenSourceIDs...)),
	)
}

func (r *repository) ResolveMissing(
	ctx context.Context,
	req *watchtowerports.ResolveMissingWatchtowerItemsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		db := r.db.Keel(ctx)
		it := watchtowerdb.Items

		rows, err := it.Pick(it.ID).
			Where(notOpenSources(req), it.ResolvedAt.IsNull()).
			UpdateReturning(ctx, db, sourceResolveSets(req.ResolvedAt)...)
		if err != nil {
			return 0, fmt.Errorf("resolve missing watchtower items: %w", err)
		}

		_, err = it.Where(
			notOpenSources(req),
			it.ResolvedAt.IsNotNull(),
			it.ResolvedByID.IsNotNull(),
			it.SourceClearedAt.IsNull(),
		).Update(ctx, db, it.SourceClearedAt.To(req.ResolvedAt), it.Version.ToExpr(it.Version))
		if err != nil {
			return 0, fmt.Errorf("mark watchtower sources cleared: %w", err)
		}

		if len(rows) == 0 {
			return 0, nil
		}
		events := make([]*watchtower.Event, 0, len(rows))
		for _, row := range rows {
			events = append(
				events,
				watchtower.NewEvent(row, watchtower.EventSourceResolved, pulid.Nil),
			)
		}
		if err = r.insertEvents(ctx, db, events); err != nil {
			return 0, err
		}

		return len(rows), nil
	})
}

func workflowSets(item *watchtower.Item) []keel.Assignment {
	it := watchtowerdb.Items
	return []keel.Assignment{
		it.Status.To(item.Status),
		it.AssigneeID.To(item.AssigneeID),
		listquery.OptionalInt64(it.DueAt, item.DueAt),
		listquery.OptionalInt64(it.SnoozedUntil, item.SnoozedUntil),
		listquery.OptionalInt64(it.AcknowledgedAt, item.AcknowledgedAt),
		it.ResolutionCode.To(item.ResolutionCode),
		it.ResolutionNote.To(item.ResolutionNote),
		it.ResolvedByID.To(item.ResolvedByID),
		listquery.OptionalInt64(it.ResolvedAt, item.ResolvedAt),
		listquery.OptionalInt64(it.SourceClearedAt, item.SourceClearedAt),
		it.Severity.To(item.Severity),
		it.EscalationLevel.To(item.EscalationLevel),
		listquery.OptionalInt64(it.LastEscalatedAt, item.LastEscalatedAt),
		it.DueLabel.To(item.DueLabel),
		it.DueSource.To(item.DueSource),
		it.AgentDefinitionID.To(item.AgentDefinitionID),
		it.AgentRunID.To(item.AgentRunID),
		listquery.OptionalInt64(it.AgentHandedAt, item.AgentHandedAt),
		it.AgentHandedByID.To(item.AgentHandedByID),
	}
}

func (r *repository) UpdateWorkflow(
	ctx context.Context,
	req *watchtowerports.UpdateWatchtowerWorkflowRequest,
) (*watchtower.Item, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		db := r.db.Keel(ctx)
		it := watchtowerdb.Items
		item := req.Item

		rows, err := it.Where(it.ID.Eq(item.ID), it.Version.Eq(req.ExpectedVersion)).
			UpdateReturning(ctx, db, workflowSets(item)...)
		if err != nil {
			return nil, fmt.Errorf("update watchtower item workflow: %w", err)
		}
		if len(rows) == 0 {
			return nil, dberror.CreateVersionMismatchError("Exception", item.ID.String())
		}
		*item = *rows[0]
		if err = r.insertEvents(ctx, db, req.Events); err != nil {
			return nil, err
		}

		return item, nil
	})
}

func (r *repository) InsertEvents(ctx context.Context, events []*watchtower.Event) error {
	if len(events) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.insertEvents(ctx, r.db.Keel(ctx), events)
	})
}

func (r *repository) insertEvents(
	ctx context.Context,
	db *keel.DB,
	events []*watchtower.Event,
) error {
	if len(events) == 0 {
		return nil
	}
	if err := watchtowerdb.Events.CreateMany(ctx, db, events); err != nil {
		return fmt.Errorf("insert watchtower item events: %w", err)
	}
	return nil
}

func (r *repository) ListEvents(
	ctx context.Context,
	req watchtowerports.ListWatchtowerEventsRequest,
) ([]*watchtower.Event, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Event, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = watchtower.DefaultEventPageSize
		}
		limit = min(limit, watchtower.MaxEventPageSize)

		ev := watchtowerdb.Events
		events, err := ev.Where(ev.ItemID.Eq(req.ItemID)).
			Order(ev.CreatedAt.Asc(), ev.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list watchtower item events: %w", err)
		}
		return events, nil
	})
}

func overduePred(req *watchtowerports.ListAttentionWatchtowerItemsRequest) keel.Pred {
	it := watchtowerdb.Items
	return keel.And(
		it.ResolvedAt.IsNull(),
		it.Status.Ne(watchtower.StatusSnoozed),
		it.DueAt.IsNotNull(),
		it.DueAt.Lt(req.Now),
		it.EscalationLevel.Lt(req.BelowEscalationCap),
		keel.Or(
			it.DueSource.IsDistinctFrom(watchtower.DueSourceSource),
			it.LastEscalatedAt.IsNull(),
			it.LastEscalatedAt.Lte(req.Now-req.EscalateEvery),
		),
	)
}

func (r *repository) ListOverdue(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
) ([]*watchtower.Item, error) {
	it := watchtowerdb.Items
	return r.listAttention(ctx, req, overduePred(&req), it.DueAt.Asc())
}

func (r *repository) ListSnoozeExpired(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
) ([]*watchtower.Item, error) {
	it := watchtowerdb.Items
	return r.listAttention(ctx, req, keel.And(
		it.ResolvedAt.IsNull(),
		it.Status.Eq(watchtower.StatusSnoozed),
		it.SnoozedUntil.Lte(req.Now),
	), it.SnoozedUntil.Asc())
}

func (r *repository) listAttention(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
	filter keel.Pred,
	order keel.Order,
) ([]*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Item, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxListLimit {
			limit = maxListLimit
		}

		it := watchtowerdb.Items
		items, err := it.Where(filter).
			Order(order, it.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list watchtower items needing attention: %w", err)
		}
		return items, nil
	})
}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func tenantsNeedingAttentionQuery(now int64) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items
	return it.AllTenants().
		Pick(it.OrganizationID, it.BusinessUnitID).
		Where(
			it.ResolvedAt.IsNull(),
			keel.Or(
				keel.And(
					it.Status.Ne(watchtower.StatusSnoozed),
					it.DueAt.Lt(now),
					it.EscalationLevel.Lt(dispatchcontrol.MaxExceptionEscalations),
				),
				keel.And(
					it.Status.Eq(watchtower.StatusSnoozed),
					it.SnoozedUntil.Lte(now),
				),
			),
		).
		Group(it.OrganizationID, it.BusinessUnitID)
}

func (r *repository) ListTenantsNeedingAttention(
	ctx context.Context,
	now int64,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with overdue or waking exceptions")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), tenantsNeedingAttentionQuery(now))
		if err != nil {
			return nil, fmt.Errorf("list tenants needing exception attention: %w", err)
		}
		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}
		return tenants, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
) (*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		item, err := watchtowerdb.Items.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("Watchtower item not found")
			}

			return nil, fmt.Errorf("get watchtower item: %w", err)
		}

		return item, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req watchtowerports.ListWatchtowerItemsQuery,
) ([]*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Item, error) {
		items, err := listQuery(&req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list watchtower items", zap.Error(err))

			return nil, fmt.Errorf("list watchtower items: %w", err)
		}

		return items, nil
	})
}

func listQuery(req *watchtowerports.ListWatchtowerItemsQuery) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items

	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var assignee keel.Pred
	switch {
	case req.Unassigned && len(req.AssigneeIDs) > 0:
		assignee = keel.Or(it.AssigneeID.IsNull(), it.AssigneeID.In(req.AssigneeIDs...))
	case req.Unassigned:
		assignee = it.AssigneeID.IsNull()
	case len(req.AssigneeIDs) > 0:
		assignee = it.AssigneeID.In(req.AssigneeIDs...)
	}

	var overdue keel.Pred
	if req.OverdueAt > 0 {
		overdue = keel.And(
			it.ResolvedAt.IsNull(),
			it.Status.Ne(watchtower.StatusSnoozed),
			it.DueAt.Lt(req.OverdueAt),
		)
	}

	dueSort := dueSortExpr()
	var before keel.Pred
	switch {
	case req.BeforeID.IsNil():
	case req.OrderByDue:
		before = keel.DynRow(dueSort, it.ID).Gt(keel.DynRow(req.BeforeDueAt, req.BeforeID.String()))
	case req.BeforeOccurredAt > 0:
		before = keel.DynRow(it.OccurredAt, it.ID).
			Lt(keel.DynRow(req.BeforeOccurredAt, req.BeforeID.String()))
	}

	q := it.Where(
		keel.When(len(req.Kinds) > 0, it.SourceKind.In(req.Kinds...)),
		keel.When(len(req.Severities) > 0, it.Severity.In(req.Severities...)),
		keel.When(req.UnresolvedOnly, it.ResolvedAt.IsNull()),
		keel.When(req.Since > 0, it.OccurredAt.Gt(req.Since)),
		keel.When(len(req.Statuses) > 0, it.Status.In(req.Statuses...)),
		assignee,
		overdue,
		keel.When(req.SubjectID.IsNotNil(), it.SubjectID.Eq(req.SubjectID)),
		before,
	)
	if req.OrderByDue {
		q = q.Order(keel.Asc(dueSort), it.ID.Asc())
	} else {
		q = q.Order(it.OccurredAt.Desc(), it.ID.Desc())
	}

	return q.Limit(limit)
}

func dueSortExpr() keel.Expr {
	return keel.DynOf(watchtowerdb.Items.DueAt).Coalesce(keel.DynConst(watchtower.NoDueSortKey))
}

type countTotals struct {
	Unresolved     int64
	Critical       int64
	Active         int64
	ActiveCritical int64
	UnseenCritical int64
	Unseen         int64
	Overdue        int64
	Unassigned     int64
	Mine           int64
	MineOverdue    int64
}

func openItems(req *watchtowerports.CountWatchtowerItemsRequest) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items
	return it.Where(
		it.ResolvedAt.IsNull(),
		keel.When(len(req.Kinds) > 0, it.SourceKind.In(req.Kinds...)),
	)
}

func countTotalsQuery(
	req *watchtowerports.CountWatchtowerItemsRequest,
) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items
	rd := watchtowerdb.Reads
	critical := it.Severity.Eq(watchtower.SeverityCritical)
	active := it.Status.Ne(watchtower.StatusSnoozed)
	overdue := it.DueAt.Lt(req.Now)
	mine := keel.Or()
	if req.AssigneeID.IsNotNil() {
		mine = it.AssigneeID.Eq(req.AssigneeID)
	}
	unseen := keel.And(
		it.OccurredAt.Gt(req.SeenAt),
		keel.NotExists(rd.Where(rd.ItemID.EqCol(it.ID), rd.UserID.Eq(req.ReaderID))),
	)

	return openItems(req).Pick(
		keel.Count(),
		keel.Count().Filter(critical),
		keel.Count().Filter(active),
		keel.Count().Filter(keel.And(active, critical)),
		keel.Count().Filter(keel.And(critical, unseen)),
		keel.Count().Filter(unseen),
		keel.Count().Filter(keel.And(active, overdue)),
		keel.Count().Filter(it.AssigneeID.IsNull()),
		keel.Count().Filter(mine),
		keel.Count().Filter(keel.And(mine, active, overdue)),
	)
}

func countByKindQuery(
	req *watchtowerports.CountWatchtowerItemsRequest,
) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items
	return openItems(req).
		Where(it.Status.Ne(watchtower.StatusSnoozed)).
		Group(it.SourceKind).
		Pick(it.SourceKind, keel.Count())
}

func (r *repository) Counts(
	ctx context.Context,
	req watchtowerports.CountWatchtowerItemsRequest,
) (*watchtowerports.WatchtowerCountsRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*watchtowerports.WatchtowerCountsRow, error) {
			var (
				totals countTotals
				byKind []watchtowerports.WatchtowerKindCount
			)
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoValue(countTotalsQuery(&req), &totals),
				keel.IntoSlice(countByKindQuery(&req), &byKind),
			); err != nil {
				return nil, fmt.Errorf("count watchtower items: %w", err)
			}
			if byKind == nil {
				byKind = make([]watchtowerports.WatchtowerKindCount, 0)
			}

			return &watchtowerports.WatchtowerCountsRow{
				Unresolved:     int(totals.Unresolved),
				Critical:       int(totals.Critical),
				Active:         int(totals.Active),
				ActiveCritical: int(totals.ActiveCritical),
				UnseenCritical: int(totals.UnseenCritical),
				Unseen:         int(totals.Unseen),
				Overdue:        int(totals.Overdue),
				Unassigned:     int(totals.Unassigned),
				Mine:           int(totals.Mine),
				MineOverdue:    int(totals.MineOverdue),
				ByKind:         byKind,
			}, nil
		},
	)
}

func (r *repository) GetCursor(
	ctx context.Context,
	req watchtowerports.GetWatchtowerCursorRequest,
) (*watchtower.Cursor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*watchtower.Cursor, error) {
		cursor, err := readCursor(ctx, r.db.Keel(ctx), req.UserID)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return &watchtower.Cursor{
					UserID:         req.UserID,
					OrganizationID: req.TenantInfo.OrgID,
					BusinessUnitID: req.TenantInfo.BuID,
				}, nil
			}

			return nil, fmt.Errorf("get watchtower cursor: %w", err)
		}

		return cursor, nil
	})
}

func readCursor(ctx context.Context, db *keel.DB, userID pulid.ID) (*watchtower.Cursor, error) {
	c := watchtowerdb.Cursors
	return c.Where(c.UserID.Eq(userID)).One(ctx, db)
}

func (r *repository) SetCursor(
	ctx context.Context,
	cursor *watchtower.Cursor,
) (*watchtower.Cursor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Cursor, error) {
		db := r.db.Keel(ctx)
		c := watchtowerdb.Cursors
		if _, err := c.Upsert(ctx, db, []*watchtower.Cursor{cursor},
			keel.OnConflict(c.UserID, c.BusinessUnitID, c.OrganizationID).Set(
				c.SeenAt.ToExpr(c.SeenAt.Greatest(keel.Int64Of(c.SeenAt.Excluded()))),
			),
		); err != nil {
			return nil, fmt.Errorf("set watchtower cursor: %w", err)
		}

		stored, err := readCursor(ctx, db, cursor.UserID)
		if err != nil {
			return nil, fmt.Errorf("read watchtower cursor back: %w", err)
		}

		return stored, nil
	})
}

func (r *repository) MarkRead(ctx context.Context, read *watchtower.Read) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rd := watchtowerdb.Reads
		if _, err := rd.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*watchtower.Read{read},
			rd.ItemID,
			rd.UserID,
			rd.BusinessUnitID,
			rd.OrganizationID,
		); err != nil {
			return fmt.Errorf("mark watchtower item read: %w", err)
		}
		return nil
	})
}

func (r *repository) ListReadItemIDs(
	ctx context.Context,
	req watchtowerports.ListWatchtowerReadsRequest,
) (map[pulid.ID]struct{}, error) {
	if len(req.ItemIDs) == 0 || req.UserID.IsNil() {
		return map[pulid.ID]struct{}{}, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]struct{}, error) {
		rd := watchtowerdb.Reads
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), rd.Pick(rd.ItemID).
			Where(rd.UserID.Eq(req.UserID), rd.ItemID.In(req.ItemIDs...)).
			Limit(len(req.ItemIDs)))
		if err != nil {
			return nil, fmt.Errorf("list watchtower item reads: %w", err)
		}
		read := make(map[pulid.ID]struct{}, len(ids))
		for _, id := range ids {
			read[id] = struct{}{}
		}
		return read, nil
	})
}

func clearedQuery(
	req *watchtowerports.CountClearedWatchtowerItemsRequest,
) *keel.Query[*watchtower.Item] {
	it := watchtowerdb.Items
	ev := watchtowerdb.Events
	snoozedByReader := ev.Where(
		ev.ItemID.EqCol(it.ID),
		ev.ActorID.Eq(req.UserID),
		ev.Kind.Eq(watchtower.EventSnoozed),
		ev.CreatedAt.Gte(req.Since),
	)

	return it.Where(
		it.SourceKind.In(req.Kinds...),
		keel.Or(
			keel.And(it.ResolvedByID.Eq(req.UserID), it.ResolvedAt.Gte(req.Since)),
			keel.And(it.Status.Eq(watchtower.StatusSnoozed), keel.Exists(snoozedByReader)),
		),
	)
}

func (r *repository) CountCleared(
	ctx context.Context,
	req watchtowerports.CountClearedWatchtowerItemsRequest,
) (int, error) {
	if req.UserID.IsNil() || len(req.Kinds) == 0 {
		return 0, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := clearedQuery(&req).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count cleared watchtower items: %w", err)
		}
		return int(count), nil
	})
}

func resolvedBefore(before int64, limit int) keel.Pred {
	it := watchtowerdb.Items
	stale := it.As("stale")
	return it.ID.InQuery(stale.AllTenants().
		Pick(stale.ID).
		Where(
			stale.ResolvedAt.IsNotNull(),
			stale.ResolvedAt.Lt(before),
			keel.Or(stale.ResolvedByID.IsNull(), stale.SourceClearedAt.IsNotNull()),
		).
		Limit(limit))
}

func (r *repository) DeleteResolvedBefore(
	ctx context.Context,
	req watchtowerports.DeleteResolvedWatchtowerItemsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete resolved watchtower items past retention across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxDeleteBatch {
			limit = maxDeleteBatch
		}

		it := watchtowerdb.Items
		n, err := it.AllTenants().
			Where(resolvedBefore(req.Before, limit)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("delete resolved watchtower items: %w", err)
		}

		return int(n), nil
	})
}
