// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/ent/customerruleprofile"
	"github.com/emoss08/trenova/internal/ent/documentclassification"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/google/uuid"
)

// CustomerRuleProfileQuery is the builder for querying CustomerRuleProfile entities.
type CustomerRuleProfileQuery struct {
	config
	ctx                              *QueryContext
	order                            []customerruleprofile.OrderOption
	inters                           []Interceptor
	predicates                       []predicate.CustomerRuleProfile
	withBusinessUnit                 *BusinessUnitQuery
	withOrganization                 *OrganizationQuery
	withCustomer                     *CustomerQuery
	withDocumentClassifications      *DocumentClassificationQuery
	modifiers                        []func(*sql.Selector)
	withNamedDocumentClassifications map[string]*DocumentClassificationQuery
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the CustomerRuleProfileQuery builder.
func (crpq *CustomerRuleProfileQuery) Where(ps ...predicate.CustomerRuleProfile) *CustomerRuleProfileQuery {
	crpq.predicates = append(crpq.predicates, ps...)
	return crpq
}

// Limit the number of records to be returned by this query.
func (crpq *CustomerRuleProfileQuery) Limit(limit int) *CustomerRuleProfileQuery {
	crpq.ctx.Limit = &limit
	return crpq
}

// Offset to start from.
func (crpq *CustomerRuleProfileQuery) Offset(offset int) *CustomerRuleProfileQuery {
	crpq.ctx.Offset = &offset
	return crpq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (crpq *CustomerRuleProfileQuery) Unique(unique bool) *CustomerRuleProfileQuery {
	crpq.ctx.Unique = &unique
	return crpq
}

// Order specifies how the records should be ordered.
func (crpq *CustomerRuleProfileQuery) Order(o ...customerruleprofile.OrderOption) *CustomerRuleProfileQuery {
	crpq.order = append(crpq.order, o...)
	return crpq
}

// QueryBusinessUnit chains the current query on the "business_unit" edge.
func (crpq *CustomerRuleProfileQuery) QueryBusinessUnit() *BusinessUnitQuery {
	query := (&BusinessUnitClient{config: crpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := crpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := crpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerruleprofile.Table, customerruleprofile.FieldID, selector),
			sqlgraph.To(businessunit.Table, businessunit.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerruleprofile.BusinessUnitTable, customerruleprofile.BusinessUnitColumn),
		)
		fromU = sqlgraph.SetNeighbors(crpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryOrganization chains the current query on the "organization" edge.
func (crpq *CustomerRuleProfileQuery) QueryOrganization() *OrganizationQuery {
	query := (&OrganizationClient{config: crpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := crpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := crpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerruleprofile.Table, customerruleprofile.FieldID, selector),
			sqlgraph.To(organization.Table, organization.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, customerruleprofile.OrganizationTable, customerruleprofile.OrganizationColumn),
		)
		fromU = sqlgraph.SetNeighbors(crpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryCustomer chains the current query on the "customer" edge.
func (crpq *CustomerRuleProfileQuery) QueryCustomer() *CustomerQuery {
	query := (&CustomerClient{config: crpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := crpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := crpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerruleprofile.Table, customerruleprofile.FieldID, selector),
			sqlgraph.To(customer.Table, customer.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, customerruleprofile.CustomerTable, customerruleprofile.CustomerColumn),
		)
		fromU = sqlgraph.SetNeighbors(crpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryDocumentClassifications chains the current query on the "document_classifications" edge.
func (crpq *CustomerRuleProfileQuery) QueryDocumentClassifications() *DocumentClassificationQuery {
	query := (&DocumentClassificationClient{config: crpq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := crpq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := crpq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customerruleprofile.Table, customerruleprofile.FieldID, selector),
			sqlgraph.To(documentclassification.Table, documentclassification.FieldID),
			sqlgraph.Edge(sqlgraph.M2M, false, customerruleprofile.DocumentClassificationsTable, customerruleprofile.DocumentClassificationsPrimaryKey...),
		)
		fromU = sqlgraph.SetNeighbors(crpq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first CustomerRuleProfile entity from the query.
// Returns a *NotFoundError when no CustomerRuleProfile was found.
func (crpq *CustomerRuleProfileQuery) First(ctx context.Context) (*CustomerRuleProfile, error) {
	nodes, err := crpq.Limit(1).All(setContextOp(ctx, crpq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{customerruleprofile.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) FirstX(ctx context.Context) *CustomerRuleProfile {
	node, err := crpq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first CustomerRuleProfile ID from the query.
// Returns a *NotFoundError when no CustomerRuleProfile ID was found.
func (crpq *CustomerRuleProfileQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = crpq.Limit(1).IDs(setContextOp(ctx, crpq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{customerruleprofile.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := crpq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single CustomerRuleProfile entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one CustomerRuleProfile entity is found.
// Returns a *NotFoundError when no CustomerRuleProfile entities are found.
func (crpq *CustomerRuleProfileQuery) Only(ctx context.Context) (*CustomerRuleProfile, error) {
	nodes, err := crpq.Limit(2).All(setContextOp(ctx, crpq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{customerruleprofile.Label}
	default:
		return nil, &NotSingularError{customerruleprofile.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) OnlyX(ctx context.Context) *CustomerRuleProfile {
	node, err := crpq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only CustomerRuleProfile ID in the query.
// Returns a *NotSingularError when more than one CustomerRuleProfile ID is found.
// Returns a *NotFoundError when no entities are found.
func (crpq *CustomerRuleProfileQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = crpq.Limit(2).IDs(setContextOp(ctx, crpq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{customerruleprofile.Label}
	default:
		err = &NotSingularError{customerruleprofile.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := crpq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of CustomerRuleProfiles.
func (crpq *CustomerRuleProfileQuery) All(ctx context.Context) ([]*CustomerRuleProfile, error) {
	ctx = setContextOp(ctx, crpq.ctx, "All")
	if err := crpq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*CustomerRuleProfile, *CustomerRuleProfileQuery]()
	return withInterceptors[[]*CustomerRuleProfile](ctx, crpq, qr, crpq.inters)
}

// AllX is like All, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) AllX(ctx context.Context) []*CustomerRuleProfile {
	nodes, err := crpq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of CustomerRuleProfile IDs.
func (crpq *CustomerRuleProfileQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if crpq.ctx.Unique == nil && crpq.path != nil {
		crpq.Unique(true)
	}
	ctx = setContextOp(ctx, crpq.ctx, "IDs")
	if err = crpq.Select(customerruleprofile.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := crpq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (crpq *CustomerRuleProfileQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, crpq.ctx, "Count")
	if err := crpq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, crpq, querierCount[*CustomerRuleProfileQuery](), crpq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) CountX(ctx context.Context) int {
	count, err := crpq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (crpq *CustomerRuleProfileQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, crpq.ctx, "Exist")
	switch _, err := crpq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (crpq *CustomerRuleProfileQuery) ExistX(ctx context.Context) bool {
	exist, err := crpq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the CustomerRuleProfileQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (crpq *CustomerRuleProfileQuery) Clone() *CustomerRuleProfileQuery {
	if crpq == nil {
		return nil
	}
	return &CustomerRuleProfileQuery{
		config:                      crpq.config,
		ctx:                         crpq.ctx.Clone(),
		order:                       append([]customerruleprofile.OrderOption{}, crpq.order...),
		inters:                      append([]Interceptor{}, crpq.inters...),
		predicates:                  append([]predicate.CustomerRuleProfile{}, crpq.predicates...),
		withBusinessUnit:            crpq.withBusinessUnit.Clone(),
		withOrganization:            crpq.withOrganization.Clone(),
		withCustomer:                crpq.withCustomer.Clone(),
		withDocumentClassifications: crpq.withDocumentClassifications.Clone(),
		// clone intermediate query.
		sql:  crpq.sql.Clone(),
		path: crpq.path,
	}
}

// WithBusinessUnit tells the query-builder to eager-load the nodes that are connected to
// the "business_unit" edge. The optional arguments are used to configure the query builder of the edge.
func (crpq *CustomerRuleProfileQuery) WithBusinessUnit(opts ...func(*BusinessUnitQuery)) *CustomerRuleProfileQuery {
	query := (&BusinessUnitClient{config: crpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	crpq.withBusinessUnit = query
	return crpq
}

// WithOrganization tells the query-builder to eager-load the nodes that are connected to
// the "organization" edge. The optional arguments are used to configure the query builder of the edge.
func (crpq *CustomerRuleProfileQuery) WithOrganization(opts ...func(*OrganizationQuery)) *CustomerRuleProfileQuery {
	query := (&OrganizationClient{config: crpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	crpq.withOrganization = query
	return crpq
}

// WithCustomer tells the query-builder to eager-load the nodes that are connected to
// the "customer" edge. The optional arguments are used to configure the query builder of the edge.
func (crpq *CustomerRuleProfileQuery) WithCustomer(opts ...func(*CustomerQuery)) *CustomerRuleProfileQuery {
	query := (&CustomerClient{config: crpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	crpq.withCustomer = query
	return crpq
}

// WithDocumentClassifications tells the query-builder to eager-load the nodes that are connected to
// the "document_classifications" edge. The optional arguments are used to configure the query builder of the edge.
func (crpq *CustomerRuleProfileQuery) WithDocumentClassifications(opts ...func(*DocumentClassificationQuery)) *CustomerRuleProfileQuery {
	query := (&DocumentClassificationClient{config: crpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	crpq.withDocumentClassifications = query
	return crpq
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
//	client.CustomerRuleProfile.Query().
//		GroupBy(customerruleprofile.FieldBusinessUnitID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (crpq *CustomerRuleProfileQuery) GroupBy(field string, fields ...string) *CustomerRuleProfileGroupBy {
	crpq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &CustomerRuleProfileGroupBy{build: crpq}
	grbuild.flds = &crpq.ctx.Fields
	grbuild.label = customerruleprofile.Label
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
//	client.CustomerRuleProfile.Query().
//		Select(customerruleprofile.FieldBusinessUnitID).
//		Scan(ctx, &v)
func (crpq *CustomerRuleProfileQuery) Select(fields ...string) *CustomerRuleProfileSelect {
	crpq.ctx.Fields = append(crpq.ctx.Fields, fields...)
	sbuild := &CustomerRuleProfileSelect{CustomerRuleProfileQuery: crpq}
	sbuild.label = customerruleprofile.Label
	sbuild.flds, sbuild.scan = &crpq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a CustomerRuleProfileSelect configured with the given aggregations.
func (crpq *CustomerRuleProfileQuery) Aggregate(fns ...AggregateFunc) *CustomerRuleProfileSelect {
	return crpq.Select().Aggregate(fns...)
}

func (crpq *CustomerRuleProfileQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range crpq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, crpq); err != nil {
				return err
			}
		}
	}
	for _, f := range crpq.ctx.Fields {
		if !customerruleprofile.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if crpq.path != nil {
		prev, err := crpq.path(ctx)
		if err != nil {
			return err
		}
		crpq.sql = prev
	}
	return nil
}

func (crpq *CustomerRuleProfileQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*CustomerRuleProfile, error) {
	var (
		nodes       = []*CustomerRuleProfile{}
		_spec       = crpq.querySpec()
		loadedTypes = [4]bool{
			crpq.withBusinessUnit != nil,
			crpq.withOrganization != nil,
			crpq.withCustomer != nil,
			crpq.withDocumentClassifications != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*CustomerRuleProfile).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &CustomerRuleProfile{config: crpq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(crpq.modifiers) > 0 {
		_spec.Modifiers = crpq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, crpq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := crpq.withBusinessUnit; query != nil {
		if err := crpq.loadBusinessUnit(ctx, query, nodes, nil,
			func(n *CustomerRuleProfile, e *BusinessUnit) { n.Edges.BusinessUnit = e }); err != nil {
			return nil, err
		}
	}
	if query := crpq.withOrganization; query != nil {
		if err := crpq.loadOrganization(ctx, query, nodes, nil,
			func(n *CustomerRuleProfile, e *Organization) { n.Edges.Organization = e }); err != nil {
			return nil, err
		}
	}
	if query := crpq.withCustomer; query != nil {
		if err := crpq.loadCustomer(ctx, query, nodes, nil,
			func(n *CustomerRuleProfile, e *Customer) { n.Edges.Customer = e }); err != nil {
			return nil, err
		}
	}
	if query := crpq.withDocumentClassifications; query != nil {
		if err := crpq.loadDocumentClassifications(ctx, query, nodes,
			func(n *CustomerRuleProfile) { n.Edges.DocumentClassifications = []*DocumentClassification{} },
			func(n *CustomerRuleProfile, e *DocumentClassification) {
				n.Edges.DocumentClassifications = append(n.Edges.DocumentClassifications, e)
			}); err != nil {
			return nil, err
		}
	}
	for name, query := range crpq.withNamedDocumentClassifications {
		if err := crpq.loadDocumentClassifications(ctx, query, nodes,
			func(n *CustomerRuleProfile) { n.appendNamedDocumentClassifications(name) },
			func(n *CustomerRuleProfile, e *DocumentClassification) { n.appendNamedDocumentClassifications(name, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (crpq *CustomerRuleProfileQuery) loadBusinessUnit(ctx context.Context, query *BusinessUnitQuery, nodes []*CustomerRuleProfile, init func(*CustomerRuleProfile), assign func(*CustomerRuleProfile, *BusinessUnit)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerRuleProfile)
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
func (crpq *CustomerRuleProfileQuery) loadOrganization(ctx context.Context, query *OrganizationQuery, nodes []*CustomerRuleProfile, init func(*CustomerRuleProfile), assign func(*CustomerRuleProfile, *Organization)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerRuleProfile)
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
func (crpq *CustomerRuleProfileQuery) loadCustomer(ctx context.Context, query *CustomerQuery, nodes []*CustomerRuleProfile, init func(*CustomerRuleProfile), assign func(*CustomerRuleProfile, *Customer)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*CustomerRuleProfile)
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
func (crpq *CustomerRuleProfileQuery) loadDocumentClassifications(ctx context.Context, query *DocumentClassificationQuery, nodes []*CustomerRuleProfile, init func(*CustomerRuleProfile), assign func(*CustomerRuleProfile, *DocumentClassification)) error {
	edgeIDs := make([]driver.Value, len(nodes))
	byID := make(map[uuid.UUID]*CustomerRuleProfile)
	nids := make(map[uuid.UUID]map[*CustomerRuleProfile]struct{})
	for i, node := range nodes {
		edgeIDs[i] = node.ID
		byID[node.ID] = node
		if init != nil {
			init(node)
		}
	}
	query.Where(func(s *sql.Selector) {
		joinT := sql.Table(customerruleprofile.DocumentClassificationsTable)
		s.Join(joinT).On(s.C(documentclassification.FieldID), joinT.C(customerruleprofile.DocumentClassificationsPrimaryKey[1]))
		s.Where(sql.InValues(joinT.C(customerruleprofile.DocumentClassificationsPrimaryKey[0]), edgeIDs...))
		columns := s.SelectedColumns()
		s.Select(joinT.C(customerruleprofile.DocumentClassificationsPrimaryKey[0]))
		s.AppendSelect(columns...)
		s.SetDistinct(false)
	})
	if err := query.prepareQuery(ctx); err != nil {
		return err
	}
	qr := QuerierFunc(func(ctx context.Context, q Query) (Value, error) {
		return query.sqlAll(ctx, func(_ context.Context, spec *sqlgraph.QuerySpec) {
			assign := spec.Assign
			values := spec.ScanValues
			spec.ScanValues = func(columns []string) ([]any, error) {
				values, err := values(columns[1:])
				if err != nil {
					return nil, err
				}
				return append([]any{new(uuid.UUID)}, values...), nil
			}
			spec.Assign = func(columns []string, values []any) error {
				outValue := *values[0].(*uuid.UUID)
				inValue := *values[1].(*uuid.UUID)
				if nids[inValue] == nil {
					nids[inValue] = map[*CustomerRuleProfile]struct{}{byID[outValue]: {}}
					return assign(columns[1:], values[1:])
				}
				nids[inValue][byID[outValue]] = struct{}{}
				return nil
			}
		})
	})
	neighbors, err := withInterceptors[[]*DocumentClassification](ctx, query, qr, query.inters)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected "document_classifications" node returned %v`, n.ID)
		}
		for kn := range nodes {
			assign(kn, n)
		}
	}
	return nil
}

func (crpq *CustomerRuleProfileQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := crpq.querySpec()
	if len(crpq.modifiers) > 0 {
		_spec.Modifiers = crpq.modifiers
	}
	_spec.Node.Columns = crpq.ctx.Fields
	if len(crpq.ctx.Fields) > 0 {
		_spec.Unique = crpq.ctx.Unique != nil && *crpq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, crpq.driver, _spec)
}

func (crpq *CustomerRuleProfileQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(customerruleprofile.Table, customerruleprofile.Columns, sqlgraph.NewFieldSpec(customerruleprofile.FieldID, field.TypeUUID))
	_spec.From = crpq.sql
	if unique := crpq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if crpq.path != nil {
		_spec.Unique = true
	}
	if fields := crpq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, customerruleprofile.FieldID)
		for i := range fields {
			if fields[i] != customerruleprofile.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if crpq.withBusinessUnit != nil {
			_spec.Node.AddColumnOnce(customerruleprofile.FieldBusinessUnitID)
		}
		if crpq.withOrganization != nil {
			_spec.Node.AddColumnOnce(customerruleprofile.FieldOrganizationID)
		}
		if crpq.withCustomer != nil {
			_spec.Node.AddColumnOnce(customerruleprofile.FieldCustomerID)
		}
	}
	if ps := crpq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := crpq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := crpq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := crpq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (crpq *CustomerRuleProfileQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(crpq.driver.Dialect())
	t1 := builder.Table(customerruleprofile.Table)
	columns := crpq.ctx.Fields
	if len(columns) == 0 {
		columns = customerruleprofile.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if crpq.sql != nil {
		selector = crpq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if crpq.ctx.Unique != nil && *crpq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range crpq.modifiers {
		m(selector)
	}
	for _, p := range crpq.predicates {
		p(selector)
	}
	for _, p := range crpq.order {
		p(selector)
	}
	if offset := crpq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := crpq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// Modify adds a query modifier for attaching custom logic to queries.
func (crpq *CustomerRuleProfileQuery) Modify(modifiers ...func(s *sql.Selector)) *CustomerRuleProfileSelect {
	crpq.modifiers = append(crpq.modifiers, modifiers...)
	return crpq.Select()
}

// WithNamedDocumentClassifications tells the query-builder to eager-load the nodes that are connected to the "document_classifications"
// edge with the given name. The optional arguments are used to configure the query builder of the edge.
func (crpq *CustomerRuleProfileQuery) WithNamedDocumentClassifications(name string, opts ...func(*DocumentClassificationQuery)) *CustomerRuleProfileQuery {
	query := (&DocumentClassificationClient{config: crpq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	if crpq.withNamedDocumentClassifications == nil {
		crpq.withNamedDocumentClassifications = make(map[string]*DocumentClassificationQuery)
	}
	crpq.withNamedDocumentClassifications[name] = query
	return crpq
}

// CustomerRuleProfileGroupBy is the group-by builder for CustomerRuleProfile entities.
type CustomerRuleProfileGroupBy struct {
	selector
	build *CustomerRuleProfileQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (crpgb *CustomerRuleProfileGroupBy) Aggregate(fns ...AggregateFunc) *CustomerRuleProfileGroupBy {
	crpgb.fns = append(crpgb.fns, fns...)
	return crpgb
}

// Scan applies the selector query and scans the result into the given value.
func (crpgb *CustomerRuleProfileGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, crpgb.build.ctx, "GroupBy")
	if err := crpgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerRuleProfileQuery, *CustomerRuleProfileGroupBy](ctx, crpgb.build, crpgb, crpgb.build.inters, v)
}

func (crpgb *CustomerRuleProfileGroupBy) sqlScan(ctx context.Context, root *CustomerRuleProfileQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(crpgb.fns))
	for _, fn := range crpgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*crpgb.flds)+len(crpgb.fns))
		for _, f := range *crpgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*crpgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := crpgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// CustomerRuleProfileSelect is the builder for selecting fields of CustomerRuleProfile entities.
type CustomerRuleProfileSelect struct {
	*CustomerRuleProfileQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (crps *CustomerRuleProfileSelect) Aggregate(fns ...AggregateFunc) *CustomerRuleProfileSelect {
	crps.fns = append(crps.fns, fns...)
	return crps
}

// Scan applies the selector query and scans the result into the given value.
func (crps *CustomerRuleProfileSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, crps.ctx, "Select")
	if err := crps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerRuleProfileQuery, *CustomerRuleProfileSelect](ctx, crps.CustomerRuleProfileQuery, crps, crps.inters, v)
}

func (crps *CustomerRuleProfileSelect) sqlScan(ctx context.Context, root *CustomerRuleProfileQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(crps.fns))
	for _, fn := range crps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*crps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := crps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// Modify adds a query modifier for attaching custom logic to queries.
func (crps *CustomerRuleProfileSelect) Modify(modifiers ...func(s *sql.Selector)) *CustomerRuleProfileSelect {
	crps.modifiers = append(crps.modifiers, modifiers...)
	return crps
}
