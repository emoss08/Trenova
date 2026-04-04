package dbhelper

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

var ErrSelectOptionsConfigRequired = errors.New("select options config is required")

type SelectOptionsConfig struct {
	Columns          []string
	ColumnRefs       []buncolgen.Column
	OrgColumn        string
	OrgColumnRef     *buncolgen.Column
	BuColumn         string
	BuColumnRef      *buncolgen.Column
	SearchColumns    []string
	SearchColumnRefs []buncolgen.Column
	EntityName       string
	QueryModifier    func(q *bun.SelectQuery) *bun.SelectQuery
}

func (c *SelectOptionsConfig) columns() []string {
	if len(c.ColumnRefs) == 0 {
		return c.Columns
	}

	cols := make([]string, 0, len(c.ColumnRefs))
	for _, col := range c.ColumnRefs {
		cols = append(cols, col.Bare())
	}

	return cols
}

func (c *SelectOptionsConfig) orgColumn() string {
	if c.OrgColumnRef != nil {
		return c.OrgColumnRef.Qualified()
	}
	if c.OrgColumn == "" {
		return "organization_id"
	}
	return c.OrgColumn
}

func (c *SelectOptionsConfig) buColumn() string {
	if c.BuColumnRef != nil {
		return c.BuColumnRef.Qualified()
	}
	if c.BuColumn == "" {
		return "business_unit_id"
	}
	return c.BuColumn
}

func (c *SelectOptionsConfig) searchColumns() []string {
	if len(c.SearchColumnRefs) == 0 {
		return c.SearchColumns
	}

	cols := make([]string, 0, len(c.SearchColumnRefs))
	for _, col := range c.SearchColumnRefs {
		cols = append(cols, col.Qualified())
	}

	return cols
}

func SelectOptions[T any](
	ctx context.Context,
	db bun.IDB,
	req *pagination.SelectQueryRequest,
	cfg *SelectOptionsConfig,
) (*pagination.ListResult[T], error) {
	entities := make([]T, 0, req.Pagination.SafeLimit())

	if cfg == nil {
		return nil, ErrSelectOptionsConfigRequired
	}

	q := db.NewSelect().
		Model(&entities).
		Column(cfg.columns()...).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cfg.orgColumn()+" = ?", req.TenantInfo.OrgID).
				Where(cfg.buColumn()+" = ?", req.TenantInfo.BuID)
		}).
		Limit(req.Pagination.SafeLimit()).
		Offset(req.Pagination.SafeOffset())

	searchColumns := cfg.searchColumns()
	if req.Query != "" && len(searchColumns) > 0 {
		q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			for _, col := range searchColumns {
				sq.WhereOr("LOWER("+col+") LIKE LOWER(?)", WrapWildcard(req.Query))
			}
			return sq
		})
	}

	if cfg.QueryModifier != nil {
		q = cfg.QueryModifier(q)
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[T]{
		Items: entities,
		Total: total,
	}, nil
}
