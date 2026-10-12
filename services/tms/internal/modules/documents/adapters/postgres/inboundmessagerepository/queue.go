//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

func NewQueueRepository(p Params) inboundmessageports.TriageQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-queue-repository"),
	}
}

func queueStateColumns() []string {
	cols := inboundmessagecol.InboundMessageColumns

	return []string{
		cols.ID.Bare(),
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.MailboxID.Bare(),
		cols.Status.Bare(),
		cols.Classification.Bare(),
		cols.ReceivedAt.Bare(),
		cols.FromAddress.Bare(),
		cols.FromName.Bare(),
		cols.Subject.Bare(),
		cols.SenderDomain.Bare(),
		cols.ThreadKey.Bare(),
		cols.OwnerID.Bare(),
		cols.DueAt.Bare(),
		cols.Parked.Bare(),
		cols.SnoozedUntil.Bare(),
		cols.SnoozeEvent.Bare(),
		cols.SnoozeWatchID.Bare(),
		cols.SnoozeDocumentTypeID.Bare(),
		cols.AwaitingUntil.Bare(),
		cols.MatchedShipmentID.Bare(),
		cols.MatchedCustomerID.Bare(),
		cols.MatchedCarrierID.Bare(),
		cols.MatchReason.Bare(),
		cols.FailureText.Bare(),
		cols.ReviewNote.Bare(),
		cols.NovaRead.Bare(),
	}
}

var stateColumns = queueStateColumns()

func returningState() string {
	return strings.Join(stateColumns, ", ")
}

var stateReturning = returningState()

func (r *repository) QueueStates(
	ctx context.Context,
	req inboundmessageports.QueueStatesRequest,
) ([]*inboundmessageports.QueueState, error) {
	if req.ForUpdate {
		return dbtx.Write(
			ctx,
			r.db,
			func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
				return r.queueStates(ctx, req)
			},
		)
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			return r.queueStates(ctx, req)
		},
	)
}

func (r *repository) queueStates(
	ctx context.Context,
	req inboundmessageports.QueueStatesRequest,
) ([]*inboundmessageports.QueueState, error) {
	states := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
	if len(req.IDs) == 0 {
		return states, nil
	}
	if err := queueStatesQuery(r.db.DBForContext(ctx), req).Scan(ctx, &states); err != nil {
		return nil, err
	}

	return states, nil
}

func queueStatesQuery(db bun.IDB, req inboundmessageports.QueueStatesRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	query := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		Column(stateColumns...).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ID.In(), bun.List(req.IDs)).
		Order(cols.ID.OrderAsc())
	if req.ForUpdate {
		query = query.For("UPDATE")
	}

	return query
}

func (r *repository) SelectIDs(
	ctx context.Context,
	req inboundmessageports.SelectInboundMessageIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids := make([]pulid.ID, 0, req.Limit)
		dba := r.db.DBForContext(ctx)
		if err := selectIDsQuery(dba, req).Scan(ctx, &ids); err != nil {
			return nil, err
		}

		return ids, nil
	})
}

func selectIDsQuery(
	db bun.IDB,
	req inboundmessageports.SelectInboundMessageIDsRequest,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	folder := req.Folder
	query := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(cols.ID.Qualified())
	query = querybuilder.ApplyFiltersWithoutSort(
		query,
		inboundmessagecol.InboundMessageTable.Alias,
		folder.Filter,
		(*inboundmessage.InboundMessage)(nil),
	)
	query = narrow(db, query, folder)
	for _, term := range folderOrder(folder.Sort) {
		query = query.Order(term)
	}

	return query.Limit(int64(req.Limit))
}

func folderOrder(sort inboundmessageports.Sort) []string {
	cols := inboundmessagecol.InboundMessageColumns
	if sort == inboundmessageports.SortDue {
		return []string{
			cols.DueAt.OrderAsc() + " NULLS LAST",
			cols.ReceivedAt.OrderDesc(),
			cols.ID.OrderDesc(),
		}
	}

	return []string{cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()}
}

