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

type DBGetter func() bun.IDB

type BunUniquenessChecker struct {
	db       bun.IDB
	dbGetter DBGetter
}

func NewBunUniquenessChecker(db bun.IDB) *BunUniquenessChecker {
	return &BunUniquenessChecker{db: db}
}

func NewBunUniquenessCheckerLazy(getter DBGetter) *BunUniquenessChecker {
	return &BunUniquenessChecker{dbGetter: getter}
}

func (c *BunUniquenessChecker) getDB() bun.IDB {
	if c.dbGetter != nil {
		return c.dbGetter()
	}
	return c.db
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

	db := c.getDB()
	if db == nil {
		return false, errors.New("database connection is not initialized")
	}

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

	return q.Exists(ctx)
}
