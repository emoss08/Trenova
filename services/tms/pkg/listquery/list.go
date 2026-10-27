package listquery

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Request[T any] struct {
	Filter   *pagination.QueryOptions
	Cursor   pagination.CursorInfo
	Columns  []string
	Shape    func(*keel.Query[*T]) *keel.Query[*T]
	Then     []keel.Order
	Keyset   *Keyset
	Computed map[string]keel.Expr
}

type Keyset struct {
	Orders []keel.Order
	Shape  []pagination.CursorSortField
}

func Connection[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[*T],
	req *Request[T],
) (*pagination.CursorListResult[*T], error) {
	p := newPlan(q, req.Computed)
	read := narrow(p, q, req)

	orders, shape, err := connectionOrder(p, req)
	if err != nil {
		return nil, err
	}

	after, err := keysetAfter(&req.Cursor, shape)
	if err != nil {
		return nil, err
	}

	page := keysetPage{orders: orders, shape: shape, after: after}
	var matched *keel.Query[*T]
	if read != nil && after != "" {
		matched = q.Clone()
	}

	result, err := connectionPage(ctx, db, q, req, page)
	if err != nil {
		return nil, err
	}
	if read == nil || len(result.Items) > 0 {
		return result, nil
	}

	if matched != nil {
		exists, existsErr := matched.Exists(ctx, db)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return result, nil
		}
	}

	return fuzzyConnection(ctx, db, read, page)
}

func narrow[T any](p *plan, q *keel.Query[*T], req *Request[T]) *fuzzyRead[T] {
	q.Where(p.filters(req.Filter)...)

	var read *fuzzyRead[T]
	if match, ok := p.fuzzy(req.Filter); ok {
		read = &fuzzyRead[T]{match: match, unmatched: q.Clone(), req: req}
	}

	if search := p.search(queryOf(req.Filter)); search != nil {
		q.Where(search)
	}

	return read
}

type keysetPage struct {
	orders []keel.Order
	shape  []pagination.CursorSortField
	after  string
}

func connectionPage[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[*T],
	req *Request[T],
	kp keysetPage,
) (*pagination.CursorListResult[*T], error) {
	counted := q.Clone()
	rows, err := shapeRows(q, req)
	if err != nil {
		return nil, err
	}
	rows.Order(kp.orders...)

	limit := cursorLimit(req.Filter, &req.Cursor)
	var page keel.Page[*T]
	var total keel.RowCount
	items := []keel.BatchItem{
		keel.PageInto(rows, keel.After(kp.after, limit), &page),
	}
	if req.Cursor.IncludeTotalCount {
		items = append(items, keel.CountEstimateInto(counted, pagination.ExactTotalUpTo, &total))
	}
	if err = keel.Batch(ctx, db, items...); err != nil {
		return nil, err
	}

	result, err := connectionResult(&page, kp.shape)
	if err != nil {
		return nil, err
	}
	if req.Cursor.IncludeTotalCount {
		withTotal(result, total)
	}

	return result, nil
}

func withTotal[T any](result *pagination.CursorListResult[T], total keel.RowCount) {
	result.TotalCount = new(int(total.N))
	result.TotalCountExact = total.Exact
}

func connectionOrder[T any](
	p *plan,
	req *Request[T],
) ([]keel.Order, []pagination.CursorSortField, error) {
	if req.Keyset != nil {
		return req.Keyset.Orders, req.Keyset.Shape, nil
	}

	return p.keysetOrder(sortsOf(req.Filter))
}

func List[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[*T],
	req *Request[T],
) (*pagination.ListResult[*T], error) {
	p := newPlan(q, req.Computed)
	read := narrow(p, q, req)

	rows, counted, err := listRows(p, q, req, p.rank(queryOf(req.Filter)))
	if err != nil {
		return nil, err
	}

	result, err := listAndCount(ctx, db, rows, counted)
	if err != nil {
		return nil, err
	}
	if read == nil || result.Total > 0 {
		return result, nil
	}

	return fuzzyList(ctx, db, p, read)
}