func (r *repository) UpdateQueueState(
	ctx context.Context,
	req inboundmessageports.UpdateQueueStateRequest,
) ([]*inboundmessageports.QueueState, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			states := make([]*inboundmessageports.QueueState, 0, len(req.IDs))
			if len(req.IDs) == 0 {
				return states, nil
			}
			if err := updateQueueStateQuery(
				r.db.DBForContext(ctx),
				req,
			).Scan(ctx, &states); err != nil {
				return nil, err
			}

			return states, nil
		},
	)
}

func updateQueueStateQuery(
	db bun.IDB,
	req inboundmessageports.UpdateQueueStateRequest,
) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	query := db.NewUpdate().
		Model((*inboundmessage.InboundMessage)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.IDs))
		}).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), req.Now)

	query = applyPatch(query, req.Patch, req.Now)

	return query.Returning(stateReturning)
}

func applyPatch(
	query *bun.UpdateQuery,
	patch inboundmessageports.QueuePatch,
	now int64,
) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	if patch.Status != "" {
		query = query.Set(cols.Status.Set(), patch.Status)
	}
	if patch.Review != nil {
		query = applyReviewPatch(query, patch.Review, now)
	}
	if patch.Owner != nil {
		query = query.Set(cols.OwnerID.Set(), *patch.Owner)
	}
	switch {
	case patch.Park != nil:
		query = applyParkPatch(query, patch.Park, now)
	case patch.Unpark:
		query = clearParking(query)
	}
	query = applyDuePatch(query, patch)
	if patch.ClearHandled {
		query = query.Set(cols.HandledByRuleID.SetNull()).Set(cols.HandledAction.SetNull())
	}
	if patch.HandOff != nil {
		query = query.Set(
			cols.ActionState.SetExpr(
				"{} || jsonb_build_object('handedOffTo', ?::text, 'handedOffAt', ?::bigint)",
			),
			patch.HandOff.To.String(),
			now,
		)
	}
	if patch.ClearShipment {
		query = query.Set(cols.MatchedShipmentID.SetNull())
	}
	if patch.ClearCustomer {
		query = query.Set(cols.MatchedCustomerID.SetNull())
	}
	if patch.ClearCarrier {
		query = query.Set(cols.MatchedCarrierID.SetNull())
	}
	if patch.ClearReason {
		query = query.Set(cols.MatchReason.SetNull())
	}
	if patch.NovaUnread {
		query = query.Set(cols.NovaRead.Set(), false)
	}

	return query
}

func applyReviewPatch(
	query *bun.UpdateQuery,
	review *inboundmessageports.ReviewPatch,
	now int64,
) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	if review.Clear {
		return query.Set(cols.ReviewedBy.SetNull()).Set(cols.ReviewedAt.SetNull())
	}

	return query.
		Set(cols.ReviewedBy.Set(), review.ReviewerID).
		Set(cols.ReviewedAt.Set(), now).
		Set(cols.ReviewNote.SetExpr("COALESCE(NULLIF(?, ''), {})"), review.Note)
}

func clearParking(query *bun.UpdateQuery) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return query.
		Set(cols.Parked.SetNull()).
		Set(cols.SnoozedUntil.SetNull()).
		Set(cols.SnoozeEvent.SetNull()).
		Set(cols.SnoozeWatchID.SetNull()).
		Set(cols.SnoozeDocumentTypeID.SetNull()).
		Set(cols.SnoozedAt.SetNull()).
		Set(cols.SnoozedBy.SetNull()).
		Set(cols.AwaitingSince.SetNull()).
		Set(cols.AwaitingUntil.SetNull())
}

