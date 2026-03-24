package dbhelper

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

var ErrSelectOptionsConfigRequired = errors.New("select options config is required")

type SelectOptionsConfig struct {
	Columns       []string
	OrgColumn     string
	BuColumn      string
	SearchColumns []string
	EntityName    string
	QueryModifier func(q *bun.SelectQuery) *bun.SelectQuery
}

func (c *SelectOptionsConfig) orgColumn() string {
	if c.OrgColumn == "" {
		return "organization_id"
	}
	return c.OrgColumn
}

func (c *SelectOptionsConfig) buColumn() string {
	if c.BuColumn == "" {
		return "business_unit_id"
	}
	return c.BuColumn
}

func SelectOptions[T any](
	ctx context.Context,
	db bun.IDB,
	req *pagination.SelectQueryRequest,
	cfg *SelectOptionsConfig,
) (*pagination.ListResult[T], error) {
	entities := make([]T, 0, req.Pagination.Limit)

	if cfg == nil {
		return nil, ErrSelectOptionsConfigRequired
	}

	q := db.NewSelect().
		Model(&entities).
		Column(cfg.Columns...).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cfg.orgColumn()+" = ?", req.TenantInfo.OrgID).
				Where(cfg.buColumn()+" = ?", req.TenantInfo.BuID)
		}).
		Limit(req.Pagination.Limit).
		Offset(req.Pagination.Offset)

	if req.Query != "" && len(cfg.SearchColumns) > 0 {
		q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			for _, col := range cfg.SearchColumns {
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
