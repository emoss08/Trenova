package conversationrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
)

type mentionTerms struct {
	contains string
	prefix   string
	limit    int
	offset   int
}

type mentionQuery func(terms mentionTerms) keel.Projection

func humanize(e keel.Expr) keel.DynExpr {
	return keel.DynFunc(
		"regexp_replace",
		keel.DynOf(e).Cast(keel.TypeString),
		"([a-z])([A-Z])",
		`\1 \2`,
		"g",
	)
}

func mentionID(e keel.Expr) keel.DynExpr {
	return keel.DynOf(e).Cast(keel.TypeString)
}

func shipmentEndPlace(s *shipmentdb.ShipmentsTable, last bool) keel.Expr {
	m := shipmentdb.ShipmentMoves
	sp := shipmentdb.Stops
	l := locationdb.Locations
	st := usstatedb.UsStates
	order := []keel.Order{m.Sequence.Asc(), sp.Sequence.Asc()}
	if last {
		order = []keel.Order{m.Sequence.Desc(), sp.Sequence.Desc()}
	}

	return keel.Scalar(m.Join(sp, sp.ShipmentMoveID.EqCol(m.ID)).
		Join(l, l.ID.EqCol(sp.LocationID)).
		LeftJoin(st, st.ID.EqCol(l.StateID)).
		Pick(keel.DynFunc("concat_ws", ", ", l.City, st.Abbreviation)).
		Where(m.ShipmentID.EqCol(s.ID)).
		Order(order...).
		Limit(1))
}

