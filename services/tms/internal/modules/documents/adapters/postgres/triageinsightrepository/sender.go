package triageinsightrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *repository) SenderStats(
	ctx context.Context,
	req *triageports.SenderStatsRequest,
) (*triageports.SenderStats, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.SenderStats, error) {
		db := r.db.Keel(ctx)

		kinds, err := keel.Into[senderKindScan](ctx, db, senderKindsQuery(req))
		if err != nil {
			return nil, err
		}

		stats := &triageports.SenderStats{
			Kinds:  make([]triageports.SenderKindRow, 0, len(kinds)),
			Recent: []*triageports.MessageRow{},
		}
		for _, kind := range kinds {
			stats.Messages += int(kind.Count)
			if kind.Classification != "" {
				stats.Kinds = append(stats.Kinds, triageports.SenderKindRow{
					Classification: kind.Classification,
					Count:          int(kind.Count),
				})
			}
		}
		if stats.Messages == 0 {
			return stats, nil
		}

		summary, err := keel.IntoOne[senderSummary](ctx, db, senderSummaryQuery(req))
		if err != nil {
			return nil, err
		}
		stats.UsualSettleSeconds = summary.UsualSettleSeconds
		stats.CustomerID = summary.CustomerID
		stats.CarrierID = summary.CarrierID

		if req.RecentLimit > 0 {
			recent, recentErr := keel.Into[scannedRow[bool]](ctx, db, senderRecentQuery(req))
			if recentErr != nil {
				return nil, recentErr
			}
			stats.Recent = messageRows(recent, false)
		}

		return stats, nil
	})
}

type senderKindScan struct {
	Classification inboundmessage.Classification
	Count          int64
}

type senderSummary struct {
	UsualSettleSeconds *float64
	CustomerID         pulid.ID
	CarrierID          pulid.ID
}

func fromSender(req *triageports.SenderStatsRequest) []keel.Pred {
	m := inboundmessagedb.InboundMessages

	return []keel.Pred{m.SenderDomain.Eq(req.SenderDomain), m.ReceivedAt.Gte(req.Since)}
}

func senderKindsQuery(
	req *triageports.SenderStatsRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	classification := text(m.Classification)

	return messages().
		Pick(classification, keel.Count()).
		Where(fromSender(req)...).
		Group(classification).
		Order(keel.Count().Desc(), keel.Asc(classification))
}

func senderSummaryQuery(
	req *triageports.SenderStatsRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return messages().
		Pick(
			keel.PercentileCont(0.5, keel.DynOf(m.ReviewedAt).Sub(m.ReceivedAt).Asc()).Filter(
				keel.And(settledByPerson(), keel.Int64Of(m.ReviewedAt).GteExpr(m.ReceivedAt)),
			),
			text(keel.Mode(m.MatchedCustomerID.Asc())),
			text(keel.Mode(m.MatchedCarrierID.Asc())),
		).
		Where(fromSender(req)...)
}

func senderRecentQuery(
	req *triageports.SenderStatsRequest,
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return withReference(messages()).
		Pick(messageRowColumns(keel.Bool(false))...).
		Where(fromSender(req)...).
		Where(keel.When(req.ExcludeID.IsNotNil(), m.ID.Ne(req.ExcludeID))).
		Order(m.ReceivedAt.Desc(), m.ID.Desc()).
		Limit(req.RecentLimit)
}

func (r *repository) SenderParty(
	ctx context.Context,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.SenderParty, error) {
		db := r.db.Keel(ctx)

		if req.CustomerID.IsNotNil() {
			party, found, err := customerParty(ctx, db, req)
			if err != nil || found {
				return party, err
			}
		}
		if req.CarrierID.IsNotNil() {
			party, found, err := carrierParty(ctx, db, req)
			if err != nil || found {
				return party, err
			}
		}

		return driverParty(ctx, db, req)
	})
}

func customerParty(
	ctx context.Context,
	db *keel.DB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, bool, error) {
	type customerRow struct {
		Broker    bool
		CreatedAt int64
	}

	c := customerdb.Customers
	rows, err := keel.Into[customerRow](ctx, db, c.
		Pick(c.BrokerVettingEnabled, c.CreatedAt).
		Where(c.ID.Eq(req.CustomerID)).
		Limit(1))
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}

	role := inboundmessageports.SenderRoleCustomer
	if rows[0].Broker {
		role = inboundmessageports.SenderRoleBroker
	}

	return &triageports.SenderParty{Role: role, Since: rows[0].CreatedAt}, true, nil
}

func carrierParty(
	ctx context.Context,
	db *keel.DB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, bool, error) {
	c := carrierdb.Carriers
	since, err := keel.Into[int64](
		ctx,
		db,
		c.Pick(c.CreatedAt).Where(c.ID.Eq(req.CarrierID)).Limit(1),
	)
	if err != nil {
		return nil, false, err
	}
	if len(since) == 0 {
		return nil, false, nil
	}

	return &triageports.SenderParty{
		Role:  inboundmessageports.SenderRoleCarrier,
		Since: since[0],
	}, true, nil
}

func driverParty(
	ctx context.Context,
	db *keel.DB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, error) {
	unknown := &triageports.SenderParty{Role: inboundmessageports.SenderRoleUnknown}
	address := stringutils.NormalizeEmailAddress(req.Address)
	if address == "" && req.SenderDomain == "" {
		return unknown, nil
	}

	w := workerdb.Workers
	email := w.Email.Trim().Lower()
	exists, err := w.Where(
		w.Status.Eq(domaintypes.StatusActive),
		keel.Or(
			keel.When(address != "", email.Eq(address)),
			keel.When(
				req.SenderDomain != "",
				email.Like("%@"+stringutils.EscapeLikePattern(strings.ToLower(req.SenderDomain))),
			),
		),
	).Exists(ctx, db)
	if err != nil {
		return nil, err
	}
	if !exists {
		return unknown, nil
	}

	return &triageports.SenderParty{Role: inboundmessageports.SenderRoleDriver}, nil
}
