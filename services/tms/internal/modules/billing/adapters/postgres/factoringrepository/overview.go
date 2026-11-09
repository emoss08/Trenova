package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger/factoringledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	maxOverviewPace      = 50
	maxOverviewAttention = 50
	overviewAwaiting     = 3
	overviewRemittances  = 3
)

var overviewErrors = dberror.Mapping{Entity: "FactoringCompany"}

type overviewRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOverviewRepository(p Params) factoringports.OverviewRepository {
	return &overviewRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-overview"),
	}
}

type factorRow struct {
	Name     string
	Status   domaintypes.Status
	Timezone string
}

type movedRow struct {
	Kind      factoringledger.ReserveEntryKind
	Invoices  int
	CashMinor int64
}

func (r *overviewRepository) Get(
	ctx context.Context,
	req *factoringports.GetOverviewRequest,
) (*factoringports.OverviewRead, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringports.OverviewRead, error) {
		var (
			factor      []factorRow
			summary     summaryRead
			moved       []movedRow
			awaiting    []factoringports.AwaitingSubmission
			awaitingN   int64
			overdue     []factoringwatchtowerports.UnfundedSubmission
			failed      []factoringwatchtowerports.FailedSubmission
			funded      []factoringports.FundedSubmission
			remittances []factoringports.OpenRemittance
			remitN      int64
			notices     factoringports.NoticeCounts
			transports  []*factoringsubmission.Transport
			agreements  []*factoring.Agreement
			soonestRows []factoringwatchtowerports.ChargebackDue
		)

		summaryReq := &factoringports.GetSummaryRequest{
			TenantInfo:         req.TenantInfo,
			FactoringCompanyID: req.FactoringCompanyID,
			ReadyBefore:        req.ReadyBefore,
			ChargebackBefore:   req.ChargebackBefore,
		}
		attention := boundedLimit(req.AttentionLimit, maxOverviewAttention)
		items := append(
			[]keel.BatchItem{keel.IntoSlice(overviewFactor(req), &factor)},
			summary.items(summaryReq)...,
		)
		items = append(
			items,
			keel.IntoSlice(reserveMoves(req), &moved),
			keel.IntoSlice(chargebacksDueSoon(&factoringwatchtowerports.ListChargebacksDueRequest{
				TenantInfo:         req.TenantInfo,
				FactoringCompanyID: req.FactoringCompanyID,
				Before:             req.ChargebackBefore,
				Limit:              1,
			}), &soonestRows),
			keel.IntoSlice(awaitingSubmissions(req.FactoringCompanyID), &awaiting),
			keel.CountInto(awaitingScope(req.FactoringCompanyID), &awaitingN),
			keel.IntoSlice(
				unfundedSubmissions(&factoringwatchtowerports.ListUnfundedSubmissionsRequest{
					TenantInfo:         req.TenantInfo,
					FactoringCompanyID: req.FactoringCompanyID,
					At:                 req.At,
					Limit:              attention,
				}),
				&overdue,
			),
			keel.IntoSlice(failedSubmissions(&factoringwatchtowerports.ListFailedSubmissionsRequest{
				TenantInfo:         req.TenantInfo,
				FactoringCompanyID: req.FactoringCompanyID,
				Limit:              attention,
			}), &failed),
			keel.IntoSlice(fundedSubmissions(req), &funded),
			keel.IntoSlice(factorRemittances(req.FactoringCompanyID), &remittances),
			keel.CountInto(openRemittanceScope(req.FactoringCompanyID), &remitN),
			keel.IntoValue(noticeCounts(req.FactoringCompanyID, req.At), &notices),
			keel.ListInto(transportFor(req.FactoringCompanyID), &transports),
			keel.ListInto(agreementInForce(req), &agreements),
		)

		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			r.l.Error("failed to read the factoring overview", zap.Error(err))
			return nil, err
		}
		if len(factor) == 0 {
			return nil, overviewErrors.Map(keel.ErrNotFound, req.FactoringCompanyID.String())
		}

		read := &factoringports.OverviewRead{
			Factor: factoringports.OverviewFactor{
				Name:     factor[0].Name,
				Status:   factor[0].Status,
				Timezone: factor[0].Timezone,
			},
			Summary:         summary.summary(),
			Awaiting:        pointersTo(awaiting),
			AwaitingCount:   int(awaitingN),
			Overdue:         pointersTo(overdue),
			Failed:          pointersTo(failed),
			Funded:          pointersTo(funded),
			Remittances:     pointersTo(remittances),
			RemittanceCount: int(remitN),
			Notices:         notices,
		}
		if len(soonestRows) > 0 {
			read.SoonestChargeback = &soonestRows[0]
		}
		for _, row := range moved {
			totals := factoringports.VerdictTotals{Count: row.Invoices, AmountMinor: row.CashMinor}
			switch row.Kind {
			case factoringledger.ReserveEntryRelease:
				read.Released = totals
			case factoringledger.ReserveEntryChargeback:
				totals.AmountMinor = -totals.AmountMinor
				read.ChargedBack = totals
			case factoringledger.ReserveEntryFunding,
				factoringledger.ReserveEntryChargebackReversal:
			}
		}
		if len(transports) > 0 {
			read.Transport = transports[0]
		}
		if len(agreements) > 0 {
			read.Agreement = agreements[0]
		}

		return read, nil
	})
}

