// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/accessorialcharge"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/commodity"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/customerdetentionpolicy"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/emoss08/trenova/internal/ent/revenuecode"
	"github.com/emoss08/trenova/internal/ent/shipmenttype"
	"github.com/google/uuid"
)

// CustomerDetentionPolicyQuery is the builder for querying CustomerDetentionPolicy entities.
type CustomerDetentionPolicyQuery struct {
	config
	ctx                   *QueryContext
	order                 []customerdetentionpolicy.OrderOption
	inters                []Interceptor
	predicates            []predicate.CustomerDetentionPolicy
	withBusinessUnit      *BusinessUnitQuery
	withOrganization      *OrganizationQuery
	withCustomer          *CustomerQuery
	withCommodity         *CommodityQuery
	withRevenueCode       *RevenueCodeQuery
	withShipmentType      *ShipmentTypeQuery
	withAccessorialCharge *AccessorialChargeQuery
	modifiers             []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the CustomerDetentionPolicyQuery builder.
func (cdpq *CustomerDetentionPolicyQuery) Where(ps ...predicate.CustomerDetentionPolicy) *CustomerDetentionPolicyQuery {
	cdpq.predicates = append(cdpq.predicates, ps...)
	return cdpq
}

// Limit the number of records to be returned by this query.
func (cdpq *CustomerDetentionPolicyQuery) Limit(limit int) *CustomerDetentionPolicyQuery {
	cdpq.ctx.Limit = &limit
	return cdpq
}

// Offset to start from.
func (cdpq *CustomerDetentionPolicyQuery) Offset(offset int) *CustomerDetentionPolicyQuery {
	cdpq.ctx.Offset = &offset
	return cdpq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (cdpq *CustomerDetentionPolicyQuery) Unique(unique bool) *CustomerDetentionPolicyQuery {
	cdpq.ctx.Unique = &unique
	return cdpq
}

// Order specifies how the records should be ordered.
func (cdpq *CustomerDetentionPolicyQuery) Order(o ...customerdetentionpolicy.OrderOption) *CustomerDetentionPolicyQuery {
	cdpq.order = append(cdpq.order, o...)
	return cdpq
}

// QueryBusinessUnit chains the current query on the "business_unit" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryBusinessUnit() *BusinessUnitQuery {
	query := (&BusinessUnitClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(businessunit.Table, businessunit.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.BusinessUnitTable, customerdetentionpolicy.BusinessUnitColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryOrganization chains the current query on the "organization" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryOrganization() *OrganizationQuery {
	query := (&OrganizationClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(organization.Table, organization.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.OrganizationTable, customerdetentionpolicy.OrganizationColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryCustomer chains the current query on the "customer" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryCustomer() *CustomerQuery {
	query := (&CustomerClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(customer.Table, customer.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, customerdetentionpolicy.CustomerTable, customerdetentionpolicy.CustomerColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryCommodity chains the current query on the "commodity" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryCommodity() *CommodityQuery {
	query := (&CommodityClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(commodity.Table, commodity.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.CommodityTable, customerdetentionpolicy.CommodityColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRevenueCode chains the current query on the "revenue_code" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryRevenueCode() *RevenueCodeQuery {
	query := (&RevenueCodeClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(revenuecode.Table, revenuecode.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.RevenueCodeTable, customerdetentionpolicy.RevenueCodeColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryShipmentType chains the current query on the "shipment_type" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryShipmentType() *ShipmentTypeQuery {
	query := (&ShipmentTypeClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(shipmenttype.Table, shipmenttype.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.ShipmentTypeTable, customerdetentionpolicy.ShipmentTypeColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryAccessorialCharge chains the current query on the "accessorial_charge" edge.
func (cdpq *CustomerDetentionPolicyQuery) QueryAccessorialCharge() *AccessorialChargeQuery {
	query := (&AccessorialChargeClient{config: cdpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := cdpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := cdpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerdetentionpolicy.Table, customerdetentionpolicy.FieldID, selector),
			sqlgraph.To(accessorialcharge.Table, accessorialcharge.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerdetentionpolicy.AccessorialChargeTable, customerdetentionpolicy.AccessorialChargeColumn),
		)
		fromU = sqlgraph.SetNeighbors(cdpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first CustomerDetentionPolicy entity from the query.
// Returns a *NotFoundError when no CustomerDetentionPolicy was found.
func (cdpq *CustomerDetentionPolicyQuery) First(ctx context.Context) (*CustomerDetentionPolicy, error) {
	nodes, err := cdpq.Limit(1).All(setContextOp(ctx, cdpq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{customerdetentionpolicy.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) FirstX(ctx context.Context) *CustomerDetentionPolicy {
	node, err := cdpq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first CustomerDetentionPolicy ID from the query.
// Returns a *NotFoundError when no CustomerDetentionPolicy ID was found.
func (cdpq *CustomerDetentionPolicyQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = cdpq.Limit(1).IDs(setContextOp(ctx, cdpq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{customerdetentionpolicy.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := cdpq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single CustomerDetentionPolicy entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one CustomerDetentionPolicy entity is found.
// Returns a *NotFoundError when no CustomerDetentionPolicy entities are found.
func (cdpq *CustomerDetentionPolicyQuery) Only(ctx context.Context) (*CustomerDetentionPolicy, error) {
	nodes, err := cdpq.Limit(2).All(setContextOp(ctx, cdpq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{customerdetentionpolicy.Label}
	default:
		return nil, &NotSingularError{customerdetentionpolicy.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) OnlyX(ctx context.Context) *CustomerDetentionPolicy {
	node, err := cdpq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only CustomerDetentionPolicy ID in the query.
// Returns a *NotSingularError when more than one CustomerDetentionPolicy ID is found.
// Returns a *NotFoundError when no entities are found.
func (cdpq *CustomerDetentionPolicyQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = cdpq.Limit(2).IDs(setContextOp(ctx, cdpq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{customerdetentionpolicy.Label}
	default:
		err = &NotSingularError{customerdetentionpolicy.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := cdpq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of CustomerDetentionPolicies.
func (cdpq *CustomerDetentionPolicyQuery) All(ctx context.Context) ([]*CustomerDetentionPolicy, error) {
	ctx = setContextOp(ctx, cdpq.ctx, "All")
	if err := cdpq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*CustomerDetentionPolicy, *CustomerDetentionPolicyQuery]()
	return withInterceptors[[]*CustomerDetentionPolicy](ctx, cdpq, qr, cdpq.inters)
}

// AllX is like All, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) AllX(ctx context.Context) []*CustomerDetentionPolicy {
	nodes, err := cdpq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of CustomerDetentionPolicy IDs.
func (cdpq *CustomerDetentionPolicyQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if cdpq.ctx.Unique == nil && cdpq.path != nil {
		cdpq.Unique(true)
	}
	ctx = setContextOp(ctx, cdpq.ctx, "IDs")
	if err = cdpq.Select(customerdetentionpolicy.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := cdpq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (cdpq *CustomerDetentionPolicyQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, cdpq.ctx, "Count")
	if err := cdpq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, cdpq, querierCount[*CustomerDetentionPolicyQuery](), cdpq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) CountX(ctx context.Context) int {
	count, err := cdpq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (cdpq *CustomerDetentionPolicyQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, cdpq.ctx, "Exist")
	switch _, err := cdpq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (cdpq *CustomerDetentionPolicyQuery) ExistX(ctx context.Context) bool {
	exist, err := cdpq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the CustomerDetentionPolicyQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (cdpq *CustomerDetentionPolicyQuery) Clone() *CustomerDetentionPolicyQuery {
	if cdpq == nil {
		return nil
	}
	return &CustomerDetentionPolicyQuery{
		config:                cdpq.config,
		ctx:                   cdpq.ctx.Clone(),
		order:                 append([]customerdetentionpolicy.OrderOption{}, cdpq.order...),
		inters:                append([]Interceptor{}, cdpq.inters...),
		predicates:            append([]predicate.CustomerDetentionPolicy{}, cdpq.predicates...),
		withBusinessUnit:      cdpq.withBusinessUnit.Clone(),
		withOrganization:      cdpq.withOrganization.Clone(),
		withCustomer:          cdpq.withCustomer.Clone(),
		withCommodity:         cdpq.withCommodity.Clone(),
		withRevenueCode:       cdpq.withRevenueCode.Clone(),
		withShipmentType:      cdpq.withShipmentType.Clone(),
		withAccessorialCharge: cdpq.withAccessorialCharge.Clone(),
		// clone intermediate query.
		sql:  cdpq.sql.Clone(),
		path: cdpq.path,
	}
}

// WithBusinessUnit tells the query-builder to eager-load the nodes that are connected to
// the "business_unit" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithBusinessUnit(opts ...func(*BusinessUnitQuery)) *CustomerDetentionPolicyQuery {
	query := (&BusinessUnitClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withBusinessUnit = query
	return cdpq
}

// WithOrganization tells the query-builder to eager-load the nodes that are connected to
// the "organization" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithOrganization(opts ...func(*OrganizationQuery)) *CustomerDetentionPolicyQuery {
	query := (&OrganizationClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withOrganization = query
	return cdpq
}

// WithCustomer tells the query-builder to eager-load the nodes that are connected to
// the "customer" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithCustomer(opts ...func(*CustomerQuery)) *CustomerDetentionPolicyQuery {
	query := (&CustomerClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withCustomer = query
	return cdpq
}

// WithCommodity tells the query-builder to eager-load the nodes that are connected to
// the "commodity" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithCommodity(opts ...func(*CommodityQuery)) *CustomerDetentionPolicyQuery {
	query := (&CommodityClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withCommodity = query
	return cdpq
}

// WithRevenueCode tells the query-builder to eager-load the nodes that are connected to
// the "revenue_code" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithRevenueCode(opts ...func(*RevenueCodeQuery)) *CustomerDetentionPolicyQuery {
	query := (&RevenueCodeClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withRevenueCode = query
	return cdpq
}

// WithShipmentType tells the query-builder to eager-load the nodes that are connected to
// the "shipment_type" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithShipmentType(opts ...func(*ShipmentTypeQuery)) *CustomerDetentionPolicyQuery {
	query := (&ShipmentTypeClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withShipmentType = query
	return cdpq
}

// WithAccessorialCharge tells the query-builder to eager-load the nodes that are connected to
// the "accessorial_charge" edge. The optional arguments are used to configure the query builder of the edge.
func (cdpq *CustomerDetentionPolicyQuery) WithAccessorialCharge(opts ...func(*AccessorialChargeQuery)) *CustomerDetentionPolicyQuery {
	query := (&AccessorialChargeClient{config: cdpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	cdpq.withAccessorialCharge = query
	return cdpq
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
//	client.CustomerDetentionPolicy.Query().
//		GroupBy(customerdetentionpolicy.FieldBusinessUnitID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (cdpq *CustomerDetentionPolicyQuery) GroupBy(field string, fields ...string) *CustomerDetentionPolicyGroupBy {
	cdpq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &CustomerDetentionPolicyGroupBy{build: cdpq}
	grbuild.flds = &cdpq.ctx.Fields
	grbuild.label = customerdetentionpolicy.Label
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
//	client.CustomerDetentionPolicy.Query().
//		Select(customerdetentionpolicy.FieldBusinessUnitID).
//		Scan(ctx, &v)
func (cdpq *CustomerDetentionPolicyQuery) Select(fields ...string) *CustomerDetentionPolicySelect {
	cdpq.ctx.Fields = append(cdpq.ctx.Fields, fields...)
	sbuild := &CustomerDetentionPolicySelect{CustomerDetentionPolicyQuery: cdpq}
	sbuild.label = customerdetentionpolicy.Label
	sbuild.flds, sbuild.scan = &cdpq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a CustomerDetentionPolicySelect configured with the given aggregations.
func (cdpq *CustomerDetentionPolicyQuery) Aggregate(fns ...AggregateFunc) *CustomerDetentionPolicySelect {
	return cdpq.Select().Aggregate(fns...)
}

func (cdpq *CustomerDetentionPolicyQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range cdpq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, cdpq); err != nil {
				return err
			}
		}
	}
	for _, f := range cdpq.ctx.Fields {
		if !customerdetentionpolicy.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if cdpq.path != nil {
		prev, err := cdpq.path(ctx)
		if err != nil {
			return err
		}
		cdpq.sql = prev
	}
	return nil
}

func (cdpq *CustomerDetentionPolicyQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*CustomerDetentionPolicy, error) {
	var (
		nodes       = []*CustomerDetentionPolicy{}
		_spec       = cdpq.querySpec()
		loadedTypes = [7]bool{
			cdpq.withBusinessUnit != nil,
			cdpq.withOrganization != nil,
			cdpq.withCustomer != nil,
			cdpq.withCommodity != nil,
			cdpq.withRevenueCode != nil,
			cdpq.withShipmentType != nil,
			cdpq.withAccessorialCharge != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*CustomerDetentionPolicy).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &CustomerDetentionPolicy{config: cdpq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(cdpq.modifiers) > 0 {
		_spec.Modifiers = cdpq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, cdpq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := cdpq.withBusinessUnit; query != nil {
		if err := cdpq.loadBusinessUnit(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *BusinessUnit) { n.Edges.BusinessUnit = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withOrganization; query != nil {
		if err := cdpq.loadOrganization(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *Organization) { n.Edges.Organization = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withCustomer; query != nil {
		if err := cdpq.loadCustomer(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *Customer) { n.Edges.Customer = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withCommodity; query != nil {
		if err := cdpq.loadCommodity(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *Commodity) { n.Edges.Commodity = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withRevenueCode; query != nil {
		if err := cdpq.loadRevenueCode(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *RevenueCode) { n.Edges.RevenueCode = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withShipmentType; query != nil {
		if err := cdpq.loadShipmentType(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *ShipmentType) { n.Edges.ShipmentType = e }); err != nil {
			return nil, err
		}
	}
	if query := cdpq.withAccessorialCharge; query != nil {
		if err := cdpq.loadAccessorialCharge(ctx, query, nodes, nil,
			func(n *CustomerDetentionPolicy, e *AccessorialCharge) { n.Edges.AccessorialCharge = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (cdpq *CustomerDetentionPolicyQuery) loadBusinessUnit(ctx context.Context, query *BusinessUnitQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *BusinessUnit)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
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
func (cdpq *CustomerDetentionPolicyQuery) loadOrganization(ctx context.Context, query *OrganizationQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *Organization)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
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
func (cdpq *CustomerDetentionPolicyQuery) loadCustomer(ctx context.Context, query *CustomerQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *Customer)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
	for i := range nodes {
		fk := nodes[i].CustomerID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(customer.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "customer_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (cdpq *CustomerDetentionPolicyQuery) loadCommodity(ctx context.Context, query *CommodityQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *Commodity)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
	for i := range nodes {
		if nodes[i].CommodityID == nil {
			continue
		}
		fk := *nodes[i].CommodityID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(commodity.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "commodity_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (cdpq *CustomerDetentionPolicyQuery) loadRevenueCode(ctx context.Context, query *RevenueCodeQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *RevenueCode)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
	for i := range nodes {
		if nodes[i].RevenueCodeID == nil {
			continue
		}
		fk := *nodes[i].RevenueCodeID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(revenuecode.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "revenue_code_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (cdpq *CustomerDetentionPolicyQuery) loadShipmentType(ctx context.Context, query *ShipmentTypeQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *ShipmentType)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
	for i := range nodes {
		if nodes[i].ShipmentTypeID == nil {
			continue
		}
		fk := *nodes[i].ShipmentTypeID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(shipmenttype.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "shipment_type_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (cdpq *CustomerDetentionPolicyQuery) loadAccessorialCharge(ctx context.Context, query *AccessorialChargeQuery, nodes []*CustomerDetentionPolicy, init func(*CustomerDetentionPolicy), assign func(*CustomerDetentionPolicy, *AccessorialCharge)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerDetentionPolicy)
	for i := range nodes {
		if nodes[i].AccessorialChargeID == nil {
			continue
		}
		fk := *nodes[i].AccessorialChargeID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(accessorialcharge.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "accessorial_charge_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (cdpq *CustomerDetentionPolicyQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := cdpq.querySpec()
	if len(cdpq.modifiers) > 0 {
		_spec.Modifiers = cdpq.modifiers
	}
	_spec.Node.Columns = cdpq.ctx.Fields
	if len(cdpq.ctx.Fields) > 0 {
		_spec.Unique = cdpq.ctx.Unique != nil && *cdpq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, cdpq.driver, _spec)
}

func (cdpq *CustomerDetentionPolicyQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(customerdetentionpolicy.Table, customerdetentionpolicy.Columns, sqlgraph.NewFieldSpec(customerdetentionpolicy.FieldID, field.TypeUUID))
	_spec.From = cdpq.sql
	if unique := cdpq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if cdpq.path != nil {
		_spec.Unique = true
	}
	if fields := cdpq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, customerdetentionpolicy.FieldID)
		for i := range fields {
			if fields[i] != customerdetentionpolicy.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if cdpq.withBusinessUnit != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldBusinessUnitID)
		}
		if cdpq.withOrganization != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldOrganizationID)
		}
		if cdpq.withCustomer != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldCustomerID)
		}
		if cdpq.withCommodity != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldCommodityID)
		}
		if cdpq.withRevenueCode != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldRevenueCodeID)
		}
		if cdpq.withShipmentType != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldShipmentTypeID)
		}
		if cdpq.withAccessorialCharge != nil {
			_spec.Node.AddColumnOnce(customerdetentionpolicy.FieldAccessorialChargeID)
		}
	}
	if ps := cdpq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := cdpq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := cdpq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := cdpq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (cdpq *CustomerDetentionPolicyQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(cdpq.driver.Dialect())
	t1 := builder.Table(customerdetentionpolicy.Table)
	columns := cdpq.ctx.Fields
	if len(columns) == 0 {
		columns = customerdetentionpolicy.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if cdpq.sql != nil {
		selector = cdpq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if cdpq.ctx.Unique != nil && *cdpq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range cdpq.modifiers {
		m(selector)
	}
	for _, p := range cdpq.predicates {
		p(selector)
	}
	for _, p := range cdpq.order {
		p(selector)
	}
	if offset := cdpq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := cdpq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// Modify adds a query modifier for attaching custom logic to queries.
func (cdpq *CustomerDetentionPolicyQuery) Modify(modifiers ...func(s *sql.Selector)) *CustomerDetentionPolicySelect {
	cdpq.modifiers = append(cdpq.modifiers, modifiers...)
	return cdpq.Select()
}

// CustomerDetentionPolicyGroupBy is the group-by builder for CustomerDetentionPolicy entities.
type CustomerDetentionPolicyGroupBy struct {
	selector
	build *CustomerDetentionPolicyQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (cdpgb *CustomerDetentionPolicyGroupBy) Aggregate(fns ...AggregateFunc) *CustomerDetentionPolicyGroupBy {
	cdpgb.fns = append(cdpgb.fns, fns...)
	return cdpgb
}

// Scan applies the selector query and scans the result into the given value.
func (cdpgb *CustomerDetentionPolicyGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, cdpgb.build.ctx, "GroupBy")
	if err := cdpgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerDetentionPolicyQuery, *CustomerDetentionPolicyGroupBy](ctx, cdpgb.build, cdpgb, cdpgb.build.inters, v)
}

func (cdpgb *CustomerDetentionPolicyGroupBy) sqlScan(ctx context.Context, root *CustomerDetentionPolicyQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(cdpgb.fns))
	for _, fn := range cdpgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*cdpgb.flds)+len(cdpgb.fns))
		for _, f := range *cdpgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*cdpgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := cdpgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// CustomerDetentionPolicySelect is the builder for selecting fields of CustomerDetentionPolicy entities.
type CustomerDetentionPolicySelect struct {
	*CustomerDetentionPolicyQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (cdps *CustomerDetentionPolicySelect) Aggregate(fns ...AggregateFunc) *CustomerDetentionPolicySelect {
	cdps.fns = append(cdps.fns, fns...)
	return cdps
}

// Scan applies the selector query and scans the result into the given value.
func (cdps *CustomerDetentionPolicySelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, cdps.ctx, "Select")
	if err := cdps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerDetentionPolicyQuery, *CustomerDetentionPolicySelect](ctx, cdps.CustomerDetentionPolicyQuery, cdps, cdps.inters, v)
}

func (cdps *CustomerDetentionPolicySelect) sqlScan(ctx context.Context, root *CustomerDetentionPolicyQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(cdps.fns))
	for _, fn := range cdps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*cdps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := cdps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// Modify adds a query modifier for attaching custom logic to queries.
func (cdps *CustomerDetentionPolicySelect) Modify(modifiers ...func(s *sql.Selector)) *CustomerDetentionPolicySelect {
	cdps.modifiers = append(cdps.modifiers, modifiers...)
	return cdps
}