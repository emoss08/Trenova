//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewQueueRepository(p Params) inboundmessageports.TriageQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-queue-repository"),
	}
}

func stateColumns() []keel.Expr {
	m := inboundmessagedb.InboundMessages

	return []keel.Expr{
		m.ID,
		m.OrganizationID,
		m.BusinessUnitID,
		m.MailboxID,
		m.Status,
		m.Classification,
		m.ReceivedAt,
		m.FromAddress,
		m.FromName,
		m.Subject,
		m.SenderDomain,
		m.ThreadKey,
		m.OwnerID,
		m.DueAt,
		m.Parked,
		m.SnoozedUntil,
		m.SnoozeEvent,
		m.SnoozeWatchID,
		m.SnoozeDocumentTypeID,
		m.AwaitingUntil,
		m.MatchedShipmentID,
		m.MatchedCustomerID,
		m.MatchedCarrierID,
		m.MatchReason,
		m.FailureText,
		m.ReviewNote,
		m.NovaRead,
	}
}

func stateOf(m *inboundmessage.InboundMessage) *inboundmessageports.QueueState {
	return &inboundmessageports.QueueState{
		ID:                   m.ID,
		OrganizationID:       m.OrganizationID,
		BusinessUnitID:       m.BusinessUnitID,
		MailboxID:            m.MailboxID,
		Status:               m.Status,
		Classification:       m.Classification,
		ReceivedAt:           m.ReceivedAt,
		FromAddress:          m.FromAddress,
		FromName:             m.FromName,
		Subject:              m.Subject,
		SenderDomain:         m.SenderDomain,
		ThreadKey:            m.ThreadKey,
		OwnerID:              m.OwnerID,
		DueAt:                m.DueAt,
		Parked:               m.Parked,
		SnoozedUntil:         m.SnoozedUntil,
		SnoozeEvent:          m.SnoozeEvent,
		SnoozeWatchID:        m.SnoozeWatchID,
		SnoozeDocumentTypeID: m.SnoozeDocumentTypeID,
		AwaitingUntil:        m.AwaitingUntil,
		MatchedShipmentID:    m.MatchedShipmentID,
		MatchedCustomerID:    m.MatchedCustomerID,
		MatchedCarrierID:     m.MatchedCarrierID,
		MatchReason:          m.MatchReason,
		FailureText:          m.FailureText,
		ReviewNote:           m.ReviewNote,
		NovaRead:             m.NovaRead,
	}
}

func statesOf(
	messages []*inboundmessage.InboundMessage,
	err error,
) ([]*inboundmessageports.QueueState, error) {
	if err != nil {
		return nil, err
	}

	states := make([]*inboundmessageports.QueueState, 0, len(messages))
	for _, m := range messages {
		states = append(states, stateOf(m))
	}

	return states, nil
}

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
	if len(req.IDs) == 0 {
		return []*inboundmessageports.QueueState{}, nil
	}

	return statesOf(queueStatesQuery(req).List(ctx, r.db.Keel(ctx)))
}

func queueStatesQuery(
	req inboundmessageports.QueueStatesRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	q := m.Pick(stateColumns()...).
		Where(m.ID.In(req.IDs...)).
		Order(m.ID.Asc()).
		Limit(len(req.IDs))
	if req.ForUpdate {
		q = q.Lock(keel.ForUpdate)
	}

	return q
}

func (r *repository) SelectIDs(
	ctx context.Context,
	req inboundmessageports.SelectInboundMessageIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req.Limit <= 0 {
			return []pulid.ID{}, nil
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), selectIDsQuery(req))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func selectIDsQuery(
	req inboundmessageports.SelectInboundMessageIDsRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	folder := req.Folder

	return listquery.Filter(m.Where(narrow(folder)...), folder.Filter).
		Pick(m.ID).
		Order(folderOrder(folder.Sort)...).
		Limit(req.Limit)
}

func folderOrder(sort inboundmessageports.Sort) []keel.Order {
	m := inboundmessagedb.InboundMessages
	if sort == inboundmessageports.SortDue {
		return []keel.Order{m.DueAt.Asc().NullsLast(), m.ReceivedAt.Desc(), m.ID.Desc()}
	}

	return []keel.Order{m.ReceivedAt.Desc(), m.ID.Desc()}
}

