package factoringrepository

import (
	"context"

	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	invoiceStateEntity = "FactoringInvoiceState"
	maxStateBatch      = 500
	maxCandidateBatch  = 1000
)

var (
	invoiceStateErrors = dberror.Mapping{Entity: invoiceStateEntity}
	invoiceErrors      = dberror.Mapping{Entity: "Invoice"}
)

type invoiceStateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewInvoiceStateRepository(p Params) factoringports.InvoiceStateRepository {
	return &invoiceStateRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-invoice-state"),
	}
}

func (r *invoiceStateRepository) SetInvoiceOverride(
	ctx context.Context,
	req *factoringports.SetInvoiceOverrideRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		inv := invoicedb.Invoices
		n, err := inv.Where(inv.ID.Eq(req.InvoiceID)).Update(ctx, r.db.Keel(ctx),
			inv.FactoringOverride.To(req.Override),
			inv.FactoringOverrideReason.ToExpr(keel.Val(stringutils.NilIfEmpty(req.Reason))),
			inv.FactoringCompanyID.ToExpr(keel.Val(req.FactoringCompanyID)),
			inv.FactoringAgreementID.ToExpr(keel.Val(req.FactoringAgreementID)),
		)
		if err != nil {
			return invoiceErrors.Map(err, req.InvoiceID.String())
		}

		return dberror.CheckFoundCount(n, "Invoice")
	})
}

func (r *invoiceStateRepository) Get(
	ctx context.Context,
	req factoringports.GetInvoiceStateRequest,
) (*factoring.InvoiceState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.InvoiceState, error) {
		st := factoringdb.InvoiceStates
		entity, err := st.Where(st.InvoiceID.Eq(req.InvoiceID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceStateErrors.Map(err, req.InvoiceID.String())
		}

		return entity, nil
	})
}

func (r *invoiceStateRepository) ListByInvoiceIDs(
	ctx context.Context,
	req *factoringports.ListInvoiceStatesByInvoiceIDsRequest,
) (map[pulid.ID]*factoring.InvoiceState, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoring.InvoiceState, error) {
			result := make(map[pulid.ID]*factoring.InvoiceState, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			st := factoringdb.InvoiceStates
			rows, err := st.Where(st.InvoiceID.In(req.InvoiceIDs...)).
				Limit(len(req.InvoiceIDs)).
				List(ctx, r.db.Keel(ctx))
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

func (r *invoiceStateRepository) Upsert(
	ctx context.Context,
	entity *factoring.InvoiceState,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		st := factoringdb.InvoiceStates
		ready := factoring.VerdictReady
		stillReady := func(col keel.Int64Col) keel.Int64Expr {
			return keel.CaseInt64().
				When(
					keel.And(
						keel.StringOf(st.Verdict.Excluded()).Eq(string(ready)),
						st.Verdict.Eq(ready),
					),
					col,
				).
				Else(keel.Int64Of(col.Excluded()))
		}

		_, err := st.Upsert(ctx, db, []*factoring.InvoiceState{entity},
			keel.OnConflict(st.InvoiceID, st.OrganizationID, st.BusinessUnitID).
				Update(
					st.FactoringCompanyID,
					st.CustomerID,
					st.Verdict,
					st.Checks,
					st.FailedChecks,
					st.OpenBalanceMinor,
					st.EvaluatedAt,
				).
				Set(
					st.ReadySince.ToExpr(stillReady(st.ReadySince)),
					st.ReadyAnnouncedAt.ToExpr(stillReady(st.ReadyAnnouncedAt)),
				),
		)
		if err != nil {
			return invoiceStateErrors.Map(err, entity.InvoiceID.String())
		}

		stored, err := st.Where(st.InvoiceID.Eq(entity.InvoiceID)).One(ctx, db)
		if err != nil {
			return invoiceStateErrors.Map(err, entity.InvoiceID.String())
		}
		*entity = *stored

		return nil
	})
}

