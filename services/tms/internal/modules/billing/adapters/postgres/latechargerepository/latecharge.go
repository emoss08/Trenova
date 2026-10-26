package latechargerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge/latechargedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultTenantLimit = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) latechargeports.LateChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("repository.late-charge"),
	}
}

type candidateRecord struct {
	CustomerID       pulid.ID
	CustomerName     string
	InvoiceID        pulid.ID
	InvoiceNumber    string
	CurrencyCode     string
	DueDate          int64
	GracePeriodDays  int
	RatePercent      decimal.Decimal
	OpenBalanceMinor int64
	AssessedPeriods  []int64
}

func candidatesQuery(asOf int64, customerIDs []pulid.ID) *keel.Query[*invoice.Invoice] {
	inv := invoicedb.Invoices
	cus := customerdb.Customers
	cbp := customerdb.CustomerBillingProfiles
	dispute := invoicedb.InvoiceDisputes
	graceEnds := inv.DueDate.Add(
		keel.CastInt64(cbp.GracePeriodDays).Mul(keel.Int64(receivablesql.SecondsPerDay)),
	)

	return inv.
		Join(cus, cus.ID.EqCol(inv.CustomerID)).
		Join(cbp, cbp.CustomerID.EqCol(cus.ID)).
		Where(
			receivablesql.Open(inv),
			inv.DueDate.IsNotNull(),
			graceEnds.Lte(asOf),
			cbp.ApplyLateCharges.Eq(true),
			cbp.LateChargeRate.IsNotNull(),
			keel.NumericOf(cbp.LateChargeRate).GtExpr(keel.ConstNumeric("0")),
			inv.DisputeStatus.Ne(invoice.DisputeStatusDisputed),
			inv.MemoKind.Ne(invoice.MemoKindLateCharge),
			keel.NotExists(dispute.Where(
				dispute.InvoiceID.EqCol(inv.ID),
				dispute.Status.Eq(invoice.DisputeCaseStatusOpen),
			)),
			keel.When(len(customerIDs) > 0, inv.CustomerID.In(customerIDs...)),
		)
}

func (r *repository) ListCandidateCustomers(
	ctx context.Context,
	req *latechargeports.ListLateChargeCustomersRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		inv := invoicedb.Invoices
		customers := candidatesQuery(req.AsOfDate, req.CustomerIDs).
			Pick(inv.CustomerID).
			Where(inv.CustomerID.Gt(req.AfterCustomerID)).
			Group(inv.CustomerID).
			Order(inv.CustomerID.Asc()).
			Limit(req.Limit)
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), customers)
		if err != nil {
			return nil, fmt.Errorf("list late charge customers: %w", err)
		}

		return ids, nil
	})
}

func (r *repository) ListCandidates(
	ctx context.Context,
	req *latechargeports.ListLateChargeCandidatesRequest,
) ([]*latechargeports.LateChargeCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*latechargeports.LateChargeCandidate, error) {
			inv := invoicedb.Invoices
			cus := customerdb.Customers
			cbp := customerdb.CustomerBillingProfiles
			lca := latechargedb.LateChargeAssessments
			assessed := keel.Scalar(lca.
				Pick(keel.ArrayAgg(lca.PeriodIndex).OrderBy(lca.PeriodIndex.Asc())).
				Where(lca.SourceInvoiceID.EqCol(inv.ID)))
			records, err := keel.Into[candidateRecord](ctx, r.db.Keel(ctx), candidatesQuery(
				req.AsOfDate,
				req.CustomerIDs,
			).
				Pick(
					cus.ID,
					cus.Name,
					inv.ID,
					inv.Number,
					inv.CurrencyCode,
					inv.DueDate,
					cbp.GracePeriodDays,
					cbp.LateChargeRate,
					receivablesql.OpenMinor(inv),
					keel.Int64ArrayOf(assessed).Coalesce(keel.Int64Array(nil)),
				).
				Order(cus.Name.Asc(), inv.DueDate.Asc(), inv.Number.Asc()))
			if err != nil {
				return nil, fmt.Errorf("list late charge candidates: %w", err)
			}

			candidates := make([]*latechargeports.LateChargeCandidate, 0, len(records))
			for _, rec := range records {
				assessed := make([]int, 0, len(rec.AssessedPeriods))
				for _, idx := range rec.AssessedPeriods {
					assessed = append(assessed, int(idx))
				}
				candidates = append(candidates, &latechargeports.LateChargeCandidate{
					CustomerID:       rec.CustomerID,
					CustomerName:     rec.CustomerName,
					InvoiceID:        rec.InvoiceID,
					InvoiceNumber:    rec.InvoiceNumber,
					CurrencyCode:     rec.CurrencyCode,
					DueDate:          rec.DueDate,
					GracePeriodDays:  rec.GracePeriodDays,
					RatePercent:      rec.RatePercent,
					OpenBalanceMinor: rec.OpenBalanceMinor,
					AssessedPeriods:  assessed,
				})
			}

			return candidates, nil
		},
	)
}

