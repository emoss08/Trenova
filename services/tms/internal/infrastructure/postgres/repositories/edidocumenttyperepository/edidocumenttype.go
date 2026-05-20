package edidocumenttyperepository

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

func New(p Params) repositories.EDIDocumentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-document-type-repository"),
	}
}

func (r *repository) ListDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	entities := make([]*edi.EDIDocumentType, 0, 8)
	cols := buncolgen.EDIDocumentTypeColumns
	query := r.db.DBForContext(ctx).NewSelect().Model(&entities).Order(cols.Code.OrderAsc())
	query = filterDocumentTypesQuery(query, req)
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) SelectDocumentTypeOptions(
	ctx context.Context,
	req *repositories.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	entities := make([]*edi.EDIDocumentType, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	cols := buncolgen.EDIDocumentTypeColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.Code.Bare(),
			cols.Name.Bare(),
			cols.Standard.Bare(),
			cols.TransactionSet.Bare(),
			cols.Direction.Bare(),
			cols.DefaultVersion.Bare(),
			cols.Status.Bare(),
		).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return filterDocumentTypesQuery(sq, repositories.ListEDIDocumentTypesRequest{
				Standard:       req.Standard,
				TransactionSet: req.TransactionSet,
				Direction:      req.Direction,
				Status:         req.Status,
			})
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyDocumentTypeSearch(sq, req.SelectQueryRequest.Query)
		})

	total, err := query.
		Order(cols.Code.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIDocumentType]{Items: entities, Total: total}, nil
}

func filterDocumentTypesQuery(
	query *bun.SelectQuery,
	req repositories.ListEDIDocumentTypesRequest,
) *bun.SelectQuery {
	cols := buncolgen.EDIDocumentTypeColumns
	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	return query
}

func applyDocumentTypeSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDIDocumentTypeColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Code.LowerLike(), term).
			WhereOr(cols.Name.LowerLike(), term).
			WhereOr(cols.DefaultVersion.LowerLike(), term)
	})
}