func shipmentMentions(terms mentionTerms) keel.Projection {
	s := shipmentdb.Shipments
	route := keel.DynFunc(
		"concat_ws",
		" → ",
		shipmentEndPlace(s, false),
		shipmentEndPlace(s, true),
	).NullIf("")

	return s.Pick(
		keel.DynLit("shipment"),
		mentionID(s.ID),
		s.ProNumber,
		keel.DynFunc("concat_ws", " · ", route, humanize(s.Status)),
	).
		Where(keel.Or(s.ProNumber.ILike(terms.contains), s.BOL.ILike(terms.contains))).
		Order(keel.DynOf(s.ProNumber).ILike(terms.prefix).Desc(), s.UpdatedAt.Desc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func customerMentions(terms mentionTerms) keel.Projection {
	c := customerdb.Customers

	return c.Pick(
		keel.DynLit("customer"),
		mentionID(c.ID),
		c.Name,
		keel.DynFunc("concat_ws", " · ", c.Code, keel.DynOf(c.City).NullIf("")),
	).
		Where(keel.Or(c.Name.ILike(terms.contains), c.Code.ILike(terms.contains))).
		Order(keel.DynOf(c.Name).ILike(terms.prefix).Desc(), c.Name.Asc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func invoiceMentions(terms mentionTerms) keel.Projection {
	i := invoicedb.Invoices

	return i.Pick(
		keel.DynLit("invoice"),
		mentionID(i.ID),
		i.Number,
		keel.DynFunc(
			"concat_ws",
			" · ",
			keel.DynOf(i.BillToName).NullIf(""),
			humanize(i.Status),
		),
	).
		Where(keel.Or(i.Number.ILike(terms.contains), i.BillToName.ILike(terms.contains))).
		Order(keel.DynOf(i.Number).ILike(terms.prefix).Desc(), i.UpdatedAt.Desc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func invoiceDisputeMentions(terms mentionTerms) keel.Projection {
	d := invoicedb.InvoiceDisputes
	i := invoicedb.Invoices

	return d.Join(i, i.ID.EqCol(d.InvoiceID)).
		Pick(
			keel.DynLit("invoice_dispute"),
			mentionID(d.ID),
			i.Number,
			keel.DynFunc(
				"concat_ws",
				" · ",
				keel.DynOf(i.BillToName).NullIf(""),
				humanize(d.ReasonCode),
				humanize(d.Status),
			),
		).
		Where(keel.Or(i.Number.ILike(terms.contains), i.BillToName.ILike(terms.contains))).
		Order(keel.DynOf(i.Number).ILike(terms.prefix).Desc(), d.UpdatedAt.Desc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func billingQueueMentions(terms mentionTerms) keel.Projection {
	q := billingqueuedb.BillingQueueItems
	s := shipmentdb.Shipments

	return q.LeftJoin(s, s.ID.EqCol(q.ShipmentID)).
		Pick(
			keel.DynLit("billing_queue_item"),
			mentionID(q.ID),
			q.Number,
			keel.DynFunc(
				"concat_ws",
				" · ",
				keel.DynOf(s.ProNumber).NullIf(""),
				humanize(q.Status),
			),
		).
		Where(keel.Or(q.Number.ILike(terms.contains), s.ProNumber.ILike(terms.contains))).
		Order(keel.DynOf(q.Number).ILike(terms.prefix).Desc(), q.UpdatedAt.Desc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func workerMentions(terms mentionTerms) keel.Projection {
	w := workerdb.Workers
	fullName := keel.DynFunc("concat_ws", " ", w.FirstName, w.LastName)

	return w.Pick(
		keel.DynLit("worker"),
		mentionID(w.ID),
		fullName,
		keel.DynFunc("concat_ws", " · ", humanize(w.Type), keel.DynOf(w.City).NullIf("")),
	).
		Where(keel.Or(fullName.ILike(terms.contains), w.LastName.ILike(terms.contains))).
		Order(
			keel.DynOf(w.FirstName).ILike(terms.prefix).Desc(),
			w.FirstName.Asc(),
			w.LastName.Asc(),
		).
		Limit(terms.limit).
		Offset(terms.offset)
}

func carrierMentions(terms mentionTerms) keel.Projection {
	c := carrierdb.Carriers

	return c.Pick(
		keel.DynLit("carrier"),
		mentionID(c.ID),
		c.Name,
		keel.DynFunc(
			"concat_ws",
			" · ",
			keel.DynLit("MC ").Concat(keel.DynOf(c.MCNumber).NullIf("")),
			humanize(c.ComplianceStatus),
		),
	).
		Where(keel.Or(
			c.Name.ILike(terms.contains),
			c.Code.ILike(terms.contains),
			c.MCNumber.ILike(terms.contains),
		)).
		Order(keel.DynOf(c.Name).ILike(terms.prefix).Desc(), c.Name.Asc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

func locationMentions(terms mentionTerms) keel.Projection {
	l := locationdb.Locations
	st := usstatedb.UsStates

	return l.LeftJoin(st, st.ID.EqCol(l.StateID)).
		Pick(
			keel.DynLit("location"),
			mentionID(l.ID),
			l.Name,
			keel.DynFunc(
				"concat_ws",
				" · ",
				l.Code,
				keel.DynFunc(
					"concat_ws",
					", ",
					keel.DynOf(l.City).NullIf(""),
					st.Abbreviation,
				).NullIf(""),
			),
		).
		Where(keel.Or(
			l.Name.ILike(terms.contains),
			l.Code.ILike(terms.contains),
			l.City.ILike(terms.contains),
		)).
		Order(keel.DynOf(l.Name).ILike(terms.prefix).Desc(), l.Name.Asc()).
		Limit(terms.limit).
		Offset(terms.offset)
}

var mentionQueries = map[string]mentionQuery{
	"shipment":           shipmentMentions,
	"customer":           customerMentions,
	"invoice":            invoiceMentions,
	"invoice_dispute":    invoiceDisputeMentions,
	"billing_queue_item": billingQueueMentions,
	"worker":             workerMentions,
	"carrier":            carrierMentions,
	"location":           locationMentions,
}

func likeEscape(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func (r *repository) SearchMentions(
	ctx context.Context,
	req agentports.SearchMentionsRequest,
) ([]agentports.MentionRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MentionRow, error) {
		if req.LimitPerKind <= 0 {
			return []agentports.MentionRow{}, nil
		}
		term := likeEscape(strings.TrimSpace(req.Query))
		terms := mentionTerms{
			contains: "%" + term + "%",
			prefix:   term + "%",
			limit:    req.LimitPerKind,
			offset:   max(req.Offset, 0),
		}
		results := make([][]agentports.MentionRow, 0, len(req.Kinds))
		items := make([]keel.BatchItem, 0, len(req.Kinds))
		for _, kind := range req.Kinds {
			query, ok := mentionQueries[kind]
			if !ok {
				continue
			}
			results = append(results, nil)
			items = append(items, keel.IntoSlice(query(terms), &results[len(results)-1]))
		}
		if len(items) == 0 {
			return []agentports.MentionRow{}, nil
		}
		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			return nil, fmt.Errorf("search mentions: %w", err)
		}
		out := make([]agentports.MentionRow, 0, len(req.Kinds)*req.LimitPerKind)
		for _, rows := range results {
			out = append(out, rows...)
		}
		return out, nil
	})
}