func applyParkPatch(
	query *bun.UpdateQuery,
	park *inboundmessageports.ParkPatch,
	now int64,
) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	query = query.Set(cols.Parked.Set(), park.Parking)
	if park.Parking == inboundmessage.ParkingAwaitingReply {
		return query.
			Set(cols.AwaitingSince.Set(), now).
			Set(cols.AwaitingUntil.Set(), park.AwaitingUntil).
			Set(cols.SnoozedUntil.SetNull()).
			Set(cols.SnoozeEvent.SetNull()).
			Set(cols.SnoozeWatchID.SetNull()).
			Set(cols.SnoozeDocumentTypeID.SetNull()).
			Set(cols.SnoozedAt.SetNull()).
			Set(cols.SnoozedBy.SetNull())
	}

	query = query.
		Set(cols.SnoozedUntil.Set(), intutils.NilIfZero(park.SnoozedUntil)).
		Set(cols.SnoozeEvent.Set(), stringutils.NilIfEmpty(park.SnoozeEvent)).
		Set(cols.SnoozeDocumentTypeID.Set(), park.DocumentTypeID).
		Set(cols.SnoozedAt.Set(), now).
		Set(cols.SnoozedBy.Set(), park.By).
		Set(cols.AwaitingSince.SetNull()).
		Set(cols.AwaitingUntil.SetNull())
	if park.WatchShipment {
		return query.Set(cols.SnoozeWatchID.SetExpr(cols.MatchedShipmentID.Bare()))
	}

	return query.Set(cols.SnoozeWatchID.SetNull())
}

func applyDuePatch(
	query *bun.UpdateQuery,
	patch inboundmessageports.QueuePatch,
) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	switch patch.Due {
	case inboundmessageports.DueClear:
		return query.Set(cols.DueAt.SetNull())
	case inboundmessageports.DueFromTargets:
		expr, args := dueFromTargetsExpr(patch.Targets)
		return query.Set(cols.DueAt.SetExpr(expr), args...)
	case inboundmessageports.DueKeep:
		return query
	default:
		return query
	}
}

func dueFromTargetsExpr(targets triage.Targets) (string, []any) {
	cols := inboundmessagecol.InboundMessageColumns
	kinds := inboundmessage.AllClassifications()
	var b strings.Builder
	args := make([]any, 0, len(kinds)*2)
	b.WriteString(cols.ReceivedAt.Bare())
	b.WriteString(" + 60 * CASE COALESCE(")
	b.WriteString(cols.Classification.Bare())
	b.WriteString(", ?)")
	args = append(args, inboundmessage.ClassificationOther)
	for _, kind := range kinds {
		b.WriteString(" WHEN ? THEN ?::bigint")
		args = append(args, kind, targets.For(kind))
	}
	b.WriteString(" ELSE ?::bigint END")
	args = append(args, targets.For(inboundmessage.ClassificationOther))

	return b.String(), args
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req inboundmessageports.ListInboundMessagesByIDsRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			entities := make([]*inboundmessage.InboundMessage, 0, len(req.IDs))
			if len(req.IDs) == 0 {
				return entities, nil
			}
			cols := inboundmessagecol.InboundMessageColumns
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Relation(inboundmessagecol.InboundMessageRelations.Mailbox).
				Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
				Where(cols.ID.In(), bun.List(req.IDs)).
				Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) ThreadKeysByMessageKeys(
	ctx context.Context,
	req inboundmessageports.ThreadKeysRequest,
) (map[string]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]string, error) {
		found := make(map[string]string, len(req.MessageKeys))
		if len(req.MessageKeys) == 0 {
			return found, nil
		}

		rows := make([]struct {
			MessageKey string `bun:"message_key"`
			ThreadKey  string `bun:"thread_key"`
		}, 0, len(req.MessageKeys))
		cols := inboundmessagecol.InboundMessageColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(cols.MessageKey.Qualified()).
			ColumnExpr(cols.ThreadKey.Qualified()).
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.MessageKey.In(), bun.List(req.MessageKeys)).
			Order(cols.ReceivedAt.OrderAsc()).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, ok := found[row.MessageKey]; !ok {
				found[row.MessageKey] = row.ThreadKey
			}
		}

		return found, nil
	})
}

