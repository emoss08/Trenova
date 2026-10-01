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
	source dbSource
}

func NewBunReferenceChecker(db bun.IDB) *BunReferenceChecker {
	return &BunReferenceChecker{source: dbSource{db: db}}
}

func NewBunReferenceCheckerScoped(conn ScopedDB) *BunReferenceChecker {
	return &BunReferenceChecker{source: dbSource{scoped: conn}}
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

	var exists bool
	err := c.source.read(ctx, func(ctx context.Context, db bun.IDB) error {
		var queryErr error
		exists, queryErr = referenceQuery(db, req).Exists(ctx)
		return queryErr
	})

	return exists, err
}

func referenceQuery(db bun.IDB, req *ReferenceRequest) *bun.SelectQuery {
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

	return q
}

type CustomReferenceCheckFunc func(ctx context.Context, orgID, buID, refID pulid.ID) (bool, error)

// NewUSStateReferenceCheck builds the shared existence check for the global
// us_states table (which carries no tenant columns).
func NewUSStateReferenceCheck(conn ScopedDB) CustomReferenceCheckFunc {
	source := dbSource{scoped: conn}

	return func(ctx context.Context, _, _ pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		var exists bool
		err := source.read(ctx, func(ctx context.Context, db bun.IDB) error {
			var queryErr error
			exists, queryErr = db.NewSelect().
				TableExpr("us_states").
				ColumnExpr("1").
				Where("id = ?", refID).
				Exists(ctx)
			return queryErr
		})

		return exists, err
	}
}

type ReferenceFieldConfig[T TenantedEntity] struct {
	FieldName   string
	TableName   string
	Message     string
	Optional    bool
	GetID       func(T) pulid.ID
	CustomCheck CustomReferenceCheckFunc
}
