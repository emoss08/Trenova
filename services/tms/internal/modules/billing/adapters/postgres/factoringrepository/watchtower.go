package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxAttentionRows = 500

type watchtowerReader struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWatchtowerReader(p Params) factoringports.WatchtowerReader {
	return &watchtowerReader{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-watchtower"),
	}
}

func blank(e keel.Expr) keel.Expr {
	return keel.DynOf(e).Coalesce("")
}

func intoPointers[R any](ctx context.Context, db *keel.DB, p keel.Projection) ([]*R, error) {
	rows, err := keel.Into[R](ctx, db, p)
	if err != nil {
		return nil, err
	}

	out := make([]*R, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out, nil
}

func (r *watchtowerReader) ListInvoiceLabels(
	ctx context.Context,
	req *factoringports.ListInvoiceLabelsRequest,
) (map[pulid.ID]*factoringports.InvoiceLabel, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringports.InvoiceLabel, error) {
			result := make(map[pulid.ID]*factoringports.InvoiceLabel, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			rows, err := intoPointers[factoringports.InvoiceLabel](
				ctx,
				r.db.Keel(ctx),
				invoiceLabels(req.InvoiceIDs),
			)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				result[row.InvoiceID] = row
			}

			return result, nil
		},
	)
}

func invoiceLabels(invoiceIDs []pulid.ID) *keel.Query[*invoice.Invoice] {
	inv := invoicedb.Invoices
	return inv.Pick(
		inv.ID,
		inv.Number,
		inv.CurrencyCode,
		blank(inv.Customer().Name),
		blank(inv.FactoringCompany().Name),
	).Where(inv.ID.In(invoiceIDs...)).Limit(len(invoiceIDs))
}

func (r *watchtowerReader) ListCustomersMissingNoa(
	ctx context.Context,
	req *factoringports.ListCustomersMissingNoaRequest,
) ([]*factoringports.CustomerMissingNoa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.CustomerMissingNoa, error) {
			return intoPointers[factoringports.CustomerMissingNoa](
				ctx,
				r.db.Keel(ctx),
				customersMissingNoa(req),
			)
		},
	)
}

func customersMissingNoa(
	req *factoringports.ListCustomersMissingNoaRequest,
) *keel.Query[*customer.CustomerBillingProfile] {
	cbp := customerdb.CustomerBillingProfiles
	cus := customerdb.Customers
	fac := factoringcompanydb.FactoringCompanies
	noa := factoringdb.Noas

	return cbp.Pick(
		cbp.CustomerID,
		cus.Code,
		cus.Name,
		cbp.FactoringCompanyID,
		fac.Name,
		cbp.UpdatedAt,
		blank(noa.ID),
		blank(noa.Status),
		noa.SentAt,
		noa.ExpiresAt,
		noa.CreatedAt,
	).
		Join(cus, cus.ID.EqCol(cbp.CustomerID)).
		Join(fac, fac.ID.EqCol(cbp.FactoringCompanyID)).
		LeftJoin(noa,
			noa.CustomerID.EqCol(cbp.CustomerID),
			noa.FactoringCompanyID.EqCol(cbp.FactoringCompanyID),
			noa.Status.Ne(factoring.NoaStatusReleased),
		).
		Where(
			cbp.FactoringCompanyID.IsNotNull(),
			cus.Status.Eq(domaintypes.StatusActive),
			keel.Or(
				noa.ID.IsNull(),
				noa.Status.Ne(factoring.NoaStatusAcknowledged),
				noa.ExpiresAt.Lte(req.At),
			),
		).
		Order(cbp.UpdatedAt.Asc(), cbp.CustomerID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}

func (r *watchtowerReader) ListNoasExpiringSoon(
	ctx context.Context,
	req *factoringports.ListNoasExpiringSoonRequest,
) ([]*factoringports.ExpiringNoa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoringports.ExpiringNoa, error) {
		return intoPointers[factoringports.ExpiringNoa](ctx, r.db.Keel(ctx), noasExpiringSoon(req))
	})
}

func noasExpiringSoon(req *factoringports.ListNoasExpiringSoonRequest) *keel.Query[*factoring.Noa] {
	noa := factoringdb.Noas
	return noa.Pick(
		noa.ID,
		noa.CustomerID,
		blank(noa.Customer().Name),
		noa.FactoringCompanyID,
		blank(noa.FactoringCompany().Name),
		noa.AcknowledgedAt,
		noa.ExpiresAt,
	).
		Where(
			noa.Status.Eq(factoring.NoaStatusAcknowledged),
			noa.ExpiresAt.IsNotNull(),
			noa.ExpiresAt.Gt(req.At),
			noa.ExpiresAt.Lte(req.Before),
		).
		Order(noa.ExpiresAt.Asc(), noa.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}

func (r *watchtowerReader) ListFailedSubmissions(
	ctx context.Context,
	req *factoringports.ListFailedSubmissionsRequest,
) ([]*factoringports.FailedSubmission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.FailedSubmission, error) {
			return intoPointers[factoringports.FailedSubmission](
				ctx,
				r.db.Keel(ctx),
				failedSubmissions(req.Limit),
			)
		},
	)
}

