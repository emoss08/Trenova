package integritycheck

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/keelcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	idColumn             = "id"
	organizationIDColumn = "organization_id"
	businessUnitIDColumn = "business_unit_id"
)

var errTableNameRequired = errors.New("integritycheck: table name is required")

type Checker struct {
	conn *postgres.Connection
}

func New(conn *postgres.Connection) validationframework.IntegrityChecker {
	return &Checker{conn: conn}
}

type referenceGroup struct {
	table      *schema.Table
	column     string
	conditions []validationframework.Condition
	ids        []string
	positions  map[string][]int
}

func (c *Checker) CheckIntegrity(
	ctx context.Context,
	req *validationframework.IntegrityRequest,
) (*validationframework.IntegrityResult, error) {
	result := &validationframework.IntegrityResult{
		Duplicate: make([]bool, len(req.Unique)),
		Exists:    make([]bool, len(req.References)),
	}
	if req.Empty() {
		return result, nil
	}

	items := make([]keel.BatchItem, 0, len(req.Unique)+len(req.References))
	if len(req.Unique) > 0 {
		table, err := tableNamed(req.Table)
		if err != nil {
			return nil, err
		}
		for i, check := range req.Unique {
			q, qErr := uniquenessQuery(ctx, table, req, check)
			if qErr != nil {
				return nil, qErr
			}
			items = append(items, keel.DynExistsInto(q, &result.Duplicate[i]))
		}
	}

	groups, err := groupReferences(req.References)
	if err != nil {
		return nil, err
	}
	found := make([]*keel.DynResult, len(groups))
	for i, group := range groups {
		q, qErr := referenceQuery(ctx, group, req)
		if qErr != nil {
			return nil, qErr
		}
		items = append(items, keel.DynInto(q, &found[i]))
	}

	if err = c.conn.RunScoped(ctx, true, func(ctx context.Context) error {
		return keel.Batch(ctx, c.conn.Keel(ctx), items...)
	}); err != nil {
		return nil, err
	}

	for i, group := range groups {
		markFound(found[i], group, result.Exists)
	}

	return result, nil
}

func tableNamed(name string) (*schema.Table, error) {
	if name == "" {
		return nil, errTableNameRequired
	}
	table, ok := keelcatalog.Named(name)
	if !ok {
		return nil, fmt.Errorf("integritycheck: no generated table is named %q", name)
	}

	return table, nil
}

