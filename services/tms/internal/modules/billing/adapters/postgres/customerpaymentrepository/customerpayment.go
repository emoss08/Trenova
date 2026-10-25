//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package customerpaymentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultMatchCandidateLimit = 10

var paymentErrors = dberror.Mapping{Entity: "CustomerPayment"}

var creditMemoApplicationErrors = dberror.Mapping{Entity: "CreditMemoApplication"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) customerpaymentports.CustomerPaymentRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-payment-repository")}
}

func orderedApplications() keel.Loader {
	apps := customerpaymentdb.Payments.Applications()
	return apps.Order(apps.LineNumber.Asc())
}

func (r *repository) List(
	ctx context.Context,
	req *customerpaymentports.ListCustomerPaymentsRequest,
) (*pagination.ListResult[*customerpayment.Payment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*customerpayment.Payment], error) {
			cp := customerpaymentdb.Payments
			q := cp.Where(
				keel.When(req.Filter.Query != "", keel.Or(
					cp.ReferenceNumber.ContainsFold(req.Filter.Query),
					cp.Memo.ContainsFold(req.Filter.Query),
				)),
				keel.When(req.CustomerID.IsNotNil(), cp.CustomerID.Eq(req.CustomerID)),
				keel.When(req.Status != "", cp.Status.Eq(req.Status)),
			)
			counted := q.Clone()
			rows := q.Load(orderedApplications()).
				Order(cp.CreatedAt.Desc()).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())

			var items []*customerpayment.Payment
			var total int64
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountInto(counted, &total),
			); err != nil {
				return nil, fmt.Errorf("list customer payments: %w", err)
			}

			return &pagination.ListResult[*customerpayment.Payment]{
				Items: items,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *customerpaymentports.ListCustomerPaymentConnectionRequest,
) (*pagination.CursorListResult[*customerpayment.Payment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*customerpayment.Payment], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				customerpaymentdb.Payments.Query(),
				&listquery.Request[customerpayment.Payment]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[customerpayment.Payment],
					) *keel.Query[customerpayment.Payment] {
						return q.Load(orderedApplications())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list customer payments", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req customerpaymentports.GetCustomerPaymentByIDRequest,
) (*customerpayment.Payment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customerpayment.Payment, error) {
		cp := customerpaymentdb.Payments
		entity, err := cp.Where(cp.ID.Eq(req.ID)).
			Load(orderedApplications()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, paymentErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func unmatchedPayment(cp *customerpaymentdb.PaymentsTable) keel.Pred {
	br := bankreceiptdb.BankReceipts.As("br")
	return keel.NotExists(br.Where(
		br.MatchedCustomerPaymentID.EqCol(cp.ID),
		br.Status.Eq(bankreceipt.StatusMatched),
	))
}

func (r *repository) FindMatchCandidates(
	ctx context.Context,
	req customerpaymentports.FindCustomerPaymentMatchCandidatesRequest,
) ([]*customerpayment.Payment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customerpayment.Payment, error) {
		cp := customerpaymentdb.Payments
		items, err := cp.Where(
			cp.Status.Eq(customerpayment.StatusPosted),
			cp.ReferenceNumber.Eq(req.ReferenceNumber),
			cp.AmountMinor.Eq(req.AmountMinor),
			unmatchedPayment(cp),
		).
			Order(cp.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("find customer payment match candidates: %w", err)
		}
		return items, nil
	})
}

func (r *repository) FindSuggestedMatchCandidates(
	ctx context.Context,
	req customerpaymentports.FindCustomerPaymentMatchCandidatesRequest,
) ([]*customerpayment.Payment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customerpayment.Payment, error) {
		cp := customerpaymentdb.Payments
		limit := req.Limit
		if limit <= 0 {
			limit = defaultMatchCandidateLimit
		}
		items, err := cp.Where(
			cp.Status.Eq(customerpayment.StatusPosted),
			unmatchedPayment(cp),
			keel.When(req.ReferenceNumber != "", keel.Or(
				cp.ReferenceNumber.Eq(req.ReferenceNumber),
				cp.ReferenceNumber.ContainsFold(req.ReferenceNumber),
			)),
			keel.When(req.AmountMinor > 0, keel.Or(
				cp.AmountMinor.Eq(req.AmountMinor),
				keel.DynFunc("abs", keel.DynOf(cp.AmountMinor).Sub(req.AmountMinor)).Lte(100),
			)),
			keel.When(
				req.ReceiptDate > 0,
				keel.DynFunc("abs", keel.DynOf(cp.PaymentDate).Sub(req.ReceiptDate)).Lte(604800),
			),
		).
			Order(cp.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("find suggested customer payment matches: %w", err)
		}
		return items, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *customerpayment.Payment,
) (*customerpayment.Payment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*customerpayment.Payment, error) {
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("cpay_")
		}
		entity.SyncAmounts()
		assignApplicationFields(entity)
		db := r.db.Keel(ctx)
		if err := customerpaymentdb.Payments.Create(ctx, db, entity); err != nil {
			return nil, paymentErrors.Map(err, entity.ID.String())
		}
		if err := r.insertApplications(ctx, db, entity); err != nil {
			return nil, err
		}
		return r.GetByID(ctx, customerpaymentports.GetCustomerPaymentByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
	})
}

