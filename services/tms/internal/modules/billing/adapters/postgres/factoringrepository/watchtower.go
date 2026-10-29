package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
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

func NewWatchtowerReader(p Params) factoringwatchtowerports.WatchtowerReader {
	return &watchtowerReader{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-watchtower"),
	}
}

func blank(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).Coalesce("")
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
	req *factoringwatchtowerports.ListInvoiceLabelsRequest,
) (map[pulid.ID]*factoringwatchtowerports.InvoiceLabel, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringwatchtowerports.InvoiceLabel, error) {
			result := make(map[pulid.ID]*factoringwatchtowerports.InvoiceLabel, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			rows, err := intoPointers[factoringwatchtowerports.InvoiceLabel](
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
	req *factoringwatchtowerports.ListCustomersMissingNoaRequest,
) ([]*factoringwatchtowerports.CustomerMissingNoa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringwatchtowerports.CustomerMissingNoa, error) {
			return intoPointers[factoringwatchtowerports.CustomerMissingNoa](
				ctx,
				r.db.Keel(ctx),
				customersMissingNoa(req),
			)
		},
	)
}

func customersMissingNoa(
	req *factoringwatchtowerports.ListCustomersMissingNoaRequest,
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
	req *factoringwatchtowerports.ListNoasExpiringSoonRequest,
) ([]*factoringwatchtowerports.ExpiringNoa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringwatchtowerports.ExpiringNoa, error) {
			return intoPointers[factoringwatchtowerports.ExpiringNoa](
				ctx,
				r.db.Keel(ctx),
				noasExpiringSoon(req),
			)
		},
	)
}

func noasExpiringSoon(
	req *factoringwatchtowerports.ListNoasExpiringSoonRequest,
) *keel.Query[*factoring.Noa] {
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
	req *factoringwatchtowerports.ListFailedSubmissionsRequest,
) ([]*factoringwatchtowerports.FailedSubmission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringwatchtowerports.FailedSubmission, error) {
			return intoPointers[factoringwatchtowerports.FailedSubmission](
				ctx,
				r.db.Keel(ctx),
				failedSubmissions(req.Limit),
			)
		},
	)
}

func failedSubmissions(limit int) *keel.Query[*factoringsubmission.Submission] {
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
		Where(sub.Status.Eq(factoringsubmission.SubmissionStatusFailed), sub.FailedAt.IsNotNull()).
		Order(sub.FailedAt.Asc(), sub.ID.Asc()).
		Limit(boundedLimit(limit, maxAttentionRows))
}

func (r *watchtowerReader) ListUnfundedSubmissions(
	ctx context.Context,
	req *factoringwatchtowerports.ListUnfundedSubmissionsRequest,
) ([]*factoringwatchtowerports.UnfundedSubmission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringwatchtowerports.UnfundedSubmission, error) {
			return intoPointers[factoringwatchtowerports.UnfundedSubmission](
				ctx,
				r.db.Keel(ctx),
				unfundedSubmissions(req),
			)
		},
	)
}

func unfundedSubmissions(
	req *factoringwatchtowerports.ListUnfundedSubmissionsRequest,
) *keel.Query[*factoringsubmission.Submission] {
	sub := factoringdb.Submissions
	item := factoringdb.SubmissionItems
	agr := factoringdb.Agreements
	fac := sub.FactoringCompany()
	dueAt := sub.SentAt.Add(agr.FundingDays.Mul(keel.Int64(86400)))

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
			item.Status.Eq(factoringsubmission.SubmissionItemStatusPending),
		).
		Join(agr, agr.ID.EqCol(sub.AgreementID)).
		Where(
			sub.Status.In(factoringsubmission.AwaitingFundingStatuses()...),
			sub.SentAt.IsNotNull(),
			dueAt.Lt(req.At),
		).
		Group(sub.ID, sub.OrganizationID, sub.BusinessUnitID, fac.Name, agr.FundingDays).
		Order(dueAt.Asc(), sub.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}

func (r *watchtowerReader) ListChargebacksDue(
	ctx context.Context,
	req *factoringwatchtowerports.ListChargebacksDueRequest,
) ([]*factoringwatchtowerports.ChargebackDue, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringwatchtowerports.ChargebackDue, error) {
			return intoPointers[factoringwatchtowerports.ChargebackDue](
				ctx,
				r.db.Keel(ctx),
				chargebacksDueSoon(req),
			)
		},
	)
}

func chargebacksDueSoon(
	req *factoringwatchtowerports.ListChargebacksDueRequest,
) *keel.Query[*factoringsubmission.SubmissionItem] {
	item := factoringdb.SubmissionItems
	sub := item.Submission()
	agr := sub.Agreement()
	deadline := item.FundedOn.Add(agr.ChargebackWindowDays.Mul(keel.Int64(86400)))

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
		item.AdvanceMinor.Coalesce(0).Add(item.FeeMinor.Coalesce(0)),
		item.FundedOn,
		deadline,
	).
		Where(
			sub.ID.IsNotNull(),
			item.Status.Eq(factoringsubmission.SubmissionItemStatusFunded),
			item.FundedOn.IsNotNull(),
			agr.Recourse.Eq(factoring.RecourseFull),
			deadline.Lte(req.Before),
			keel.When(req.UnwarnedOnly, item.ChargebackWarnedAt.IsNull()),
		).
		Order(deadline.Asc(), item.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxAttentionRows))
}