func (r *repository) InsertAssessments(
	ctx context.Context,
	assessments []*latecharge.LateChargeAssessment,
) ([]*latecharge.LateChargeAssessment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*latecharge.LateChargeAssessment, error) {
			if len(assessments) == 0 {
				return []*latecharge.LateChargeAssessment{}, nil
			}
			runKey := assessments[0].RunKey

			lca := latechargedb.LateChargeAssessments
			db := r.db.Keel(ctx)
			if _, err := lca.CreateOrSkip(
				ctx,
				db,
				assessments,
				lca.OrganizationID,
				lca.BusinessUnitID,
				lca.SourceInvoiceID,
				lca.PeriodIndex,
			); err != nil {
				return nil, fmt.Errorf("insert late charge assessments: %w", err)
			}

			rows, err := lca.Where(lca.RunKey.Eq(runKey)).
				Order(lca.SourceInvoiceID.Asc(), lca.PeriodIndex.Asc()).
				All(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("list late charge assessments by run: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) SetDebitMemoLines(
	ctx context.Context,
	assessments []*latecharge.LateChargeAssessment,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		lca := latechargedb.LateChargeAssessments
		db := r.db.Keel(ctx)
		for _, assessment := range assessments {
			if assessment == nil || assessment.DebitMemoLineID.IsNil() {
				continue
			}
			if err := lca.Save(
				ctx,
				db,
				assessment,
				lca.DebitMemoInvoiceID,
				lca.DebitMemoLineID,
			); err != nil {
				return fmt.Errorf("stamp late charge assessment memo line: %w", err)
			}
		}

		return nil
	})
}

func (r *repository) DeleteByRunKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	runKey string,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		lca := latechargedb.LateChargeAssessments
		n, err := lca.Where(lca.RunKey.Eq(runKey)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("delete late charge assessments by run: %w", err)
		}

		return n, nil
	})
}

func (r *repository) ListBySourceInvoiceIDs(
	ctx context.Context,
	req *latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	lca := latechargedb.LateChargeAssessments
	return r.listGrouped(
		ctx,
		req,
		lca.SourceInvoiceID.In(req.InvoiceIDs...),
		func(a *latecharge.LateChargeAssessment) pulid.ID {
			return a.SourceInvoiceID
		},
	)
}

func (r *repository) ListByDebitMemoIDs(
	ctx context.Context,
	req *latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	lca := latechargedb.LateChargeAssessments
	return r.listGrouped(
		ctx,
		req,
		lca.DebitMemoInvoiceID.In(req.InvoiceIDs...),
		func(a *latecharge.LateChargeAssessment) pulid.ID {
			return a.DebitMemoInvoiceID
		},
	)
}

func (r *repository) listGrouped(
	ctx context.Context,
	req *latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest,
	match keel.Pred,
	keyOf func(*latecharge.LateChargeAssessment) pulid.ID,
) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
			result := make(map[pulid.ID][]*latecharge.LateChargeAssessment, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			lca := latechargedb.LateChargeAssessments
			rows, err := lca.Where(match).
				Order(lca.AsOfDate.Desc(), lca.PeriodIndex.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list late charge assessments: %w", err)
			}

			for _, row := range rows {
				key := keyOf(row)
				result[key] = append(result[key], row)
			}

			return result, nil
		},
	)
}

func (r *repository) CountBySourceInvoiceID(
	ctx context.Context,
	_ pagination.TenantInfo,
	invoiceID pulid.ID,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		lca := latechargedb.LateChargeAssessments
		count, err := lca.Where(lca.SourceInvoiceID.Eq(invoiceID)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count late charge assessments: %w", err)
		}

		return count, nil
	})
}

type tenantRecord struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListLateChargeTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with late charges to assess")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		if limit <= 0 {
			limit = defaultTenantLimit
		}
		bc := tenantdb.BillingControls
		records, err := keel.Into[tenantRecord](ctx, r.db.Keel(ctx), bc.
			AllTenants().
			Pick(bc.OrganizationID, bc.BusinessUnitID).
			Where(bc.LateChargeAssessmentMode.Ne(tenant.LateChargeAssessmentModeDisabled)).
			Order(bc.OrganizationID.Asc()).
			Limit(limit))
		if err != nil {
			return nil, fmt.Errorf("list late charge tenants: %w", err)
		}

		tenants := make([]pagination.TenantInfo, 0, len(records))
		for _, rec := range records {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: rec.OrganizationID,
				BuID:  rec.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}
