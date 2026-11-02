package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const assistIssueAlias = "ast_iss"

var briefErrors = dberror.Mapping{Entity: "Billing queue brief"}

func (r *repository) RecordAgentReview(
	ctx context.Context,
	req *billingqueueports.RecordAgentReviewRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		bq := billingqueuedb.BillingQueueItems
		verdict := bq.AgentVerdict.To(req.Verdict)
		if req.Verdict == "" {
			verdict = bq.AgentVerdict.ToNull()
		}
		summary := bq.AgentSummary.To(req.Summary)
		if req.Summary == "" {
			summary = bq.AgentSummary.ToNull()
		}
		reads := bq.AgentReads.To(req.Reads)
		if len(req.Reads) == 0 {
			reads = bq.AgentReads.ToNull()
		}
		_, err := bq.Where(bq.ID.Eq(req.ItemID)).Update(
			ctx,
			r.db.Keel(ctx),
			verdict,
			summary,
			reads,
			bq.AgentReviewedAt.To(req.ReviewedAt),
			bq.AgentReviewHash.To(req.InputHash),
			bq.Version.ToExpr(bq.Version),
			bq.UpdatedAt.ToExpr(bq.UpdatedAt),
		)

		return itemErrors.Map(err, req.ItemID.String())
	})
}

type openIssueCountRow struct {
	Code  billingqueue.IssueCode
	Count int
}

func (r *repository) ListOpenIssueCounts(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*billingqueueports.OpenIssueCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.OpenIssueCount, error) {
			bq := billingqueuedb.BillingQueueItems
			iss := billingqueuedb.Issues.As(assistIssueAlias)
			rows, err := keel.Into[openIssueCountRow](
				ctx,
				r.db.Keel(ctx),
				iss.Pick(iss.Code, keel.Count()).
					Join(
						bq,
						bq.ID.EqCol(iss.ItemID),
						bq.OrganizationID.EqCol(iss.OrganizationID),
						bq.BusinessUnitID.EqCol(iss.BusinessUnitID),
					).
					Where(
						iss.ResolutionKey.IsNull(),
						iss.Code.NotIn(billingqueue.SuggestionIssueCodes()...),
						bq.Lane.In(billingqueue.ReviewLanes()...),
					).
					Group(iss.Code),
			)
			if err != nil {
				return nil, err
			}

			out := make([]*billingqueueports.OpenIssueCount, 0, len(rows))
			for _, row := range rows {
				out = append(
					out,
					&billingqueueports.OpenIssueCount{Code: row.Code, Count: row.Count},
				)
			}

			return out, nil
		},
	)
}

type agentConcernRow struct {
	ID        pulid.ID
	Number    string
	ProNumber string
	Summary   string
}

func (r *repository) ListAgentConcerns(
	ctx context.Context,
	_ pagination.TenantInfo,
	limit int,
) ([]*billingqueueports.AgentConcern, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.AgentConcern, error) {
			bq := billingqueuedb.BillingQueueItems
			sp := shipmentdb.Shipments
			rows, err := keel.Into[agentConcernRow](
				ctx,
				r.db.Keel(ctx),
				bq.Pick(bq.ID, bq.Number, sp.ProNumber.Coalesce(""), bq.AgentSummary).
					LeftJoin(
						sp,
						sp.ID.EqCol(bq.ShipmentID),
						sp.OrganizationID.EqCol(bq.OrganizationID),
						sp.BusinessUnitID.EqCol(bq.BusinessUnitID),
					).
					Where(
						bq.Lane.In(billingqueue.ReviewLanes()...),
						bq.AgentVerdict.Eq(billingqueue.AgentVerdictConcern),
						bq.AgentSummary.IsNotNull(),
					).
					Order(bq.CreatedAt.Asc(), bq.ID.Asc()).
					Limit(limit),
			)
			if err != nil {
				return nil, err
			}

			out := make([]*billingqueueports.AgentConcern, 0, len(rows))
			for _, row := range rows {
				shipmentRef := row.ProNumber
				if shipmentRef == "" {
					shipmentRef = row.Number
				}
				out = append(out, &billingqueueports.AgentConcern{
					ItemID:   row.ID,
					Shipment: shipmentRef,
					Summary:  row.Summary,
				})
			}

			return out, nil
		},
	)
}

