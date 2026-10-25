package triageinsightrepository

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
)

const (
	handledSamplesPerGroup = 3
	rankedCTE              = "ranked"
)

type briefingScan struct {
	Waiting             int64
	Overdue             int64
	ClosedByPeopleToday int64
	HandledByNovaToday  int64
	ComingBack          int64
	AnyMessage          bool
}

func (r *repository) BriefingCounts(
	ctx context.Context,
	req triageports.BriefingWindow,
) (*triageports.BriefingCounts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.BriefingCounts, error) {
		s, err := keel.IntoOne[briefingScan](ctx, r.db.Keel(ctx), briefingCountsQuery(req))
		if err != nil {
			return nil, err
		}

		return &triageports.BriefingCounts{
			Waiting:             int(s.Waiting),
			Overdue:             int(s.Overdue),
			ClosedByPeopleToday: int(s.ClosedByPeopleToday),
			HandledByNovaToday:  int(s.HandledByNovaToday),
			ComingBack:          int(s.ComingBack),
			AnyMessage:          s.AnyMessage,
		}, nil
	})
}

func briefingCountsQuery(
	req triageports.BriefingWindow,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	anyMessage := inboundmessagedb.InboundMessages.As("any_message")

	open := m.Status.In(needsReview...)
	waiting := keel.And(open, m.Parked.IsNull())
	overdue := keel.And(waiting, m.DueAt.Lt(req.Now))
	closed := keel.And(
		m.Status.In(settled...),
		m.ReviewedAt.Gte(req.DayStart),
		m.ReviewedBy.IsNotNull(),
	)
	handled := keel.And(m.HandledByRuleID.IsNotNull(), m.ReceivedAt.Gte(req.DayStart))
	comingBack := keel.And(open, m.Parked.IsNotNull())

	return messages().
		Pick(
			keel.Count().Filter(waiting),
			keel.Count().Filter(overdue),
			keel.Count().Filter(closed),
			keel.Count().Filter(handled),
			keel.Count().Filter(comingBack),
			keel.Exists(anyMessage.Pick(anyMessage.ID).Limit(1)),
		).
		Where(keel.Or(open, closed, handled))
}

func (r *repository) HandledByNova(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.HandledRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.HandledRow, error) {
		scanned, err := keel.Into[scannedRow[int64]](ctx, r.db.Keel(ctx), handledQuery(req))
		if err != nil {
			return nil, err
		}

		rows := make([]*triageports.HandledRow, 0, len(scanned))
		for i := range scanned {
			rows = append(rows, &triageports.HandledRow{
				MessageRow: scanned[i].row(),
				GroupCount: int(scanned[i].Extra),
			})
		}

		return rows, nil
	})
}

func handledQuery(req triageports.BriefingWindow) *keel.Query[keel.Row] {
	m := inboundmessagedb.InboundMessages
	partition := []keel.Expr{m.HandledAction, m.Classification}

	ranked := keel.CTE(
		rankedCTE,
		append(slices.Clone(messageRowNames), "group_count", "sample_rank"),
		withReference(messages()).
			Pick(messageRowColumns(
				keel.Count().Over(partition),
				keel.RowNumber().Over(partition, m.ReceivedAt.Desc(), m.ID.Desc()),
			)...).
			Where(m.HandledByRuleID.IsNotNull(), m.ReceivedAt.Gte(req.DayStart)),
	)

	return ranked.Query().
		With(ranked).
		Pick(rankedColumns(ranked, "group_count")...).
		Where(ranked.Col("sample_rank").Lte(handledSamplesPerGroup)).
		Order(
			ranked.Col("group_count").Desc(),
			ranked.Col("handled_action").Asc(),
			ranked.Col("classification").Asc(),
			ranked.Col("sample_rank").Asc(),
		)
}

type arrivalScan struct {
	Hour          int64
	HandledByNova int64
	LeftForPerson int64
}

func (r *repository) Arrivals(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.ArrivalRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.ArrivalRow, error) {
		scanned, err := keel.Into[arrivalScan](ctx, r.db.Keel(ctx), arrivalsQuery(req))
		if err != nil {
			return nil, err
		}

		rows := make([]*triageports.ArrivalRow, 0, len(scanned))
		for _, s := range scanned {
			rows = append(rows, &triageports.ArrivalRow{
				Hour:          int(s.Hour),
				HandledByNova: int(s.HandledByNova),
				LeftForPerson: int(s.LeftForPerson),
			})
		}

		return rows, nil
	})
}