func (r *repository) UpdateQueueState(
	ctx context.Context,
	req inboundmessageports.UpdateQueueStateRequest,
) ([]*inboundmessageports.QueueState, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			if len(req.IDs) == 0 {
				return []*inboundmessageports.QueueState{}, nil
			}

			m := inboundmessagedb.InboundMessages
			sets := append(
				[]keel.Assignment{m.UpdatedAt.To(req.Now)},
				patchAssignments(req.Patch, req.Now)...,
			)

			return statesOf(m.Pick(stateColumns()...).
				Where(m.ID.In(req.IDs...)).
				UpdateReturning(ctx, r.db.Keel(ctx), sets...))
		},
	)
}

func patchAssignments(patch inboundmessageports.QueuePatch, now int64) []keel.Assignment {
	m := inboundmessagedb.InboundMessages
	sets := make([]keel.Assignment, 0, 16)

	if patch.Status != "" {
		sets = append(sets, m.Status.To(patch.Status))
	}
	if patch.Review != nil {
		sets = append(sets, reviewAssignments(patch.Review, now)...)
	}
	if patch.Owner != nil {
		sets = append(sets, m.OwnerID.To(*patch.Owner))
	}
	switch {
	case patch.Park != nil:
		sets = append(sets, parkAssignments(patch.Park, now)...)
	case patch.Unpark:
		sets = append(sets, clearParking()...)
	}
	sets = append(sets, dueAssignments(patch)...)
	if patch.ClearHandled {
		sets = append(sets, m.HandledByRuleID.ToNull(), m.HandledAction.ToNull())
	}
	if patch.HandOff != nil {
		sets = append(sets, m.ActionState.ToExpr(keel.DynOf(m.ActionState).Concat(keel.DynFunc(
			"jsonb_build_object",
			keel.DynConst("handedOffTo"),
			keel.DynLit(patch.HandOff.To.String()),
			keel.DynConst("handedOffAt"),
			keel.DynLit(now).Cast(keel.TypeInt64),
		))))
	}
	if patch.ClearShipment {
		sets = append(sets, m.MatchedShipmentID.ToNull())
	}
	if patch.ClearCustomer {
		sets = append(sets, m.MatchedCustomerID.ToNull())
	}
	if patch.ClearCarrier {
		sets = append(sets, m.MatchedCarrierID.ToNull())
	}
	if patch.ClearReason {
		sets = append(sets, m.MatchReason.ToNull())
	}
	if patch.NovaUnread {
		sets = append(sets, m.NovaRead.To(false))
	}

	return sets
}

func reviewAssignments(review *inboundmessageports.ReviewPatch, now int64) []keel.Assignment {
	m := inboundmessagedb.InboundMessages
	if review.Clear {
		return []keel.Assignment{m.ReviewedBy.ToNull(), m.ReviewedAt.ToNull()}
	}

	return []keel.Assignment{
		m.ReviewedBy.To(review.ReviewerID),
		m.ReviewedAt.To(now),
		m.ReviewNote.ToExpr(keel.DynLit(review.Note).NullIf("").Coalesce(m.ReviewNote)),
	}
}

func clearParking() []keel.Assignment {
	m := inboundmessagedb.InboundMessages

	return []keel.Assignment{
		m.Parked.ToNull(),
		m.SnoozedUntil.ToNull(),
		m.SnoozeEvent.ToNull(),
		m.SnoozeWatchID.ToNull(),
		m.SnoozeDocumentTypeID.ToNull(),
		m.SnoozedAt.ToNull(),
		m.SnoozedBy.ToNull(),
		m.AwaitingSince.ToNull(),
		m.AwaitingUntil.ToNull(),
	}
}

func parkAssignments(park *inboundmessageports.ParkPatch, now int64) []keel.Assignment {
	m := inboundmessagedb.InboundMessages
	if park.Parking == inboundmessage.ParkingAwaitingReply {
		return []keel.Assignment{
			m.Parked.To(park.Parking),
			m.AwaitingSince.To(now),
			m.AwaitingUntil.To(park.AwaitingUntil),
			m.SnoozedUntil.ToNull(),
			m.SnoozeEvent.ToNull(),
			m.SnoozeWatchID.ToNull(),
			m.SnoozeDocumentTypeID.ToNull(),
			m.SnoozedAt.ToNull(),
			m.SnoozedBy.ToNull(),
		}
	}

	watch := m.SnoozeWatchID.ToNull()
	if park.WatchShipment {
		watch = m.SnoozeWatchID.ToExpr(m.MatchedShipmentID)
	}

	return []keel.Assignment{
		m.Parked.To(park.Parking),
		m.SnoozedUntil.To(park.SnoozedUntil),
		m.SnoozeEvent.To(park.SnoozeEvent),
		m.SnoozeDocumentTypeID.To(park.DocumentTypeID),
		m.SnoozedAt.To(now),
		m.SnoozedBy.To(park.By),
		m.AwaitingSince.ToNull(),
		m.AwaitingUntil.ToNull(),
		watch,
	}
}