func failedSubmissions(limit int) *keel.Query[*factoring.Submission] {
	sub := factoringdb.Submissions
	return sub.Pick(
		sub.ID,
		sub.Number,
		sub.FactoringCompanyID,
		blank(sub.FactoringCompany().Name),
		sub.CurrencyCode,
		sub.InvoiceCount,
		sub.FaceMinor,
		blank(sub.TransportTarget),
		blank(sub.LastError),
		sub.FailedAt,
	).
		Where(sub.Status.Eq(factoring.SubmissionStatusFailed), sub.FailedAt.IsNotNull()).
		Order(sub.FailedAt.Asc(), sub.ID.Asc()).
		Limit(boundedLimit(limit, maxAttentionRows))
}

func (r *watchtowerReader) ListUnfundedSubmissions(
	ctx context.Context,
	req *factoringports.ListUnfundedSubmissionsRequest,
) ([]*factoringports.UnfundedSubmission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.UnfundedSubmission, error) {
			return intoPointers[factoringports.UnfundedSubmission](
				ctx,
				r.db.Keel(ctx),
				unfundedSubmissions(req),
			)
		},
	)
}

func unfundedSubmissions(
	req *factoringports.ListUnfundedSubmissionsRequest,
) *keel.Query[*factoring.Submission] {
	sub := factoringdb.Submissions
	item := factoringdb.SubmissionItems
	agr := factoringdb.Agreements
	fac := sub.FactoringCompany()
	dueAt := keel.DynOf(sub.SentAt).Add(keel.DynOf(agr.FundingDays).Mul(86400))

	return sub.Pick(
		sub.ID,
		sub.Number,
		sub.Status,
		sub.FactoringCompanyID,
		blank(fac.Name),
		sub.CurrencyCode,
		keel.Count(),
		item.ExpectedAdvanceMinor.Sum(),
		sub.SentAt,
		dueAt,
	).
		Join(item,
			item.SubmissionID.EqCol(sub.ID),
			item.Status.Eq(factoring.SubmissionItemStatusPending),
		).
		Join(agr, agr.ID.EqCol(sub.AgreementID)).
		Where(
			sub.Status.In(factoring.AwaitingFundingStatuses()...),
			sub.SentAt.IsNotNull(),
			keel.DynOf(dueAt).Lt(req.At),
		).
		Group(sub.ID, sub.OrganizationID, sub.BusinessUnitID, fac.Name, agr.FundingDays).
		Order(keel.Asc(dueAt), sub.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}

func (r *watchtowerReader) ListChargebacksDue(
	ctx context.Context,
	req *factoringports.ListChargebacksDueRequest,
) ([]*factoringports.ChargebackDue, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.ChargebackDue, error) {
			return intoPointers[factoringports.ChargebackDue](
				ctx,
				r.db.Keel(ctx),
				chargebacksDueSoon(req),
			)
		},
	)
}

func chargebacksDueSoon(
	req *factoringports.ListChargebacksDueRequest,
) *keel.Query[*factoring.SubmissionItem] {
	item := factoringdb.SubmissionItems
	sub := item.Submission()
	agr := sub.Agreement()
	deadline := keel.DynOf(item.FundedOn).Add(keel.DynOf(agr.ChargebackWindowDays).Mul(86400))

	return item.Pick(
		item.ID,
		item.SubmissionID,
		sub.Number,
		item.InvoiceID,
		item.InvoiceNumber,
		item.CustomerID,
		blank(item.Customer().Name),
		sub.FactoringCompanyID,
		blank(sub.FactoringCompany().Name),
		sub.CurrencyCode,
		item.FaceMinor,
		keel.DynOf(item.AdvanceMinor).Coalesce(0).Add(keel.DynOf(item.FeeMinor).Coalesce(0)),
		item.FundedOn,
		deadline,
	).
		Where(
			sub.ID.IsNotNull(),
			item.Status.Eq(factoring.SubmissionItemStatusFunded),
			item.FundedOn.IsNotNull(),
			agr.Recourse.Eq(factoring.RecourseFull),
			keel.DynOf(deadline).Lte(req.Before),
			keel.When(req.UnwarnedOnly, item.ChargebackWarnedAt.IsNull()),
		).
		Order(keel.Asc(deadline), item.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}
