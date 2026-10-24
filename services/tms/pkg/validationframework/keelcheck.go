package validationframework

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

var errDBNotInitialized = errors.New("database connection is not initialized")

const (
	idColumn             = "id"
	organizationIDColumn = "organization_id"
	businessUnitIDColumn = "business_unit_id"
)

type KeelDB interface {
	Keel(ctx context.Context) *keel.DB
	RunScoped(ctx context.Context, readOnly bool, fn func(context.Context) error) error
}

type KeelChecker struct {
	conn   KeelDB
	tables map[string]*schema.Table
}

var (
	_ UniquenessChecker = (*KeelChecker)(nil)
	_ ReferenceChecker  = (*KeelChecker)(nil)
)

func NewKeelChecker(conn KeelDB, table *schema.Table) *KeelChecker {
	tables := make(map[string]*schema.Table, len(table.Relations)+1)
	tables[table.Name] = table
	for i := range table.Relations {
		if target := table.Relations[i].Target; target != nil {
			tables[target.Name] = target
		}
	}

	return &KeelChecker{conn: conn, tables: tables}
}

func (c *KeelChecker) CheckUniqueness(ctx context.Context, req *UniquenessRequest) (bool, error) {
	q, err := c.uniquenessQuery(ctx, req)
	if err != nil {
		return false, err
	}

	return existsIn(ctx, c.conn, q)
}

func (c *KeelChecker) CheckReference(ctx context.Context, req *ReferenceRequest) (bool, error) {
	q, err := c.referenceQuery(ctx, req)
	if err != nil {
		return false, err
	}

	return existsIn(ctx, c.conn, q)
}

func (c *KeelChecker) uniquenessQuery(
	ctx context.Context,
	req *UniquenessRequest,
) (*keel.DynQuery, error) {
	if len(req.Fields) == 0 {
		return nil, errors.New("at least one field is required")
	}

	table, err := c.table(req.TableName)
	if err != nil {
		return nil, err
	}

	q := keel.DynFrom(table)
	preds := tenantPredicates(ctx, q, req.OrganizationID, req.BusinessUnitID)
	for _, fields := range [][]FieldCheck{req.Fields, req.ScopeFields} {
		for _, field := range fields {
			pred, fieldErr := fieldPredicate(q, field)
			if fieldErr != nil {
				return nil, fieldErr
			}
			preds = append(preds, pred)
		}
	}
	if req.ExcludeID.IsNotNil() {
		preds = append(preds, q.Path(idColumn).Ne(req.ExcludeID.String()))
	}

	return existsQuery(ctx, q.Where(preds...)), nil
}

func (c *KeelChecker) referenceQuery(
	ctx context.Context,
	req *ReferenceRequest,
) (*keel.DynQuery, error) {
	if req.ID.IsNil() {
		return nil, errors.New("reference ID is required")
	}

	table, err := c.table(req.TableName)
	if err != nil {
		return nil, err
	}

	return keelReferenceQuery(ctx, table, req.OrganizationID, req.BusinessUnitID, req.ID), nil
}

func (c *KeelChecker) table(name string) (*schema.Table, error) {
	if name == "" {
		return nil, errors.New("table name is required")
	}

	table, ok := c.tables[name]
	if !ok {
		return nil, fmt.Errorf("table %q is not known to this checker", name)
	}

	return table, nil
}

func KeelReferenceCheck(conn KeelDB, table *schema.Table) CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		return existsIn(ctx, conn, keelReferenceQuery(ctx, table, orgID, buID, refID))
	}
}

func keelReferenceQuery(
	ctx context.Context,
	table *schema.Table,
	orgID, buID, id pulid.ID,
) *keel.DynQuery {
	q := keel.DynFrom(table)
	preds := tenantPredicates(ctx, q, orgID, buID)
	preds = append(preds, q.Path(idColumn).Eq(id.String()))

	return existsQuery(ctx, q.Where(preds...))
}

func existsQuery(ctx context.Context, q *keel.DynQuery) *keel.DynQuery {
	if dbscope.IsSystem(ctx) && q.Table().HasTenant() {
		q.AllTenants()
	}

	return q.Select(q.Path(idColumn)).Limit(1)
}

func existsIn(ctx context.Context, conn KeelDB, q *keel.DynQuery) (bool, error) {
	if conn == nil {
		return false, errDBNotInitialized
	}

	var exists bool
	err := conn.RunScoped(ctx, true, func(ctx context.Context) error {
		res, err := q.Run(ctx, conn.Keel(ctx))
		if err != nil {
			return err
		}
		exists = res.Rows > 0
		return nil
	})

	return exists, err
}

func tenantPredicates(ctx context.Context, q *keel.DynQuery, orgID, buID pulid.ID) []keel.Pred {
	preds := make([]keel.Pred, 0, 3)
	table := q.Table()
	if !dbscope.IsSystem(ctx) || !table.HasTenant() {
		return preds
	}

	if orgID.IsNotNil() && table.Column(organizationIDColumn) >= 0 {
		preds = append(preds, q.Path(organizationIDColumn).Eq(orgID.String()))
	}
	if buID.IsNotNil() && table.Column(businessUnitIDColumn) >= 0 {
		preds = append(preds, q.Path(businessUnitIDColumn).Eq(buID.String()))
	}

	return preds
}

func fieldPredicate(q *keel.DynQuery, field FieldCheck) (keel.Pred, error) {
	table := q.Table()
	if table.Column(field.Column) < 0 {
		return nil, fmt.Errorf("table %s has no column %q", table.Name, field.Column)
	}

	col := q.Path(field.Column)
	if field.CaseSensitive {
		return col.Eq(field.Value), nil
	}

	return keel.DynFunc("lower", col).
		Eq(keel.DynFunc("lower", keel.DynLit(field.Value).Cast(keel.TypeString))), nil
}
