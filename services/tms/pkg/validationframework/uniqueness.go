package validationframework

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type UniquenessChecker interface {
	CheckUniqueness(ctx context.Context, req *UniquenessRequest) (bool, error)
}

type UniquenessRequest struct {
	TableName      string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ExcludeID      pulid.ID
	ScopeFields    []FieldCheck
	Fields         []FieldCheck
}

type FieldCheck struct {
	Column        string
	Value         any
	CaseSensitive bool
}

type BunUniquenessChecker struct {
	source dbSource
}

func NewBunUniquenessChecker(db bun.IDB) *BunUniquenessChecker {
	return &BunUniquenessChecker{source: dbSource{db: db}}
}

func NewBunUniquenessCheckerScoped(conn ScopedDB) *BunUniquenessChecker {
	return &BunUniquenessChecker{source: dbSource{scoped: conn}}
}

func (c *BunUniquenessChecker) CheckUniqueness(
	ctx context.Context,
	req *UniquenessRequest,
) (bool, error) {
	if req.TableName == "" {
		return false, errors.New("table name is required")
	}

	if len(req.Fields) == 0 {
		return false, errors.New("at least one field is required")
	}

	var exists bool
	err := c.source.read(ctx, func(ctx context.Context, db bun.IDB) error {
		var queryErr error
		exists, queryErr = uniquenessQuery(db, req).Exists(ctx)
		return queryErr
	})

	return exists, err
}

func uniquenessQuery(db bun.IDB, req *UniquenessRequest) *bun.SelectQuery {
	q := db.NewSelect().
		TableExpr(req.TableName).
		ColumnExpr("1")

	if req.OrganizationID.IsNotNil() {
		q = q.Where(
			fmt.Sprintf("%s.organization_id = ?", req.TableName),
			req.OrganizationID,
		)
	}

	if req.BusinessUnitID.IsNotNil() {
		q = q.Where(
			fmt.Sprintf("%s.business_unit_id = ?", req.TableName),
			req.BusinessUnitID,
		)
	}

	for _, field := range req.Fields {
		if field.CaseSensitive {
			q = q.Where(
				fmt.Sprintf("%s.%s = ?", req.TableName, field.Column),
				field.Value,
			)
		} else {
			q = q.Where(
				fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", req.TableName, field.Column),
				field.Value,
			)
		}
	}

	for _, field := range req.ScopeFields {
		if field.CaseSensitive {
			q = q.Where(
				fmt.Sprintf("%s.%s = ?", req.TableName, field.Column),
				field.Value,
			)
		} else {
			q = q.Where(
				fmt.Sprintf("LOWER(%s.%s) = LOWER(?)", req.TableName, field.Column),
				field.Value,
			)
		}
	}

	if req.ExcludeID.IsNotNil() {
		q = q.Where(fmt.Sprintf("%s.id != ?", req.TableName), req.ExcludeID)
	}

	return q
}