func uniquenessQuery(
	ctx context.Context,
	table *schema.Table,
	req *validationframework.IntegrityRequest,
	check validationframework.UniqueCheck,
) (*keel.DynQuery, error) {
	if len(check.Fields) == 0 {
		return nil, fmt.Errorf("integritycheck: a uniqueness check on %s names no fields", table.Name)
	}

	q := keel.DynFrom(table)
	preds, err := tenantPredicates(ctx, q, req.OrganizationID, req.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	for _, field := range check.Fields {
		pred, err := fieldPredicate(q, field)
		if err != nil {
			return nil, err
		}
		preds = append(preds, pred)
	}
	if req.ExcludeID.IsNotNil() {
		preds = append(preds, q.Path(idColumn).Ne(req.ExcludeID.String()))
	}

	return scoped(ctx, q.Where(preds...)).Select(q.Path(idColumn)).Limit(1), nil
}

func groupReferences(checks []validationframework.ReferenceCheck) ([]*referenceGroup, error) {
	if len(checks) == 0 {
		return nil, nil
	}

	byKey := make(map[string]*referenceGroup, len(checks))
	groups := make([]*referenceGroup, 0, len(checks))
	for position, check := range checks {
		column := check.Column
		if column == "" {
			column = idColumn
		}
		key := groupKey(check.Table, column, check.Conditions)
		group, ok := byKey[key]
		if !ok {
			table, err := tableNamed(check.Table)
			if err != nil {
				return nil, err
			}
			group = &referenceGroup{
				table:      table,
				column:     column,
				conditions: check.Conditions,
				positions:  make(map[string][]int, 1),
			}
			byKey[key] = group
			groups = append(groups, group)
		}

		id := check.ID.String()
		if _, seen := group.positions[id]; !seen {
			group.ids = append(group.ids, id)
		}
		group.positions[id] = append(group.positions[id], position)
	}

	return groups, nil
}

func groupKey(table, column string, conditions []validationframework.Condition) string {
	var b strings.Builder
	b.WriteString(table)
	b.WriteByte(0)
	b.WriteString(column)
	for _, condition := range conditions {
		b.WriteByte(0)
		b.WriteString(condition.Column)
		b.WriteByte('=')
		fmt.Fprint(&b, condition.Value)
	}

	return b.String()
}

func referenceQuery(
	ctx context.Context,
	group *referenceGroup,
	req *validationframework.IntegrityRequest,
) (*keel.DynQuery, error) {
	q := keel.DynFrom(group.table)
	if group.table.Column(group.column) < 0 {
		return nil, fmt.Errorf(
			"integritycheck: table %s has no column %q",
			group.table.Name,
			group.column,
		)
	}

	preds, err := tenantPredicates(ctx, q, req.OrganizationID, req.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	preds = append(preds, q.Path(group.column).In(group.ids))
	for _, condition := range group.conditions {
		if group.table.Column(condition.Column) < 0 {
			return nil, fmt.Errorf(
				"integritycheck: table %s has no column %q",
				group.table.Name,
				condition.Column,
			)
		}
		preds = append(preds, q.Path(condition.Column).Eq(condition.Value))
	}

	return scoped(ctx, q.Where(preds...)).
		Select(q.Path(group.column)).
		Limit(len(group.ids)), nil
}

func markFound(found *keel.DynResult, group *referenceGroup, exists []bool) {
	if found == nil || len(found.Columns) == 0 {
		return
	}

	column := &found.Columns[0]
	for row := range found.Rows {
		for _, position := range group.positions[cellText(column, row)] {
			exists[position] = true
		}
	}
}

func cellText(column *keel.DynColumn, row int) string {
	switch value := column.Value(row).(type) {
	case nil:
		return ""
	case string:
		return value
	case []byte:
		return string(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return fmt.Sprint(value)
	}
}

func scoped(ctx context.Context, q *keel.DynQuery) *keel.DynQuery {
	if dbscope.IsSystem(ctx) && q.Table().HasTenant() {
		q.AllTenants()
	}

	return q
}

func tenantPredicates(
	ctx context.Context,
	q *keel.DynQuery,
	orgID, buID pulid.ID,
) ([]keel.Pred, error) {
	table := q.Table()
	if !dbscope.IsSystem(ctx) || !table.HasTenant() {
		return nil, nil
	}

	preds := make([]keel.Pred, 0, len(table.Tenant))
	for _, index := range table.Tenant {
		name := table.Columns[index].Name
		var value pulid.ID
		switch name {
		case organizationIDColumn:
			value = orgID
		case businessUnitIDColumn:
			value = buID
		default:
			return nil, fmt.Errorf("integritycheck: %s has tenant column %q", table.Name, name)
		}
		if value.IsNil() {
			return nil, fmt.Errorf(
				"integritycheck: a system-scope check on %s needs its %s",
				table.Name,
				name,
			)
		}
		preds = append(preds, q.Path(name).Eq(value.String()))
	}

	return preds, nil
}

func fieldPredicate(q *keel.DynQuery, field validationframework.FieldCheck) (keel.Pred, error) {
	table := q.Table()
	if table.Column(field.Column) < 0 {
		return nil, fmt.Errorf("integritycheck: table %s has no column %q", table.Name, field.Column)
	}

	col := q.Path(field.Column)
	if field.CaseSensitive {
		return col.Eq(field.Value), nil
	}

	return keel.DynFunc("lower", col).
		Eq(keel.DynFunc("lower", keel.DynLit(field.Value).Cast(keel.TypeString))), nil
}