func (r *repository) GetBrief(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*billingqueue.QueueBrief, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingqueue.QueueBrief, error) {
		brief, err := billingqueuedb.QueueBriefs.Query().One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, briefErrors.Map(err, "")
		}

		return brief, nil
	})
}

func (r *repository) SaveBrief(ctx context.Context, brief *billingqueue.QueueBrief) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		qb := billingqueuedb.QueueBriefs
		n, err := qb.Where(
			qb.OrganizationID.Eq(brief.OrganizationID),
			qb.BusinessUnitID.Eq(brief.BusinessUnitID),
		).Update(
			ctx,
			r.db.Keel(ctx),
			qb.Text.To(brief.Text),
			qb.InputHash.To(brief.InputHash),
			qb.GeneratedAt.To(brief.GeneratedAt),
		)
		if err != nil {
			return briefErrors.Map(err, "")
		}
		if n > 0 {
			return nil
		}

		_, err = qb.CreateOrSkip(ctx, r.db.Keel(ctx), []*billingqueue.QueueBrief{brief},
			qb.OrganizationID, qb.BusinessUnitID)

		return briefErrors.Map(err, "")
	})
}

func (r *repository) RecentCustomerInvoices(
	ctx context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	limit int,
) ([]*billingqueueports.RecentInvoice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.RecentInvoice, error) {
			inv := invoicedb.Invoices
			rows, err := inv.Pick(
				inv.ID,
				inv.Number,
				inv.InvoiceDate,
				inv.SubtotalAmount,
				inv.OtherAmount,
				inv.TotalAmount,
			).
				Where(
					inv.CustomerID.Eq(customerID),
					inv.Status.Eq(invoice.StatusPosted),
					inv.BillType.Eq(billingvocab.BillTypeInvoice),
					inv.IsAdjustmentArtifact.IsFalse(),
				).
				Order(inv.InvoiceDate.Desc(), inv.ID.Desc()).
				Limit(limit).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			out := make([]*billingqueueports.RecentInvoice, 0, len(rows))
			byID := make(map[pulid.ID]*billingqueueports.RecentInvoice, len(rows))
			ids := make([]pulid.ID, 0, len(rows))
			for _, row := range rows {
				recent := &billingqueueports.RecentInvoice{
					ID:          row.ID,
					Number:      row.Number,
					InvoiceDate: row.InvoiceDate,
					Subtotal:    row.SubtotalAmount,
					Other:       row.OtherAmount,
					Total:       row.TotalAmount,
				}
				out = append(out, recent)
				byID[row.ID] = recent
				ids = append(ids, row.ID)
			}
			if len(ids) == 0 {
				return out, nil
			}

			charges, err := r.recentCharges(ctx, ids)
			if err != nil {
				return nil, err
			}
			for _, charge := range charges {
				if recent := byID[charge.InvoiceID]; recent != nil {
					recent.Charges = append(recent.Charges, &billingqueueports.RecentCharge{
						Type:        charge.Type,
						Code:        charge.Code,
						Description: charge.Description,
						Amount:      charge.Amount,
					})
				}
			}

			return out, nil
		},
	)
}

type recentChargeRow struct {
	InvoiceID   pulid.ID
	Type        invoice.InvoiceLineType
	Code        string
	Description string
	Amount      decimal.Decimal
}

func (r *repository) recentCharges(ctx context.Context, invoiceIDs []pulid.ID) ([]recentChargeRow, error) {
	il := invoicedb.InvoiceLines

	return keel.Into[recentChargeRow](ctx, r.db.Keel(ctx), il.
		Pick(il.InvoiceID, il.Type, il.ChargeCode.Coalesce(""), il.Description.Min(), il.Amount.Sum()).
		Where(il.InvoiceID.In(invoiceIDs...)).
		Group(il.InvoiceID, il.Type, il.ChargeCode).
		Order(il.InvoiceID.Asc(), il.Type.Asc(), il.ChargeCode.Asc()))
}