func (r *repository) insertApplications(
	ctx context.Context,
	db *keel.DB,
	entity *customerpayment.Payment,
) error {
	apps := make([]*customerpayment.Application, 0, len(entity.Applications))
	for _, app := range entity.Applications {
		if app != nil {
			apps = append(apps, app)
		}
	}
	if len(apps) == 0 {
		return nil
	}
	if err := customerpaymentdb.Applications.CreateMany(ctx, db, apps); err != nil {
		return fmt.Errorf("create customer payment applications: %w", err)
	}
	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *customerpayment.Payment,
) (*customerpayment.Payment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*customerpayment.Payment, error) {
		entity.SyncAmounts()
		assignApplicationFields(entity)
		cp := customerpaymentdb.Payments
		db := r.db.Keel(ctx)
		if err := cp.Save(
			ctx,
			db,
			entity,
			cp.AppliedAmountMinor,
			cp.UnappliedAmountMinor,
			cp.Status,
			cp.PostedBatchID,
			cp.ReversalBatchID,
			cp.ReversedByID,
			cp.ReversedAt,
			cp.ReversalReason,
			cp.UpdatedByID,
		); err != nil {
			return nil, paymentErrors.Map(err, entity.ID.String())
		}
		if entity.Applications != nil {
			cpa := customerpaymentdb.Applications
			if _, err := cpa.Where(cpa.CustomerPaymentID.Eq(entity.ID)).
				Delete(ctx, db); err != nil {
				return nil, fmt.Errorf("replace customer payment applications: %w", err)
			}
			if err := r.insertApplications(ctx, db, entity); err != nil {
				return nil, err
			}
		}
		return r.GetByID(ctx, customerpaymentports.GetCustomerPaymentByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
	})
}

func assignApplicationFields(entity *customerpayment.Payment) {
	for idx, app := range entity.Applications {
		if app == nil {
			continue
		}
		app.OrganizationID = entity.OrganizationID
		app.BusinessUnitID = entity.BusinessUnitID
		app.CustomerPaymentID = entity.ID
		app.LineNumber = idx + 1
	}
}

func (r *repository) SumReceived(
	ctx context.Context,
	req customerpaymentports.SumPaymentsReceivedRequest,
) ([]*customerpaymentports.PaymentsReceived, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerpaymentports.PaymentsReceived, error) {
			cp := customerpaymentdb.Payments
			rows, err := keel.Into[customerpaymentports.PaymentsReceived](ctx, r.db.Keel(ctx), cp.
				Pick(
					cp.CurrencyCode,
					keel.DynOf(cp.AmountMinor.Sum()).Coalesce(0),
					keel.Count(),
				).
				Where(
					cp.Status.Eq(customerpayment.StatusPosted),
					cp.PaymentDate.Gte(req.From),
					cp.PaymentDate.Lte(req.To),
				).
				Group(cp.CurrencyCode))
			if err != nil {
				r.l.Error("failed to sum payments received", zap.Error(err))
				return nil, fmt.Errorf("sum payments received: %w", err)
			}

			received := make([]*customerpaymentports.PaymentsReceived, 0, len(rows))
			for i := range rows {
				received = append(received, &rows[i])
			}
			return received, nil
		},
	)
}

func (r *repository) StampExchangeRate(
	ctx context.Context,
	req *kernelports.StampExchangeRateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cp := customerpaymentdb.Payments
		n, err := cp.Where(cp.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			cp.ExchangeRate.To(decimal.NewNullDecimal(req.Rate)),
			cp.ExchangeRateDate.To(req.Date),
		)
		if err != nil {
			return fmt.Errorf("stamp customer payment exchange rate: %w", err)
		}

		return dberror.CheckFoundCount(n, "Customer payment")
	})
}
