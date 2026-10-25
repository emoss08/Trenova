package bankreceiptbatchrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch/bankreceiptbatchdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var batchErrors = dberror.Mapping{Entity: "BankReceiptBatch"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) bankreceiptbatchports.BankReceiptBatchRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-batch-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req bankreceiptbatchports.GetBankReceiptBatchByIDRequest,
) (*bankreceiptbatch.BankReceiptBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*bankreceiptbatch.BankReceiptBatch, error) {
			entity, err := bankreceiptbatchdb.BankReceiptBatches.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, batchErrors.Map(err, req.ID.String())
			}
			return entity, nil
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*bankreceiptbatch.BankReceiptBatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*bankreceiptbatch.BankReceiptBatch, error) {
			b := bankreceiptbatchdb.BankReceiptBatches
			items, err := b.Order(b.CreatedAt.Desc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list bank receipt batches: %w", err)
			}
			return items, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceiptbatch.BankReceiptBatch,
) (*bankreceiptbatch.BankReceiptBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*bankreceiptbatch.BankReceiptBatch, error) {
			if err := bankreceiptbatchdb.BankReceiptBatches.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, batchErrors.Map(err, entity.ID.String())
			}
			return entity, nil
		},
	)
}

func (r *repository) DistinctSources(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*bankreceiptbatchports.BankReceiptBatchSourceOption], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*bankreceiptbatchports.BankReceiptBatchSourceOption], error) {
			b := bankreceiptbatchdb.BankReceiptBatches
			search := strings.TrimSpace(req.Query)
			q := b.Group(b.Source).
				Pick(b.Source).
				Where(b.Source.Ne(""), keel.When(search != "", b.Source.ContainsFold(search))).
				Order(b.Source.Asc())

			sources, err := keel.Into[string](ctx, r.db.Keel(ctx), q)
			if err != nil {
				return nil, fmt.Errorf("distinct sources: %w", err)
			}

			options := make([]*bankreceiptbatchports.BankReceiptBatchSourceOption, len(sources))
			for i, s := range sources {
				options[i] = &bankreceiptbatchports.BankReceiptBatchSourceOption{Value: s, Label: s}
			}

			limit := req.Pagination.SafeLimit()
			offset := req.Pagination.SafeOffset()
			total := len(options)

			if offset > total {
				offset = total
			}
			end := min(offset+limit, total)

			return &pagination.ListResult[*bankreceiptbatchports.BankReceiptBatchSourceOption]{
				Items: options[offset:end],
				Total: total,
			}, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *bankreceiptbatch.BankReceiptBatch,
) (*bankreceiptbatch.BankReceiptBatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*bankreceiptbatch.BankReceiptBatch, error) {
			b := bankreceiptbatchdb.BankReceiptBatches
			db := r.db.Keel(ctx)
			if err := b.Save(
				ctx,
				db,
				entity,
				b.Status,
				b.ImportedCount,
				b.MatchedCount,
				b.ExceptionCount,
				b.ImportedAmountMinor,
				b.MatchedAmountMinor,
				b.ExceptionAmountMinor,
				b.UpdatedByID,
			); err != nil {
				return nil, batchErrors.Map(err, entity.ID.String())
			}

			updated, err := b.Get(ctx, db, entity.ID)
			if err != nil {
				return nil, batchErrors.Map(err, entity.ID.String())
			}
			return updated, nil
		},
	)
}
