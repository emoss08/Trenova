package triageinsightrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *repository) RuleCandidates(
	ctx context.Context,
	req *triageports.RuleCandidatesRequest,
) (*triageports.RuleCandidates, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.RuleCandidates, error) {
		db := r.db.Keel(ctx)

		counted, err := messages().Where(ruleCandidates(req)...).Count(ctx, db)
		if err != nil {
			return nil, err
		}
		total := int(counted)

		rows := []*triageports.MessageRow{}
		if total > 0 && req.Limit > 0 {
			scanned, scanErr := keel.Into[scannedRow[bool]](ctx, db, ruleCandidatesQuery(req))
			if scanErr != nil {
				return nil, scanErr
			}
			rows = messageRows(scanned, true)
		}

		return &triageports.RuleCandidates{Total: total, Rows: rows}, nil
	})
}

func ruleCandidatesQuery(
	req *triageports.RuleCandidatesRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	att := inboundmessagedb.InboundAttachments

	personFiled := keel.Exists(att.Pick(att.ID).Where(
		att.MessageID.EqCol(m.ID),
		att.FiledAt.IsNotNull(),
		att.FiledBy.IsNotNull(),
	))

	return withReference(messages()).
		Pick(messageRowColumns(personFiled)...).
		Where(ruleCandidates(req)...).
		Order(m.ReceivedAt.Desc(), m.ID.Desc()).
		Limit(req.Limit)
}

func ruleCandidates(req *triageports.RuleCandidatesRequest) []keel.Pred {
	m := inboundmessagedb.InboundMessages

	preds := []keel.Pred{
		m.Classification.Eq(req.Classification),
		m.ReceivedAt.Gte(req.Since),
		m.NovaRead.IsTrue(),
		keel.When(req.MailboxID.IsNotNil(), m.MailboxID.Eq(req.MailboxID)),
	}

	switch req.SenderScope {
	case triage.SenderDomain:
		domain := strings.ToLower(req.Sender)
		preds = append(
			preds,
			keel.Or(m.SenderDomain.Eq(domain), m.SenderDomain.HasSuffix("."+domain)),
		)
	case triage.SenderAddress:
		preds = append(preds,
			m.SenderDomain.Eq(emailutils.Domain(req.Sender)),
			keel.DynFunc("lower", keel.DynFunc("btrim", m.FromAddress)).Eq(strings.ToLower(req.Sender)),
		)
	case triage.SenderAnyCustomer:
		preds = append(preds, m.MatchedCustomerID.IsNotNull())
	case triage.SenderAnyCarrier:
		preds = append(preds, m.MatchedCarrierID.IsNotNull())
	case triage.SenderAnyone:
	}

	return preds
}

type manualOutcomeScan struct {
	SenderDomain   string
	Classification inboundmessage.Classification
	Outcome        triage.ActivityKind
	Count          int64
	Arrived        int64
	Mailboxes      int64
	MailboxID      pulid.ID
	AllCustomers   bool
	AllCarriers    bool
}

func (r *repository) ManualOutcomes(
	ctx context.Context,
	req *triageports.ManualOutcomesRequest,
) ([]*triageports.ManualOutcomeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.ManualOutcomeRow, error) {
		scanned, err := keel.Into[manualOutcomeScan](ctx, r.db.Keel(ctx), manualOutcomesQuery(req))
		if err != nil {
			return nil, err
		}

		rows := make([]*triageports.ManualOutcomeRow, 0, len(scanned))
		for _, s := range scanned {
			rows = append(rows, &triageports.ManualOutcomeRow{
				SenderDomain:   s.SenderDomain,
				Classification: s.Classification,
				Outcome:        s.Outcome,
				Count:          int(s.Count),
				Arrived:        int(s.Arrived),
				Mailboxes:      int(s.Mailboxes),
				MailboxID:      s.MailboxID,
				AllCustomers:   s.AllCustomers,
				AllCarriers:    s.AllCarriers,
			})
		}

		return rows, nil
	})
}