func pointersTo[R any](rows []R) []*R {
	out := make([]*R, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out
}

func overviewFactor(req *factoringports.GetOverviewRequest) keel.Projection {
	fac := factoringcompanydb.FactoringCompanies
	org := tenantdb.Organizations
	return fac.Pick(
		fac.Name,
		fac.Status,
		blank(org.Timezone),
	).
		LeftJoin(org, org.ID.EqCol(fac.OrganizationID)).
		Where(fac.ID.Eq(req.FactoringCompanyID)).
		Limit(1)
}

func reserveMoves(req *factoringports.GetOverviewRequest) keel.Projection {
	re := factoringledgerdb.ReserveEntries
	return re.Group(re.Kind).
		Pick(
			re.Kind,
			re.InvoiceID.CountDistinct(),
			keel.NumericOf(re.CashMinor.Sum()).Coalesce(0),
		).
		Where(
			re.FactoringCompanyID.Eq(req.FactoringCompanyID),
			re.OccurredOn.Gte(req.MovedSince),
			re.Kind.In(factoringledger.ReserveEntryRelease, factoringledger.ReserveEntryChargeback),
		)
}

func awaitingScope(factorID pulid.ID) keel.Query[*factoringsubmission.Submission] {
	sub := factoringsubmissiondb.Submissions
	return sub.Where(
		sub.FactoringCompanyID.Eq(factorID),
		sub.Status.In(factoringsubmission.AwaitingFundingStatuses()...),
	)
}

func awaitingSubmissions(factorID pulid.ID) keel.Projection {
	sub := factoringsubmissiondb.Submissions
	agr := factoringdb.Agreements
	return sub.Pick(
		sub.ID,
		sub.Number,
		sub.InvoiceCount,
		sub.FaceMinor,
		sub.CurrencyCode,
		sub.SentAt.Coalesce(0),
		sub.SentAt.Coalesce(0).Add(agr.FundingDays.Mul(keel.Int64(86400))),
	).
		Join(agr, agr.ID.EqCol(sub.AgreementID)).
		Where(
			sub.FactoringCompanyID.Eq(factorID),
			sub.Status.In(factoringsubmission.AwaitingFundingStatuses()...),
			sub.SentAt.IsNotNull(),
		).
		Order(sub.SentAt.Desc(), sub.ID.Desc()).
		Limit(overviewAwaiting)
}

func fundedSubmissions(req *factoringports.GetOverviewRequest) keel.Projection {
	sub := factoringsubmissiondb.Submissions
	return sub.Pick(
		sub.ID,
		sub.Number,
		sub.SentAt.Coalesce(0),
		sub.FundedAt.Coalesce(0),
	).
		Where(
			sub.FactoringCompanyID.Eq(req.FactoringCompanyID),
			sub.FundedAt.IsNotNull(),
			sub.SentAt.IsNotNull(),
		).
		Order(sub.FundedAt.Desc(), sub.ID.Desc()).
		Limit(boundedLimit(req.PaceLimit, maxOverviewPace))
}

func openRemittanceStatuses() []factoringremittance.RemittanceStatus {
	return []factoringremittance.RemittanceStatus{
		factoringremittance.RemittanceStatusNeedsMapping,
		factoringremittance.RemittanceStatusParsed,
		factoringremittance.RemittanceStatusPartiallyApplied,
	}
}

func openRemittanceScope(factorID pulid.ID) keel.Query[*factoringremittance.Remittance] {
	rem := factoringremittancedb.Remittances
	return rem.Where(
		rem.Status.In(openRemittanceStatuses()...),
		rem.FactoringCompanyID.Eq(factorID),
	)
}

func factorRemittances(factorID pulid.ID) keel.Projection {
	rem := factoringremittancedb.Remittances
	return rem.Pick(
		rem.ID,
		blank(rem.FileName),
		rem.Status,
		rem.RemittedOn,
		rem.CreatedAt,
		rem.LineCount,
		rem.MatchedCount,
		rem.ExceptionCount,
		rem.TotalCashMinor,
		rem.CurrencyCode,
	).
		Where(
			rem.Status.In(openRemittanceStatuses()...),
			rem.FactoringCompanyID.Eq(factorID),
		).
		Order(rem.CreatedAt.Desc(), rem.ID.Desc()).
		Limit(overviewRemittances)
}

func noticeCounts(factorID pulid.ID, at int64) keel.Query[*customer.CustomerBillingProfile] {
	cbp := customerdb.CustomerBillingProfiles
	noa := factoringdb.Noas
	expired := keel.And(
		noa.Status.Eq(factoring.NoaStatusAcknowledged),
		noa.ExpiresAt.IsNotNull(),
		noa.ExpiresAt.Lte(at),
	)
	standing := keel.And(
		noa.Status.Eq(factoring.NoaStatusAcknowledged),
		keel.Or(noa.ExpiresAt.IsNull(), noa.ExpiresAt.Gt(at)),
	)

	return cbp.Pick(
		keel.Count(),
		keel.Count().Filter(keel.Or(noa.ID.IsNull(), expired)),
		keel.Count().Filter(noa.Status.In(factoring.NoaStatusDraft, factoring.NoaStatusSent)),
		keel.Count().Filter(standing),
	).
		LeftJoin(noa,
			noa.CustomerID.EqCol(cbp.CustomerID),
			noa.FactoringCompanyID.EqCol(cbp.FactoringCompanyID),
			noa.Status.Ne(factoring.NoaStatusReleased),
		).
		Where(cbp.FactoringCompanyID.Eq(factorID))
}

func transportFor(factorID pulid.ID) keel.Query[*factoringsubmission.Transport] {
	tr := factoringsubmissiondb.Transports
	return tr.Where(tr.FactoringCompanyID.Eq(factorID)).Limit(1)
}

func agreementInForce(req *factoringports.GetOverviewRequest) keel.Query[*factoring.Agreement] {
	a := factoringdb.Agreements
	return withChildren().
		Where(inForce(req.At), a.FactoringCompanyID.Eq(req.FactoringCompanyID)).
		Order(a.EffectiveFrom.Desc(), a.ID.Desc()).
		Limit(1)
}

func (r *overviewRepository) AgreementActivity(
	ctx context.Context,
	req *factoringports.GetAgreementActivityRequest,
) (*factoringports.AgreementActivity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringports.AgreementActivity, error) {
			activity, err := keel.IntoOne[factoringports.AgreementActivity](
				ctx,
				r.db.Keel(ctx),
				agreementActivity(req.AgreementID),
			)
			if err != nil {
				r.l.Error("failed to read the agreement's activity", zap.Error(err))
				return nil, err
			}

			return &activity, nil
		},
	)
}

func agreementActivity(agreementID pulid.ID) keel.Projection {
	sub := factoringsubmissiondb.Submissions
	sent := sub.SentAt.IsNotNull()
	funded := sub.Status.In(
		factoringsubmission.SubmissionStatusFunded,
		factoringsubmission.SubmissionStatusPartially,
	)
	return sub.Pick(
		keel.Count().Filter(sent),
		keel.Count().Filter(funded),
		keel.NumericOf(sub.InvoiceCount.Sum().Filter(sent)).Coalesce(0),
		keel.NumericOf(sub.FaceMinor.Sum().Filter(sent)).Coalesce(0),
		keel.NumericOf(sub.ExpectedAdvanceMinor.Sum().Filter(sent)).Coalesce(0),
		keel.NumericOf(sub.ExpectedFeeMinor.Sum().Filter(sent)).Coalesce(0),
		sub.SentAt.Min(),
		sub.SentAt.Max(),
	).Where(
		sub.AgreementID.Eq(agreementID),
		sub.Status.Ne(factoringsubmission.SubmissionStatusCancelled),
	)
}