func (r *invoiceStateRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListInvoiceStateConnectionRequest,
) (*pagination.CursorListResult[*factoring.InvoiceState], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.InvoiceState], error) {
			filter, term := withoutQuery(req.Filter)
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				stateScope(req, term),
				&listquery.Request[factoring.InvoiceState]{
					Filter: filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list factoring invoice states", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func stateScope(
	req *factoringports.ListInvoiceStateConnectionRequest,
	term string,
) *keel.Query[*factoring.InvoiceState] {
	st := factoringdb.InvoiceStates
	return st.Where(
		keel.When(term != "", searchStates(term)),
		keel.When(len(req.Verdicts) > 0, st.Verdict.In(req.Verdicts...)),
		keel.When(
			req.FactoringCompanyID.IsNotNil(),
			st.FactoringCompanyID.Eq(req.FactoringCompanyID),
		),
		keel.When(req.CustomerID.IsNotNil(), st.CustomerID.Eq(req.CustomerID)),
	)
}

func searchStates(term string) keel.Pred {
	st := factoringdb.InvoiceStates
	cus := st.Customer()
	return keel.Or(
		st.Invoice().Number.ContainsFold(term),
		cus.Name.ContainsFold(term),
		cus.Code.ContainsFold(term),
		st.FactoringCompany().Name.ContainsFold(term),
	)
}

func (r *invoiceStateRepository) ClaimReadyAnnouncement(
	ctx context.Context,
	req *factoringports.ClaimReadyAnnouncementRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		st := factoringdb.InvoiceStates
		n, err := st.Where(
			st.InvoiceID.Eq(req.InvoiceID),
			st.Verdict.Eq(factoring.VerdictReady),
			st.ReadySince.Eq(req.ReadySince),
			st.ReadyAnnouncedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), st.ReadyAnnouncedAt.To(req.At))
		if err != nil {
			return false, invoiceStateErrors.Map(err, req.InvoiceID.String())
		}

		return n > 0, nil
	})
}

func (r *invoiceStateRepository) ListReadyUnsubmitted(
	ctx context.Context,
	req factoringports.ListReadyUnsubmittedRequest,
) ([]*factoring.InvoiceState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.InvoiceState, error) {
		st := factoringdb.InvoiceStates
		rows, err := st.Where(
			st.Verdict.Eq(factoring.VerdictReady),
			st.ReadySince.Lte(req.ReadyBefore),
		).
			Order(st.ReadySince.Asc(), st.InvoiceID.Asc()).
			Limit(boundedLimit(req.Limit, maxStateBatch)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if rows == nil {
			return []*factoring.InvoiceState{}, nil
		}

		return rows, nil
	})
}

func openFactoredInvoices(inv *invoicedb.InvoicesTable) keel.Pred {
	return keel.And(
		inv.Status.Eq(invoice.StatusPosted),
		inv.FactoringCompanyID.IsNotNull(),
		inv.BillType.Ne(billingqueue.BillTypeCreditMemo),
		keel.Int64Of(inv.TotalAmountMinor).GtExpr(inv.AppliedAmountMinor),
	)
}

func (r *invoiceStateRepository) ListCandidateInvoiceIDs(
	ctx context.Context,
	req factoringports.ListCandidateInvoicesRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), candidateInvoices(&req))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			return []pulid.ID{}, nil
		}

		return ids, nil
	})
}

func candidateInvoices(req *factoringports.ListCandidateInvoicesRequest) keel.Projection {
	limit := boundedLimit(req.Limit, maxCandidateBatch)
	inv := invoicedb.Invoices
	st := factoringdb.InvoiceStates
	after := req.AfterInvoiceID.IsNotNil()

	candidates := keel.CTE("candidates", []string{"id"}, keel.Union(
		inv.Pick(inv.ID).
			Where(openFactoredInvoices(inv), keel.When(after, inv.ID.Gt(req.AfterInvoiceID))).
			Order(inv.ID.Asc()).
			Limit(limit),
		st.Pick(st.InvoiceID).
			Where(
				st.Verdict.Ne(factoring.VerdictExcluded),
				keel.When(after, st.InvoiceID.Gt(req.AfterInvoiceID)),
			).
			Order(st.InvoiceID.Asc()).
			Limit(limit),
	))

	return candidates.Query().
		With(candidates).
		Pick(candidates.Col("id")).
		Order(candidates.Col("id").Asc()).
		Limit(limit)
}

func (r *invoiceStateRepository) ListCustomerInvoiceIDs(
	ctx context.Context,
	req *factoringports.ListCustomerInvoiceIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.CustomerIDs) == 0 && req.FactoringCompanyID.IsNil() {
			return nil, nil
		}

		inv := invoicedb.Invoices
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), inv.
			Pick(inv.ID).
			Where(
				openFactoredInvoices(inv),
				keel.When(len(req.CustomerIDs) > 0, inv.CustomerID.In(req.CustomerIDs...)),
				keel.When(
					req.FactoringCompanyID.IsNotNil(),
					inv.FactoringCompanyID.Eq(req.FactoringCompanyID),
				),
			).
			Order(inv.ID.Asc()).
			Limit(boundedLimit(req.Limit, maxCandidateBatch)))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			return []pulid.ID{}, nil
		}

		return ids, nil
	})
}