func dueAssignments(patch inboundmessageports.QueuePatch) []keel.Assignment {
	m := inboundmessagedb.InboundMessages
	switch patch.Due {
	case inboundmessageports.DueClear:
		return []keel.Assignment{m.DueAt.ToNull()}
	case inboundmessageports.DueFromTargets:
		return []keel.Assignment{m.DueAt.ToExpr(dueFromTargetsExpr(patch.Targets))}
	case inboundmessageports.DueKeep:
		return nil
	default:
		return nil
	}
}

func dueFromTargetsExpr(targets triage.Targets) keel.Expr {
	m := inboundmessagedb.InboundMessages
	kinds := inboundmessage.AllClassifications()

	classification := keel.DynOf(m.Classification).Coalesce(string(inboundmessage.ClassificationOther))
	minutes := keel.DynCase()
	for _, kind := range kinds {
		minutes = minutes.When(
			classification.Eq(string(kind)),
			keel.DynLit(int64(targets.For(kind))).Cast(keel.TypeInt64),
		)
	}
	fallback := keel.DynLit(int64(targets.For(inboundmessage.ClassificationOther))).Cast(keel.TypeInt64)

	return keel.DynOf(m.ReceivedAt).Add(keel.DynLit(60).Mul(minutes.Else(fallback)))
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req inboundmessageports.ListInboundMessagesByIDsRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			if len(req.IDs) == 0 {
				return []*inboundmessage.InboundMessage{}, nil
			}

			m := inboundmessagedb.InboundMessages
			return orEmptyMessages(m.Where(m.ID.In(req.IDs...)).
				Load(m.Mailbox()).
				Order(m.ReceivedAt.Desc(), m.ID.Desc()).
				Limit(len(req.IDs)).
				List(ctx, r.db.Keel(ctx)))
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

		type threadRow struct {
			MessageKey string
			ThreadKey  string
		}

		m := inboundmessagedb.InboundMessages
		rows, err := keel.Into[threadRow](ctx, r.db.Keel(ctx), m.
			Pick(m.MessageKey, m.ThreadKey).
			Where(m.MessageKey.In(req.MessageKeys...)).
			Order(m.ReceivedAt.Asc()))
		if err != nil {
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

		type threadCount struct {
			ThreadKey string
			Count     int64
		}

		m := inboundmessagedb.InboundMessages
		rows, err := keel.Into[threadCount](ctx, r.db.Keel(ctx), m.
			Pick(m.ThreadKey, keel.Count()).
			Where(m.ThreadKey.In(req.ThreadKeys...)).
			Group(m.ThreadKey))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.ThreadKey] = int(row.Count)
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
			return orEmptyMessages(listThreadQuery(req).List(ctx, r.db.Keel(ctx)))
		},
	)
}

func listThreadQuery(
	req inboundmessageports.ListThreadRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	newest := inboundmessagedb.InboundMessages.As("newest")
	limit := max(req.Limit, 1)

	return m.Where(m.ID.InQuery(newest.
		Pick(newest.ID).
		Where(newest.ThreadKey.Eq(req.ThreadKey)).
		Order(newest.ReceivedAt.Desc(), newest.ID.Desc()).
		Limit(limit),
	)).
		Load(m.Mailbox()).
		Order(m.ReceivedAt.Asc(), m.ID.Asc()).
		Limit(limit)
}

