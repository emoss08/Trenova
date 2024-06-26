// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/equipmentmanufactuer"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/predicate"
	"github.com/google/uuid"
)

// EquipmentManufactuerUpdate is the builder for updating EquipmentManufactuer entities.
type EquipmentManufactuerUpdate struct {
	config
	hooks     []Hook
	mutation  *EquipmentManufactuerMutation
	modifiers []func(*sql.UpdateBuilder)
}

// Where appends a list predicates to the EquipmentManufactuerUpdate builder.
func (emu *EquipmentManufactuerUpdate) Where(ps ...predicate.EquipmentManufactuer) *EquipmentManufactuerUpdate {
	emu.mutation.Where(ps...)
	return emu
}

// SetOrganizationID sets the "organization_id" field.
func (emu *EquipmentManufactuerUpdate) SetOrganizationID(u uuid.UUID) *EquipmentManufactuerUpdate {
	emu.mutation.SetOrganizationID(u)
	return emu
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (emu *EquipmentManufactuerUpdate) SetNillableOrganizationID(u *uuid.UUID) *EquipmentManufactuerUpdate {
	if u != nil {
		emu.SetOrganizationID(*u)
	}
	return emu
}

// SetUpdatedAt sets the "updated_at" field.
func (emu *EquipmentManufactuerUpdate) SetUpdatedAt(t time.Time) *EquipmentManufactuerUpdate {
	emu.mutation.SetUpdatedAt(t)
	return emu
}

// SetVersion sets the "version" field.
func (emu *EquipmentManufactuerUpdate) SetVersion(i int) *EquipmentManufactuerUpdate {
	emu.mutation.ResetVersion()
	emu.mutation.SetVersion(i)
	return emu
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (emu *EquipmentManufactuerUpdate) SetNillableVersion(i *int) *EquipmentManufactuerUpdate {
	if i != nil {
		emu.SetVersion(*i)
	}
	return emu
}

// AddVersion adds i to the "version" field.
func (emu *EquipmentManufactuerUpdate) AddVersion(i int) *EquipmentManufactuerUpdate {
	emu.mutation.AddVersion(i)
	return emu
}

// SetStatus sets the "status" field.
func (emu *EquipmentManufactuerUpdate) SetStatus(e equipmentmanufactuer.Status) *EquipmentManufactuerUpdate {
	emu.mutation.SetStatus(e)
	return emu
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (emu *EquipmentManufactuerUpdate) SetNillableStatus(e *equipmentmanufactuer.Status) *EquipmentManufactuerUpdate {
	if e != nil {
		emu.SetStatus(*e)
	}
	return emu
}

// SetName sets the "name" field.
func (emu *EquipmentManufactuerUpdate) SetName(s string) *EquipmentManufactuerUpdate {
	emu.mutation.SetName(s)
	return emu
}

// SetNillableName sets the "name" field if the given value is not nil.
func (emu *EquipmentManufactuerUpdate) SetNillableName(s *string) *EquipmentManufactuerUpdate {
	if s != nil {
		emu.SetName(*s)
	}
	return emu
}

// SetDescription sets the "description" field.
func (emu *EquipmentManufactuerUpdate) SetDescription(s string) *EquipmentManufactuerUpdate {
	emu.mutation.SetDescription(s)
	return emu
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (emu *EquipmentManufactuerUpdate) SetNillableDescription(s *string) *EquipmentManufactuerUpdate {
	if s != nil {
		emu.SetDescription(*s)
	}
	return emu
}

// ClearDescription clears the value of the "description" field.
func (emu *EquipmentManufactuerUpdate) ClearDescription() *EquipmentManufactuerUpdate {
	emu.mutation.ClearDescription()
	return emu
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (emu *EquipmentManufactuerUpdate) SetOrganization(o *Organization) *EquipmentManufactuerUpdate {
	return emu.SetOrganizationID(o.ID)
}

// Mutation returns the EquipmentManufactuerMutation object of the builder.
func (emu *EquipmentManufactuerUpdate) Mutation() *EquipmentManufactuerMutation {
	return emu.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (emu *EquipmentManufactuerUpdate) ClearOrganization() *EquipmentManufactuerUpdate {
	emu.mutation.ClearOrganization()
	return emu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (emu *EquipmentManufactuerUpdate) Save(ctx context.Context) (int, error) {
	emu.defaults()
	return withHooks(ctx, emu.sqlSave, emu.mutation, emu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (emu *EquipmentManufactuerUpdate) SaveX(ctx context.Context) int {
	affected, err := emu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (emu *EquipmentManufactuerUpdate) Exec(ctx context.Context) error {
	_, err := emu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (emu *EquipmentManufactuerUpdate) ExecX(ctx context.Context) {
	if err := emu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (emu *EquipmentManufactuerUpdate) defaults() {
	if _, ok := emu.mutation.UpdatedAt(); !ok {
		v := equipmentmanufactuer.UpdateDefaultUpdatedAt()
		emu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (emu *EquipmentManufactuerUpdate) check() error {
	if v, ok := emu.mutation.Status(); ok {
		if err := equipmentmanufactuer.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`ent: validator failed for field "EquipmentManufactuer.status": %w`, err)}
		}
	}
	if v, ok := emu.mutation.Name(); ok {
		if err := equipmentmanufactuer.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "EquipmentManufactuer.name": %w`, err)}
		}
	}
	if _, ok := emu.mutation.BusinessUnitID(); emu.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EquipmentManufactuer.business_unit"`)
	}
	if _, ok := emu.mutation.OrganizationID(); emu.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EquipmentManufactuer.organization"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (emu *EquipmentManufactuerUpdate) Modify(modifiers ...func(u *sql.UpdateBuilder)) *EquipmentManufactuerUpdate {
	emu.modifiers = append(emu.modifiers, modifiers...)
	return emu
}

func (emu *EquipmentManufactuerUpdate) sqlSave(ctx context.Context) (n int, err error) {
	if err := emu.check(); err != nil {
		return n, err
	}
	_spec := sqlgraph.NewUpdateSpec(equipmentmanufactuer.Table, equipmentmanufactuer.Columns, sqlgraph.NewFieldSpec(equipmentmanufactuer.FieldID, field.TypeUUID))
	if ps := emu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := emu.mutation.UpdatedAt(); ok {
		_spec.SetField(equipmentmanufactuer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := emu.mutation.Version(); ok {
		_spec.SetField(equipmentmanufactuer.FieldVersion, field.TypeInt, value)
	}
	if value, ok := emu.mutation.AddedVersion(); ok {
		_spec.AddField(equipmentmanufactuer.FieldVersion, field.TypeInt, value)
	}
	if value, ok := emu.mutation.Status(); ok {
		_spec.SetField(equipmentmanufactuer.FieldStatus, field.TypeEnum, value)
	}
	if value, ok := emu.mutation.Name(); ok {
		_spec.SetField(equipmentmanufactuer.FieldName, field.TypeString, value)
	}
	if value, ok := emu.mutation.Description(); ok {
		_spec.SetField(equipmentmanufactuer.FieldDescription, field.TypeString, value)
	}
	if emu.mutation.DescriptionCleared() {
		_spec.ClearField(equipmentmanufactuer.FieldDescription, field.TypeString)
	}
	if emu.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   equipmentmanufactuer.OrganizationTable,
			Columns: []string{equipmentmanufactuer.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := emu.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   equipmentmanufactuer.OrganizationTable,
			Columns: []string{equipmentmanufactuer.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(emu.modifiers...)
	if n, err = sqlgraph.UpdateNodes(ctx, emu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{equipmentmanufactuer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	emu.mutation.done = true
	return n, nil
}

// EquipmentManufactuerUpdateOne is the builder for updating a single EquipmentManufactuer entity.
type EquipmentManufactuerUpdateOne struct {
	config
	fields    []string
	hooks     []Hook
	mutation  *EquipmentManufactuerMutation
	modifiers []func(*sql.UpdateBuilder)
}

// SetOrganizationID sets the "organization_id" field.
func (emuo *EquipmentManufactuerUpdateOne) SetOrganizationID(u uuid.UUID) *EquipmentManufactuerUpdateOne {
	emuo.mutation.SetOrganizationID(u)
	return emuo
}

// SetNillableOrganizationID sets the "organization_id" field if the given value is not nil.
func (emuo *EquipmentManufactuerUpdateOne) SetNillableOrganizationID(u *uuid.UUID) *EquipmentManufactuerUpdateOne {
	if u != nil {
		emuo.SetOrganizationID(*u)
	}
	return emuo
}

// SetUpdatedAt sets the "updated_at" field.
func (emuo *EquipmentManufactuerUpdateOne) SetUpdatedAt(t time.Time) *EquipmentManufactuerUpdateOne {
	emuo.mutation.SetUpdatedAt(t)
	return emuo
}

// SetVersion sets the "version" field.
func (emuo *EquipmentManufactuerUpdateOne) SetVersion(i int) *EquipmentManufactuerUpdateOne {
	emuo.mutation.ResetVersion()
	emuo.mutation.SetVersion(i)
	return emuo
}

// SetNillableVersion sets the "version" field if the given value is not nil.
func (emuo *EquipmentManufactuerUpdateOne) SetNillableVersion(i *int) *EquipmentManufactuerUpdateOne {
	if i != nil {
		emuo.SetVersion(*i)
	}
	return emuo
}

// AddVersion adds i to the "version" field.
func (emuo *EquipmentManufactuerUpdateOne) AddVersion(i int) *EquipmentManufactuerUpdateOne {
	emuo.mutation.AddVersion(i)
	return emuo
}

// SetStatus sets the "status" field.
func (emuo *EquipmentManufactuerUpdateOne) SetStatus(e equipmentmanufactuer.Status) *EquipmentManufactuerUpdateOne {
	emuo.mutation.SetStatus(e)
	return emuo
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (emuo *EquipmentManufactuerUpdateOne) SetNillableStatus(e *equipmentmanufactuer.Status) *EquipmentManufactuerUpdateOne {
	if e != nil {
		emuo.SetStatus(*e)
	}
	return emuo
}

// SetName sets the "name" field.
func (emuo *EquipmentManufactuerUpdateOne) SetName(s string) *EquipmentManufactuerUpdateOne {
	emuo.mutation.SetName(s)
	return emuo
}

// SetNillableName sets the "name" field if the given value is not nil.
func (emuo *EquipmentManufactuerUpdateOne) SetNillableName(s *string) *EquipmentManufactuerUpdateOne {
	if s != nil {
		emuo.SetName(*s)
	}
	return emuo
}

// SetDescription sets the "description" field.
func (emuo *EquipmentManufactuerUpdateOne) SetDescription(s string) *EquipmentManufactuerUpdateOne {
	emuo.mutation.SetDescription(s)
	return emuo
}

// SetNillableDescription sets the "description" field if the given value is not nil.
func (emuo *EquipmentManufactuerUpdateOne) SetNillableDescription(s *string) *EquipmentManufactuerUpdateOne {
	if s != nil {
		emuo.SetDescription(*s)
	}
	return emuo
}

// ClearDescription clears the value of the "description" field.
func (emuo *EquipmentManufactuerUpdateOne) ClearDescription() *EquipmentManufactuerUpdateOne {
	emuo.mutation.ClearDescription()
	return emuo
}

// SetOrganization sets the "organization" edge to the Organization entity.
func (emuo *EquipmentManufactuerUpdateOne) SetOrganization(o *Organization) *EquipmentManufactuerUpdateOne {
	return emuo.SetOrganizationID(o.ID)
}

// Mutation returns the EquipmentManufactuerMutation object of the builder.
func (emuo *EquipmentManufactuerUpdateOne) Mutation() *EquipmentManufactuerMutation {
	return emuo.mutation
}

// ClearOrganization clears the "organization" edge to the Organization entity.
func (emuo *EquipmentManufactuerUpdateOne) ClearOrganization() *EquipmentManufactuerUpdateOne {
	emuo.mutation.ClearOrganization()
	return emuo
}

// Where appends a list predicates to the EquipmentManufactuerUpdate builder.
func (emuo *EquipmentManufactuerUpdateOne) Where(ps ...predicate.EquipmentManufactuer) *EquipmentManufactuerUpdateOne {
	emuo.mutation.Where(ps...)
	return emuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (emuo *EquipmentManufactuerUpdateOne) Select(field string, fields ...string) *EquipmentManufactuerUpdateOne {
	emuo.fields = append([]string{field}, fields...)
	return emuo
}

// Save executes the query and returns the updated EquipmentManufactuer entity.
func (emuo *EquipmentManufactuerUpdateOne) Save(ctx context.Context) (*EquipmentManufactuer, error) {
	emuo.defaults()
	return withHooks(ctx, emuo.sqlSave, emuo.mutation, emuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (emuo *EquipmentManufactuerUpdateOne) SaveX(ctx context.Context) *EquipmentManufactuer {
	node, err := emuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (emuo *EquipmentManufactuerUpdateOne) Exec(ctx context.Context) error {
	_, err := emuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (emuo *EquipmentManufactuerUpdateOne) ExecX(ctx context.Context) {
	if err := emuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (emuo *EquipmentManufactuerUpdateOne) defaults() {
	if _, ok := emuo.mutation.UpdatedAt(); !ok {
		v := equipmentmanufactuer.UpdateDefaultUpdatedAt()
		emuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (emuo *EquipmentManufactuerUpdateOne) check() error {
	if v, ok := emuo.mutation.Status(); ok {
		if err := equipmentmanufactuer.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`ent: validator failed for field "EquipmentManufactuer.status": %w`, err)}
		}
	}
	if v, ok := emuo.mutation.Name(); ok {
		if err := equipmentmanufactuer.NameValidator(v); err != nil {
			return &ValidationError{Name: "name", err: fmt.Errorf(`ent: validator failed for field "EquipmentManufactuer.name": %w`, err)}
		}
	}
	if _, ok := emuo.mutation.BusinessUnitID(); emuo.mutation.BusinessUnitCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EquipmentManufactuer.business_unit"`)
	}
	if _, ok := emuo.mutation.OrganizationID(); emuo.mutation.OrganizationCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "EquipmentManufactuer.organization"`)
	}
	return nil
}

// Modify adds a statement modifier for attaching custom logic to the UPDATE statement.
func (emuo *EquipmentManufactuerUpdateOne) Modify(modifiers ...func(u *sql.UpdateBuilder)) *EquipmentManufactuerUpdateOne {
	emuo.modifiers = append(emuo.modifiers, modifiers...)
	return emuo
}

func (emuo *EquipmentManufactuerUpdateOne) sqlSave(ctx context.Context) (_node *EquipmentManufactuer, err error) {
	if err := emuo.check(); err != nil {
		return _node, err
	}
	_spec := sqlgraph.NewUpdateSpec(equipmentmanufactuer.Table, equipmentmanufactuer.Columns, sqlgraph.NewFieldSpec(equipmentmanufactuer.FieldID, field.TypeUUID))
	id, ok := emuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "EquipmentManufactuer.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := emuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, equipmentmanufactuer.FieldID)
		for _, f := range fields {
			if !equipmentmanufactuer.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != equipmentmanufactuer.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := emuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := emuo.mutation.UpdatedAt(); ok {
		_spec.SetField(equipmentmanufactuer.FieldUpdatedAt, field.TypeTime, value)
	}
	if value, ok := emuo.mutation.Version(); ok {
		_spec.SetField(equipmentmanufactuer.FieldVersion, field.TypeInt, value)
	}
	if value, ok := emuo.mutation.AddedVersion(); ok {
		_spec.AddField(equipmentmanufactuer.FieldVersion, field.TypeInt, value)
	}
	if value, ok := emuo.mutation.Status(); ok {
		_spec.SetField(equipmentmanufactuer.FieldStatus, field.TypeEnum, value)
	}
	if value, ok := emuo.mutation.Name(); ok {
		_spec.SetField(equipmentmanufactuer.FieldName, field.TypeString, value)
	}
	if value, ok := emuo.mutation.Description(); ok {
		_spec.SetField(equipmentmanufactuer.FieldDescription, field.TypeString, value)
	}
	if emuo.mutation.DescriptionCleared() {
		_spec.ClearField(equipmentmanufactuer.FieldDescription, field.TypeString)
	}
	if emuo.mutation.OrganizationCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   equipmentmanufactuer.OrganizationTable,
			Columns: []string{equipmentmanufactuer.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := emuo.mutation.OrganizationIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: false,
			Table:   equipmentmanufactuer.OrganizationTable,
			Columns: []string{equipmentmanufactuer.OrganizationColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(organization.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_spec.AddModifiers(emuo.modifiers...)
	_node = &EquipmentManufactuer{config: emuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, emuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{equipmentmanufactuer.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	emuo.mutation.done = true
	return _node, nil
}