func manualOutcomesQuery(
	req *triageports.ManualOutcomesRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	act := triagedb.Activities
	arrivedMessages := inboundmessagedb.InboundMessages.As("arrived_messages")

	outcomes := keel.CTE("manual_outcomes", []string{"message_id", "outcome"}, act.
		Pick(
			act.MessageID,
			keel.DynCase().
				When(
					keel.DynFunc("bool_or", act.Kind.Eq(triage.ActivityIgnored)),
					keel.DynLit(string(triage.ActivityIgnored)),
				).
				When(
					keel.DynFunc("bool_or", act.Kind.Eq(triage.ActivityFiled)),
					keel.DynLit(string(triage.ActivityFiled)),
				).
				Else(keel.DynLit(string(triage.ActivityHandled))),
		).
		Where(
			act.ActorKind.Eq(triage.ActorPerson),
			act.Kind.In(triage.ActivityFiled, triage.ActivityHandled, triage.ActivityIgnored),
			act.CreatedAt.Gte(req.Since),
		).
		Group(act.MessageID))

	arrivals := keel.CTE(
		"manual_arrivals",
		[]string{"sender_domain", "classification", "arrived"},
		arrivedMessages.
			Pick(arrivedMessages.SenderDomain, arrivedMessages.Classification, keel.Count()).
			Where(
				arrivedMessages.ReceivedAt.Gte(req.Since),
				arrivedMessages.Classification.IsNotNull(),
				arrivedMessages.SenderDomain.Ne(""),
			).
			Group(arrivedMessages.SenderDomain, arrivedMessages.Classification),
	)

	outcome := outcomes.Col("outcome")

	return messages().
		With(outcomes, arrivals).
		Join(outcomes, outcomes.Col("message_id").EqCol(m.ID)).
		Join(
			arrivals,
			arrivals.Col("sender_domain").EqCol(m.SenderDomain),
			arrivals.Col("classification").EqCol(m.Classification),
		).
		Pick(
			m.SenderDomain,
			m.Classification,
			outcome,
			keel.Count(),
			keel.Max(arrivals.Col("arrived")),
			m.MailboxID.CountDistinct(),
			m.MailboxID.Min(),
			keel.DynFunc("bool_and", keel.DynOf(m.MatchedCustomerID).IsNotNull()),
			keel.DynFunc("bool_and", keel.DynOf(m.MatchedCarrierID).IsNotNull()),
		).
		Where(m.ReceivedAt.Gte(req.Since)).
		Group(m.SenderDomain, m.Classification, outcome)
}

func (r *repository) MessageRows(
	ctx context.Context,
	req *triageports.MessageRowsRequest,
) ([]*triageports.MessageRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.MessageRow, error) {
		if len(req.IDs) == 0 {
			return []*triageports.MessageRow{}, nil
		}

		scanned, err := keel.Into[scannedRow[bool]](ctx, r.db.Keel(ctx), messageRowsQuery(req))
		if err != nil {
			return nil, err
		}

		return messageRows(scanned, false), nil
	})
}

func messageRowsQuery(
	req *triageports.MessageRowsRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return withReference(messages()).
		Pick(messageRowColumns(keel.DynLit(false).Cast(keel.TypeBool))...).
		Where(m.ID.In(req.IDs...)).
		Limit(len(req.IDs))
}

func (r *repository) SenderDomains(
	ctx context.Context,
	req *triageports.SenderDomainsRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		if req.Limit <= 0 {
			return []string{}, nil
		}

		domains, err := keel.Into[string](ctx, r.db.Keel(ctx), senderDomainsQuery(req))
		if err != nil {
			return nil, err
		}
		if domains == nil {
			domains = []string{}
		}

		return domains, nil
	})
}

func senderDomainsQuery(
	req *triageports.SenderDomainsRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	term := strings.ToLower(strings.TrimSpace(req.Query))

	return messages().
		Pick(m.SenderDomain).
		Where(
			m.ReceivedAt.Gte(req.Since),
			m.SenderDomain.Ne(""),
			keel.When(term != "", m.SenderDomain.Contains(term)),
		).
		Group(m.SenderDomain).
		Order(keel.Count().Desc(), m.SenderDomain.Asc()).
		Limit(req.Limit)
}