func arrivalsQuery(req triageports.BriefingWindow) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	hour := keel.DynFunc("floor", keel.DynOf(m.ReceivedAt).Sub(req.DayStart).Div(3600)).
		Cast(keel.TypeInt64)
	nova := settledByRule()

	return messages().
		Pick(hour, keel.Count().Filter(nova), keel.Count().Filter(keel.Not(nova))).
		Where(m.ReceivedAt.Gte(req.DayStart), m.ReceivedAt.Lte(req.Now)).
		Group(hour).
		Order(keel.Asc(hour))
}

func (r *repository) River(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.RiverRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.RiverRow, error) {
		scanned, err := keel.Into[scannedRow[bool]](ctx, r.db.Keel(ctx), riverQuery(req))
		if err != nil {
			return nil, err
		}

		rows := make([]*triageports.RiverRow, 0, len(scanned))
		for i := range scanned {
			rows = append(rows, &triageports.RiverRow{
				MessageRow: scanned[i].row(),
				Nova:       scanned[i].Extra,
			})
		}

		return rows, nil
	})
}

func riverQuery(req triageports.BriefingWindow) *keel.Query[keel.Row] {
	m := inboundmessagedb.InboundMessages
	nova := settledByRule()

	ranked := keel.CTE(rankedCTE, append(slices.Clone(messageRowNames), "nova", "lane_rank"),
		withReference(messages()).
			Pick(messageRowColumns(
				nova,
				keel.RowNumber().Over([]keel.Expr{nova}, m.ReceivedAt.Desc(), m.ID.Desc()),
			)...).
			Where(
				m.ReceivedAt.Gte(req.DayStart),
				m.ReceivedAt.Lte(req.Now),
				keel.Or(nova, m.Status.In(needsReview...), settledByPerson()),
			))

	return ranked.Query().
		With(ranked).
		Pick(rankedColumns(ranked, "nova")...).
		Where(ranked.Col("lane_rank").Lte(req.Limit)).
		Order(ranked.Col("received_at").Asc(), ranked.Col("id").Asc())
}

func (r *repository) Waiting(
	ctx context.Context,
	req triageports.WaitingRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			entities, err := waitingQuery(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			if entities == nil {
				entities = []*inboundmessage.InboundMessage{}
			}

			return entities, nil
		},
	)
}

func waitingQuery(req triageports.WaitingRequest) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return m.Omit(m.TextBody, m.HTMLKey).
		Where(m.Status.In(needsReview...), m.Parked.IsNull()).
		Order(m.DueAt.Asc().NullsLast(), m.ReceivedAt.Asc(), m.ID.Asc()).
		Limit(max(req.Limit, 1))
}

type correctionScan struct {
	CorrectionsThisWeek int64
	ReadThisWeek        int64
	CorrectedThisWeek   int64
	ReadLastWeek        int64
	CorrectedLastWeek   int64
}

func (r *repository) CorrectionTotals(
	ctx context.Context,
	req triageports.CorrectionTotalsRequest,
) (*triageports.CorrectionTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.CorrectionTotals, error) {
		s, err := keel.IntoOne[correctionScan](ctx, r.db.Keel(ctx), correctionTotalsQuery(req))
		if err != nil {
			return nil, err
		}

		return &triageports.CorrectionTotals{
			CorrectionsThisWeek: int(s.CorrectionsThisWeek),
			ReadThisWeek:        int(s.ReadThisWeek),
			CorrectedThisWeek:   int(s.CorrectedThisWeek),
			ReadLastWeek:        int(s.ReadLastWeek),
			CorrectedLastWeek:   int(s.CorrectedLastWeek),
		}, nil
	})
}

func correctionTotalsQuery(
	req triageports.CorrectionTotalsRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	act := triagedb.Activities

	corrections := keel.Scalar(act.Pick(keel.Count()).Where(
		act.Kind.In(triage.ActivityCorrected, triage.ActivityFactFlagged),
		act.ActorKind.Eq(triage.ActorPerson),
		act.CreatedAt.Gte(req.WeekStart),
	))
	corrected := keel.DynOf(m.ClassificationCorrected).Or(
		keel.DynOf(m.Facts.Contains([]map[string]bool{{"flagged": true}})).Coalesce(false),
	)
	thisWeek := m.ReceivedAt.Gte(req.WeekStart)
	lastWeek := m.ReceivedAt.Lt(req.WeekStart)

	return messages().
		Pick(
			corrections,
			keel.Count().Filter(thisWeek),
			keel.Count().Filter(keel.And(thisWeek, corrected)),
			keel.Count().Filter(lastWeek),
			keel.Count().Filter(keel.And(lastWeek, corrected)),
		).
		Where(m.NovaRead.IsTrue(), m.ReceivedAt.Gte(req.LastWeekStart))
}