func (r *repository) CountByThreadKeys(
	ctx context.Context,
	req inboundmessageports.ThreadCountsRequest,
) (map[string]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]int, error) {
		counts := make(map[string]int, len(req.ThreadKeys))
		if len(req.ThreadKeys) == 0 {
			return counts, nil
		}

		rows := make([]struct {
			ThreadKey string `bun:"thread_key"`
			Count     int    `bun:"count"`
		}, 0, len(req.ThreadKeys))
		cols := inboundmessagecol.InboundMessageColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(cols.ThreadKey.Qualified()).
			ColumnExpr(buncolgen.Count("count")).
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.ThreadKey.In(), bun.List(req.ThreadKeys)).
			GroupExpr(cols.ThreadKey.Qualified()).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.ThreadKey] = row.Count
		}

		return counts, nil
	})
}

func (r *repository) ListThread(
	ctx context.Context,
	req inboundmessageports.ListThreadRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			entities := make([]*inboundmessage.InboundMessage, 0, req.Limit)
			if err := listThreadQuery(
				r.db.DBForContext(ctx),
				req,
				&entities,
			).Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func listThreadQuery(
	db bun.IDB,
	req inboundmessageports.ListThreadRequest,
	entities *[]*inboundmessage.InboundMessage,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	newest := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ThreadKey.Eq(), req.ThreadKey).
		Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(int64(req.Limit))

	return db.NewSelect().
		Model(entities).
		Relation(inboundmessagecol.InboundMessageRelations.Mailbox).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ID.In(), newest).
		Order(cols.ReceivedAt.OrderAsc(), cols.ID.OrderAsc())
}

func (r *repository) AwaitingOnThread(
	ctx context.Context,
	req inboundmessageports.AwaitingOnThreadRequest,
) ([]*inboundmessageports.QueueState, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			states := make([]*inboundmessageports.QueueState, 0, 2)
			cols := inboundmessagecol.InboundMessageColumns
			query := r.db.DBForContext(ctx).
				NewSelect().
				Model((*inboundmessage.InboundMessage)(nil)).
				Column(stateColumns...).
				Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
				Where(cols.ThreadKey.Eq(), req.ThreadKey).
				Where(cols.Parked.Eq(), inboundmessage.ParkingAwaitingReply).
				Limit(inboundmessageports.MaxThreadMessages)
			if req.ExcludeID.IsNotNil() {
				query = query.Where(cols.ID.NotEq(), req.ExcludeID)
			}
			if err := query.Scan(ctx, &states); err != nil {
				return nil, err
			}

			return states, nil
		},
	)
}

func (r *repository) WakeByEvent(
	ctx context.Context,
	req inboundmessageports.WakeByEventRequest,
) ([]*inboundmessageports.QueueState, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			states := make([]*inboundmessageports.QueueState, 0, 2)
			if err := wakeByEventQuery(r.db.DBForContext(ctx), req).Scan(ctx, &states); err != nil {
				return nil, err
			}

			return states, nil
		},
	)
}

func wakeByEventQuery(db bun.IDB, req inboundmessageports.WakeByEventRequest) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns
	query := db.NewUpdate().
		Model((*inboundmessage.InboundMessage)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.SnoozeEvent.Eq(), req.Event).
				Where(cols.SnoozeWatchID.Eq(), req.WatchID).
				Where(cols.Parked.Eq(), inboundmessage.ParkingSnoozed)
			if req.DocumentTypeID.IsNotNil() {
				uq = uq.Where(cols.SnoozeDocumentTypeID.Eq(), req.DocumentTypeID)
			}

			return uq
		}).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), req.Now)

	return clearParking(query).Returning(stateReturning)
}

func (r *repository) ListOverdueParked(
	ctx context.Context,
	req inboundmessageports.ListOverdueParkedRequest,
) ([]inboundmessageports.ParkedRef, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"find snoozed and waiting inbound messages in every tenant whose wake-up timer was lost",
	)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]inboundmessageports.ParkedRef, error) {
		refs := make([]inboundmessageports.ParkedRef, 0, req.Limit)
		if err := overdueParkedQuery(r.db.DBForContext(ctx), req).Scan(ctx, &refs); err != nil {
			return nil, err
		}

		return refs, nil
	})
}