func (r *repository) AwaitingOnThread(
	ctx context.Context,
	req inboundmessageports.AwaitingOnThreadRequest,
) ([]*inboundmessageports.QueueState, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessageports.QueueState, error) {
			m := inboundmessagedb.InboundMessages
			return statesOf(m.Pick(stateColumns()...).
				Where(
					m.ThreadKey.Eq(req.ThreadKey),
					m.Parked.Eq(inboundmessage.ParkingAwaitingReply),
					keel.When(req.ExcludeID.IsNotNil(), m.ID.Ne(req.ExcludeID)),
				).
				Limit(inboundmessageports.MaxThreadMessages).
				List(ctx, r.db.Keel(ctx)))
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
			m := inboundmessagedb.InboundMessages
			sets := append([]keel.Assignment{m.UpdatedAt.To(req.Now)}, clearParking()...)

			return statesOf(m.Pick(stateColumns()...).
				Where(
					m.SnoozeEvent.Eq(req.Event),
					m.SnoozeWatchID.Eq(req.WatchID),
					m.Parked.Eq(inboundmessage.ParkingSnoozed),
					keel.When(
						req.DocumentTypeID.IsNotNil(),
						m.SnoozeDocumentTypeID.Eq(req.DocumentTypeID),
					),
				).
				UpdateReturning(ctx, r.db.Keel(ctx), sets...))
		},
	)
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
		if req.Limit <= 0 {
			return []inboundmessageports.ParkedRef{}, nil
		}

		refs, err := keel.Into[inboundmessageports.ParkedRef](
			ctx,
			r.db.Keel(ctx),
			overdueParkedQuery(req),
		)
		if err != nil {
			return nil, err
		}

		return refs[:min(len(refs), req.Limit)], nil
	})
}

func overdueParkedQuery(req inboundmessageports.ListOverdueParkedRequest) keel.Compound {
	m := inboundmessagedb.InboundMessages

	snoozed := m.AllTenants().
		Pick(m.ID, m.OrganizationID, m.BusinessUnitID).
		Where(
			m.Parked.Eq(inboundmessage.ParkingSnoozed),
			m.SnoozedUntil.IsNotNull(),
			m.SnoozedUntil.Lte(req.Now),
		).
		Limit(req.Limit)
	awaiting := m.AllTenants().
		Pick(m.ID, m.OrganizationID, m.BusinessUnitID).
		Where(
			m.Parked.Eq(inboundmessage.ParkingAwaitingReply),
			m.AwaitingUntil.Lte(req.Now),
		).
		Limit(req.Limit)

	return keel.UnionAll(snoozed, awaiting)
}

func (r *repository) RecomputeDue(
	ctx context.Context,
	req inboundmessageports.RecomputeDueRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		m := inboundmessagedb.InboundMessages
		classification := m.Classification.Eq(req.Classification)
		if req.Classification == inboundmessage.ClassificationOther {
			classification = keel.Or(classification, m.Classification.IsNull())
		}

		return m.Where(m.Status.In(needsReviewStatuses...), classification).Update(
			ctx,
			r.db.Keel(ctx),
			m.DueAt.ToExpr(keel.DynOf(m.ReceivedAt).Add(keel.DynLit(int64(req.Minutes)*60).Cast(keel.TypeInt64))),
			m.Version.ToExpr(m.Version),
			m.UpdatedAt.ToExpr(m.UpdatedAt),
		)
	})
}

func (r *repository) UnreadInReview(
	ctx context.Context,
	req inboundmessageports.UnreadInReviewRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req.Limit <= 0 {
			return []pulid.ID{}, nil
		}

		m := inboundmessagedb.InboundMessages
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), m.
			Pick(m.ID).
			Where(m.Status.Eq(inboundmessage.StatusInReview), m.NovaRead.IsFalse()).
			Order(m.ReceivedAt.Asc(), m.ID.Asc()).
			Limit(req.Limit))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func (r *repository) HandOffCandidates(
	ctx context.Context,
	req inboundmessageports.HandOffCandidatesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if req.Limit <= 0 {
			return []pulid.ID{}, nil
		}

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), handOffCandidatesQuery(req))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func handOffCandidatesQuery(
	req inboundmessageports.HandOffCandidatesRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return m.Pick(m.ID).
		Where(
			m.Status.In(needsReviewStatuses...),
			m.Parked.IsNull(),
			keel.DynOf(m.OwnerID).IsDistinctFrom(req.To.String()),
		).
		Order(m.DueAt.Asc().NullsLast(), m.ID.Asc()).
		Limit(req.Limit)
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

		type ownerCount struct {
			OwnerID pulid.ID
			Count   int64
		}

		m := inboundmessagedb.InboundMessages
		rows, err := keel.Into[ownerCount](ctx, r.db.Keel(ctx), m.
			Pick(m.OwnerID, keel.Count()).
			Where(
				m.Status.In(needsReviewStatuses...),
				m.Parked.IsNull(),
				m.OwnerID.In(req.OwnerIDs...),
			).
			Group(m.OwnerID))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.OwnerID] = int(row.Count)
		}

		return counts, nil
	})
}
