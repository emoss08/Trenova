// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/customreport"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

// CustomReportCreate is the builder for creating a CustomReport entity.
type CustomReportCreate struct {
	config
	mutation *CustomReportMutation
	hooks    []Hook
}

// SetBusinessUnitID sets the "business_unit_id" field.
func (crc *CustomReportCreate) SetBusinessUnitID(u uuid.UUID) *CustomReportCreate {
	crc.mutation.SetBusinessUnitID(u)
	return crc
}

// SetOrganizationID sets the "organization_id" field.
func (crc *CustomReportCreate) SetOrganizationID(u uuid.UUID) *CustomReportCreate {
	crc.mutation.SetOrganizationID(u)
	return crc
}

// SetCreatedAt sets the "created_at" field.
func (crc *CustomReportCreate) SetCreatedAt(t time.Time) *CustomReportCreate {
	crc.mutation.SetCreatedAt(t)
	return crc
}

// SetNillableCreatedAt sets the "created_at" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableCreatedAt(t *time.Time) *CustomReportCreate {
	if t != nil {
		crc.SetCreatedAt(*t)
	}
	return crc
}

// SetUpdatedAt sets the "updated_at" field.
func (crc *CustomReportCreate) SetUpdatedAt(t time.Time) *CustomReportCreate {
	crc.mutation.SetUpdatedAt(t)
	return crc
}

// SetNillableUpdatedAt sets the "updated_at" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableUpdatedAt(t *time.Time) *CustomReportCreate {
	if t != nil {
		crc.SetUpdatedAt(*t)
	}
	return crc
}