func listRows[T any](
	p *plan,
	q *keel.Query[*T],
	req *Request[T],
	rank keel.Expr,
) (rows, counted *keel.Query[*T], err error) {
	counted = q.Clone()
	rows, err = shapeRows(q, req)
	if err != nil {
		return nil, nil, err
	}

	if rank != nil {
		rows.Order(keel.Desc(rank).NullsLast())
	}
	rows.Order(p.orders(sortsOf(req.Filter))...)
	rows.Order(req.Then...)

	info := paginationOf(req.Filter)
	rows.Limit(info.SafeLimit()).Offset(info.SafeOffset())

	return rows, counted, nil
}

func SelectOptions[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[*T],
	req *pagination.SelectQueryRequest,
	search func(query string) keel.Pred,
) (*pagination.ListResult[*T], error) {
	if req.Query != "" && search != nil {
		q.Where(search(req.Query))
	}

	counted := q.Clone()
	q.Limit(req.Pagination.SafeLimit()).Offset(req.Pagination.SafeOffset())

	return listAndCount(ctx, db, q, counted)
}

func listAndCount[T any](
	ctx context.Context,
	db *keel.DB,
	rows, counted *keel.Query[*T],
) (*pagination.ListResult[*T], error) {
	var items []*T
	var total keel.RowCount
	err := keel.Batch(ctx, db,
		keel.ListInto(rows, &items),
		keel.CountEstimateInto(counted, pagination.ExactTotalUpTo, &total),
	)
	if err != nil {
		return nil, err
	}

	return Result(items, total), nil
}

func Result[T any](items []T, total keel.RowCount) *pagination.ListResult[T] {
	return &pagination.ListResult[T]{Items: items, Total: int(total.N), TotalExact: total.Exact}
}

func Pick[T any](q *keel.Query[*T], columns []string) (*keel.Query[*T], error) {
	if len(columns) == 0 {
		return q, nil
	}

	table := q.TableRef().Table
	picks := make([]keel.Expr, 0, len(columns))
	for _, name := range columns {
		index := table.Column(name)
		if index < 0 || !table.Columns[index].Stored() {
			return nil, fmt.Errorf("listquery: %s has no column %q", table.Name, name)
		}
		picks = append(picks, keel.Col{Ref: q.TableRef(), Index: index})
	}

	return q.Pick(picks...), nil
}

func shapeRows[T any](q *keel.Query[*T], req *Request[T]) (*keel.Query[*T], error) {
	rows, err := Pick(q, req.Columns)
	if err != nil {
		return nil, err
	}
	if req.Shape != nil {
		rows = req.Shape(rows)
	}

	return rows, nil
}

func keysetAfter(
	cursor *pagination.CursorInfo,
	shape []pagination.CursorSortField,
) (string, error) {
	if cursor.After == "" {
		return "", nil
	}

	if !cursor.Cursor.MatchesSort(shape) {
		return "", errortypes.NewValidationError(
			"after",
			errortypes.ErrInvalid,
			"Cursor sort does not match request sort",
		)
	}

	return cursor.Cursor.Keyset, nil
}

func connectionResult[T any](
	page *keel.Page[*T],
	shape []pagination.CursorSortField,
) (*pagination.CursorListResult[*T], error) {
	cursors := make([]string, len(page.Items))
	for i := range page.Items {
		cursor, err := pagination.EncodeKeysetCursor(shape, page.CursorAt(i))
		if err != nil {
			return nil, err
		}
		cursors[i] = cursor
	}

	return &pagination.CursorListResult[*T]{
		Items:       page.Items,
		HasNextPage: page.Next != "",
		EdgeCursors: cursors,
	}, nil
}

func sortsOf(filter *pagination.QueryOptions) []domaintypes.SortField {
	if filter == nil {
		return nil
	}

	return filter.Sort
}

func paginationOf(filter *pagination.QueryOptions) pagination.Info {
	if filter == nil {
		return pagination.Info{}
	}

	return filter.Pagination
}

func cursorLimit(filter *pagination.QueryOptions, cursor *pagination.CursorInfo) int {
	if cursor.Limit > 0 {
		return cursor.Limit
	}
	if filter != nil {
		return filter.Pagination.SafeLimit()
	}

	return pagination.DefaultLimit
}

func Filter[T any](q *keel.Query[*T], filter *pagination.QueryOptions) *keel.Query[*T] {
	return q.Where(newPlan(q, nil).where(filter)...)
}