func overdueParkedQuery(
	db bun.IDB,
	req inboundmessageports.ListOverdueParkedRequest,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	refColumns := []string{cols.ID.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()}

	snoozed := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		Column(refColumns...).
		Where(cols.Parked.Eq(), inboundmessage.ParkingSnoozed).
		Where(cols.SnoozedUntil.IsNotNull()).
		Where(cols.SnoozedUntil.Lte(), req.Now).
		Limit(int64(req.Limit))
	awaiting := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		Column(refColumns...).
		Where(cols.Parked.Eq(), inboundmessage.ParkingAwaitingReply).
		Where(cols.AwaitingUntil.Lte(), req.Now).
		Limit(int64(req.Limit))

	return db.NewSelect().
		TableExpr("((?) UNION ALL (?)) AS overdue", snoozed, awaiting).
		ColumnExpr("*").
		Limit(int64(req.Limit))
}

func (r *repository) RecomputeDue(
	ctx context.Context,
	req inboundmessageports.RecomputeDueRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		result, err := recomputeDueQuery(r.db.DBForContext(ctx), req).Exec(ctx)
		if err != nil {
			return 0, err
		}

		return result.RowsAffected()
	})
}

func recomputeDueQuery(db bun.IDB, req inboundmessageports.RecomputeDueRequest) *bun.UpdateQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return db.NewUpdate().
		Model((*inboundmessage.InboundMessage)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = inboundmessagecol.InboundMessageScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.Status.In(), bun.List(needsReviewStatuses))
			if req.Classification == inboundmessage.ClassificationOther {
				return uq.WhereGroup(" AND ", func(g *bun.UpdateQuery) *bun.UpdateQuery {
					return g.Where(cols.Classification.Eq(), req.Classification).
						WhereOr(cols.Classification.IsNull())
				})
			}

			return uq.Where(cols.Classification.Eq(), req.Classification)
		}).
		Set(cols.DueAt.SetExpr(cols.ReceivedAt.Bare()+" + ?::bigint"), int64(req.Minutes)*60)
}

func (r *repository) UnreadInReview(
	ctx context.Context,
	req inboundmessageports.UnreadInReviewRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids := make([]pulid.ID, 0, req.Limit)
		cols := inboundmessagecol.InboundMessageColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(cols.ID.Qualified()).
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.Status.Eq(), inboundmessage.StatusInReview).
			Where(cols.NovaRead.IsFalse()).
			Order(cols.ReceivedAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &ids); err != nil {
			return nil, err
		}

		return ids, nil
	})
}

func (r *repository) HandOffCandidates(
	ctx context.Context,
	req inboundmessageports.HandOffCandidatesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids := make([]pulid.ID, 0, req.Limit)
		if err := handOffCandidatesQuery(r.db.DBForContext(ctx), req).Scan(ctx, &ids); err != nil {
			return nil, err
		}

		return ids, nil
	})
}

func handOffCandidatesQuery(
	db bun.IDB,
	req inboundmessageports.HandOffCandidatesRequest,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.Status.In(), bun.List(needsReviewStatuses)).
		Where(cols.Parked.IsNull()).
		Where(cols.OwnerID.Expr("{} IS DISTINCT FROM ?"), req.To).
		Order(cols.DueAt.OrderAsc()+" NULLS LAST", cols.ID.OrderAsc()).
		Limit(int64(req.Limit))
}

func (r *repository) WaitingByOwners(
	ctx context.Context,
	req inboundmessageports.WaitingByOwnersRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(req.OwnerIDs))
		if len(req.OwnerIDs) == 0 {
			return counts, nil
		}

		rows := make([]struct {
			OwnerID pulid.ID `bun:"owner_id"`
			Count   int      `bun:"count"`
		}, 0, len(req.OwnerIDs))
		cols := inboundmessagecol.InboundMessageColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(cols.OwnerID.Qualified()).
			ColumnExpr(buncolgen.Count("count")).
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.Status.In(), bun.List(needsReviewStatuses)).
			Where(cols.Parked.IsNull()).
			Where(cols.OwnerID.In(), bun.List(req.OwnerIDs)).
			GroupExpr(cols.OwnerID.Qualified()).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.OwnerID] = row.Count
		}

		return counts, nil
	})
}
