package bankreceiptrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var bankReceiptErrors = dberror.Mapping{
	Entity: "BankReceipt",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_bank_receipts_matched_payment",
			Field:   "matchedCustomerPaymentId",
			Message: "Another bank receipt is already matched to this payment",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

type summaryRecord struct {
	ImportedCount         int64
	ImportedAmount        int64
	MatchedCount          int64
	MatchedAmount         int64
	ExceptionCount        int64
	ExceptionAmount       int64
	ActiveWorkItemCount   int64
	AssignedWorkItemCount int64
	InReviewWorkItemCount int64
	CurrentCount          int64
	CurrentAmount         int64
	Days1To3Count         int64
	Days1To3Amount        int64
	Days4To7Count         int64
	Days4To7Amount        int64
	DaysOver7Count        int64
	DaysOver7Amount       int64
}

func New(p Params) bankreceiptports.BankReceiptRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req bankreceiptports.GetBankReceiptByIDRequest,
) (*bankreceipt.BankReceipt, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bankreceipt.BankReceipt, error) {
		entity, err := bankreceiptdb.BankReceipts.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, bankReceiptErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) LockForUpdate(
	ctx context.Context,
	req bankreceiptports.GetBankReceiptByIDRequest,
) (*bankreceipt.BankReceipt, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bankreceipt.BankReceipt, error) {
		br := bankreceiptdb.BankReceipts
		entity, err := br.Where(br.ID.Eq(req.ID)).Lock(keel.ForUpdate).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, bankReceiptErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) ListMatchedToPayment(
	ctx context.Context,
	req bankreceiptports.ListBankReceiptsMatchedToPaymentRequest,
) ([]*bankreceipt.BankReceipt, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*bankreceipt.BankReceipt, error) {
		br := bankreceiptdb.BankReceipts
		items, err := br.Where(br.MatchedCustomerPaymentID.Eq(req.PaymentID)).
			Order(br.CreatedAt.Asc()).
			Lock(keel.ForUpdate).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list bank receipts matched to payment: %w", err)
		}
		return items, nil
	})
}

func (r *repository) ListByImportBatchID(
	ctx context.Context,
	req bankreceiptports.ListBankReceiptsByImportBatchRequest,
) ([]*bankreceipt.BankReceipt, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bankreceipt.BankReceipt, error) {
		br := bankreceiptdb.BankReceipts
		items, err := br.Where(br.ImportBatchID.Eq(req.BatchID)).
			Order(br.ReceiptDate.Asc(), br.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list bank receipts by import batch: %w", err)
		}
		return items, nil
	})
}

func (r *repository) ListExceptions(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*bankreceipt.BankReceipt, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bankreceipt.BankReceipt, error) {
		br := bankreceiptdb.BankReceipts
		items, err := br.Where(br.Status.Eq(bankreceipt.StatusException)).
			Order(br.ReceiptDate.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list bank receipt exceptions: %w", err)
		}
		return items, nil
	})
}

func summaryQuery(asOf int64) *keel.Query[bankreceipt.BankReceipt] {
	br := bankreceiptdb.BankReceipts
	wi := bankreceiptworkitemdb.WorkItems.As("wi")
	count := func(p keel.Pred) keel.Expr { return keel.Count().Filter(p) }
	amount := func(p keel.Pred) keel.Expr {
		return keel.DynOf(br.AmountMinor.Sum().Filter(p)).Coalesce(0)
	}
	workItems := func(statuses ...bankreceiptworkitem.Status) keel.Expr {
		return keel.Scalar(wi.Pick(keel.Count()).Where(wi.Status.In(statuses...)))
	}
	days := keel.DynLit(asOf).Sub(br.ReceiptDate).Div(86400)
	exception := br.Status.Eq(bankreceipt.StatusException)
	aged := func(band keel.Pred) keel.Pred {
		return keel.And(exception, band)
	}
	imported := br.Status.Eq(bankreceipt.StatusImported)
	matched := br.Status.Eq(bankreceipt.StatusMatched)
	current := aged(days.Lte(0))
	days1To3 := aged(days.Between(1, 3))
	days4To7 := aged(days.Between(4, 7))
	daysOver7 := aged(days.Gt(7))

	return br.
		Pick(
			count(imported), amount(imported),
			count(matched), amount(matched),
			count(exception), amount(exception),
			workItems(
				bankreceiptworkitem.StatusOpen,
				bankreceiptworkitem.StatusAssigned,
				bankreceiptworkitem.StatusInReview,
			),
			workItems(bankreceiptworkitem.StatusAssigned),
			workItems(bankreceiptworkitem.StatusInReview),
			count(current), amount(current),
			count(days1To3), amount(days1To3),
			count(days4To7), amount(days4To7),
			count(daysOver7), amount(daysOver7),
		).
		Where(br.ReceiptDate.Lte(asOf))
}

func (r *repository) GetSummary(
	ctx context.Context,
	req bankreceiptports.GetBankReceiptSummaryRequest,
) (*bankreceiptports.BankReceiptReconciliationSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*bankreceiptports.BankReceiptReconciliationSummary, error) {
			rec, err := keel.IntoOne[summaryRecord](ctx, r.db.Keel(ctx), summaryQuery(req.AsOfDate))
			if err != nil {
				return nil, fmt.Errorf("get bank receipt summary: %w", err)
			}
			return &bankreceiptports.BankReceiptReconciliationSummary{
				AsOfDate:              req.AsOfDate,
				ImportedCount:         rec.ImportedCount,
				ImportedAmount:        rec.ImportedAmount,
				MatchedCount:          rec.MatchedCount,
				MatchedAmount:         rec.MatchedAmount,
				ExceptionCount:        rec.ExceptionCount,
				ExceptionAmount:       rec.ExceptionAmount,
				ActiveWorkItemCount:   rec.ActiveWorkItemCount,
				AssignedWorkItemCount: rec.AssignedWorkItemCount,
				InReviewWorkItemCount: rec.InReviewWorkItemCount,
				ExceptionAging: bankreceiptports.BankReceiptExceptionAging{
					CurrentCount:    rec.CurrentCount,
					CurrentAmount:   rec.CurrentAmount,
					Days1To3Count:   rec.Days1To3Count,
					Days1To3Amount:  rec.Days1To3Amount,
					Days4To7Count:   rec.Days4To7Count,
					Days4To7Amount:  rec.Days4To7Amount,
					DaysOver7Count:  rec.DaysOver7Count,
					DaysOver7Amount: rec.DaysOver7Amount,
				},
			}, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceipt.BankReceipt,
) (*bankreceipt.BankReceipt, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bankreceipt.BankReceipt, error) {
		if err := bankreceiptdb.BankReceipts.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, bankReceiptErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *bankreceipt.BankReceipt,
) (*bankreceipt.BankReceipt, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bankreceipt.BankReceipt, error) {
		br := bankreceiptdb.BankReceipts
		db := r.db.Keel(ctx)
		if err := br.Save(
			ctx,
			db,
			entity,
			br.Status,
			br.MatchedCustomerPaymentID,
			br.MatchedAt,
			br.MatchedByID,
			br.ExceptionReason,
			br.UpdatedByID,
		); err != nil {
			return nil, bankReceiptErrors.Map(err, entity.ID.String())
		}
		updated, err := br.Get(ctx, db, entity.ID)
		if err != nil {
			return nil, bankReceiptErrors.Map(err, entity.ID.String())
		}
		return updated, nil
	})
}