// SetVersion sets the "version" field.
func (crc *CustomReportCreate) SetVersion(i int) *CustomReportCreate {
	crc.mutation.SetVersion(i)
	return crc
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableVersion(i *int) *CustomReportCreate {
	if i != nil {
		crc.SetVersion(*i)
	}
	return crc
}

// SetName sets the "name" field.
func (crc *CustomReportCreate) SetName(s string) *CustomReportCreate {
	crc.mutation.SetName(s)
	return crc
}

// SetDescription sets the "description" field.
func (crc *CustomReportCreate) SetDescription(s string) *CustomReportCreate {
	crc.mutation.SetDescription(s)
	return crc
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableDescription(s *string) *CustomReportCreate {
	if s != nil {
		crc.SetDescription(*s)
	}
	return crc
}

// SetTable sets the "table" field.
func (crc *CustomReportCreate) SetTable(s string) *CustomReportCreate {
	crc.mutation.SetTable(s)
	return crc
}

// SetNillableTable sets the "table" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableTable(s *string) *CustomReportCreate {
	if s != nil {
		crc.SetTable(*s)
	}
	return crc
}

// SetID sets the "id" field.
func (crc *CustomReportCreate) SetID(u uuid.UUID) *CustomReportCreate {
	crc.mutation.SetID(u)
	return crc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (crc *CustomReportCreate) SetNillableID(u *uuid.UUID) *CustomReportCreate {
	if u != nil {
		crc.SetID(*u)
	}
	return crc
}

// SetBusinessUnit sets the "business_unit" edge to the BusinessUnit entity.
func (crc *CustomReportCreate) SetBusinessUnit(b *BusinessUnit) *CustomReportCreate {
	return crc.SetBusinessUnitID(b.ID)
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (crc *CustomReportCreate) SetOrganization(o *Organization) *CustomReportCreate {
	return crc.SetOrganizationID(o.ID)
}

// Mutation returns the CustomReportMutation object of the builder.
func (crc *CustomReportCreate) Mutation() *CustomReportMutation {
	return crc.mutation
}

// Save creates the CustomReport in the database.
func (crc *CustomReportCreate) Save(ctx context.Context) (*CustomReport, error) {
	crc.defaults()
	return withHooks(ctx, crc.sqlSave, crc.mutation, crc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (crc *CustomReportCreate) SaveX(ctx context.Context) *CustomReport {
	v, err := crc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (crc *CustomReportCreate) Exec(ctx context.Context) error {
	_, err := crc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (crc *CustomReportCreate) ExecX(ctx context.Context) {
	if err := crc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (crc *CustomReportCreate) defaults() {
	if _, ok := crc.mutation.CreatedAt(); !ok {
		v := customreport.DefaultCreatedAt()
		crc.mutation.SetCreatedAt(v)
	}
	if _, ok := crc.mutation.UpdatedAt(); !ok {
		v := customreport.DefaultUpdatedAt()
		crc.mutation.SetUpdatedAt(v)
	}
	if _, ok := crc.mutation.Version(); !ok {
		v := customreport.DefaultVersion
		crc.mutation.SetVersion(v)
	}
	if _, ok := crc.mutation.ID(); !ok {
		v := customreport.DefaultID()
		crc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (crc *CustomReportCreate) check() error {
	if _, ok := crc.mutation.BusinessUnitID(); !ok {
		return &ValidationError{Name: "business_unit_id", err: errors.New(`ent: missing required field "CustomReport.business_unit_id"`)}
	}
	if _, ok := crc.mutation.OrganizationID(); !ok {
		return &ValidationError{Name: "organization_id", err: errors.New(`ent: missing required field "CustomReport.organization_id"`)}
	}
	if _, ok := crc.mutation.CreatedAt(); !ok {
		return &ValidationError{Name: "created_at", err: errors.New(`ent: missing required field "CustomReport.created_at"`)}
	}
	if _, ok := crc.mutation.UpdatedAt(); !ok {
		return &ValidationError{Name: "updated_at", err: errors.New(`ent: missing required field "CustomReport.updated_at"`)}
	}
	if _, ok := crc.mutation.Version(); !ok {
		return &ValidationError{Name: "version", err: errors.New(`ent: missing required field "CustomReport.version"`)}
	}
	if _, ok := crc.mutation.Name(); !ok {
		return &ValidationError{Name: "name", err: errors.New(`ent: missing required field "CustomReport.name"`)}
	}
	if v, ok := crc.mutation.Name(); ok {
		if err := customreport.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "CustomReport.name": %w`, err)}
		}
	}
	if _, ok := crc.mutation.BusinessUnitID(); !ok {
		return &ValidationError{Name: "business_unit", err: errors.New(`ent: missing required edge "CustomReport.business_unit"`)}
	}
	if _, ok := crc.mutation.OrganizationID(); !ok {
		return &ValidationError{Name: "organization", err: errors.New(`ent: missing required edge "CustomReport.organization"`)}
	}
	return nil
}

func (crc *CustomReportCreate) sqlSave(ctx context.Context) (*CustomReport, error) {
	if err := crc.check(); err != nil {
		return nil, err
	}
	_node, _spec := crc.createSpec()
	if err := sqlgraph.CreateNode(ctx, crc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	crc.mutation.id = &_node.ID
	crc.mutation.done = true
	return _node, nil
}

func (crc *CustomReportCreate) createSpec() (*CustomReport, *sqlgraph.CreateSpec) {
	var (
		_node = &CustomReport{config: crc.config}
		_spec = sqlgraph.NewCreateSpec(customreport.Table, sqlgraph.NewFieldSpec(customreport.FieldID, field.TypeUUID))
	)
	if id, ok := crc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := crc.mutation.CreatedAt(); ok {
		_spec.SetField(customreport.FieldCreatedAt, field.TypeTime, value)
		_node.CreatedAt = value
	}
	if value, ok := crc.mutation.UpdatedAt(); ok {
		_spec.SetField(customreport.FieldUpdatedAt, field.TypeTime, value)
		_node.UpdatedAt = value
	}
	if value, ok := crc.mutation.Version(); ok {
		_spec.SetField(customreport.FieldVersion, field.TypeInt, value)
		_node.Version = value
	}
	if value, ok := crc.mutation.Name(); ok {
		_spec.SetField(customreport.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := crc.mutation.Description(); ok {
		_spec.SetField(customreport.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := crc.mutation.Table(); ok {
		_spec.SetField(customreport.FieldTable, field.TypeString, value)
		_node.Table = value
	}
	if nodes := crc.mutation.BusinessUnitIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customreport.BusinessUnitTable,
			Columns: []string{customreport.BusinessUnitColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(businessunit.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.BusinessUnitID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	if nodes := crc.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   customreport.OrganizationTable,
			Columns: []string{customreport.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.OrganizationID = nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// CustomReportCreateBulk is the builder for creating many CustomReport entities in bulk.
type CustomReportCreateBulk struct {
	config
	err      error
	builders []*CustomReportCreate
}

// Save creates the CustomReport entities in the database.
func (crcb *CustomReportCreateBulk) Save(ctx context.Context) ([]*CustomReport, error) {
	if crcb.err != nil {
		return nil, crcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(crcb.builders))
	nodes := make([]*CustomReport, len(crcb.builders))
	mutators := make([]Mutator, len(crcb.builders))
	for i := range crcb.builders {
		func(i int, root context.Context) {
			builder := crcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*CustomReportMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, crcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, crcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, crcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (crcb *CustomReportCreateBulk) SaveX(ctx context.Context) []*CustomReport {
	v, err := crcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (crcb *CustomReportCreateBulk) Exec(ctx context.Context) error {
	_, err := crcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (crcb *CustomReportCreateBulk) ExecX(ctx context.Context) {
	if err := crcb.Exec(ctx); err != nil {
		panic(err)
	}
}