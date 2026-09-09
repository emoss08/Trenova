package editransactionsetrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.EDITransactionSetRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transaction-set-repository"),
	}
}

func (r *repository) SelectTransactionSetOptions(
	ctx context.Context,
	req *repositories.EDITransactionSetSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransactionSet], error) {
	entities := make([]*edi.EDITransactionSet, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	cols := buncolgen.EDITransactionSetColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.Standard.Bare(),
			cols.Code.Bare(),
			cols.Name.Bare(),
			cols.Description.Bare(),
			cols.DefaultVersion.Bare(),
			cols.Status.Bare(),
			cols.CreatedAt.Bare(),
		).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return filterTransactionSetsQuery(sq, req)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTransactionSetSearch(sq, req.SelectQueryRequest.Query)
		})

	total, err := query.
		Order(cols.Code.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITransactionSet]{Items: entities, Total: total}, nil
}

func filterTransactionSetsQuery(
	query *bun.SelectQuery,
	req *repositories.EDITransactionSetSelectOptionsRequest,
) *bun.SelectQuery {
	cols := buncolgen.EDITransactionSetColumns
	if len(req.IDs) > 0 {
		query = query.Where(cols.ID.In(), bun.List(req.IDs))
	}
	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	return query
}

func applyTransactionSetSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDITransactionSetColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Code.LowerLike(), term).
			WhereOr(cols.Name.LowerLike(), term).
			WhereOr(cols.DefaultVersion.LowerLike(), term)
	})
}