type verdictRow struct {
	Verdict     factoring.Verdict
	Count       int
	AmountMinor int64
	LateCount   int
	LateMinor   int64
}

type fundedRow struct {
	Count        int
	AmountMinor  int64
	ReserveCount int
	ReserveMinor int64
}

func (r *invoiceStateRepository) GetSummary(
	ctx context.Context,
	req factoringports.GetSummaryRequest,
) (*factoringports.Summary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringports.Summary, error) {
		var (
			verdicts []verdictRow
			funded   fundedRow
			due      factoringports.VerdictTotals
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(verdictTotals(req.ReadyBefore), &verdicts),
			keel.IntoValue(fundedTotals(), &funded),
			keel.IntoValue(chargebacksDue(req.ChargebackBefore), &due),
		)
		if err != nil {
			return nil, err
		}

		summary := &factoringports.Summary{
			Funded: factoringports.VerdictTotals{
				Count:       funded.Count,
				AmountMinor: funded.AmountMinor,
			},
			ReserveHeld: factoringports.VerdictTotals{
				Count:       funded.ReserveCount,
				AmountMinor: funded.ReserveMinor,
			},
			ChargebacksDue: due,
		}
		for _, rw := range verdicts {
			totals := factoringports.VerdictTotals{Count: rw.Count, AmountMinor: rw.AmountMinor}
			switch rw.Verdict {
			case factoring.VerdictReady:
				summary.Ready = totals
				summary.ReadyUnsubmitted = factoringports.VerdictTotals{
					Count:       rw.LateCount,
					AmountMinor: rw.LateMinor,
				}
			case factoring.VerdictNotReady:
				summary.NotReady = totals
			case factoring.VerdictSubmitted:
				summary.Submitted = totals
			case factoring.VerdictFunded, factoring.VerdictExcluded:
			}
		}

		return summary, nil
	})
}

func verdictTotals(readyBefore int64) *keel.Query[*factoring.InvoiceState] {
	st := factoringdb.InvoiceStates
	late := st.ReadySince.Lte(readyBefore)
	return st.Group(st.Verdict).
		Pick(
			st.Verdict,
			keel.Count(),
			keel.NumericOf(st.OpenBalanceMinor.Sum()).Coalesce(0),
			keel.Count().Filter(late),
			keel.NumericOf(st.OpenBalanceMinor.Sum().Filter(late)).Coalesce(0),
		).
		Where(st.Verdict.In(
			factoring.VerdictReady,
			factoring.VerdictNotReady,
			factoring.VerdictSubmitted,
		))
}

func fundedTotals() *keel.Query[*factoring.SubmissionItem] {
	item := factoringdb.SubmissionItems
	return item.Pick(
		keel.Count(),
		keel.NumericOf(item.FaceMinor.Sum()).Coalesce(0),
		keel.Count().Filter(item.ReserveMinor.Coalesce(0).Gt(0)),
		keel.NumericOf(item.ReserveMinor.Sum()).Coalesce(0),
	).Where(item.Status.Eq(factoring.SubmissionItemStatusFunded))
}

func chargebacksDue(before int64) *keel.Query[*factoring.SubmissionItem] {
	item := factoringdb.SubmissionItems
	sub := factoringdb.Submissions
	agr := factoringdb.Agreements
	return item.Pick(
		keel.Count(),
		keel.NumericOf(
			keel.Sum(item.AdvanceMinor.Coalesce(0).Add(item.FeeMinor.Coalesce(0))),
		).Coalesce(0),
	).
		Join(sub, sub.ID.EqCol(item.SubmissionID)).
		Join(agr, agr.ID.EqCol(sub.AgreementID)).
		Where(
			item.Status.Eq(factoring.SubmissionItemStatusFunded),
			item.FundedOn.IsNotNull(),
			agr.Recourse.Eq(factoring.RecourseFull),
			item.FundedOn.Add(agr.ChargebackWindowDays.Mul(keel.Int64(86400))).Lte(before),
		)
}

func boundedLimit(limit, ceiling int) int {
	if limit <= 0 || limit > ceiling {
		return ceiling
	}
	return limit
}

func withoutQuery(
	filter *pagination.QueryOptions,
) (stripped *pagination.QueryOptions, term string) {
	if filter == nil {
		return nil, ""
	}
	term = strings.TrimSpace(filter.Query)
	if term == "" {
		return filter, ""
	}
	copied := *filter
	copied.Query = ""
	return &copied, term
}
