// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/emoss08/trenova/internal/ent/userreport"
	"github.com/google/uuid"
)

// UserReportQuery is the builder for querying UserReport entities.
type UserReportQuery struct {
	config
	ctx              *QueryContext
	order            []userreport.OrderOption
	inters           []Interceptor
	predicates       []predicate.UserReport
	withBusinessUnit *BusinessUnitQuery
	withOrganization *OrganizationQuery
	withUser         *UserQuery
	modifiers        []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the UserReportQuery builder.
func (urq *UserReportQuery) Where(ps ...predicate.UserReport) *UserReportQuery {
	urq.predicates = append(urq.predicates, ps...)
	return urq
}

// Limit the number of records to be returned by this query.
func (urq *UserReportQuery) Limit(limit int) *UserReportQuery {
	urq.ctx.Limit = &limit
	return urq
}

// Offset to start from.
func (urq *UserReportQuery) Offset(offset int) *UserReportQuery {
	urq.ctx.Offset = &offset
	return urq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (urq *UserReportQuery) Unique(unique bool) *UserReportQuery {
	urq.ctx.Unique = &unique
	return urq
}

// Order specifies how the records should be ordered.
func (urq *UserReportQuery) Order(o ...userreport.OrderOption) *UserReportQuery {
	urq.order = append(urq.order, o...)
	return urq
}

// QueryBusinessUnit chains the current query on the "business_unit" edge.
func (urq *UserReportQuery) QueryBusinessUnit() *BusinessUnitQuery {
	query := (&BusinessUnitClient{config: urq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := urq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := urq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(userreport.Table, userreport.FieldID, selector),
			sqlgraph.To(businessunit.Table, businessunit.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, userreport.BusinessUnitTable, userreport.BusinessUnitColumn),
		)
		fromU = sqlgraph.SetNeighbors(urq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryOrganization chains the current query on the "organization" edge.
func (urq *UserReportQuery) QueryOrganization() *OrganizationQuery {
	query := (&OrganizationClient{config: urq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := urq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := urq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(userreport.Table, userreport.FieldID, selector),
			sqlgraph.To(organization.Table, organization.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, userreport.OrganizationTable, userreport.OrganizationColumn),
		)
		fromU = sqlgraph.SetNeighbors(urq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryUser chains the current query on the "user" edge.
func (urq *UserReportQuery) QueryUser() *UserQuery {
	query := (&UserClient{config: urq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := urq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := urq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(userreport.Table, userreport.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, userreport.UserTable, userreport.UserColumn),
		)
		fromU = sqlgraph.SetNeighbors(urq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first UserReport entity from the query.
// Returns a *NotFoundError when no UserReport was found.
func (urq *UserReportQuery) First(ctx context.Context) (*UserReport, error) {
	nodes, err := urq.Limit(1).All(setContextOp(ctx, urq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{userreport.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (urq *UserReportQuery) FirstX(ctx context.Context) *UserReport {
	node, err := urq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first UserReport ID from the query.
// Returns a *NotFoundError when no UserReport ID was found.
func (urq *UserReportQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = urq.Limit(1).IDs(setContextOp(ctx, urq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{userreport.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (urq *UserReportQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := urq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single UserReport entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one UserReport entity is found.
// Returns a *NotFoundError when no UserReport entities are found.
func (urq *UserReportQuery) Only(ctx context.Context) (*UserReport, error) {
	nodes, err := urq.Limit(2).All(setContextOp(ctx, urq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{userreport.Label}
	default:
		return nil, &NotSingularError{userreport.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (urq *UserReportQuery) OnlyX(ctx context.Context) *UserReport {
	node, err := urq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only UserReport ID in the query.
// Returns a *NotSingularError when more than one UserReport ID is found.
// Returns a *NotFoundError when no entities are found.
func (urq *UserReportQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = urq.Limit(2).IDs(setContextOp(ctx, urq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{userreport.Label}
	default:
		err = &NotSingularError{userreport.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (urq *UserReportQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := urq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of UserReports.
func (urq *UserReportQuery) All(ctx context.Context) ([]*UserReport, error) {
	ctx = setContextOp(ctx, urq.ctx, "All")
	if err := urq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*UserReport, *UserReportQuery]()
	return withInterceptors[[]*UserReport](ctx, urq, qr, urq.inters)
}

// AllX is like All, but panics if an error occurs.
func (urq *UserReportQuery) AllX(ctx context.Context) []*UserReport {
	nodes, err := urq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of UserReport IDs.
func (urq *UserReportQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if urq.ctx.Unique == nil && urq.path != nil {
		urq.Unique(true)
	}
	ctx = setContextOp(ctx, urq.ctx, "IDs")
	if err = urq.Select(userreport.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (urq *UserReportQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := urq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (urq *UserReportQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, urq.ctx, "Count")
	if err := urq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, urq, querierCount[*UserReportQuery](), urq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (urq *UserReportQuery) CountX(ctx context.Context) int {
	count, err := urq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (urq *UserReportQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, urq.ctx, "Exist")
	switch _, err := urq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (urq *UserReportQuery) ExistX(ctx context.Context) bool {
	exist, err := urq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the UserReportQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (urq *UserReportQuery) Clone() *UserReportQuery {
	if urq == nil {
		return nil
	}
	return &UserReportQuery{
		config:           urq.config,
		ctx:              urq.ctx.Clone(),
		order:            append([]userreport.OrderOption{}, urq.order...),
		inters:           append([]Interceptor{}, urq.inters...),
		predicates:       append([]predicate.UserReport{}, urq.predicates...),
		withBusinessUnit: urq.withBusinessUnit.Clone(),
		withOrganization: urq.withOrganization.Clone(),
		withUser:         urq.withUser.Clone(),
		// clone intermediate query.
		sql:  urq.sql.Clone(),
		path: urq.path,
	}
}

// WithBusinessUnit tells the query-builder to eager-load the nodes that are connected to
// the "business_unit" edge. The optional arguments are used to configure the query builder of the edge.
func (urq *UserReportQuery) WithBusinessUnit(opts ...func(*BusinessUnitQuery)) *UserReportQuery {
	query := (&BusinessUnitClient{config: urq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	urq.withBusinessUnit = query
	return urq
}

// WithOrganization tells the query-builder to eager-load the nodes that are connected to
// the "organization" edge. The optional arguments are used to configure the query builder of the edge.
func (urq *UserReportQuery) WithOrganization(opts ...func(*OrganizationQuery)) *UserReportQuery {
	query := (&OrganizationClient{config: urq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	urq.withOrganization = query
	return urq
}

// WithUser tells the query-builder to eager-load the nodes that are connected to
// the "user" edge. The optional arguments are used to configure the query builder of the edge.
func (urq *UserReportQuery) WithUser(opts ...func(*UserQuery)) *UserReportQuery {
	query := (&UserClient{config: urq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	urq.withUser = query
	return urq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		BusinessUnitID uuid.UUID `json:"businessUnitId"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.UserReport.Query().
//		GroupBy(userreport.FieldBusinessUnitID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (urq *UserReportQuery) GroupBy(field string, fields ...string) *UserReportGroupBy {
	urq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &UserReportGroupBy{build: urq}
	grbuild.flds = &urq.ctx.Fields
	grbuild.label = userreport.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		BusinessUnitID uuid.UUID `json:"businessUnitId"`
//	}
//
//	client.UserReport.Query().
//		Select(userreport.FieldBusinessUnitID).
//		Scan(ctx, &v)
func (urq *UserReportQuery) Select(fields ...string) *UserReportSelect {
	urq.ctx.Fields = append(urq.ctx.Fields, fields...)
	sbuild := &UserReportSelect{UserReportQuery: urq}
	sbuild.label = userreport.Label
	sbuild.flds, sbuild.scan = &urq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a UserReportSelect configured with the given aggregations.
func (urq *UserReportQuery) Aggregate(fns ...AggregateFunc) *UserReportSelect {
	return urq.Select().Aggregate(fns...)
}

func (urq *UserReportQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range urq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, urq); err != nil {
				return err
			}
		}
	}
	for _, f := range urq.ctx.Fields {
		if !userreport.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if urq.path != nil {
		prev, err := urq.path(ctx)
		if err != nil {
			return err
		}
		urq.sql = prev
	}
	return nil
}

func (urq *UserReportQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*UserReport, error) {
	var (
		nodes       = []*UserReport{}
		_spec       = urq.querySpec()
		loadedTypes = [3]bool{
			urq.withBusinessUnit != nil,
			urq.withOrganization != nil,
			urq.withUser != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*UserReport).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &UserReport{config: urq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(urq.modifiers) > 0 {
		_spec.Modifiers = urq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, urq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := urq.withBusinessUnit; query != nil {
		if err := urq.loadBusinessUnit(ctx, query, nodes, nil,
			func(n *UserReport, e *BusinessUnit) { n.Edges.BusinessUnit = e }); err != nil {
			return nil, err
		}
	}
	if query := urq.withOrganization; query != nil {
		if err := urq.loadOrganization(ctx, query, nodes, nil,
			func(n *UserReport, e *Organization) { n.Edges.Organization = e }); err != nil {
			return nil, err
		}
	}
	if query := urq.withUser; query != nil {
		if err := urq.loadUser(ctx, query, nodes, nil,
			func(n *UserReport, e *User) { n.Edges.User = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (urq *UserReportQuery) loadBusinessUnit(ctx context.Context, query *BusinessUnitQuery, nodes []*UserReport, init func(*UserReport), assign func(*UserReport, *BusinessUnit)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*UserReport)
	for i := range nodes {
		fk := nodes[i].BusinessUnitID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(businessunit.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "business_unit_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (urq *UserReportQuery) loadOrganization(ctx context.Context, query *OrganizationQuery, nodes []*UserReport, init func(*UserReport), assign func(*UserReport, *Organization)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*UserReport)
	for i := range nodes {
		fk := nodes[i].OrganizationID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(organization.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "organization_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (urq *UserReportQuery) loadUser(ctx context.Context, query *UserQuery, nodes []*UserReport, init func(*UserReport), assign func(*UserReport, *User)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*UserReport)
	for i := range nodes {
		fk := nodes[i].UserID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(user.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "user_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (urq *UserReportQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := urq.querySpec()
	if len(urq.modifiers) > 0 {
		_spec.Modifiers = urq.modifiers
	}
	_spec.Node.Columns = urq.ctx.Fields
	if len(urq.ctx.Fields) > 0 {
		_spec.Unique = urq.ctx.Unique != nil && *urq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, urq.driver, _spec)
}

func (urq *UserReportQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(userreport.Table, userreport.Columns, sqlgraph.NewFieldSpec(userreport.FieldID, field.TypeUUID))
	_spec.From = urq.sql
	if unique := urq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if urq.path != nil {
		_spec.Unique = true
	}
	if fields := urq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, userreport.FieldID)
		for i := range fields {
			if fields[i] != userreport.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if urq.withBusinessUnit != nil {
			_spec.Node.AddColumnOnce(userreport.FieldBusinessUnitID)
		}
		if urq.withOrganization != nil {
			_spec.Node.AddColumnOnce(userreport.FieldOrganizationID)
		}
		if urq.withUser != nil {
			_spec.Node.AddColumnOnce(userreport.FieldUserID)
		}
	}
	if ps := urq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := urq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := urq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := urq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (urq *UserReportQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(urq.driver.Dialect())
	t1 := builder.Table(userreport.Table)
	columns := urq.ctx.Fields
	if len(columns) == 0 {
		columns = userreport.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if urq.sql != nil {
		selector = urq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if urq.ctx.Unique != nil && *urq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range urq.modifiers {
		m(selector)
	}
	for _, p := range urq.predicates {
		p(selector)
	}
	for _, p := range urq.order {
		p(selector)
	}
	if offset := urq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := urq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// Modify adds a query modifier for attaching custom logic to queries.
func (urq *UserReportQuery) Modify(modifiers ...func(s *sql.Selector)) *UserReportSelect {
	urq.modifiers = append(urq.modifiers, modifiers...)
	return urq.Select()
}

// UserReportGroupBy is the group-by builder for UserReport entities.
type UserReportGroupBy struct {
	selector
	build *UserReportQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (urgb *UserReportGroupBy) Aggregate(fns ...AggregateFunc) *UserReportGroupBy {
	urgb.fns = append(urgb.fns, fns...)
	return urgb
}

// Scan applies the selector query and scans the result into the given value.
func (urgb *UserReportGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, urgb.build.ctx, "GroupBy")
	if err := urgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*UserReportQuery, *UserReportGroupBy](ctx, urgb.build, urgb, urgb.build.inters, v)
}

func (urgb *UserReportGroupBy) sqlScan(ctx context.Context, root *UserReportQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(urgb.fns))
	for _, fn := range urgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*urgb.flds)+len(urgb.fns))
		for _, f := range *urgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*urgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := urgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// UserReportSelect is the builder for selecting fields of UserReport entities.
type UserReportSelect struct {
	*UserReportQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (urs *UserReportSelect) Aggregate(fns ...AggregateFunc) *UserReportSelect {
	urs.fns = append(urs.fns, fns...)
	return urs
}

// Scan applies the selector query and scans the result into the given value.
func (urs *UserReportSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, urs.ctx, "Select")
	if err := urs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*UserReportQuery, *UserReportSelect](ctx, urs.UserReportQuery, urs, urs.inters, v)
}

func (urs *UserReportSelect) sqlScan(ctx context.Context, root *UserReportQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(urs.fns))
	for _, fn := range urs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*urs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := urs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// Modify adds a query modifier for attaching custom logic to queries.
func (urs *UserReportSelect) Modify(modifiers ...func(s *sql.Selector)) *UserReportSelect {
	urs.modifiers = append(urs.modifiers, modifiers...)
	return urs
}
