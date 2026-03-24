package validationframework

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type ReferenceChecker interface {
	CheckReference(ctx context.Context, req *ReferenceRequest) (bool, error)
}

type ReferenceRequest struct {
	TableName      string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ID             pulid.ID
}

type BunReferenceChecker struct {
	db       bun.IDB
	dbGetter DBGetter
}

func NewBunReferenceChecker(db bun.IDB) *BunReferenceChecker {
	return &BunReferenceChecker{db: db}
}

func NewBunReferenceCheckerLazy(getter DBGetter) *BunReferenceChecker {
	return &BunReferenceChecker{dbGetter: getter}
}

func (c *BunReferenceChecker) getDB() bun.IDB {
	if c.dbGetter != nil {
		return c.dbGetter()
	}
	return c.db
}

func (c *BunReferenceChecker) CheckReference(
	ctx context.Context,
	req *ReferenceRequest,
) (bool, error) {
	if req.TableName == "" {
		return false, errors.New("table name is required")
	}

	if req.ID.IsNil() {
		return false, errors.New("reference ID is required")
	}

	db := c.getDB()
	if db == nil {
		return false, errors.New("database connection is not initialized")
	}

	q := db.NewSelect().
		TableExpr(req.TableName).
		ColumnExpr("1").
		Where(fmt.Sprintf("%s.id = ?", req.TableName), req.ID)

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

	return q.Exists(ctx)
}

type CustomReferenceCheckFunc func(ctx context.Context, orgID, buID, refID pulid.ID) (bool, error)

type ReferenceFieldConfig[T TenantedEntity] struct {
	FieldName   string
	TableName   string
	Message     string
	Optional    bool
	GetID       func(T) pulid.ID
	CustomCheck CustomReferenceCheckFunc
}
